//go:build ignore

// generate_build.go compiles cmd/wasm/ to a WASM binary using TinyGo.
//
// Build strategy (in order of preference):
//   1. Dagger container with tinygo/tinygo image (reproducible everywhere)
//   2. Local tinygo binary (if available and version >= 0.31)
//   3. Standard Go WASM as fallback (larger binaries)
//
// Invoked by `go generate ./internal/wasm`.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

const (
	defaultTinyGoImage = "tinygo/tinygo:0.41.1"
	wasmPkg            = "./cmd/wasm"
	wasmOut            = "search.wasm"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	root, err := findRepoRoot()
	if err != nil {
		return err
	}

	outPath := filepath.Join(root, "internal", "wasm", "embed", wasmOut)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	if os.Getenv("BUILD_WASM_LOCAL") == "1" {
		return runLocal(root, outPath)
	}

	if err := runDagger(ctx, root, outPath); err != nil {
		if errors.Is(err, errDaggerUnavailable) {
			fmt.Fprintln(os.Stderr, "dagger unavailable, falling back to local build")
			return runLocal(root, outPath)
		}
		return err
	}
	return nil
}

var errDaggerUnavailable = errors.New("dagger: engine not reachable")

// runDagger compiles the WASM inside a TinyGo container via Dagger.
// This works on any machine with Docker — no local TinyGo install needed.
func runDagger(ctx context.Context, root, outPath string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("%w: %v", errDaggerUnavailable, err)
	}
	defer func() { _ = client.Close() }()

	image := envDefault("WASM_TINYGO_IMAGE", defaultTinyGoImage)

	// Mount only the WASM module source + its dependency on internal/wasm types.
	// The TinyGo container has its own Go stdlib, so we only need our source.
	src := client.Host().Directory(root, dagger.HostDirectoryOpts{
		Exclude: []string{
			"dist", "node_modules", ".git", ".tools",
			"ui/node_modules", "ui/dist", "ui/storybook-static",
		},
	})

	// Build with TinyGo inside the container.
	// Output to /tmp/ which is always writable.
	container := client.Container().
		From(image).
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{
			"tinygo", "build",
			"-target", "wasm",
			"-scheduler", "none",
			"-o", "/tmp/" + wasmOut,
			wasmPkg,
		})

	// Export the compiled WASM back to the host
	if _, err := container.File("/tmp/" + wasmOut).Export(ctx, outPath); err != nil {
		return fmt.Errorf("export wasm: %w", err)
	}

	reportSize(outPath, "TinyGo (dagger)")
	return nil
}

// runLocal tries TinyGo from PATH or .tools, then falls back to standard Go WASM.
func runLocal(root, outPath string) error {
	pkgPath := filepath.Join(root, "cmd", "wasm")

	// Try TinyGo first
	if tryTinyGo(pkgPath, outPath, root) {
		reportSize(outPath, "TinyGo (local)")
		return nil
	}

	// Fallback to standard Go WASM
	fmt.Fprintln(os.Stderr, "TinyGo unavailable or incompatible, falling back to standard Go WASM")
	cmd := exec.Command("go", "build", "-o", outPath, pkgPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	fmt.Println("GOOS=js GOARCH=wasm go build -o", outPath, pkgPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go wasm build: %w", err)
	}
	reportSize(outPath, "Go (local)")
	return nil
}

func tryTinyGo(pkgPath, outPath, root string) bool {
	tinygo := findTinyGo(root)
	if tinygo == "" {
		return false
	}

	// Check version
	out, err := exec.Command(tinygo, "version").Output()
	if err != nil {
		return false
	}
	versionStr := string(out)
	fmt.Println("Found:", strings.TrimSpace(versionStr))

	// TinyGo 0.31+ supports Go 1.22+
	parts := strings.Fields(versionStr)
	if len(parts) >= 3 {
		verParts := strings.Split(parts[2], ".")
		if len(verParts) >= 2 && verParts[0] == "0" {
			minorNum := 0
			fmt.Sscanf(verParts[1], "%d", &minorNum)
			if minorNum < 31 {
				fmt.Printf("TinyGo %s.%s too old (need 0.31+)\n", verParts[0], verParts[1])
				return false
			}
		}
	}

	cmd := exec.Command(tinygo, "build", "-target", "wasm", "-scheduler", "none", "-o", outPath, pkgPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set TINYGOROOT for locally-extracted .tools layout
	if strings.Contains(tinygo, ".tools") {
		cmd.Env = append(os.Environ(), "TINYGOROOT="+filepath.Join(root, ".tools", "lib"))
	}
	fmt.Println("tinygo build -target wasm -scheduler none -o", outPath, pkgPath)
	if err := cmd.Run(); err != nil {
		fmt.Println("TinyGo build failed:", err)
		return false
	}
	return true
}

func findTinyGo(root string) string {
	localTinygo := filepath.Join(root, ".tools", "tinygo")
	if _, err := os.Stat(localTinygo); err == nil {
		return localTinygo
	}
	if path, err := exec.LookPath("tinygo"); err == nil {
		return path
	}
	return ""
}

func reportSize(outPath, compiler string) {
	info, err := os.Stat(outPath)
	if err != nil {
		log.Fatal(err)
	}
	data, _ := os.ReadFile(outPath)
	gzSize := 0
	if len(data) > 0 {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(data)
		w.Close()
		gzSize = buf.Len()
	}
	fmt.Printf("generate_build: wrote %s (%.1f KB, %.1f KB gzipped) [%s]\n",
		outPath, float64(info.Size())/1024, float64(gzSize)/1024, compiler)
}

func envDefault(k, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fallback
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
