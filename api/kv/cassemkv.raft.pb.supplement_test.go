package kv

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntityTTL(t *testing.T) {
	tests := []struct {
		name    string
		ttl     int32
		expired bool
		wantTTL int32
	}{
		{
			name:    "zero ttl never expires",
			ttl:     0,
			expired: false,
			wantTTL: NEVER_EXPIRED,
		},
		{
			name:    "never expired sentinel",
			ttl:     NEVER_EXPIRED,
			expired: false,
			wantTTL: NEVER_EXPIRED,
		},
		{
			name:    "expired sentinel",
			ttl:     EXPIRED,
			expired: true,
			wantTTL: EXPIRED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := NewEntityWithCreated("ttl/key", []byte("value"), tt.ttl, time.Now().Unix())

			assert.Equal(t, tt.expired, entity.Expired())
			assert.Equal(t, tt.wantTTL, entity.GetTtl())
		})
	}
}

func TestEntityPositiveTTLExpires(t *testing.T) {
	entity := NewEntityWithCreated("ttl/key", []byte("value"), 1, time.Now().Unix())
	entity.UpdatedAt = time.Now().Add(-2 * time.Second).Unix()

	assert.True(t, entity.Expired())
	assert.Equal(t, int32(EXPIRED), entity.GetTtl())
}

func TestClusterDiscoveryMessagesCompile(t *testing.T) {
	member := &ClusterMember{
		NodeId:       1,
		RaftAddr:     "http://cassemkv1:3021",
		GrpcEndpoint: "cassemkv1:2021",
		Leader:       true,
	}
	resp := &ListMembersResponse{Members: []*ClusterMember{member}}

	assert.Equal(t, uint64(1), resp.GetMembers()[0].GetNodeId())
	assert.Equal(t, "http://cassemkv1:3021", resp.GetMembers()[0].GetRaftAddr())
	assert.Equal(t, "cassemkv1:2021", resp.GetMembers()[0].GetGrpcEndpoint())
	assert.True(t, resp.GetMembers()[0].GetLeader())
}
