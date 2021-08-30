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
	LeaderChangeCh(chan<- bool)
	ChangeNotifyCh() <-chan watcher.IChange

	Shutdown() error
}
