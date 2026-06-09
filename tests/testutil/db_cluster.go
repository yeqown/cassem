package testutil

import "testing"

func StartDBCluster(t testing.TB) *Cluster {
	t.Helper()
	return UseDBCluster(t)
}

func StartAdmCluster(t testing.TB) *Cluster {
	t.Helper()
	return UseAdmCluster(t)
}

func StartFullCluster(t testing.TB) *Cluster {
	t.Helper()
	return UseFullCluster(t)
}
