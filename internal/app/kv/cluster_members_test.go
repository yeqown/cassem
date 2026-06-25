package kv

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errorx "github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
	"github.com/yeqown/cassem/internal/app/kv/storage"
	"github.com/yeqown/cassem/pkg/conf"
)

func TestClusterMemberKey(t *testing.T) {
	assert.Equal(t, "cassem/cluster/members/7", clusterMemberKey(7))
}

func TestClusterMemberRecordRoundTrip(t *testing.T) {
	want := clusterMemberRecord{
		NodeID:       1,
		RaftAddr:     "http://cassemkv1:3021",
		GRPCEndpoint: "cassemkv1:2021",
	}

	data, err := encodeClusterMemberRecord(want)
	require.NoError(t, err)

	got, err := decodeClusterMemberRecord(data)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDecodeClusterMemberRecordRejectsMissingEndpoint(t *testing.T) {
	_, err := decodeClusterMemberRecord([]byte(`{"node_id":1,"raft_addr":"http://cassemkv1:3021"}`))
	assert.ErrorContains(t, err, "grpc endpoint is required")
}

func TestAdvertiseAddrRequiresExplicitConfig(t *testing.T) {
	assert.Equal(t, "cassemkv1:2021", advertiseAddr(&conf.CassemKVConfig{ListenAddr: "0.0.0.0:2021", AdvertiseAddr: "cassemkv1:2021"}))
	assert.Empty(t, advertiseAddr(&conf.CassemKVConfig{ListenAddr: "0.0.0.0:2021"}))
}

type fakeDiscoveryRaft struct {
	nodeID   uint64
	raftAddr string
	leaderID uint64
	store    map[string][]byte
}

func newFakeDiscoveryRaft() *fakeDiscoveryRaft {
	return &fakeDiscoveryRaft{nodeID: 1, raftAddr: "http://cassemkv1:3021", leaderID: 1, store: map[string][]byte{}}
}

func (f *fakeDiscoveryRaft) NodeID() uint64                         { return f.nodeID }
func (f *fakeDiscoveryRaft) RaftAddr() string                       { return f.raftAddr }
func (f *fakeDiscoveryRaft) Peers() []string                        { return []string{f.raftAddr} }
func (f *fakeDiscoveryRaft) LeaderID() uint64                       { return f.leaderID }
func (f *fakeDiscoveryRaft) IsLeader() bool                         { return f.leaderID == f.nodeID }
func (f *fakeDiscoveryRaft) LeaderChangeCh(chan<- bool)             {}
func (f *fakeDiscoveryRaft) ChangeNotifyCh() <-chan errorx.Change { return nil }
func (f *fakeDiscoveryRaft) AddNode(_ context.Context, addr string) (uint64, []string, error) {
	return 2, []string{f.raftAddr, addr}, nil
}
func (f *fakeDiscoveryRaft) RemoveNode(context.Context, uint64) error { return nil }
func (f *fakeDiscoveryRaft) Shutdown() error                          { return nil }
func (f *fakeDiscoveryRaft) GetKV(req *apikv.GetKVReq) (*apikv.Entity, error) {
	data, ok := f.store[req.GetKey()]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return apikv.NewEntityWithCreated(req.GetKey(), data, 0, time.Now().Unix()), nil
}
func (f *fakeDiscoveryRaft) SetKV(_ context.Context, req *apikv.SetKVReq) error {
	f.store[req.GetKey()] = append([]byte(nil), req.GetVal()...)
	return nil
}
func (f *fakeDiscoveryRaft) UnsetKV(_ context.Context, req *apikv.UnsetKVReq) error {
	delete(f.store, req.GetKey())
	return nil
}
func (f *fakeDiscoveryRaft) Range(req *apikv.RangeReq) (*apikv.RangeResp, error) {
	resp := &apikv.RangeResp{}
	for key, value := range f.store {
		if strings.HasPrefix(key, req.GetKey()+"/") {
			resp.Entities = append(resp.Entities, apikv.NewEntityWithCreated(key, value, 0, time.Now().Unix()))
		}
	}
	return resp, nil
}
func (f *fakeDiscoveryRaft) Expire(*apikv.ExpireReq) error { return nil }

func TestRegisterCurrentMember(t *testing.T) {
	raft := newFakeDiscoveryRaft()
	d := &app{config: &conf.CassemKVConfig{ListenAddr: "0.0.0.0:2021", AdvertiseAddr: "cassemkv1:2021"}, raft: raft}

	require.NoError(t, d.registerCurrentMember())

	got, err := decodeClusterMemberRecord(raft.store[clusterMemberKey(1)])
	require.NoError(t, err)
	assert.Equal(t, clusterMemberRecord{NodeID: 1, RaftAddr: "http://cassemkv1:3021", GRPCEndpoint: "cassemkv1:2021"}, got)
}

func TestListMembers(t *testing.T) {
	raft := newFakeDiscoveryRaft()
	d := &app{config: &conf.CassemKVConfig{ListenAddr: "0.0.0.0:2021", AdvertiseAddr: "cassemkv1:2021"}, raft: raft}
	require.NoError(t, d.registerCurrentMember())

	members, err := d.listMembers()
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, uint64(1), members[0].GetNodeId())
	assert.Equal(t, "http://cassemkv1:3021", members[0].GetRaftAddr())
	assert.Equal(t, "cassemkv1:2021", members[0].GetGrpcEndpoint())
	assert.True(t, members[0].GetLeader())
}

func TestListMembersSkipsStaleRaftPeers(t *testing.T) {
	raft := newFakeDiscoveryRaft()
	d := &app{config: &conf.CassemKVConfig{ListenAddr: "0.0.0.0:2021", AdvertiseAddr: "cassemkv1:2021"}, raft: raft}
	require.NoError(t, d.registerCurrentMember())
	stale, err := encodeClusterMemberRecord(clusterMemberRecord{NodeID: 2, RaftAddr: "http://removed:3022", GRPCEndpoint: "removed:2021"})
	require.NoError(t, err)
	raft.store[clusterMemberKey(2)] = stale

	members, err := d.listMembers()
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, uint64(1), members[0].GetNodeId())
}

func TestListMembersRPC(t *testing.T) {
	raft := newFakeDiscoveryRaft()
	d := &app{config: &conf.CassemKVConfig{ListenAddr: "0.0.0.0:2021", AdvertiseAddr: "cassemkv1:2021"}, raft: raft}
	require.NoError(t, d.registerCurrentMember())

	resp, err := (grpcServer{coord: d}).ListMembers(context.Background(), &apikv.ListMembersRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetMembers(), 1)
	assert.Equal(t, "cassemkv1:2021", resp.GetMembers()[0].GetGrpcEndpoint())
}
