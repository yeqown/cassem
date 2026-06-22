package adm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

func TestInstancesRespReturnsStructuredTargets(t *testing.T) {
	out := newInstancesResp(&concept.GetInstancesResult{
		CommonPager: concept.CommonPager{HasMore: true, NextSeek: "next-client"},
		Instances: []*concept.Instance{
			{
				ClientId:           "client-01",
				AgentId:            "agent-a",
				ClientIp:           "127.0.0.1",
				LastRenewTimestamp: 1_700_000_000,
				Watching: []*concept.Instance_Watching{
					{App: "demo", Env: "prod", WatchKeys: []string{"db_url", "feature_flag"}},
					{App: "demo", Env: "staging", WatchKeys: []string{"db_url"}},
				},
			},
		},
	})

	require.True(t, out.HasMore)
	require.Equal(t, "next-client", out.NextSeek)
	require.Len(t, out.Instances, 1)
	require.Equal(t, "client-01@127.0.0.1", out.Instances[0].ID)
	require.Equal(t, "client-01", out.Instances[0].ClientID)
	require.Equal(t, "agent-a", out.Instances[0].AgentID)
	require.Equal(t, "127.0.0.1", out.Instances[0].ClientIP)
	require.Equal(t, int64(1_700_000_000), out.Instances[0].LastRenewTimestamp)

	body, err := json.Marshal(out)
	require.NoError(t, err)
	require.Contains(t, string(body), `"targets":[{"app":"demo","env":"prod","key":"db_url"},{"app":"demo","env":"prod","key":"feature_flag"},{"app":"demo","env":"staging","key":"db_url"}]`)
	require.NotContains(t, string(body), `"app":"demo","env":"prod, staging"`)
	require.Contains(t, string(body), `"lastRenewTimestamp":1700000000`)
}

func TestInstancesRespKeepsEmptyFieldsInJSON(t *testing.T) {
	out := newInstancesResp(&concept.GetInstancesResult{
		Instances: []*concept.Instance{
			{
				ClientId:           "client-01",
				AgentId:            "agent-a",
				ClientIp:           "127.0.0.1",
				LastRenewTimestamp: 1_700_000_000,
			},
		},
	})

	body, err := json.Marshal(out)
	require.NoError(t, err)
	require.Contains(t, string(body), `"id":"client-01@127.0.0.1"`)
	require.Contains(t, string(body), `"targets":[]`)
	require.Contains(t, string(body), `"lastRenewTimestamp":1700000000`)
}
