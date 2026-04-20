// Command build-ts-index invokes the Node-based TypeScript extractor and
// writes its JSON output into the repository's indexfs embed directory.
//
// The default code path runs the extractor inside a Dagger container, which
// matches the go-web-dagger-pnpm-build skill pattern used by cmd/build-web
// (CacheVolume for the pnpm store + corepack-activated pinned pnpm +
// frozen-lockfile install). When Dagger is unavailable — for example, on
// developer machines without Docker — set BUILD_TS_LOCAL=1 to shell out to
// a local pnpm + node instead; both paths produce identical JSON.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

const (
	defaultImage       = "node:22-bookworm"
	defaultPNPMVersion = "10.13.1"
	defaultOut         = "internal/indexfs/embed/index-ts.json"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "build-ts-index: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	modRoot, err := filepath.Abs(envDefault("TS_MODULE_ROOT", filepath.Join(repoRoot, "ui")))
	if err != nil {
		return fmt.Errorf("abs module-root: %w", err)
	}
	outPath, err := filepath.Abs(envDefault("TS_INDEX_OUT", filepath.Join(repoRoot, defaultOut)))
	if err != nil {
		return fmt.Errorf("abs out: %w", err)
	}
	tsconfig := envDefault("TS_TSCONFIG", "tsconfig.json")
	moduleName := envDefault("TS_MODULE_NAME", filepath.Base(modRoot))

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}

	if os.Getenv("BUILD_TS_LOCAL") == "1" {
		return runLocal(ctx, repoRoot, modRoot, tsconfig, moduleName, outPath)
	}
	// Opportunistic Dagger path — falls back to local if the engine is
	// unreachable. The daemon probe is synchronous but cheap.
	if err := runDagger(ctx, repoRoot, modRoot, tsconfig, moduleName, outPath); err != nil {
		if errors.Is(err, errDaggerUnavailable) {
			fmt.Fprintln(os.Stderr, "dagger unavailable, falling back to local pnpm")
			return runLocal(ctx, repoRoot, modRoot, tsconfig, moduleName, outPath)
		}
		return err
	}
	return nil
}

var errDaggerUnavailable = errors.New("dagger: engine not reachable")

func runDagger(ctx context.Context, repoRoot, modRoot, tsconfig, moduleName, outPath string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("%w: %v", errDaggerUnavailable, err)
	}
	defer func() { _ = client.Close() }()

	image := envDefault("TS_INDEXER_IMAGE", defaultImage)
	pnpmVer := envDefault("WEB_PNPM_VERSION", defaultPNPMVersion)

	// Mount only the two directories the extractor needs: tools/ts-indexer
	// (for the extractor code + its pinned dependencies) and the caller's
	// module root (the target project). Keeping the mounts narrow also
	// narrows the implicit attack surface if someone points the tool at an
	// untrusted project.
	indexerSrc := client.Host().Directory(filepath.Join(repoRoot, "tools", "ts-indexer"), dagger.HostDirectoryOpts{
		Exclude: []string{"node_modules", "bin"},
	})
	relMod, err := filepath.Rel(repoRoot, modRoot)
	if err != nil {
		relMod = filepath.Base(modRoot)
	}
	moduleSrc := client.Host().Directory(modRoot, dagger.HostDirectoryOpts{
		Exclude: []string{
			"node_modules",
			"dist",
			"storybook-static",
			".next",
			"*.tsbuildinfo",
		},
	})

	pnpmStore := client.CacheVolume("codebase-browser-ts-indexer-pnpm-store")
	pathEnv := "/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	// Ensure the module's own dependencies are installed so TypeScript can
	// resolve external imports while walking the AST. Without this step the
	// Compiler API emits "cannot find module" diagnostics for every import;
	// the extracted *symbols* are still correct but the logs are noisy.
	container := client.Container().
		From(image).
		WithEnvVariable("PNPM_HOME", "/pnpm").
		WithEnvVariable("PATH", pathEnv).
		WithMountedCache("/pnpm/store", pnpmStore).
		WithDirectory("/ts-indexer", indexerSrc).
		WithDirectory("/module", moduleSrc).
		WithWorkdir("/ts-indexer").
		WithExec([]string{"sh", "-lc", "corepack enable && corepack prepare pnpm@" + pnpmVer + " --activate"}).
		WithExec([]string{"pnpm", "install", "--frozen-lockfile", "--prefer-offline"}).
		WithExec([]string{"pnpm", "run", "build"}).
		WithWorkdir("/module").
		WithExec([]string{"sh", "-lc",
			"if [ -f package.json ]; then pnpm install --prefer-offline --ignore-scripts || true; fi"}).
		WithExec([]string{
			"node", "/ts-indexer/bin/cli.js",
			"--module-root", "/module",
			"--tsconfig", tsconfig,
			"--module-name", moduleName,
			"--out", "/out/index-ts.json",
		})

	if _, err := container.File("/out/index-ts.json").Export(ctx, outPath); err != nil {
		return fmt.Errorf("export index-ts.json: %w", err)
	}
	fmt.Fprintf(os.Stderr, "build-ts-index: wrote %s (module=%s, repoRoot=%s, relModule=%s)\n",
		outPath, moduleName, repoRoot, relMod)
	return nil
}

func runLocal(ctx context.Context, repoRoot, modRoot, tsconfig, moduleName, outPath string) error {
	indexerDir := filepath.Join(repoRoot, "tools", "ts-indexer")
	// Make sure the extractor is compiled (idempotent; tsc is fast on 5 files).
	binPath := filepath.Join(indexerDir, "bin", "cli.js")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		build := exec.CommandContext(ctx, "pnpm", "run", "build")
		build.Dir = indexerDir
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return fmt.Errorf("pnpm run build (local): %w", err)
		}
	}
	cmd := exec.CommandContext(ctx, "node", binPath,
		"--module-root", modRoot,
		"--tsconfig", tsconfig,
		"--module-name", moduleName,
		"--out", outPath,
	)
	cmd.Stdout = os.Stderr // CLI emits progress on stderr; the index file goes to --out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("node cli.js (local): %w", err)
	}
	fmt.Fprintf(os.Stderr, "build-ts-index (local): wrote %s (module=%s)\n", outPath, moduleName)
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
