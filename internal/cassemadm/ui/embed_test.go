package ui

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDist(t *testing.T) {
	assets, err := Dist()
	require.NoError(t, err)

	index, err := fs.ReadFile(assets, "index.html")
	require.NoError(t, err)
	indexHTML := string(index)
	assert.Contains(t, indexHTML, `id="cassem-admin"`)

	for _, asset := range []string{
		"assets/app.js",
		"assets/style.css",
		"assets/alpine.min.js",
	} {
		asset := asset
		t.Run(asset, func(t *testing.T) {
			assert.Contains(t, indexHTML, `/ui/`+asset)

			contents, err := fs.ReadFile(assets, asset)
			require.NoError(t, err)
			assert.NotEmpty(t, contents)

			if asset == "assets/alpine.min.js" {
				assert.Greater(t, len(contents), 40_000)
				assert.Contains(t, string(contents), "window.Alpine")
				assert.Contains(t, string(contents), "3.15.12")
			}
		})
	}

	appJS, err := fs.ReadFile(assets, "assets/app.js")
	require.NoError(t, err)
	appSource := string(appJS)
	assert.Contains(t, appSource, "const readStoredUser = () =>")
	assert.Contains(t, appSource, "user: readStoredUser()")
	assert.Contains(t, appSource, "const isAuthError = (payload)")
	assert.Contains(t, appSource, "payload?.errcode === 16")
	assert.Contains(t, appSource, "unauthenticated|session expired|invalid session")
	assert.Contains(t, appSource, "clearSession()")
	assert.Contains(t, appSource, "resetWorkspace()")
	assert.Contains(t, appSource, "const key = this.selectedElement.metadata.key")
	assert.Contains(t, appSource, "const updated = this.elements.find(element => element.metadata.key === key)")
	assert.Contains(t, appSource, "if (updated) await this.selectElement(updated)")
	assert.Contains(t, indexHTML, `x-model.number="elementForm.contentType" :disabled="!!selectedElement"`)
}
