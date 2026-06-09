package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestDBClusterUsesFixedEndpoints(t *testing.T) {
	cluster := TestDBCluster()

	assert.Equal(t, []string{
		"127.0.0.1:2021",
		"127.0.0.1:2022",
		"127.0.0.1:2023",
	}, cluster.DBEndpoints)
	assert.Empty(t, cluster.AdmBaseURL)
	assert.Empty(t, cluster.AgentEndpoint)
}

func TestTestAdmClusterUsesFixedEndpoint(t *testing.T) {
	cluster := TestAdmCluster()

	assert.Equal(t, "http://127.0.0.1:20218", cluster.AdmBaseURL)
	assert.Empty(t, cluster.AgentEndpoint)
}

func TestTestFullClusterUsesFixedEndpoint(t *testing.T) {
	cluster := TestFullCluster()

	assert.Equal(t, "127.0.0.1:20219", cluster.AgentEndpoint)
}
