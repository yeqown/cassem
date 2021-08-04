package app

import (
	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/watcher"
)

type ICoordinator interface {
	getKV(key string) (*repository.StoreValue, error)
	setKV(*setKVParam) error
	unsetKV(*unsetKVParam) error
	watch(keys ...string) (watcher.IObserver, func())
	ttl(key string) (int32, error)
	expire(key string) error
	iterate(*rangeParam) (*repository.RangeResult, error)
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

type rangeParam struct {
	key   string
	seek  string
	limit int
}
