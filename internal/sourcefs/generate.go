// Package sourcefs also hosts generator directives. Keep `go generate`
// hygienic: each directive lives here, not sprinkled across other files.
package sourcefs

//go:generate go run generate_build.go
