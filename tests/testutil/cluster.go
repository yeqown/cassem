package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type Cluster struct {
	RepoRoot      string
	Network       string
	DBImage       string
	AdmImage      string
	AgentImage    string
	DBEndpoints   []string
	AdmBaseURL    string
	AgentEndpoint string
	containers    []string
}

func StartCluster(t testing.TB) *Cluster {
	t.Helper()
	return StartFullCluster(t)
}

func startCluster(t testing.TB, withAdm bool, withAgent bool) *Cluster {
	t.Helper()
	RequirePodman(t)

	repoRoot := RepoRoot(t)
	prefix := UniqueName("cassem-it")
	cluster := &Cluster{
		RepoRoot:   repoRoot,
		Network:    prefix,
		DBImage:    "localhost/cassemdb:" + prefix,
		AdmImage:   "localhost/cassemadm:" + prefix,
		AgentImage: "localhost/cassemagent:" + prefix,
	}

	CreateNetwork(t, cluster.Network)
	t.Cleanup(func() {
		for i := len(cluster.containers) - 1; i >= 0; i-- {
			logs := ContainerLogs(t, cluster.containers[i])
			if t.Failed() {
				t.Logf("container %s logs:\n%s", cluster.containers[i], logs)
			}
			StopAndRemoveContainer(t, cluster.containers[i])
		}
		RemoveNetwork(t, cluster.Network)
	})

	BuildBinary(t, repoRoot, filepath.Join(repoRoot, "cassemdb"), "./cmd/cassemdb")
	BuildImage(t, repoRoot, cluster.DBImage, "./.deploy/dockerfiles/cassemdb.Dockerfile")
	if withAdm {
		BuildBinary(t, repoRoot, filepath.Join(repoRoot, "cassemadm"), "./cmd/cassemadm")
		BuildImage(t, repoRoot, cluster.AdmImage, "./.deploy/dockerfiles/cassemadm.Dockerfile")
	}
	if withAgent {
		BuildBinary(t, repoRoot, filepath.Join(repoRoot, "cassemagent"), "./cmd/cassemagent")
		BuildImage(t, repoRoot, cluster.AgentImage, "./.deploy/dockerfiles/cassemagent.Dockerfile")
	}

	tmp := t.TempDir()
	dbPorts := []int{FreePort(t), FreePort(t), FreePort(t)}
	admPort := FreePort(t)
	agentPort := FreePort(t)

	raftCluster := "http://cassemdb1:3021,http://cassemdb2:3022,http://cassemdb3:3023"
	for i := 0; i < 3; i++ {
		idx := i + 1
		name := fmt.Sprintf("cassemdb%d", idx)
		confDir := filepath.Join(tmp, name, "configs")
		storageDir := filepath.Join(tmp, name, "storage")
		mustMkdir(t, confDir)
		mustMkdir(t, storageDir)
		writeFile(t, filepath.Join(confDir, "cassemdb.toml"), fmt.Sprintf(`debug = false
addr = ":2021"
[bolt]
    db = "cassemdb.kv"
[raft]
    bind = "http://%s:302%d"
    cluster = "%s"
    snapCount = 300
`, name, idx, raftCluster))

		id := RunContainer(t, ContainerOptions{
			Name:     prefix + "-" + name,
			Hostname: name,
			Alias:    name,
			Image:    cluster.DBImage,
			Network:  cluster.Network,
			Ports: map[string]string{
				fmt.Sprintf("127.0.0.1:%d", dbPorts[i]): "2021",
			},
			Volumes: map[string]string{
				confDir:    "/app/cassemdb/configs:Z",
				storageDir: "/app/cassemdb/storage:Z",
			},
			Args: []string{
				"./cassemdb",
				"--conf", "./configs/cassemdb.toml",
				"--endpoint", ":2021",
				"--raft.cluster", raftCluster,
				"--raft.bind", fmt.Sprintf("http://%s:302%d", name, idx),
				"--storage", "./storage",
			},
		})
		cluster.containers = append(cluster.containers, id)
		cluster.DBEndpoints = append(cluster.DBEndpoints, fmt.Sprintf("127.0.0.1:%d", dbPorts[i]))
	}

	for _, port := range dbPorts {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := WaitTCP(ctx, "127.0.0.1", port); err != nil {
			cancel()
			t.Fatalf("wait cassemdb tcp: %v", err)
		}
		cancel()
	}
	WaitCassemDB(t, cluster.DBEndpoints)

	if withAdm {
		admConfDir := filepath.Join(tmp, "cassemadm", "configs")
		mustMkdir(t, admConfDir)
		writeFile(t, filepath.Join(admConfDir, "cassemadm.toml"), `debug = false
cassemdb = [
    "cassemdb1:2021",
    "cassemdb2:2021",
    "cassemdb3:2021"
]
[http]
    addr = ":20218"
`)
		admID := RunContainer(t, ContainerOptions{
			Name:    prefix + "-cassemadm",
			Image:   cluster.AdmImage,
			Network: cluster.Network,
			Ports: map[string]string{
				fmt.Sprintf("127.0.0.1:%d", admPort): "20218",
			},
			Volumes: map[string]string{
				admConfDir: "/app/cassemadm/configs:Z",
			},
		})
		cluster.containers = append(cluster.containers, admID)
		cluster.AdmBaseURL = fmt.Sprintf("http://127.0.0.1:%d", admPort)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := WaitTCP(ctx, "127.0.0.1", admPort); err != nil {
			cancel()
			t.Fatalf("wait cassemadm tcp: %v", err)
		}
		cancel()
	}

	if withAgent {
		agentConfDir := filepath.Join(tmp, "cassemagent", "configs")
		mustMkdir(t, agentConfDir)
		writeFile(t, filepath.Join(agentConfDir, "cassemagent.toml"), `debug = false
ttl = 30
renewInterval = 20
elementCacheSize = 1000
cassemdb = [
    "cassemdb1:2021",
    "cassemdb2:2021",
    "cassemdb3:2021"
]
[server]
    addr = ":20219"
`)
		agentID := RunContainer(t, ContainerOptions{
			Name:    prefix + "-cassemagent",
			Image:   cluster.AgentImage,
			Network: cluster.Network,
			Ports: map[string]string{
				fmt.Sprintf("127.0.0.1:%d", agentPort): "20219",
			},
			Volumes: map[string]string{
				agentConfDir: "/app/cassemagent/configs:Z",
			},
		})
		cluster.containers = append(cluster.containers, agentID)
		cluster.AgentEndpoint = fmt.Sprintf("127.0.0.1:%d", agentPort)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := WaitTCP(ctx, "127.0.0.1", agentPort); err != nil {
			cancel()
			t.Fatalf("wait cassemagent tcp: %v", err)
		}
		cancel()
	}

	return cluster
}

func mustMkdir(t testing.TB, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create directory %s: %v", dir, err)
	}
}

func writeFile(t testing.TB, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
