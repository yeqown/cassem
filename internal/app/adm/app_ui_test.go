package adm

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uiTestAssetsFS(t *testing.T) fs.FS {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return os.DirFS(filepath.Join(filepath.Dir(file), "testdata", "ui"))
}

func TestMountUIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		path       string
		statusCode int
		body       string
		location   string
	}{
		{name: "redirect ui root", path: "/ui", statusCode: http.StatusMovedPermanently, location: "/ui/"},
		{name: "serve index", path: "/ui/", statusCode: http.StatusOK, body: "Cassem Admin Test"},
		{name: "serve asset", path: "/ui/assets/app.js", statusCode: http.StatusOK, body: "CASSEM_TEST_ASSET"},
		{name: "fallback spa route", path: "/ui/apps/demo/envs/prod/elements/db.url", statusCode: http.StatusOK, body: "Cassem Admin Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			mountUIRoutes(r, uiTestAssetsFS(t))

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			if tt.body != "" {
				assert.Contains(t, w.Body.String(), tt.body)
			}
			if tt.location != "" {
				assert.Equal(t, tt.location, w.Header().Get("Location"))
			}
		})
	}
}

func decodeGzipBody(t *testing.T, body []byte) string {
	t.Helper()

	zr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer zr.Close()

	plain, err := io.ReadAll(zr)
	require.NoError(t, err)
	return string(plain)
}

func TestMountUIRoutesGzipUIResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	mountUIRoutes(r, uiTestAssetsFS(t))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Contains(t, w.Header().Get("Vary"), "Accept-Encoding")
	assert.Contains(t, decodeGzipBody(t, w.Body.Bytes()), "Cassem Admin Test")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ui/assets/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Contains(t, decodeGzipBody(t, w.Body.Bytes()), "CASSEM_TEST_ASSET")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ui/apps/demo/envs/prod/elements/db.url", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Contains(t, decodeGzipBody(t, w.Body.Bytes()), "Cassem Admin Test")
}

func TestMountUIRoutesDoesNotCaptureAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	mountUIRoutes(r, uiTestAssetsFS(t))
	r.POST("/api/account/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"api": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/account/login", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"api":true}`, w.Body.String())
}

func TestUIFileHandlerPreservesRequestPathAfterNext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
	}{
		{name: "fallback spa route", path: "/ui/apps/demo/envs/prod/elements/db.url"},
		{name: "asset route", path: "/ui/assets/app.js"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recordedPath string
			var recordedRawPath string

			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Next()
				recordedPath = c.Request.URL.Path
				recordedRawPath = c.Request.URL.RawPath
			})
			mountUIRoutes(r, uiTestAssetsFS(t))

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.URL.RawPath = tt.path
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.path, recordedPath)
			assert.Equal(t, tt.path, recordedRawPath)
		})
	}
}
