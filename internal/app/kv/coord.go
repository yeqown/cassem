package kv

import (
	"context"

	apikv "github.com/yeqown/cassem/api/kv"
)

// ICoordinator is a interface for app API layer.
type ICoordinator interface {
	getKV(key string) (*apikv.Entity, error)
	setKV(context.Context, *setKVParam) error
	unsetKV(context.Context, *unsetKVParam) error
	watch(keys ...string) (*builtinObserver, func())
	ttl(key string) (int32, error)
	expire(key string) error
	iterate(*rangeParam) (*apikv.RangeResp, error)
	compactElementHistory(*apikv.CompactElementHistoryReq) (*apikv.CompactElementHistoryResp, error)

	// cluster management operations
	addNode(ctx context.Context, raftAddr string, grpcEndpoint string) (nodeId uint64, peers []string, err error)
	removeNode(ctx context.Context, nodeID uint64) error
	listMembers() ([]*apikv.ClusterMember, error)
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
