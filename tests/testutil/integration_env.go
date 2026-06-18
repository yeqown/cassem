package testutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultAdmHTTPAddr   = "http://127.0.0.1:20218"
	defaultAgentGRPCAddr = "127.0.0.1:20219"
	defaultDBGRPCAddrs   = "127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023"
	defaultAdminEmail    = "superadmin@example.com"
	defaultAdminPassword = "cassem"
	defaultWaitTimeout   = 90 * time.Second
	defaultPollInterval  = time.Second
)

// IntegrationEnv defines the endpoint and timing contract for integration tests.
type IntegrationEnv struct {
	// Strict enables fail-fast integration behavior used by CI release gates.
	Strict bool
	// AdmHTTPAddr is the admin HTTP base address.
	AdmHTTPAddr string
	// AgentGRPCAddr is the agent gRPC endpoint in host:port form.
	AgentGRPCAddr string
	// DBGRPCAddrs contains one or more cassemdb gRPC endpoints.
	DBGRPCAddrs []string
	// AdminEmail is the bootstrap admin account used by integration tests.
	AdminEmail string
	// AdminPassword is the bootstrap admin password used by integration tests.
	AdminPassword string
	// WaitTimeout bounds eventual checks against the shared integration cluster.
	WaitTimeout time.Duration
	// PollInterval controls retry cadence for eventual integration assertions.
	PollInterval time.Duration
}

// LoadIntegrationEnv reads and validates integration environment settings.
func LoadIntegrationEnv() (IntegrationEnv, error) {
	waitTimeout, err := lookupDuration("CASSEM_INTEGRATION_WAIT_TIMEOUT", defaultWaitTimeout)
	if err != nil {
		return IntegrationEnv{}, err
	}

	pollInterval, err := lookupDuration("CASSEM_INTEGRATION_POLL_INTERVAL", defaultPollInterval)
	if err != nil {
		return IntegrationEnv{}, err
	}

	env := IntegrationEnv{
		Strict:        os.Getenv("CASSEM_INTEGRATION_STRICT") == "1",
		AdmHTTPAddr:   lookupString("CASSEM_ADM_HTTP_ADDR", defaultAdmHTTPAddr),
		AgentGRPCAddr: lookupString("CASSEM_AGENT_GRPC_ADDR", defaultAgentGRPCAddr),
		AdminEmail:    lookupString("CASSEM_ADMIN_EMAIL", defaultAdminEmail),
		AdminPassword: lookupString("CASSEM_ADMIN_PASSWORD", defaultAdminPassword),
		WaitTimeout:   waitTimeout,
		PollInterval:  pollInterval,
	}

	dbAddrs, err := parseDBAddrs(lookupString("CASSEM_DB_GRPC_ADDRS", defaultDBGRPCAddrs))
	if err != nil {
		return IntegrationEnv{}, err
	}
	env.DBGRPCAddrs = dbAddrs

	if err := validateHTTPURL("CASSEM_ADM_HTTP_ADDR", env.AdmHTTPAddr); err != nil {
		return IntegrationEnv{}, err
	}
	if err := validateHostPort("CASSEM_AGENT_GRPC_ADDR", env.AgentGRPCAddr); err != nil {
		return IntegrationEnv{}, err
	}
	if strings.TrimSpace(env.AdminEmail) == "" {
		return IntegrationEnv{}, fmt.Errorf("CASSEM_ADMIN_EMAIL must not be empty")
	}
	if strings.TrimSpace(env.AdminPassword) == "" {
		return IntegrationEnv{}, fmt.Errorf("CASSEM_ADMIN_PASSWORD must not be empty")
	}
	if env.WaitTimeout <= 0 {
		return IntegrationEnv{}, fmt.Errorf("CASSEM_INTEGRATION_WAIT_TIMEOUT must be positive")
	}
	if env.PollInterval <= 0 {
		return IntegrationEnv{}, fmt.Errorf("CASSEM_INTEGRATION_POLL_INTERVAL must be positive")
	}

	return env, nil
}

// MustIntegrationEnv loads the integration environment or fails the test immediately.
func MustIntegrationEnv(t TB) IntegrationEnv {
	t.Helper()

	env, err := LoadIntegrationEnv()
	if err != nil {
		t.Fatalf("load integration environment: %v", err)
	}

	return env
}

func lookupString(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}

func lookupDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}

	return d, nil
}

func validateHTTPURL(key string, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https: %q", key, raw)
	}
	if parsed.Host == "" {
		return fmt.Errorf("%s must include host: %q", key, raw)
	}
	if _, _, err := net.SplitHostPort(parsed.Host); err != nil {
		return fmt.Errorf("%s must include host and port: %w", key, err)
	}

	return nil
}

func validateHostPort(key string, raw string) error {
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return fmt.Errorf("%s must be host:port: %w", key, err)
	}
	if host == "" || port == "" {
		return fmt.Errorf("%s must include host and port: %q", key, raw)
	}

	return nil
}

func parseDBAddrs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("CASSEM_DB_GRPC_ADDRS must not be empty")
	}

	parts := strings.Split(raw, ",")
	addrs := make([]string, 0, len(parts))
	for _, part := range parts {
		endpoint := strings.TrimSpace(part)
		if endpoint == "" {
			continue
		}
		if err := validateHostPort("CASSEM_DB_GRPC_ADDRS", endpoint); err != nil {
			return nil, err
		}
		addrs = append(addrs, endpoint)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("CASSEM_DB_GRPC_ADDRS must include at least one endpoint")
	}

	return addrs, nil
}
