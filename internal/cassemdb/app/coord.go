package app

import (
	"github.com/yeqown/cassem/pkg/types"
	"github.com/yeqown/cassem/pkg/watcher"
)

type ICoordinator interface {
	GetKV(key string) (*types.StoreValue, error)
	SetKV(key string, val []byte) error
	UnsetKV(key string) error
	Watch(keys ...string) (watcher.IObserver, func())

	ShouldForwardToLeader() (bool, string)

	RemoveNode(serveId string) error    // RemoveNode
	AddNode(serveId, addr string) error // AddNode
	Apply(data []byte) error            // Apply

	Heartbeat()
}
