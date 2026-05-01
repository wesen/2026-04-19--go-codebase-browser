package review

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wesen/codebase-browser/internal/review"
	"github.com/wesen/codebase-browser/internal/static"
)

func newExportCmd() *cobra.Command {
	var (
		dbPath string
		outDir string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a review database to a standalone static directory",
		Long: `Export a review database to a standalone directory that can be opened
in a browser without a server. The output contains the SPA, WASM module,
pre-computed data, and optionally the raw SQLite database.

The export requires a review database produced by 'review index' or 'review db create'.

Examples:
  codebase-browser review export --db pr-42.db --out ./pr-42-export/
  codebase-browser review export --db review.db --out /tmp/export/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return fmt.Errorf("--db is required")
			}
			if outDir == "" {
				return fmt.Errorf("--out is required")
			}

			// 1. Load review database
			store, err := review.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open review db: %w", err)
			}
			defer store.Close()

			// 2. Load latest snapshot for regular index data
			loaded, err := review.LoadLatestSnapshot(cmd.Context(), store)
			if err != nil {
				return fmt.Errorf("load snapshot: %w", err)
			}

			// 3. Build regular precomputed data
			fmt.Fprintln(os.Stderr, "Building search index...")
			searchIdx := static.BuildSearchIndexFast(loaded)

			fmt.Fprintln(os.Stderr, "Building xref index...")
			xrefIdx := static.BuildXrefIndex(loaded)

			fmt.Fprintln(os.Stderr, "Extracting snippets...")
			sourceFS := os.DirFS(".")
			snippets, err := static.ExtractSnippets(loaded, sourceFS)
			if err != nil {
				return fmt.Errorf("extract snippets: %w", err)
			}

			fmt.Fprintln(os.Stderr, "Extracting snippet refs...")
			snippetRefs := static.ExtractSnippetRefs(loaded)

			fmt.Fprintln(os.Stderr, "Extracting source refs...")
			sourceRefs := static.ExtractSourceRefs(loaded)

			fmt.Fprintln(os.Stderr, "Building file xref index...")
			fileXrefIdx := static.BuildFileXrefIndex(loaded)

			// 4. Load review-specific data
			fmt.Fprintln(os.Stderr, "Loading review data for export...")
			reviewData, err := review.LoadForExport(dbPath)
			if err != nil {
				return fmt.Errorf("load review export data: %w", err)
			}

			// 5. Assemble merged precomputed.json
			rawIndex, _ := json.Marshal(loaded.Index)
			precomputed := map[string]interface{}{
				"version":       "1",
				"module":        loaded.Index.Module,
				"generatedAt":   loaded.Index.GeneratedAt,
				"indexJSON":     json.RawMessage(rawIndex),
				"searchIndex":   searchIdx,
				"xrefIndex":     xrefIdx,
				"fileXrefIndex": fileXrefIdx,
				"snippets":      snippets,
				"snippetRefs":   snippetRefs,
				"sourceRefs":    sourceRefs,
				"docManifest":   []interface{}{}, // Review docs are in reviewData
				"docHTML":       map[string]string{},
				"reviewData":    reviewData,
			}

			// 6. Build SPA
			fmt.Fprintln(os.Stderr, "Building SPA...")
			if err := buildSPA(); err != nil {
				return fmt.Errorf("build SPA: %w", err)
			}

			// 7. Create output directory and copy assets
			fmt.Fprintln(os.Stderr, "Copying assets to output directory...")
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return fmt.Errorf("create output dir: %w", err)
			}

			// Copy dist contents
			distDir := "dist"
			if err := copyTree(distDir, outDir); err != nil {
				return fmt.Errorf("copy dist: %w", err)
			}

			// Write merged precomputed.json
			pcPath := filepath.Join(outDir, "precomputed.json")
			pcData, err := json.MarshalIndent(precomputed, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal precomputed: %w", err)
			}
			if err := os.WriteFile(pcPath, pcData, 0644); err != nil {
				return fmt.Errorf("write precomputed.json: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Wrote precomputed.json (%.1f KB)\n", float64(len(pcData))/1024)

			// Optionally copy review.db
			dbOutPath := filepath.Join(outDir, "review.db")
			if err := copyFile(dbPath, dbOutPath); err != nil {
				return fmt.Errorf("copy review.db: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Copied review.db\n")

			fmt.Fprintf(os.Stderr, "\nExport complete: %s\n", outDir)
			fmt.Fprintf(os.Stderr, "Open file://%s/index.html in a browser\n", outDir)

			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to review database (required)")
	cmd.Flags().StringVar(&outDir, "out", "", "Output directory for the static export (required)")

	_ = cmd.MarkFlagRequired("db")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}

func buildSPA() error {
	cmd := exec.Command("pnpm", "-C", "ui", "run", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
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
