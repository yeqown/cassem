package testutil

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadIntegrationEnvUsesDefaults(t *testing.T) {
	clearIntegrationEnv(t)

	env, err := LoadIntegrationEnv()

	require.NoError(t, err)
	require.False(t, env.Strict)
	require.Equal(t, "http://127.0.0.1:20218", env.AdmHTTPAddr)
	require.Equal(t, "127.0.0.1:20219", env.AgentGRPCAddr)
	require.Equal(t, []string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}, env.DBGRPCAddrs)
	require.Equal(t, "superadmin@example.com", env.AdminEmail)
	require.Equal(t, "cassem", env.AdminPassword)
	require.Equal(t, 90*time.Second, env.WaitTimeout)
	require.Equal(t, time.Second, env.PollInterval)
}

func TestLoadIntegrationEnvParsesOverrides(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv("CASSEM_INTEGRATION_STRICT", "1")
	t.Setenv("CASSEM_ADM_HTTP_ADDR", "http://127.0.0.1:30218")
	t.Setenv("CASSEM_AGENT_GRPC_ADDR", "127.0.0.1:30219")
	t.Setenv("CASSEM_DB_GRPC_ADDRS", "127.0.0.1:3021,127.0.0.1:3022")
	t.Setenv("CASSEM_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("CASSEM_ADMIN_PASSWORD", "secret")
	t.Setenv("CASSEM_INTEGRATION_WAIT_TIMEOUT", "3s")
	t.Setenv("CASSEM_INTEGRATION_POLL_INTERVAL", "250ms")

	env, err := LoadIntegrationEnv()

	require.NoError(t, err)
	require.True(t, env.Strict)
	require.Equal(t, "http://127.0.0.1:30218", env.AdmHTTPAddr)
	require.Equal(t, "127.0.0.1:30219", env.AgentGRPCAddr)
	require.Equal(t, []string{"127.0.0.1:3021", "127.0.0.1:3022"}, env.DBGRPCAddrs)
	require.Equal(t, "admin@example.com", env.AdminEmail)
	require.Equal(t, "secret", env.AdminPassword)
	require.Equal(t, 3*time.Second, env.WaitTimeout)
	require.Equal(t, 250*time.Millisecond, env.PollInterval)
}

func TestLoadIntegrationEnvRejectsMalformedAdminURL(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv("CASSEM_ADM_HTTP_ADDR", "127.0.0.1:20218")

	_, err := LoadIntegrationEnv()

	require.Error(t, err)
	require.Contains(t, err.Error(), "CASSEM_ADM_HTTP_ADDR")
}

func TestLoadIntegrationEnvRejectsMalformedAgentEndpoint(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv("CASSEM_AGENT_GRPC_ADDR", "127.0.0.1")

	_, err := LoadIntegrationEnv()

	require.Error(t, err)
	require.Contains(t, err.Error(), "CASSEM_AGENT_GRPC_ADDR")
}

func TestLoadIntegrationEnvRejectsEmptyDBEndpointList(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv("CASSEM_DB_GRPC_ADDRS", "")

	_, err := LoadIntegrationEnv()

	require.Error(t, err)
	require.Contains(t, err.Error(), "CASSEM_DB_GRPC_ADDRS")
}

func TestLoadIntegrationEnvRejectsBadDuration(t *testing.T) {
	clearIntegrationEnv(t)
	t.Setenv("CASSEM_INTEGRATION_WAIT_TIMEOUT", "slow")

	_, err := LoadIntegrationEnv()

	require.Error(t, err)
	require.Contains(t, err.Error(), "CASSEM_INTEGRATION_WAIT_TIMEOUT")
}

func clearIntegrationEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"CASSEM_INTEGRATION_STRICT",
		"CASSEM_ADM_HTTP_ADDR",
		"CASSEM_AGENT_GRPC_ADDR",
		"CASSEM_DB_GRPC_ADDRS",
		"CASSEM_ADMIN_EMAIL",
		"CASSEM_ADMIN_PASSWORD",
		"CASSEM_INTEGRATION_WAIT_TIMEOUT",
		"CASSEM_INTEGRATION_POLL_INTERVAL",
	} {
		key := key
		value, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if !ok {
				_ = os.Unsetenv(key)
				return
			}
			_ = os.Setenv(key, value)
		})
	}
}
