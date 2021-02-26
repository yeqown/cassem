package cache

import "github.com/pkg/errors"

var ErrMiss = errors.New("key missed")

// ICache represents the proxier to operate cache data,
// also need to replace data while cache size is over than it's limits.
type ICache interface {
	Set(key string, v []byte) (evicted bool, err error)

	Get(key string) ([]byte, error)
}
