package raftimpl

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/watcher"
)

// RaftNode defines the ability of what raft component should act.
// TODO(@yeqown): redesign RaftNode interface.
type RaftNode interface {
	// GetKV get value of key
	GetKV(key string) (*apicassemdb.Entity, error)
	// SetKV save key and value
	SetKV(key string, value []byte, isDir, overwrite bool, ttl int32) error
	// UnsetKV save key and value
	UnsetKV(key string, isDir bool) error
	Range(key, seek string, limit int) (*apicassemdb.RangeResp, error)
	Expire(key string) error
	ChangeNotifyCh() <-chan watcher.IChange

	// IsLeader
	// TODO(@yeqown): replace IsLeader() into Stat()
	IsLeader() bool // IsLeader
	LeaderChangeCh() <-chan bool

	Shutdown() error
}
