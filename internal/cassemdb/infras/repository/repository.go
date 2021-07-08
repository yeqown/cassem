package repository

import (
	"strings"

	"github.com/yeqown/cassem/pkg/types"
)

// Repository is a proxy who helps convert data between logic and persistence.Not only all parameters of Repository
// are logic datatype, but also all return values.
// NOTE(@yeqown): how to delete resource or mark it as deprecated, now only support container deletion.
type Repository interface {
	// GetKV get value of key
	GetKV(key types.StoreKey, isDir bool) (*types.StoreValue, error)

	// SetKV save key and value
	SetKV(key types.StoreKey, value types.StoreValue, isDir bool) error

	// UnsetKV save key and value
	UnsetKV(key types.StoreKey, isDir bool) error
}

func KeySplitter(s types.StoreKey) (nodes []string, leaf string) {
	arr := strings.Split(s.String(), "/")
	l := len(arr)
	if l < 1 {
		return
	}

	leaf = arr[l-1]
	if l > 1 {
		nodes = arr[:l-1]
	}

	return
}

func IsEmptyLeaf(leaf string) bool {
	if len(leaf) == 0 {
		return true
	}

	leaf = strings.TrimSpace(leaf)
	return len(leaf) == 0
}
