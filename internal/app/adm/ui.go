package adm

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func mountUI(r chi.Router) error {
	assets, err := Dist()
	if err != nil {
		return fmt.Errorf("load UI assets: %w", err)
	}

	mountUIRoutesHTTP(r, assets)
	return nil
}

func mountUIRoutesHTTP(r chi.Router, assets fs.FS) {
	r.Get("/ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})

	r.With(middleware.Compress(5)).Get("/ui/*", uiFileHandlerHTTP(assets))
}

func uiFileHandlerHTTP(assets fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(assets))

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/ui/")
		requestPath := "/" + path
		if path == "" || !uiFileExists(assets, path) {
			requestPath = "/"
		}

		req := r.Clone(r.Context())
		req.URL.Path = requestPath
		req.URL.RawPath = ""
		fileServer.ServeHTTP(w, req)
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
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	return err == nil && !stat.IsDir()
}
