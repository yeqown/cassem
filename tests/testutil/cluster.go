package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Cluster groups the shared endpoints used by integration tests.
type Cluster struct {
	DBEndpoints   []string
	AdmBaseURL    string
	AgentEndpoint string
}

// TestDBCluster returns the DB endpoints from the integration environment or the local defaults.
func TestDBCluster() *Cluster {
	env, err := LoadIntegrationEnv()
	if err != nil {
		return &Cluster{DBEndpoints: defaultDBEndpoints()}
	}

	return &Cluster{DBEndpoints: append([]string(nil), env.DBGRPCAddrs...)}
}

// TestAdmCluster returns the admin endpoint from the integration environment or the local default.
func TestAdmCluster() *Cluster {
	env, err := LoadIntegrationEnv()
	cluster := TestDBCluster()
	if err != nil {
		cluster.AdmBaseURL = defaultAdmHTTPAddr
		return cluster
	}

	cluster.AdmBaseURL = env.AdmHTTPAddr
	return cluster
}

// TestFullCluster returns the full cluster endpoints from the integration environment or the local defaults.
func TestFullCluster() *Cluster {
	env, err := LoadIntegrationEnv()
	cluster := TestAdmCluster()
	if err != nil {
		cluster.AgentEndpoint = defaultAgentGRPCAddr
		return cluster
	}

	cluster.AgentEndpoint = env.AgentGRPCAddr
	return cluster
}

// UseDBCluster returns a ready DB cluster or skips when the cluster is only available locally.
func UseDBCluster(t testing.TB) *Cluster {
	t.Helper()
	return useCluster(t, "cassemkv cluster", "cassemkv cluster", readyDBCluster)
}

// UseAdmCluster returns a ready admin cluster or skips when the cluster is only available locally.
func UseAdmCluster(t testing.TB) *Cluster {
	t.Helper()
	return useCluster(t, "cassemadm cluster", "cassemadm cluster", readyAdmCluster)
}

// UseFullCluster returns a ready full cluster or skips when the cluster is only available locally.
func UseFullCluster(t testing.TB) *Cluster {
	t.Helper()
	return useCluster(t, "cassem full cluster", "cassem full cluster", readyFullCluster)
}

// RequireDBCluster returns a ready DB cluster and fails immediately on errors.
func RequireDBCluster(t testing.TB) *Cluster {
	t.Helper()
	return requireCluster(t, "cassemkv cluster", readyDBCluster)
}

// RequireAdmCluster returns a ready admin cluster and fails immediately on errors.
func RequireAdmCluster(t testing.TB) *Cluster {
	t.Helper()
	return requireCluster(t, "cassemadm cluster", readyAdmCluster)
}

// RequireFullCluster returns a ready full cluster and fails immediately on errors.
func RequireFullCluster(t testing.TB) *Cluster {
	t.Helper()
	return requireCluster(t, "cassem full cluster", readyFullCluster)
}

// StartCluster preserves the legacy full-cluster entrypoint.
func StartCluster(t testing.TB) *Cluster {
	t.Helper()
	return UseFullCluster(t)
}

func useCluster(t testing.TB, skipName string, strictName string, ready func(IntegrationEnv) (*Cluster, error)) *Cluster {
	t.Helper()

	env, err := LoadIntegrationEnv()
	if err != nil {
		if integrationStrictEnabled() {
			t.Fatalf("load integration environment: %v", err)
		}
		t.Skipf("integration environment is not ready: %v; run make cluster.start", err)
		return nil
	}

	cluster, err := ready(env)
	if err == nil {
		return cluster
	}

	if env.Strict {
		t.Fatalf("%s is not ready: %v; run make cluster.start", strictName, err)
	}
	t.Skipf("external %s is not ready: %v; run make cluster.start", skipName, err)
	return nil
}

func requireCluster(t testing.TB, name string, ready func(IntegrationEnv) (*Cluster, error)) *Cluster {
	t.Helper()

	env, err := LoadIntegrationEnv()
	if err != nil {
		t.Fatalf("load integration environment: %v", err)
	}

	cluster, err := ready(env)
	if err != nil {
		t.Fatalf("%s is not ready: %v; run make cluster.start", name, err)
	}
	return cluster
}

func readyDBCluster(env IntegrationEnv) (*Cluster, error) {
	cluster := &Cluster{DBEndpoints: append([]string(nil), env.DBGRPCAddrs...)}
	if err := CheckCassemKV(cluster.DBEndpoints, env.WaitTimeout); err != nil {
		return nil, err
	}
	return cluster, nil
}

func readyAdmCluster(env IntegrationEnv) (*Cluster, error) {
	cluster := &Cluster{DBEndpoints: append([]string(nil), env.DBGRPCAddrs...), AdmBaseURL: env.AdmHTTPAddr}
	if err := CheckCassemKV(cluster.DBEndpoints, env.WaitTimeout); err != nil {
		return nil, fmt.Errorf("db readiness before adm: %w", err)
	}
	if err := CheckCassemAdm(cluster.AdmBaseURL, env.AdminEmail, env.AdminPassword, env.WaitTimeout, env.PollInterval); err != nil {
		return nil, err
	}
	return cluster, nil
}

func readyFullCluster(env IntegrationEnv) (*Cluster, error) {
	cluster := &Cluster{DBEndpoints: append([]string(nil), env.DBGRPCAddrs...), AdmBaseURL: env.AdmHTTPAddr, AgentEndpoint: env.AgentGRPCAddr}
	if err := CheckCassemKV(cluster.DBEndpoints, env.WaitTimeout); err != nil {
		return nil, fmt.Errorf("db readiness before full cluster: %w", err)
	}
	if err := CheckCassemAdm(cluster.AdmBaseURL, env.AdminEmail, env.AdminPassword, env.WaitTimeout, env.PollInterval); err != nil {
		return nil, fmt.Errorf("adm readiness before full cluster: %w", err)
	}
	if err := CheckCassemAgent(cluster.AgentEndpoint, env.WaitTimeout, env.PollInterval); err != nil {
		return nil, err
	}
	return cluster, nil
}

func defaultDBEndpoints() []string {
	parts := strings.Split(defaultDBGRPCAddrs, ",")
	endpoints := make([]string, 0, len(parts))
	for _, part := range parts {
		endpoint := strings.TrimSpace(part)
		if endpoint != "" {
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func integrationStrictEnabled() bool {
	return os.Getenv("CASSEM_INTEGRATION_STRICT") == "1"
}
