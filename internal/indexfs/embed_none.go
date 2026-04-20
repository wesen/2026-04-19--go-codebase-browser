//go:build !embed

package indexfs

import (
	"os"
	"path/filepath"
)

// Bytes reads index.json from disk. Used for `go run` development before
// `go generate` or a tagged build has produced an embedded copy.
func Bytes() []byte {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		path := filepath.Join(dir, "internal", "indexfs", "embed", "index.json")
		if data, err := os.ReadFile(path); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}
