package adm

import (
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var distCandidates = []string{
	filepath.Join("internal", "app", "adm", "dist"),
	"dist",
}

// Dist returns UI assets rooted at dist.
func Dist() (fs.FS, error) {
	for _, candidate := range distCandidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return os.DirFS(candidate), nil
		}
	}

	return nil, fmt.Errorf("load UI dist: no dist directory found in %v", distCandidates)
}

func mountUI(engi *gin.Engine) error {
	assets, err := Dist()
	if err != nil {
		return fmt.Errorf("load UI assets: %w", err)
	}

	mountUIRoutes(engi, assets)
	return nil
}

func mountUIRoutes(engi *gin.Engine, assets fs.FS) {
	engi.GET("/ui", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})

	ui := engi.Group("/ui", gzip.Gzip(gzip.DefaultCompression))
	ui.GET("/*filepath", uiFileHandler(assets))
}

func uiFileHandler(assets fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(assets))

	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("filepath"), "/")
		requestPath := "/" + path
		if path == "" || !uiFileExists(assets, path) {
			requestPath = "/"
		}

		req := c.Request.Clone(c.Request.Context())
		req.URL.Path = requestPath
		req.URL.RawPath = ""
		fileServer.ServeHTTP(c.Writer, req)
	}
}

func uiFileExists(assets fs.FS, name string) bool {
	if !fs.ValidPath(name) {
		return false
	}

	file, err := assets.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()

	stat, err := file.Stat()
	return err == nil && !stat.IsDir()
}
