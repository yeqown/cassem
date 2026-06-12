package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		{name: "fallback spa route", path: "/ui/config/apps/demo", statusCode: http.StatusOK, body: "Cassem Admin Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			mountUIRoutes(r, os.DirFS("testdata/ui"))

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

func TestMountUIRoutesDoesNotCaptureAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	mountUIRoutes(r, os.DirFS("testdata/ui"))
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
		{name: "fallback spa route", path: "/ui/config/apps/demo"},
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
			mountUIRoutes(r, os.DirFS("testdata/ui"))

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
