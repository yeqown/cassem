package raftimpl

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/watcher"
)

// RaftNode defines the ability of what raft component should act.
// TODO(@yeqown): redesign RaftNode interface.
type RaftNode interface {
	// GetKV get value of key
	GetKV(getReq *apicassemdb.GetKVReq) (*apicassemdb.Entity, error)
	// SetKV save key and value
	SetKV(setReq *apicassemdb.SetKVReq) error
	// UnsetKV save key and value
	UnsetKV(unsetReq *apicassemdb.UnsetKVReq) error
	Range(rangeReq *apicassemdb.RangeReq) (*apicassemdb.RangeResp, error)
	Expire(expireReq *apicassemdb.ExpireReq) error

	// IsLeader
	// TODO(@yeqown): replace IsLeader() into Stat()
	IsLeader() bool // IsLeader
	LeaderChangeCh() <-chan bool
	ChangeNotifyCh() <-chan watcher.IChange

	Shutdown() error
}
