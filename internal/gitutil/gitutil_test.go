package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repo with 3 commits.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %s", name, args, out)
		}
	}
	writeFile := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")

	writeFile("main.go", `package main
func main() { println("v1") }
`)
	run("git", "add", ".")
	run("git", "commit", "-m", "first commit")

	writeFile("main.go", `package main
func main() { println("v2") }
func greet() { println("hello") }
`)
	run("git", "add", ".")
	run("git", "commit", "-m", "second commit")

	writeFile("main.go", `package main
func main() { println("v3") }
func greet() { println("hello world") }
func farewell() { println("bye") }
`)
	run("git", "add", ".")
	run("git", "commit", "-m", "third commit")

	return dir
}
