//go:build ignore

// generate_build.go is invoked by `go generate ./internal/indexfs`. It finds
// the repo root and runs `codebase-browser index build --lang auto`, which
// extracts the Go AST, runs the Node TS extractor (via Dagger by default or
// local pnpm when BUILD_TS_LOCAL=1 is set), merges both indexes, and writes
// the embedded internal/indexfs/embed/index.json that embed.go consumes.
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
	cmd := exec.Command("go", "run", "./cmd/codebase-browser",
		"index", "build", "--lang", "auto")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("generate_build: wrote internal/indexfs/embed/index.json")
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
