package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

func TestInstancesRespFlattensWatching(t *testing.T) {
	out := newInstancesResp(&concept.GetInstancesResult{
		CommonPager: concept.CommonPager{HasMore: true, NextSeek: "next-client"},
		Instances: []*concept.Instance{
			{
				ClientId:           "client-01",
				AgentId:            "agent-a",
				ClientIp:           "127.0.0.1",
				LastRenewTimestamp: 1_700_000_000,
				Watching: []*concept.Instance_Watching{
					{App: "demo", Env: "prod", WatchKeys: []string{"db.url", "feature.flag"}},
					{App: "demo", Env: "staging", WatchKeys: []string{"db.url"}},
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
	require.Equal(t, "demo", out.Instances[0].App)
	require.Equal(t, "prod, staging", out.Instances[0].Env)
	require.Equal(t, "db.url, feature.flag", out.Instances[0].Key)
	require.Equal(t, int64(1_700_000_000), out.Instances[0].LastRenewTimestamp)
	require.Len(t, out.Instances[0].Watching, 2)

	body, err := json.Marshal(out)
	require.NoError(t, err)
	require.Contains(t, string(body), `"app":"demo"`)
	require.Contains(t, string(body), `"env":"prod, staging"`)
	require.Contains(t, string(body), `"key":"db.url, feature.flag"`)
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
	require.Contains(t, string(body), `"app":""`)
	require.Contains(t, string(body), `"env":""`)
	require.Contains(t, string(body), `"key":""`)
	require.Contains(t, string(body), `"lastRenewTimestamp":1700000000`)
}
