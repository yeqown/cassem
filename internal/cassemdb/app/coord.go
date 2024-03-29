package app

import (
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/watcher"
)

// ICoordinator is a interface for app API layer.
type ICoordinator interface {
	getKV(key string) (*apicassemdb.Entity, error)
	setKV(*setKVParam) error
	unsetKV(*unsetKVParam) error
	watch(keys ...string) (watcher.IObserver, func())
	ttl(key string) (int32, error)
	expire(key string) error
	iterate(*rangeParam) (*apicassemdb.RangeResp, error)

	// cluster management operations
	addNode(addr string) (nodeId uint64, peers []string, err error)
	removeNode(nodeID uint64) error
}

type setKVParam struct {
	key       string
	val       []byte
	isDir     bool
	overwrite bool
	ttl       int32
}

type unsetKVParam struct {
	key   string
	isDir bool
}

type rangeParam struct {
	key   string
	seek  string
	limit int
}
