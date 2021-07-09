package repository

import (
	"strings"
)

// KV is a proxy who helps convert data between logic and persistence.Not only all parameters of KV
// are logic datatype, but also all return values.
type KV interface {
	// GetKV get value of key
	GetKV(key StoreKey, isDir bool) (*StoreValue, error)

	// SetKV save key and value
	SetKV(key StoreKey, value *StoreValue, isDir bool) error

	// UnsetKV save key and value
	UnsetKV(key StoreKey, isDir bool) error

	// Range iterates all keys or buckets under the given key.
	Range(key StoreKey, seek string, limit int) (*RangeResult, error)
}

func keySplitter(s StoreKey) (paths []string, leaf string) {
	arr := strings.Split(s.String(), "/")
	l := len(arr)
	if l < 1 {
		return
	}

	leaf = arr[l-1]
	if l > 1 {
		paths = arr[:l-1]
	}

	return
}

func isEmptyLeaf(leaf string) bool {
	if len(leaf) == 0 {
		return true
	}

	leaf = strings.TrimSpace(leaf)
	return len(leaf) == 0
}
