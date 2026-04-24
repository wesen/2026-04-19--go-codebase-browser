//go:build ignore

// generate_build.go is invoked by `go generate ./internal/web`. It builds the
// Vite SPA and copies the built assets into internal/web/embed/public/.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

const (
	defaultImage       = "node:22-bookworm"
	defaultPNPMVersion = "10.13.1"
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
	if os.Getenv("BUILD_WEB_LOCAL") == "1" {
		return runLocal(root)
	}
	if err := runDagger(ctx, root); err != nil {
		if errors.Is(err, errDaggerUnavailable) {
			fmt.Fprintln(os.Stderr, "dagger unavailable, falling back to local pnpm")
			return runLocal(root)
		}
		return err
	}
	return nil
}

var errDaggerUnavailable = errors.New("dagger: engine not reachable")

func runDagger(ctx context.Context, root string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("%w: %v", errDaggerUnavailable, err)
	}
	defer func() { _ = client.Close() }()

	image := envDefault("WEB_BUILDER_IMAGE", defaultImage)
	pnpmVersion := envDefault("WEB_PNPM_VERSION", readPNPMVersion(filepath.Join(root, "ui", "package.json")))

	uiSrc := client.Host().Directory(filepath.Join(root, "ui"), dagger.HostDirectoryOpts{
		Exclude: []string{"node_modules", "dist", "storybook-static"},
	})
	pnpmStore := client.CacheVolume("codebase-browser-ui-pnpm-store")
	pathEnv := "/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	container := client.Container().
		From(image).
		WithEnvVariable("PNPM_HOME", "/pnpm").
		WithEnvVariable("PATH", pathEnv).
		WithMountedCache("/pnpm/store", pnpmStore).
		WithDirectory("/ui", uiSrc).
		WithWorkdir("/ui").
		WithExec([]string{"sh", "-lc", "corepack enable && corepack prepare pnpm@" + pnpmVersion + " --activate"}).
		WithExec([]string{"pnpm", "install", "--frozen-lockfile", "--prefer-offline"}).
		WithExec([]string{"pnpm", "run", "build"})

	tmpSrc, err := os.MkdirTemp("", "codebase-browser-ui-dist-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpSrc)

	if _, err := container.Directory("/ui/dist/public").Export(ctx, tmpSrc); err != nil {
		return fmt.Errorf("export ui dist: %w", err)
	}

	dst := filepath.Join(root, "internal", "web", "embed", "public")
	if err := recreate(dst); err != nil {
		return err
	}
	if err := copyTree(tmpSrc, dst); err != nil {
		return err
	}
	if err := copyServeAssets(root, dst); err != nil {
		return err
	}
	fmt.Println("generate_build: copied", tmpSrc, "->", dst)
	return nil
}

func runLocal(root string) error {
	if err := runCmd(root, "pnpm", "-C", "ui", "run", "build"); err != nil {
		return err
	}
	src := filepath.Join(root, "ui", "dist", "public")
	dst := filepath.Join(root, "internal", "web", "embed", "public")
	if err := recreate(dst); err != nil {
		return err
	}
	if err := copyTree(src, dst); err != nil {
		return err
	}
	if err := copyServeAssets(root, dst); err != nil {
		return err
	}
	fmt.Println("generate_build: copied", src, "->", dst)
	return nil
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

func envDefault(k, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fallback
}

func readPNPMVersion(packageJSON string) string {
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return defaultPNPMVersion
	}
	var payload struct {
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return defaultPNPMVersion
	}
	if payload.PackageManager == "" {
		return defaultPNPMVersion
	}
	if v := strings.TrimPrefix(payload.PackageManager, "pnpm@"); v != payload.PackageManager {
		return v
	}
	return defaultPNPMVersion
}

func runCmd(dir string, name string, args ...string) error {
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

func copyServeAssets(root, dst string) error {
	if err := copyFile(filepath.Join(root, "internal", "wasm", "embed", "search.wasm"), filepath.Join(dst, "search.wasm")); err != nil {
		return fmt.Errorf("copy search.wasm: %w", err)
	}
	if err := copyFile(filepath.Join(root, "internal", "static", "embed", "precomputed.json"), filepath.Join(dst, "precomputed.json")); err != nil {
		return fmt.Errorf("copy precomputed.json: %w", err)
	}

	wasmExecSrc := ""
	for _, candidate := range []string{
		filepath.Join(root, "ui", "public", "wasm_exec.js"),
		filepath.Join(root, "internal", "wasm", "embed", "wasm_exec.js"),
		filepath.Join(os.Getenv("GOROOT"), "lib", "wasm", "wasm_exec.js"),
		"/usr/local/go/lib/wasm/wasm_exec.js",
	} {
		if _, err := os.Stat(candidate); err == nil {
			wasmExecSrc = candidate
			break
		}
	}
	if wasmExecSrc == "" {
		return fmt.Errorf("wasm_exec.js not found")
	}
	if err := copyFile(wasmExecSrc, filepath.Join(dst, "wasm_exec.js")); err != nil {
		return fmt.Errorf("copy wasm_exec.js: %w", err)
	}
	if err := injectWasmExec(filepath.Join(dst, "index.html")); err != nil {
		return fmt.Errorf("inject wasm_exec.js: %w", err)
	}
	return nil
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

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func injectWasmExec(indexPath string) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	html := string(data)
	inject := `<script src="/wasm_exec.js"></script>`
	if strings.Contains(html, `<script type="module"`) {
		html = strings.Replace(html, `<script type="module"`, inject+`
  <script type="module"`, 1)
	} else if strings.Contains(html, `<script`) {
		html = strings.Replace(html, `<script`, inject+`
  <script`, 1)
	} else {
		html = strings.Replace(html, "</body>", inject+"\n</body>", 1)
	}
	return os.WriteFile(indexPath, []byte(html), 0o644)
}
