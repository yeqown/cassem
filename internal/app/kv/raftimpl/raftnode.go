package raftimpl

import (
	"context"

	errorx "github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
)

// RaftNode defines the ability of what raft component should act.
type RaftNode interface {
	// GetKV get value of key
	GetKV(getReq *apikv.GetKVReq) (*apikv.Entity, error)
	// SetKV save key and value
	SetKV(ctx context.Context, setReq *apikv.SetKVReq) error
	// UnsetKV save key and value
	UnsetKV(ctx context.Context, unsetReq *apikv.UnsetKVReq) error
	Range(rangeReq *apikv.RangeReq) (*apikv.RangeResp, error)
	Expire(expireReq *apikv.ExpireReq) error

	// IsLeader returns current node is leader or not. true mean leader.
	IsLeader() bool
	// NodeID returns current raft member id.
	NodeID() uint64
	// RaftAddr returns current node raft address.
	RaftAddr() string
	// Peers returns known raft peer addresses.
	Peers() []string
	// LeaderID returns current leader id, or 0 when unknown.
	LeaderID() uint64
	LeaderChangeCh(chan<- bool)
	ChangeNotifyCh() <-chan errorx.Change
	AddNode(ctx context.Context, addr string) (nodeID uint64, peers []string, err error)
	RemoveNode(ctx context.Context, nodeID uint64) error

	Shutdown() error
}
