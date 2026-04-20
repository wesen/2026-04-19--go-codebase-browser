//go:build embed

package sourcefs

import (
	"embed"
	"io/fs"
)

//go:embed embed/source
var embeddedFS embed.FS

// FS exposes the embedded snapshot of the module's source tree.
func FS() fs.FS {
	sub, err := fs.Sub(embeddedFS, "embed/source")
	if err != nil {
		panic(err)
	}
	return sub
}
