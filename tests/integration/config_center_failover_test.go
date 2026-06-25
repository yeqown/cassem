//go:build integration

package integration_test

import (
	"context"
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/tests/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestConfigCenterReleaseGate_RaftLeaderFailoverPreservesWriteAvailability(t *testing.T) {
	cluster := testutil.RequireFullCluster(t)
	scope := testutil.NewRunScope(t, "raft leader failover preserves write availability")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("risk-service")
	env := "production"
	key := scope.Key("fraud-scoring-model-router")
	v1 := `{"model":"stable-v1","traffic":100}`
	v2 := `{"model":"stable-v2","traffic":100}`

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, v1)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	reader, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(scope.ClientID("risk-reader")), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(reader.Quit)
	waitAgentRaw(t, reader, app, env, key, v1)

	leaderEndpoint, service := currentRaftLeader(t, cluster.DBEndpoints)
	t.Logf("stopping raft leader %s service=%s", leaderEndpoint, service)
	compose(t, "stop", service)
	t.Cleanup(func() {
		compose(t, "start", service)
		testutil.RequireEventually(t, 90*time.Second, time.Second, func() (bool, string) {
			if err := testDBWrite(cluster.DBEndpoints); err != nil {
				return false, err.Error()
			}
			return true, ""
		})
	})

	testutil.RequireEventually(t, 90*time.Second, time.Second, func() (bool, string) {
		if err := testDBWrite(cluster.DBEndpoints); err != nil {
			return false, err.Error()
		}
		return true, ""
	})

	updateElementRaw(t, adm, app, env, key, v2)
	publishReleaseGateElement(t, adm, app, env, key, 2, concept.PublishMode_FULL, nil)
	waitAgentRaw(t, reader, app, env, key, v2)
}

func currentRaftLeader(t testing.TB, endpoints []string) (string, string) {
	t.Helper()
	serviceByEndpoint := map[string]string{
		"127.0.0.1:2021": "cassemkv1",
		"127.0.0.1:2022": "cassemkv2",
		"127.0.0.1:2023": "cassemkv3",
	}
	for _, endpoint := range endpoints {
		cc, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			continue
		}
		client := healthpb.NewHealthClient(cc)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "cassemkv.RaftLeader"})
		cancel()
		_ = cc.Close()
		if err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING {
			service := serviceByEndpoint[endpoint]
			if service == "" {
				t.Fatalf("no compose service mapping for leader endpoint %s", endpoint)
			}
			return endpoint, service
		}
	}
	t.Fatalf("raft leader was not found from endpoints %v", endpoints)
	return "", ""
}

func compose(t testing.TB, args ...string) {
	t.Helper()
	root := testutil.RepoRoot(t)
	tool := os.Getenv("CONTAINER_TOOL")
	if tool == "" {
		tool = "podman"
	}
	imageTag := os.Getenv("IMAGE_TAG")
	if imageTag == "" {
		imageTag = "latest"
	}

	cmdArgs := []string{
		"IMAGE_TAG=" + imageTag,
		"CASSEM_EXAMPLES_DIR=" + filepath.Join(root, "examples"),
		tool,
		"compose",
		"-p",
		"cassem",
		"-f",
		"examples/compose.cluster.yaml",
	}
	cmdArgs = append(cmdArgs, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := testutil.Run(ctx, root, "env", cmdArgs...)
	if err != nil {
		t.Fatalf("compose %v: %v\n%s", args, err, out)
	}
}

func testDBWrite(endpoints []string) error {
	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
	if err != nil {
		return err
	}
	defer cc.Close()
	client := apikv.NewKVClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.SetKV(ctx, &apikv.SetKVReq{
		Key:       fmt.Sprintf("tests/failover/%d", time.Now().UnixNano()),
		Val:       []byte("ok"),
		Overwrite: true,
	})
	return err
}
