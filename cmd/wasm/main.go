//go:build wasm

// Package main is the Go WASM entry point compiled with TinyGo.
// RegisterExports() sets up window.codebaseBrowser, then main
// blocks on a channel so the WASM instance stays alive for callbacks.
package main

import "github.com/wesen/codebase-browser/internal/wasm"

func main() {
	wasm.RegisterExports()
	// Block forever using a channel. TinyGo keeps the goroutine alive
	// so JS can call the registered callbacks at any time.
	<-make(chan struct{})
}
