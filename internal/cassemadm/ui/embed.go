package ui

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed dist
var dist embed.FS

// Dist returns embedded UI assets rooted at dist.
func Dist() (fs.FS, error) {
	assets, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, fmt.Errorf("load embedded UI dist: %w", err)
	}

	return assets, nil
}
