package cache

import "github.com/pkg/errors"

var ErrMiss = errors.New("key missed")

type IStore interface {
	// Persist serializes cache into []byte data.
	Persist() ([]byte, error)

	// Restore apply data to override all the cache's data, data is comes from Persist.
	Restore(data []byte) error
}

// ICache represents the proxy to operate cache data,
// also need to replace data while cache size is over than it's limits.
type ICache interface {
	IStore

	// Set returns wasSet means this key need to synchronous, err means set failed.
	Set(key string, v []byte) SetResult

	// Get
	Get(key string) ([]byte, error)

	Del(key string) error
}

// SetResult represents what operations would be caused by ICache.Set.
//
// NeedSync tells users that them should trigger setting apply, NeedDeleteKey should be set while
// Cache-Replacing happened.
type SetResult struct {
	err           error
	NeedSync      bool
	NeedDeleteKey string
}

func (s SetResult) Error() error {
	return s.err
}
