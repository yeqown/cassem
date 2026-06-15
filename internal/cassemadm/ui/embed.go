package ui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var distCandidates = []string{
	filepath.Join("internal", "cassemadm", "ui", "dist"),
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
