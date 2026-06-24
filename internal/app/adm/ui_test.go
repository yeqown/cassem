package adm

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestMountUIRoutesHTTPRedirectsUIRoot(t *testing.T) {
	r := chi.NewRouter()
	mountUIRoutesHTTP(r, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("index")}})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui", nil))

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/ui/", w.Header().Get("Location"))
}

func TestMountUIRoutesHTTPServesAsset(t *testing.T) {
	r := chi.NewRouter()
	mountUIRoutesHTTP(r, fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")}})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/assets/app.js", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "console.log(1)")
}

func TestMountUIRoutesHTTPFallsBackToIndex(t *testing.T) {
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("spa index")}}
	sub, err := fs.Sub(assets, ".")
	require.NoError(t, err)
	r := chi.NewRouter()
	mountUIRoutesHTTP(r, sub)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ui/missing/path", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "spa index")
}
