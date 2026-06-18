//go:build integration

package testutil

import "testing"

func TestIntegrationClusterReady(t *testing.T) {
	RequireFullCluster(t)
}
