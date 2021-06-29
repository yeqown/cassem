package persistence

import "strings"

// Repository is a proxy who helps convert data between logic and persistence.Not only all parameters of Repository
// are logic datatype, but also all return values.
// NOTE(@yeqown): how to delete resource or mark it as deprecated, now only support container deletion.
type Repository interface {
	// Get get value of key
	Get(key string) ([]byte, error)

	// Set save key and value
	Set(key string, value []byte) error

	// Unset save key and value
	Unset(key string) error
}

func KeySplitter(s string) (nodes []string, leaf string) {
	arr := strings.Split(s, "/")
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
	if len(leaf) == 0 {
		return true
	}

	return false
}
