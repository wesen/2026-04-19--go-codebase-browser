// Package indexfs also hosts generator directives. Keep `go generate`
// hygienic: each directive lives here, not sprinkled across other files.
package indexfs

//go:generate go run generate_build.go
