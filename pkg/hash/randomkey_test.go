package hash_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yeqown/cassem/pkg/hash"
)

func Test_RandKey(t *testing.T) {
	key := hash.RandKey(-1)

	assert.Equal(t, 6, len(key))
	assert.NotEmpty(t, key)
	t.Log(key)

	key = hash.RandKey(12)
	assert.Equal(t, 12, len(key))
	assert.NotEmpty(t, key)
	t.Log(key)
}
