//go:build !embed

package sourcefs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FS returns an on-disk view of the repo root, so `go run` can serve source
// without running `go generate` first.
func FS() fs.FS {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.DirFS(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return os.DirFS(".")
}
