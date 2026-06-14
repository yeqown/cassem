package ui

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var assetRefPattern = regexp.MustCompile(`/ui/assets/[^"' )]+`)

func TestDist(t *testing.T) {
	assets, err := Dist()
	require.NoError(t, err)

	index, err := fs.ReadFile(assets, "index.html")
	require.NoError(t, err)
	indexHTML := string(index)
	assert.Contains(t, indexHTML, `id="cassem-admin"`)
	assert.Contains(t, indexHTML, `/ui/assets/`)

	assetRefs := assetRefPattern.FindAllString(indexHTML, -1)
	require.NotEmpty(t, assetRefs)

	seen := make(map[string]struct{}, len(assetRefs))
	for _, assetRef := range assetRefs {
		assetPath := strings.TrimPrefix(assetRef, "/ui/")
		if _, ok := seen[assetPath]; ok {
			continue
		}
		seen[assetPath] = struct{}{}

		contents, err := fs.ReadFile(assets, assetPath)
		require.NoError(t, err)
		assert.NotEmpty(t, contents)
	}
}
