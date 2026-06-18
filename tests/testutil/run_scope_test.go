package testutil

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunScopeBuildsValidBusinessNames(t *testing.T) {
	scope := NewRunScope(t, "Online Payment Feature Flag")

	require.Contains(t, scope.ID, "online-payment-feature-flag")
	require.Regexp(t, `^[a-z0-9][a-z0-9-]*$`, scope.ID)
	require.Contains(t, scope.App("payment-service"), "payment-service")
	require.Contains(t, scope.Env("production"), "production")
	require.Equal(t, "checkout-feature-dynamic-risk-control", scope.Key("checkout-feature-dynamic-risk-control"))
	require.True(t, strings.HasPrefix(scope.ClientID("checkout-worker-a"), "checkout-worker-a-"))
	require.LessOrEqual(t, len(scope.ClientID("checkout-worker-a")), 64)
	require.Contains(t, scope.Account("release.manager"), "@cassem.local")
	require.Contains(t, scope.TTLKey("leases", "payment-service", "release-lock"), "cassem/integration/leases/payment-service/release-lock")
}

func TestRunScopePreservesUniqueSuffixForLongNames(t *testing.T) {
	scope := NewRunScope(t, "Online Payment Feature Flag")
	suffix := shortScope(scope.ID)

	app := scope.App("regional-checkout-payment-service")
	env := scope.Env("production-canary-environment")
	clientID := scope.ClientID("checkout-worker-primary-eu-west-1")

	require.Contains(t, app, suffix)
	require.LessOrEqual(t, len(app), 30)
	require.Contains(t, env, suffix)
	require.LessOrEqual(t, len(env), 30)
	require.Contains(t, clientID, suffix)
	require.LessOrEqual(t, len(clientID), 64)
}

func TestRequireEventuallyReportsLastReason(t *testing.T) {
	fake := &recordingTB{}
	attempts := 0
	RequireEventually(fake, 30*time.Millisecond, 5*time.Millisecond, func() (bool, string) {
		attempts++
		if attempts == 1 {
			return false, "not visible yet"
		}
		return false, "latest value still stale"
	})

	require.True(t, fake.failed)
	require.Contains(t, fake.message, "latest value still stale")
	require.NotContains(t, fake.message, "not visible yet")
}

type recordingTB struct {
	failed  bool
	message string
}

func (t *recordingTB) Helper() {}
func (t *recordingTB) Fatalf(format string, args ...any) {
	t.failed = true
	t.message = fmt.Sprintf(format, args...)
}
func (t *recordingTB) Logf(format string, args ...any)  {}
func (t *recordingTB) Skipf(format string, args ...any) {}
func (t *recordingTB) TempDir() string                  { return "" }
