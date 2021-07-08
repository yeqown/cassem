package app

import (
	"github.com/yeqown/cassem/pkg/types"
	"github.com/yeqown/cassem/pkg/watcher"
)

type ICoordinator interface {
	getKV(key string) (*types.StoreValue, error)
	setKV(*setKVParam) error
	unsetKV(param *unsetKVParam) error
	watch(keys ...string) (watcher.IObserver, func())
	ttl(key string) (uint32, error)
	expire(key string) error
	iter(key string) error
}

type setKVParam struct {
	key       string
	val       []byte
	isDir     bool
	overwrite bool
	ttl       uint32
}

type unsetKVParam struct {
	key   string
	isDir bool
}
