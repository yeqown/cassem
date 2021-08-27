package storage

import (
	"strings"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

// KV is a proxy who helps convert data between logic and persistence.Not only all parameters of KV
// are logic datatype, but also all return values.
type KV interface {
	// GetKV get value of key
	GetKV(key string, isDir bool) (*apicassemdb.Entity, error)

	// SetKV save key and value
	SetKV(key string, value *apicassemdb.Entity, isDir bool) error

	// UnsetKV save key and value
	UnsetKV(key string, isDir bool) error

	// Range iterates all keys or buckets under the given key.
	Range(key string, seek string, limit int) (*RangeResult, error)
}

type RangeResult struct {
	Items       []*apicassemdb.Entity
	HasMore     bool
	NextSeekKey string
	ExpiredKeys []string
}

func KeySplitter(s string) (paths []string, leaf string) {
	arr := strings.Split(s, "/")
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
