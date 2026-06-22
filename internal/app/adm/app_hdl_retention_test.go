package adm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/pkg/conf"
)

func TestRetentionPolicyResponseFromConfig(t *testing.T) {
	config := conf.DefaultRetentionConfig()

	resp := retentionPolicyResponseFromConfig(config)

	require.True(t, resp.Enabled)
	require.Equal(t, 20, resp.KeepVersionCount)
	require.Equal(t, 30, resp.KeepVersionDays)
	require.Equal(t, 180, resp.KeepOperationDays)
	require.Equal(t, "Versions keep current, draft, latest 20, and versions from the last 30 days.", resp.VersionPolicy)
	require.Equal(t, "Operation logs keep 180 days.", resp.OperationPolicy)
}

func TestRetentionPolicyResponseFromNilConfig(t *testing.T) {
	resp := retentionPolicyResponseFromConfig(nil)

	require.True(t, resp.Enabled)
	require.Equal(t, 20, resp.KeepVersionCount)
	require.Equal(t, 30, resp.KeepVersionDays)
	require.Equal(t, 180, resp.KeepOperationDays)
}
