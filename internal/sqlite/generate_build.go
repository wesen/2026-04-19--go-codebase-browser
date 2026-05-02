//go:build ignore

// generate_build.go is invoked by `go generate ./internal/sqlite`. It reads
// the generated index.json and writes internal/sqlite/embed/codebase.db.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-go-golems/codebase-browser/internal/indexer"
	cbsqlite "github.com/go-go-golems/codebase-browser/internal/sqlite"
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}

	indexPath := filepath.Join(root, "internal", "indexfs", "embed", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		log.Fatalf("read index: %v", err)
	}
	var idx indexer.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		log.Fatalf("decode index: %v", err)
	}

	outDir := filepath.Join(root, "internal", "sqlite", "embed")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create sqlite embed dir: %v", err)
	}
	outPath := filepath.Join(outDir, "codebase.db")
	_ = os.Remove(outPath)

	store, err := cbsqlite.Create(outPath)
	if err != nil {
		log.Fatalf("create sqlite db: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.LoadFromIndex(ctx, &idx); err != nil {
		log.Fatalf("load index into sqlite: %v", err)
	}
	if err := store.EnableFTS5(ctx); err != nil {
		log.Fatalf("enable fts5: %v", err)
	}

	fmt.Printf("generate_build: wrote %s (%d packages, %d files, %d symbols, %d refs)\n",
		outPath, len(idx.Packages), len(idx.Files), len(idx.Symbols), len(idx.Refs))
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
