package testutil

import "testing"

func StartDBCluster(t testing.TB) *Cluster {
	t.Helper()
	return startCluster(t, false, false)
}

func StartAdmCluster(t testing.TB) *Cluster {
	t.Helper()
	return startCluster(t, true, false)
}

func StartFullCluster(t testing.TB) *Cluster {
	t.Helper()
	return startCluster(t, true, true)
}
