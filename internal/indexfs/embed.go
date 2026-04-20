//go:build embed

package indexfs

import (
	_ "embed"
)

//go:embed embed/index.json
var indexJSON []byte

// Bytes returns the embedded index.json payload.
func Bytes() []byte { return indexJSON }
