package review

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/wesen/codebase-browser/internal/review"
	"github.com/wesen/codebase-browser/internal/server"
	"github.com/wesen/codebase-browser/internal/web"
)

func newServeCmd() *cobra.Command {
	var (
		dbPath   string
		addr     string
		repoRoot string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve review docs and history API from a review database",
		Long: `Start an HTTP server that serves review documents and the history API.

The server reads from a review database produced by 'review index' or 'review db create'.
It serves the React SPA, the base API (index, symbol, source, xref), and review-specific
routes (/api/review/docs, /api/review/commits, /api/review/stats).

Examples:
  codebase-browser review serve --db pr-42.db --addr :3002
  codebase-browser review serve --db review.db --repo-root .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if repoRoot == "" {
				repoRoot = "."
			}

			store, err := review.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open review db: %w", err)
			}
			defer store.Close()

			loaded, err := review.LoadLatestSnapshot(ctx, store)
			if err != nil {
				return fmt.Errorf("load latest snapshot: %w", err)
			}

			sourceFS := os.DirFS(repoRoot)
			spaFS := web.FS()

			baseSrv := server.New(loaded, sourceFS, spaFS, nil, nil)
			baseSrv.History = store.History
			baseSrv.RepoRoot = repoRoot

			rs := review.NewReviewServer(baseSrv, store)
			h := rs.Handler()

			fmt.Fprintf(os.Stderr, "review server listening on %s\n", addr)
			fmt.Fprintf(os.Stderr, "  db: %s\n", dbPath)
			fmt.Fprintf(os.Stderr, "  commits: latest from snapshot\n")
			fmt.Fprintf(os.Stderr, "  docs: %d in review_docs\n", countDocs(store))

			srv := &http.Server{Addr: addr, Handler: h}
			return srv.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "review.db", "Path to review database")
	cmd.Flags().StringVar(&addr, "addr", ":3002", "Bind address")
	cmd.Flags().StringVar(&repoRoot, "repo-root", ".", "Path to git repository root")

	return cmd
}

func countDocs(store *review.Store) int {
	var count int
	_ = store.DB().QueryRow(`SELECT COUNT(*) FROM review_docs`).Scan(&count)
	return count
}

func defaultPatterns() []string {
	return []string{"./cmd/...", "./internal/..."}
}
