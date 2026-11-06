package protogen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
