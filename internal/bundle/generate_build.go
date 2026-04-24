//go:build ignore

// generate_build.go bundles the static WASM artifact.
// It builds the SPA, copies all assets into dist/, and injects wasm_exec.js
// into the HTML entry point.
// Invoked by `go generate ./internal/bundle`.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}

	distDir := filepath.Join(root, "dist")
	if err := os.RemoveAll(distDir); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		log.Fatal(err)
	}

	// 1. Build SPA
	fmt.Println("Building SPA...")
	if err := buildSPA(root); err != nil {
		log.Fatal("build SPA:", err)
	}

	// 2. Copy SPA build output to dist/
	spaDir := filepath.Join(root, "ui", "dist", "public")
	if err := copyTree(spaDir, distDir); err != nil {
		log.Fatal("copy SPA:", err)
	}
	fmt.Println("Copied SPA to", distDir)

	// 3. Copy WASM
	wasmSrc := filepath.Join(root, "internal", "wasm", "embed", "search.wasm")
	wasmDst := filepath.Join(distDir, "search.wasm")
	if err := copyFile(wasmSrc, wasmDst); err != nil {
		log.Fatal("copy WASM:", err)
	}
	fmt.Println("Copied search.wasm")

	// 4. Copy precomputed.json
	pcSrc := filepath.Join(root, "internal", "static", "embed", "precomputed.json")
	pcDst := filepath.Join(distDir, "precomputed.json")
	if err := copyFile(pcSrc, pcDst); err != nil {
		log.Fatal("copy precomputed:", err)
	}
	fmt.Println("Copied precomputed.json")

	// 5. Copy source tree
	sourceSrc := filepath.Join(root, "internal", "sourcefs", "embed", "source")
	sourceDst := filepath.Join(distDir, "source")
	if err := copyTree(sourceSrc, sourceDst); err != nil {
		log.Fatal("copy source:", err)
	}
	fmt.Println("Copied source tree")

	// 6. Copy wasm_exec.js (TinyGo version, NOT Go's)
	// TinyGo's wasm_exec.js lives in the project at ui/public/wasm_exec.js
	// (downloaded from TinyGo release). Fallback to Go's version.
	wasmExecDst := filepath.Join(distDir, "wasm_exec.js")
	wasmExecSrc := "" // will be set below
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
		log.Fatal("wasm_exec.js not found")
	}
	if err := copyFile(wasmExecSrc, wasmExecDst); err != nil {
		log.Fatal("copy wasm_exec.js:", err)
	}
	fmt.Println("Copied wasm_exec.js from", wasmExecSrc)

	// 7. Inject wasm_exec.js into index.html
	indexPath := filepath.Join(distDir, "index.html")
	if err := injectWasmExec(indexPath); err != nil {
		log.Fatal("inject wasm_exec:", err)
	}
	fmt.Println("Injected wasm_exec.js into index.html")

	// Report
	reportSize(distDir)
	fmt.Println("\nStatic artifact ready at:", distDir)
	fmt.Println("Open file://" + distDir + "/index.html in a browser")
}

func buildSPA(root string) error {
	cmd := exec.Command("pnpm", "-C", "ui", "run", "build")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
			return os.MkdirAll(target, 0755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
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

	// Inject wasm_exec.js before the first script tag
	inject := `<script src="/wasm_exec.js"></script>`
	if strings.Contains(html, `<script type="module"`) {
		html = strings.Replace(html, `<script type="module"`, inject+`
  <script type="module"`, 1)
	} else if strings.Contains(html, `<script`) {
		html = strings.Replace(html, `<script`, inject+`
  <script`, 1)
	} else {
		// Append before closing body
		html = strings.Replace(html, "</body>", inject+"\n</body>", 1)
	}

	return os.WriteFile(indexPath, []byte(html), 0644)
}

func reportSize(dir string) {
	var total int64
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, _ := d.Info()
		if info != nil {
			total += info.Size()
			rel, _ := filepath.Rel(dir, p)
			fmt.Printf("  %-40s %8.1f KB\n", rel, float64(info.Size())/1024)
		}
		return nil
	})
	fmt.Printf("  %-40s %8.1f KB\n", "TOTAL", float64(total)/1024)
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
