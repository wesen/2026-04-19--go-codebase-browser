//go:build ignore

// generate_build.go is invoked by `go generate ./internal/web`. It builds the
// Vite SPA and copies ui/dist/public/* into internal/web/embed/public/.
package main

import (
	"fmt"
	"io"
	"io/fs"
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
	if err := run(root, "pnpm", "-C", "ui", "run", "build"); err != nil {
		log.Fatal(err)
	}
	src := filepath.Join(root, "ui", "dist", "public")
	dst := filepath.Join(root, "internal", "web", "embed", "public")
	if err := recreate(dst); err != nil {
		log.Fatal(err)
	}
	if err := copyTree(src, dst); err != nil {
		log.Fatal(err)
	}
	fmt.Println("generate_build: copied", src, "->", dst)
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
	return "", fmt.Errorf("go.mod not found from %s", dir)
}

func run(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func recreate(dir string) error {
	// Preserve .keep files the way the repo's .gitignore expects.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() == ".keep" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return os.MkdirAll(dir, 0o755)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
