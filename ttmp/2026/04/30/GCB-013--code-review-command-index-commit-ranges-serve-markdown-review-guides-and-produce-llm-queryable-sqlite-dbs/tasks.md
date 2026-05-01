# Tasks

## Preparation & Decisions

- [ ] **Task 0.1:** Resolve open questions before coding begins
  - Decide: should diffs be cached in review.db during `review index`? (affects Phase 2 design)
  - Decide: max practical commit range for export? Start with 10-20 commits, benchmark precomputed.json size
  - Decide: SPA route structure for review docs (`/review/:slug` vs `/docs/:slug`)
  - Decide: should `review index` also build a flat index.json in the DB for faster snapshot loading?
  - Decide: how to handle TypeScript extraction in review worktrees (node_modules may not exist at historical commits)
  - Owner: —
  - Depends on: —
  - See: design-doc/01 §10.3 Open Questions

---

## Phase 1: Foundation — Review Schema & Store

**Goal:** Create the `internal/review/` package with SQLite schema extensions and a unified store that wraps the history store.

- [x] **Task 1.1:** Refactor `internal/history/store.go` to support external DB connections
  - Add `NewFromDB(db *sql.DB) (*Store, error)` constructor
  - Ensure `configure(db)` is reusable or extracted
  - Keep backward compatibility with existing `Open(path)` and `Create(path)`
  - Validation: `go test ./internal/history/...` passes
  - Files: `internal/history/store.go`
  - Owner: —
  - Depends on: —
  - Est: 2–4h

- [x] **Task 1.2:** Create `internal/review/schema.go`
  - Define `dropReviewSchemaSQL` and `createReviewSchemaSQL` constants
  - Tables: `review_docs`, `review_doc_snippets` with all columns and indexes
  - Match column types to existing history schema conventions
  - Files: `internal/review/schema.go`
  - Owner: —
  - Depends on: Task 1.1
  - Est: 2–4h

- [x] **Task 1.3:** Create `internal/review/store.go`
  - `Store` struct wrapping `*sql.DB` and `*history.Store`
  - `Open(path string) (*Store, error)` — open existing review DB
  - `Create(path string) (*Store, error)` — create new, drop + recreate schema
  - `ResetSchema(ctx) error` — drop and recreate history + review tables
  - `DB() *sql.DB` and `Close() error`
  - Validation: `go test ./internal/review/...` passes
  - Files: `internal/review/store.go`, `internal/review/store_test.go`
  - Owner: —
  - Depends on: Task 1.2
  - Est: 4–6h

---

## Phase 2: Review Indexer — Commits + Docs → Unified DB

**Goal:** Implement `review.IndexReview` that indexes a git commit range and markdown docs into a single review database.

- [x] **Task 2.1:** Create `internal/review/indexer.go` — core indexing logic
  - Define `IndexOptions`, `IndexResult`, `IndexError` structs
  - Implement `IndexReview(ctx, store, opts) (*IndexResult, error)`
  - Phase 1: `gitutil.ParseCommitRange` to resolve commit range
  - Phase 2: delegate to `history.IndexCommits` for per-commit snapshot indexing
  - Phase 3: `discoverDocs(paths)` — resolve files and directories to .md file list
  - Phase 4: for each doc, render and store
  - Validation: `codebase-browser review db create --commits HEAD~2..HEAD --db /tmp/test.db` works
  - Files: `internal/review/indexer.go`
  - Owner: —
  - Depends on: Phase 1
  - Est: 6–10h

- [x] **Task 2.2:** Create `internal/review/loader.go` — snapshot reconstruction
  - Implement `loadLatestSnapshot(ctx, store) (*browser.Loaded, error)`
  - Query latest commit hash from `commits` table by `author_time DESC`
  - Reconstruct `*indexer.Index` from `snapshot_packages`, `snapshot_files`, `snapshot_symbols`, `snapshot_refs`
  - Return `browser.LoadFromBytes(marshalIndex(idx))`
  - Consider: add `history.Store.ReconstructIndex(commitHash)` helper for reuse
  - Validation: unit test with a fixture DB containing 2 commits
  - Files: `internal/review/loader.go`, `internal/review/loader_test.go`
  - Owner: —
  - Depends on: Task 2.1
  - Est: 4–6h

- [x] **Task 2.3:** Implement doc indexing in `internal/review/indexer.go`
  - `indexDoc(ctx, store, path, loaded)` function
  - Read markdown file, derive slug from basename
  - Call `docs.Render()` with the reconstructed `Loaded` index
  - Extract YAML frontmatter (if present) into `frontmatter_json`
  - Insert into `review_docs` with `ON CONFLICT(slug) DO UPDATE`
  - Insert all `SnippetRef` entries into `review_doc_snippets`
  - Validation: after indexing, `sqlite3` shows docs and snippets in DB
  - Files: `internal/review/indexer.go`
  - Owner: —
  - Depends on: Task 2.2
  - Est: 4–6h

- [x] **Task 2.4:** Add `IndexOptions.SkipDocs` and `IndexCommitsOnly`
  - Support `db create` skipping the doc phase entirely
  - Validation: `codebase-browser review db create --commits HEAD~2..HEAD --db /tmp/test.db` produces DB with no `review_docs` table rows
  - Files: `internal/review/indexer.go`
  - Owner: —
  - Depends on: Task 2.3
  - Est: 1–2h

---

## Phase 3: Review Server — HTTP Routes for Review Docs

**Goal:** Create `internal/review/server.go` that wraps the base `server.Server` and adds `/api/review/*` routes.

- [x] **Task 3.1:** Create `internal/review/server.go`
  - Define `ReviewServer` struct embedding `*server.Server` + `Store *Store`
  - `Handler() http.Handler` — mounts review routes before base routes
  - `handleReviewDocList` — SELECT slug, title, path, indexed_at FROM review_docs
  - `handleReviewDocPage` — SELECT content FROM review_docs WHERE slug = ?, then `docs.Render()`
  - `handleReviewCommits` — list commits from review DB
  - `handleReviewStats` — counts of commits, docs, snippets, symbols, files, refs
  - Validation: curl endpoints return valid JSON
  - Files: `internal/review/server.go`
  - Owner: —
  - Depends on: Phase 2
  - Est: 4–6h

---

## Phase 4: CLI Commands — `review index`, `review serve`, `review db create`

**Goal:** Wire up the `review` command tree with three subcommands.

- [x] **Task 4.1:** Create `cmd/codebase-browser/cmds/review/root.go`
  - `Register(root *cobra.Command) error` — registers the review command tree
  - Glazed command description with `cmds.NewCommandDescription`
  - Files: `cmd/codebase-browser/cmds/review/root.go`
  - Owner: —
  - Depends on: Phase 3
  - Est: 2–4h

- [x] **Task 4.2:** Create `cmd/codebase-browser/cmds/review/db.go`
  - `review db create` subcommand
  - Flags: `--commits`, `--db`, `--repo-root`, `--patterns`, `--include-tests`, `--parallelism`
  - Runs `review.Create` then `review.IndexCommitsOnly` (SkipDocs = true)
  - Glazed output row: db path, commits indexed, duration
  - Files: `cmd/codebase-browser/cmds/review/db.go`
  - Owner: —
  - Depends on: Task 4.1
  - Est: 2–4h

- [x] **Task 4.3:** Create `cmd/codebase-browser/cmds/review/index.go`
  - `review index` subcommand
  - Flags: all db.go flags + `--docs` (paths to markdown files or directories)
  - Runs `review.Create` then `review.IndexReview` (full: commits + docs)
  - Glazed output row: db path, commits, docs, snippets, duration
  - Files: `cmd/codebase-browser/cmds/review/index.go`
  - Owner: —
  - Depends on: Task 4.2
  - Est: 2–4h

- [x] **Task 4.4:** Create `cmd/codebase-browser/cmds/review/serve.go`
  - `review serve` subcommand
  - Flags: `--db`, `--addr` (default :3002), `--repo-root`
  - Opens review DB with `review.Open`
  - Loads latest snapshot with `review.LoadLatestSnapshot`
  - Constructs `review.ReviewServer`, starts `http.ListenAndServe`
  - Files: `cmd/codebase-browser/cmds/review/serve.go`
  - Owner: —
  - Depends on: Task 4.3
  - Est: 2–4h

- [x] **Task 4.5:** Register review command in `cmd/codebase-browser/main.go`
  - Add `cobra.CheckErr(review.Register(rootCmd))` after existing registrations
  - Files: `cmd/codebase-browser/main.go`
  - Owner: —
  - Depends on: Task 4.4
  - Est: 0.5h

- [x] **Task 4.6:** Integration test — full CLI workflow
  - Create temp git repo with 3 commits
  - Write a markdown review file with `codebase-snippet` directive
  - Run `review index`, `review serve`, `review db create`
  - Assert DB contents via sqlite3 CLI
  - Validation: `make test` passes, no regressions
  - Files: test script or Go integration test
  - Owner: —
  - Depends on: Task 4.5
  - Est: 4–6h

---

## Phase 5: Glazed Help Entries — Reference & User Guide

**Goal:** Write and embed Glazed help entries for the review feature.

- [x] **Task 5.1:** Create `docs/help/embed.go`
  - `//go:embed *.md` for help markdown files
  - `AddDocToHelpSystem(helpSystem)` registration function
  - Files: `docs/help/embed.go`
  - Owner: —
  - Depends on: —
  - Est: 1–2h

- [x] **Task 5.2:** Write `docs/help/review-reference.md`
  - Glazed frontmatter with Slug: `review-db-reference`, SectionType: `GeneralTopic`
  - Sections: overview, history tables, review tables, common SQL queries, symbol ID scheme, commit range syntax
  - Troubleshooting table and See Also cross-references
  - Files: `docs/help/review-reference.md`
  - Owner: —
  - Depends on: Task 5.1
  - Est: 2–4h

- [x] **Task 5.3:** Write `docs/help/review-user-guide.md`
  - Glazed frontmatter with Slug: `review-user-guide`, SectionType: `Tutorial`
  - Sections: quick start, writing review markdown files, available directives, commit ranges, sharing DBs, LLM queries
  - Troubleshooting table and See Also cross-references
  - Files: `docs/help/review-user-guide.md`
  - Owner: —
  - Depends on: Task 5.2
  - Est: 2–4h

- [x] **Task 5.4:** Wire help entries into `main.go`
  - Import `reviewhelp "github.com/wesen/codebase-browser/docs/help"`
  - Add `cobra.CheckErr(reviewhelp.AddDocToHelpSystem(helpSystem))`
  - Validation: `codebase-browser help review-db-reference` and `codebase-browser help review-user-guide` render correctly
  - Files: `cmd/codebase-browser/main.go`
  - Owner: —
  - Depends on: Task 5.3
  - Est: 1–2h

---

## Phase 6: Static WASM Export Foundation — Pre-computation & Bundler Fixes

**Goal:** Extend the existing GCB-006 static build to support history-aware data and fix paths for `file://` usage.

- [x] **Task 6.1:** Fix bundler paths for `file://` compatibility
  - Change `internal/bundle/generate_build.go` to use relative paths in HTML/JS output
  - Replace absolute `/search.wasm`, `/precomputed.json`, `/source/` with relative paths
  - Switch React `BrowserRouter` to `HashRouter` in `App.tsx` for file:// support
  - Validation: open `dist/index.html` directly in browser (no HTTP server), verify SPA loads
  - Files: `internal/bundle/generate_build.go`, `ui/src/app/App.tsx`
  - Owner: —
  - Depends on: —
  - Est: 2–4h

- [x] **Task 6.2:** Create `internal/review/export.go` — build-time pre-computation
  - Define `PrecomputedReview` struct with Commits, Diffs, Histories, Impacts, Docs
  - Implement `LoadForExport(dbPath) (*PrecomputedReview, error)`
  - Query review.db for commits, then compute diffs/histories/impacts
  - Render all review docs to HTML
  - Marshal to JSON
  - Files: `internal/review/export.go`
  - Owner: —
  - Depends on: Phase 2 (review indexer produces the DB)
  - Est: 4–6h

- [x] **Task 6.3:** Pre-compute diffs at export time
  - For each adjacent commit pair in the review range, run diff computation
  - Reuse `internal/history/diff.go` logic or add diff caching during `review index`
  - Store in `PrecomputedReview.Diffs` keyed by `"oldHash..newHash"`
  - Validation: export JSON contains correct diff entries
  - Files: `internal/review/export.go`
  - Owner: —
  - Depends on: Task 6.2
  - Est: 4–6h

- [x] **Task 6.4:** Pre-compute symbol histories at export time
  - For each symbol that appears in >1 commit, build timeline of body_hash/signature changes
  - Store in `PrecomputedReview.Histories`
  - Validation: export JSON contains history for symbols that changed
  - Files: `internal/review/export.go`
  - Owner: —
  - Depends on: Task 6.3
  - Est: 2–4h

- [x] **Task 6.5:** Pre-compute impact analysis at export time
  - For each symbol referenced in review docs, run BFS over `snapshot_refs`
  - Store in `PrecomputedReview.Impacts`
  - Validation: export JSON contains impact nodes
  - Files: `internal/review/export.go`
  - Owner: —
  - Depends on: Task 6.4
  - Est: 2–4h

---

## Phase 7: TinyGo WASM History Exports

**Goal:** Extend `internal/wasm/search.go` and `exports.go` to support history query exports.

- [x] **Task 7.1:** Add review data structures to `internal/wasm/search.go`
  - Add `ReviewData *PrecomputedReview` field to `SearchCtx`
  - Define lightweight WASM-compatible types: `CommitLite`, `CommitDiffLite`, `HistoryEntryLite`, `ImpactLite`, `ReviewDocLite`
  - Add `LoadReviewData(jsonData []byte) error` method
  - Files: `internal/wasm/search.go`, `internal/wasm/index_types.go`
  - Owner: —
  - Depends on: Phase 6
  - Est: 2–4h

- [x] **Task 7.2:** Implement history query methods in `SearchCtx`
  - `GetCommitDiff(oldHash, newHash) []byte`
  - `GetSymbolHistory(symbolID) []byte`
  - `GetImpact(symbolID, direction, depth) []byte`
  - `GetReviewDocs() []byte`
  - `GetReviewDoc(slug) []byte`
  - All return JSON-encoded data (or `"null"`)
  - Files: `internal/wasm/search.go`
  - Owner: —
  - Depends on: Task 7.1
  - Est: 2–4h

- [x] **Task 7.3:** Register new exports in `internal/wasm/exports.go`
  - Update `initWasm` to accept a 7th parameter for review JSON data
  - Register `getCommitDiff`, `getSymbolHistory`, `getImpact`, `getReviewDocs`, `getReviewDoc`
  - Files: `internal/wasm/exports.go`
  - Owner: —
  - Depends on: Task 7.2
  - Est: 2–4h

- [x] **Task 7.4:** Update frontend WASM client
  - Extend `ui/src/api/wasmClient.ts` to load `precomputed.json` including review data
  - Add RTK-Query endpoints for history queries (or extend existing endpoints)
  - Validation: `pnpm -C ui run typecheck` passes
  - Files: `ui/src/api/wasmClient.ts`
  - Owner: —
  - Depends on: Task 7.3
  - Est: 2–4h

- [x] **Task 7.5:** Build and test TinyGo WASM with history exports
  - Run `go generate ./internal/wasm`
  - Verify `search.wasm` builds with TinyGo (not falling back to standard Go)
  - Test in browser with a fixture precomputed.json containing review data
  - Files: `internal/wasm/embed/search.wasm`
  - Owner: —
  - Depends on: Task 7.4
  - Est: 2–4h

---

## Phase 8: sql.js Integration — Optional SQL Console (Optional)

**Goal:** Add sql.js for ad-hoc SQL/LLM queries. This phase is optional — the core review export works without it.

- [ ] **Task 8.1:** Add sql.js as a dependency
  - `pnpm -C ui add sql.js` or use CDN
  - Files: `ui/package.json`
  - Owner: —
  - Depends on: Phase 7
  - Est: 0.5h

- [ ] **Task 8.2:** Create `ui/src/features/review/SqlConsole.tsx`
  - Lazy-load sql.js assets only when console is opened
  - Fetch `review.db` as ArrayBuffer
  - Initialize sql.js database
  - Text input for SQL query, table output for results
  - Error display for invalid SQL
  - Files: `ui/src/features/review/SqlConsole.tsx`
  - Owner: —
  - Depends on: Task 8.1
  - Est: 4–6h

- [ ] **Task 8.3:** Add SQL console route or modal
  - Route: `/review/sql` or modal triggered from review doc page
  - Only shown when `review.db` is present in the export
  - Files: `ui/src/app/App.tsx` or relevant page component
  - Owner: —
  - Depends on: Task 8.2
  - Est: 1–2h

- [ ] **Task 8.4:** Include sql.js assets in bundler
  - Update `internal/bundle/generate_build.go` to copy sql.js WASM + JS files
  - Ensure `review.db` is copied to export directory
  - Files: `internal/bundle/generate_build.go`
  - Owner: —
  - Depends on: Task 8.3
  - Est: 1–2h

---

## Phase 9: `review export` CLI Command

**Goal:** Wire everything into a single `review export` command that produces a standalone directory.

- [x] **Task 9.1:** Create `cmd/codebase-browser/cmds/review/export.go`
  - `review export` subcommand
  - Flags: `--commits`, `--docs`, `--db`, `--out`, `--repo-root`, `--patterns`, `--include-tests`, `--parallelism`
  - If `--db` provided, skip indexing and use existing DB
  - Otherwise, run `review index` internally first
  - Call `review.LoadForExport` to build precomputed JSON
  - Run `make build-static` (or equivalent Vite build with `VITE_STATIC_EXPORT=1`)
  - Copy all assets to `--out` directory
  - Files: `cmd/codebase-browser/cmds/review/export.go`
  - Owner: —
  - Depends on: Phase 7 (and optionally Phase 8)
  - Est: 4–6h

- [x] **Task 9.2:** Add `VITE_STATIC_EXPORT` environment variable to SPA build
  - In Vite config or `ui/src/api/store.ts`, detect static export mode
  - Use `wasmBaseQuery` instead of `fetchBaseQuery` for all API calls
  - Files: `ui/src/api/store.ts`, `ui/vite.config.ts`
  - Owner: —
  - Depends on: Task 9.1
  - Est: 1–2h

---

## Phase 10: Integration & End-to-End Testing

**Goal:** Verify the full workflow on real commit ranges.

- [x] **Task 10.1:** End-to-end test — server-based workflow
  - `codebase-browser review index --commits HEAD~5..HEAD --docs ./reviews/ --db /tmp/e2e.db`
  - `codebase-browser review serve --db /tmp/e2e.db --addr :3003 &`
  - Verify: `/api/review/docs`, `/api/review/docs/<slug>`, `/api/review/commits`, `/api/review/stats`
  - Open browser, verify widgets hydrate correctly
  - Files: manual test or script
  - Owner: —
  - Depends on: Phases 1–5
  - Est: 4–6h

- [x] **Task 10.2:** End-to-end test — static export workflow
  - `codebase-browser review export --commits HEAD~5..HEAD --docs ./reviews/ --out /tmp/e2e-export/`
  - Open `/tmp/e2e-export/index.html` directly in browser (file://)
  - Or serve via `python3 -m http.server`
  - Verify: review docs render, diff widgets show correct code, history timeline works, impact analysis works
  - Verify offline: disconnect network, refresh page, everything still works
  - Verify zip portability: zip the export, unzip elsewhere, open index.html
  - Files: manual test or script
  - Owner: —
  - Depends on: Phases 6–9
  - Est: 4–6h

- [x] **Task 10.3:** End-to-end test — sql.js console (if Phase 8 implemented)
  - Skipped: Phase 8 (sql.js) is optional and not implemented
  - Open SQL console in exported artifact
  - Run: `SELECT COUNT(*) FROM commits;`
  - Run: `SELECT name, signature FROM snapshot_symbols WHERE commit_hash = '...';`
  - Verify results display correctly
  - Files: manual test
  - Owner: —
  - Depends on: Phase 8
  - Est: 2–4h

- [x] **Task 10.4:** Performance benchmarking
  - Measure `precomputed.json` size for various commit ranges (5, 10, 20, 50 commits)
  - Measure TinyGo `search.wasm` binary size with history exports
  - Measure initial page load time in browser
  - Document max practical commit range
  - Files: benchmark script or notes
  - Owner: —
  - Depends on: Tasks 10.1–10.2
  - Est: 2–4h

- [x] **Task 10.5:** Final handoff — update ticket docs
  - Update diary with what worked, what didn't, exact errors, solutions
  - Update changelog with all implementation steps
  - Run `docmgr doctor --ticket GCB-013`
  - Upload final design + implementation bundle to reMarkable
  - Files: ticket reference docs
  - Owner: —
  - Depends on: All above
  - Est: 2–4h

---

## Completed

- [x] **Task 0.0:** Create docmgr ticket GCB-013
- [x] **Task 0.1a:** Write comprehensive design doc (server-based architecture) — 63KB, 11 sections
- [x] **Task 0.1b:** Write standalone WASM export design doc — 32KB, comparison of 3 approaches
- [x] **Task 0.1c:** Write TinyGo vs sql.js feasibility assessment — resolves open questions, recommends TinyGo default + optional sql.js
- [x] **Task 0.1d:** Update ticket bookkeeping — relate files, update changelog, resolve vocabulary, pass `docmgr doctor`
- [x] **Task 0.1e:** Upload 3-document bundle to reMarkable at `/ai/2026/04/30/GCB-013`
