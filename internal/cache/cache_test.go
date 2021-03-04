package cache_test

import (
	"testing"

	"github.com/yeqown/cassem/internal/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PersistAndRestore(t *testing.T) {
	var c = cache.NewNonCache()

	err := c.Set("a", []byte("aaaaa")).Error()

	if assert.NoError(t, err) {
		buf, err := c.Persist()

		if assert.NoError(t, err) {
			c2 := cache.NewNonCache()
			err = c2.Restore(buf)
			require.Nil(t, err)

			val, err := c2.Get("a")
			require.Nil(t, err)
			assert.Equal(t, []byte("aaaaa"), val)
		}
	}
}
