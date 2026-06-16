package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yeqown/cassem/api/concept"
)

func TestTopologyInstancePageLimitFitsRangeValidation(t *testing.T) {
	assert.LessOrEqual(t, topologyInstancePageLimit, 100)
	assert.GreaterOrEqual(t, topologyInstancePageLimit, 1)
}

func TestTopologyHealthHelpers(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	tests := []struct {
		name string
		got  topologyHealth
		want topologyHealth
	}{
		{
			name: "agent with valid renewal annotations is healthy",
			got: agentHealth(&concept.AgentInstance{
				Addr:        "127.0.0.1:2030",
				Annotations: map[string]string{"ttl": "60", "renewInterval": "30"},
			}),
			want: topologyHealthHealthy,
		},
		{
			name: "agent with invalid renewal interval is unhealthy",
			got: agentHealth(&concept.AgentInstance{
				Addr:        "127.0.0.1:2030",
				Annotations: map[string]string{"ttl": "30", "renewInterval": "30"},
			}),
			want: topologyHealthUnhealthy,
		},
		{name: "agent without address is offline", got: agentHealth(&concept.AgentInstance{}), want: topologyHealthOffline},
		{name: "fresh instance is healthy", got: instanceHealth(now.Add(-30*time.Second).Unix(), now), want: topologyHealthHealthy},
		{name: "stale instance is unhealthy", got: instanceHealth(now.Add(-90*time.Second).Unix(), now), want: topologyHealthUnhealthy},
		{name: "expired instance is offline", got: instanceHealth(now.Add(-3*time.Minute).Unix(), now), want: topologyHealthOffline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestExtractTopologyHost(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{addr: "127.0.0.1:2021", want: "127.0.0.1"},
		{addr: "http://10.0.0.1:2030", want: "10.0.0.1"},
		{addr: "agent.internal", want: "agent.internal"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			assert.Equal(t, tt.want, extractHost(tt.addr))
		})
	}
}
