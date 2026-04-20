//go:build !embed

package docs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// PagesFS returns an on-disk view of internal/docs/embed/pages so `go run`
// dev picks up edits without rebuilding.
func PagesFS() fs.FS {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		path := filepath.Join(dir, "internal", "docs", "embed", "pages")
		if _, err := os.Stat(path); err == nil {
			return os.DirFS(path)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return os.DirFS(".")
}
