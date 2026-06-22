package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

func TestQueryFromCache(t *testing.T) {
	elem := &concept.Element{Metadata: &concept.ElementMetadata{Key: "key-1"}, Raw: []byte("value")}
	d := app{cache: NewCache(10)}
	d.cache.Set("app", "dev", "key-1", elem)
	d.cache.Set("app", "dev", "key-1", elem)

	got := d.queryFromCache("app", "dev", "key-1", "missing")

	require.NotNil(t, got)
	assert.Equal(t, []*concept.Element{elem}, got.elems)
	assert.Equal(t, []string{"missing"}, got.miss)
}

func TestUpdateCache(t *testing.T) {
	d := app{cache: NewCache(10)}
	elem1 := &concept.Element{Metadata: &concept.ElementMetadata{Key: "key-1"}, Raw: []byte("value-1")}
	elem2 := &concept.Element{Metadata: &concept.ElementMetadata{Key: "key-2"}, Raw: []byte("value-2")}

	d.updateCache("app", "dev", elem1, elem2)
	d.updateCache("app", "dev", elem1, elem2)

	got1, ok := d.cache.Query("app", "dev", "key-1")
	require.True(t, ok)
	assert.Same(t, elem1, got1)
	got2, ok := d.cache.Query("app", "dev", "key-2")
	require.True(t, ok)
	assert.Same(t, elem2, got2)
}
