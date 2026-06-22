package adm

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var assetRefPattern = regexp.MustCompile(`/ui/assets/[^"' )]+`)

func TestDist(t *testing.T) {
	root := t.TempDir()
	distDir := filepath.Join(root, "dist")
	require.NoError(t, os.MkdirAll(filepath.Join(distDir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "index.html"), []byte(`<!doctype html><div id="cassem-admin"></div><script type="module" src="/ui/assets/app.js"></script>`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "assets", "app.js"), []byte(`console.log('ok')`), 0o644))

	origCandidates := distCandidates
	distCandidates = []string{filepath.Join(root, "missing"), distDir}
	t.Cleanup(func() { distCandidates = origCandidates })

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

func TestDistReturnsErrorWhenNoCandidateExists(t *testing.T) {
	origCandidates := distCandidates
	distCandidates = []string{filepath.Join(t.TempDir(), "missing")}
	t.Cleanup(func() { distCandidates = origCandidates })

	_, err := Dist()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load UI dist")
}
