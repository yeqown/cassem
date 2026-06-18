package testutil

import "time"

// RequireEventually fails with the last observed reason when a condition never becomes true.
func RequireEventually(t TB, timeout time.Duration, interval time.Duration, check func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	lastReason := "condition not satisfied"
	for time.Now().Before(deadline) {
		ok, reason := check()
		if ok {
			return
		}
		if reason != "" {
			lastReason = reason
		}
		time.Sleep(interval)
	}
	t.Fatalf("condition was not satisfied within %s: %s", timeout, lastReason)
}
