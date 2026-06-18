package raftimpl

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/watcher"
)

// RaftNode defines the ability of what raft component should act.
type RaftNode interface {
	// GetKV get value of key
	GetKV(getReq *apicassemdb.GetKVReq) (*apicassemdb.Entity, error)
	// SetKV save key and value
	SetKV(setReq *apicassemdb.SetKVReq) error
	// UnsetKV save key and value
	UnsetKV(unsetReq *apicassemdb.UnsetKVReq) error
	Range(rangeReq *apicassemdb.RangeReq) (*apicassemdb.RangeResp, error)
	Expire(expireReq *apicassemdb.ExpireReq) error

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
	ChangeNotifyCh() <-chan watcher.IChange
	AddNode(addr string) (nodeID uint64, peers []string, err error)
	RemoveNode(nodeID uint64) error

	Shutdown() error
}
