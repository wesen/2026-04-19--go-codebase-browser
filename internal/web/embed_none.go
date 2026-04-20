//go:build !embed

package web

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS exposes an on-disk fallback when the `embed` build tag is not set.
// The path is relative to the repo root, chosen to match the Vite output
// copied by `go generate ./internal/web`.
func FS() fs.FS {
	root := findRoot()
	return os.DirFS(filepath.Join(root, "internal", "web", "embed", "public"))
}

func findRoot() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}
