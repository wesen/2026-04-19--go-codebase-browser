//go:build ignore

// generate_build.go compiles internal/wasm/ to a WASM binary using TinyGo.
// Invoked by `go generate ./internal/wasm`.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}

	wasmPkg := filepath.Join(root, "cmd", "wasm")
	outPath := filepath.Join(root, "internal", "wasm", "embed", "search.wasm")

	// Ensure embed directory exists
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		log.Fatal(err)
	}

	// Build with standard Go targeting WASM (TinyGo doesn't support Go 1.22 yet)
	cmd := exec.Command("go", "build",
		"-o", outPath,
		wasmPkg,
	)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"GOOS=js",
		"GOARCH=wasm",
	)

	fmt.Println("GOOS=js GOARCH=wasm go build -o", outPath, wasmPkg)
	if err := cmd.Run(); err != nil {
		log.Fatal("wasm build failed:", err)
	}

	// Report size
	info, err := os.Stat(outPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("generate_build: wrote %s (%.1f KB)\n", outPath, float64(info.Size())/1024)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found (started from %s)", dir)
}
