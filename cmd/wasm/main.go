//go:build wasm

// Package main is the Go WASM entry point.
// Registers all JS exports from internal/wasm and blocks forever.
package main

import "github.com/wesen/codebase-browser/internal/wasm"

func main() {
	wasm.RegisterExports()
	// Block forever to prevent main from returning (which terminates WASM).
	select {}
}
