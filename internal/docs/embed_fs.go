//go:build embed

package docs

import (
	"embed"
	"io/fs"
)

//go:embed embed/pages
var pagesFS embed.FS

// PagesFS returns the embedded doc pages filesystem.
func PagesFS() fs.FS {
	sub, err := fs.Sub(pagesFS, "embed/pages")
	if err != nil {
		panic(err)
	}
	return sub
}
