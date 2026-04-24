package builtinconcepts

import "embed"

// Files contains the built-in structured SQL concept catalog.
//
//go:embed **/*.sql
var Files embed.FS
