---
Title: Investigation Diary
Ticket: GCB-013
Status: active
Topics:
    - codebase-browser
    - pr-review
    - code-review
    - sqlite-index
    - markdown-docs
    - literate-programming
    - glazed-help
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological diary of the GCB-013 design and investigation process.
LastUpdated: 2026-04-30T12:00:00Z
WhatFor: Track design decisions, evidence gathering, and open questions for the code review tool ticket
WhenToUse: Read when continuing implementation or when design decisions need context
---

# Diary

## Goal

Capture the step-by-step investigation and design process for GCB-013: turning codebase-browser into a standalone code review tool with a unified SQLite database, markdown review guide serving, and Glazed help documentation.

## Step 1: Ticket creation and repo exploration

The user asked to create a new docmgr ticket for a code review tool feature and to write a detailed analysis/design/implementation guide before any code is written. The feature idea: point at a commit range and markdown files, index into a DB, serve the markdown files with widget support, and provide a verb to create the DB alone for LLM consumption.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to turn this into a code review tool. The idea is the following: - you point it at a range of commits and a set of markdown files, and it indexes it into a db, and serves the given md files (or directory), which are literate programming markdown files allowing pointing out different things. It also gives a verb to create that sqlite db on its own so that it can easily be queried and used as code review basis for an LLM. For that, we also need a proper reference and user guide documentation in the glazed help entry format. Before you implement anything, create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and detailed. Store in the ticket and then upload to remarkable."

**Assistant interpretation:** Create a docmgr ticket GCB-013, write a comprehensive design doc for a new intern covering the entire existing architecture plus the new review command feature, then upload to reMarkable.

**Inferred user intent:** The user wants a thorough design artifact that can be handed to a new team member or used as the basis for implementation. The emphasis on "before you implement anything" means this is a design-first phase.

### What I did
- Read the ticket-research-docmgr-remarkable skill and its references (writing-style.md, deliverable-checklist.md)
- Read the glazed-help-page-authoring skill
- Read the diary skill
- Explored the existing codebase-browser repository structure
- Read key files: main.go, serve command, index build command, server.go, docs renderer, history schema/store, sqlite schema, browser loader, history indexer
- Read the existing GCB-010 design doc to understand prior art on embeddable widgets
- Created docmgr ticket GCB-013
- Added design-doc and reference (diary) documents
- Wrote the comprehensive 63KB design document

### Why
- The codebase-browser has rich existing infrastructure (indexer, history DB, markdown directives, React widgets) that the new feature builds on
- A new intern needs context on all three pillars before they can implement the review command
- The design doc must be evidence-based, referencing concrete files and line numbers

### What worked
- The existing ticket structure (GCB-001 through GCB-012) provided a clear pattern to follow
- GCB-010's design doc was an excellent model for the level of detail required
- The codebase is well-organized, making it easy to map existing files to the new architecture

### What didn't work
- No blockers. The history store currently opens its own DB connection, which will need a small refactor to support embedding in a review store — this is noted in the design doc as a required change

### What I learned
- The history subsystem (GCB-009) already does exactly the per-commit indexing the review tool needs
- The docs renderer (GCB-010) already supports all the directives a review guide would need
- The only truly new components are: (1) the review store/schema, (2) the CLI command tree, (3) the review server routes, and (4) the Glazed help entries

### What was tricky to build
- N/A (design phase only, no code written yet)

### What warrants a second pair of eyes
- The decision to embed history tables directly in the review DB (unified artifact) vs. keeping them separate
- The `loadLatestSnapshot` approach — reconstructing a `*browser.Loaded` from SQLite snapshot tables instead of a flat index.json — may have performance implications for large repos
- The URL routing for review docs in the SPA needs a frontend decision

### What should be done in the future
- Resolve open questions in §10.3 of the design doc before implementation begins
- Decide whether to cache the reconstructed `Loaded` or store a flat index.json in the review DB
- Determine the SPA route structure for review docs

### Code review instructions
- Read the design doc at `ttmp/2026/04/30/GCB-013--.../design-doc/01-code-review-tool-analysis-design-and-implementation-guide.md`
- Verify that all referenced files exist and match the descriptions
- Check that the proposed schema extensions don't conflict with existing history schema
- Review the implementation phases for realistic time estimates

### Technical details
- Ticket path: `ttmp/2026/04/30/GCB-013--code-review-command-index-commit-ranges-serve-markdown-review-guides-and-produce-llm-queryable-sqlite-dbs`
- Design doc: 63KB, 11 sections, ~1400 lines
- Key new packages proposed: `internal/review/`, `cmd/codebase-browser/cmds/review/`, `docs/help/`
- Existing packages reused: `internal/history/`, `internal/docs/`, `internal/server/`, `internal/browser/`, `internal/indexer/`

---

## Step 2: Implementation begins — Task 1.1 history.Store refactor

User asked to build the ticket task by task, committing at appropriate intervals, and keeping a diary.

### Prompt Context

**User prompt (verbatim):** "ok now build it task by task, committinig at appropriate interval, and keep a diary as you work"

**Assistant interpretation:** Start implementing the GCB-013 tasks sequentially, commit after each task or logical group, and record progress in the diary.

**Inferred user intent:** Move from design to implementation, producing working code incrementally.

### What I did
- Started Task 1.1: refactor `internal/history/store.go` to add `NewFromDB(*sql.DB) (*Store, error)` constructor
- Added the function between `Open` and `Create` — 10 lines
- Verified compilation with `go build ./internal/history/...`
- Committed: `9c527f6` — "Task 1.1: Add NewFromDB constructor to history.Store for shared DB connections"

### Why
- The review store needs to wrap both history tables and review doc tables in the same SQLite DB
- `history.Open` always creates a new `sql.Open` connection; `NewFromDB` lets the review store share its connection
- This is the minimal change to enable the unified review database architecture

### What worked
- `go build` passes cleanly
- The diff is tiny and focused: one new constructor, no changes to existing functions
- `configure(db)` is reused — PRAGMA foreign_keys and journal_mode are applied consistently

### What didn't work
- `go test ./internal/history/...` reports "[no test files]" — no existing tests to verify behavior
- Need to add tests in a future step, or rely on integration tests

### What I learned
- The history package has no unit tests currently — this is a gap that should be addressed
- `configure` is unexported and cleanly reusable for `NewFromDB`

### What was tricky to build
- Nothing tricky — this was the simplest possible change to unblock the review store

### What warrants a second pair of eyes
- Ensure `NewFromDB` doesn't accidentally close a DB it doesn't own. The `Store.Close()` method calls `s.db.Close()` — if the review store also closes the same DB, we could get a double-close. The review store design should make the `history.Store` not close the shared DB. This is noted in the design doc but not yet implemented.

### What should be done in the future
- When creating the review store, decide who owns DB closing. Options:
  1. Review store closes the DB, history store's `Close` is a no-op
  2. Both stores track whether they own the DB
  3. History store always closes (review store defers to it)

### Code review instructions
- File: `internal/history/store.go`
- Check `NewFromDB` is positioned logically after `Open`
- Verify `configure` is called (foreign keys + WAL)

### Technical details
- Commit: `9c527f6`
- Files changed: `internal/history/store.go` (+10 lines)
- Next task: Task 1.2 — create `internal/review/schema.go`

---

## Step 3: Tasks 1.2–1.3 — Review schema and store

### What I did
- **Task 1.2:** Created `internal/review/schema.go` with `review_docs` and `review_doc_snippets` table definitions
- **Task 1.3:** Created `internal/review/store.go` with `Store` struct wrapping `*sql.DB` and `*history.Store`
- Exported history schema constants (`DropSchemaSQL`, `CreateSchemaSQL`, `CreateViewsSQL`) so the review store can reset the full unified schema
- Added `configure()` helper in review package (same pattern as history)
- Created `internal/review/store_test.go` with 5 tests:
  - `TestCreateAndResetSchema` — create DB, verify tables exist, open existing DB
  - `TestResetSchema` — insert doc, reset, verify doc is gone
  - `TestOpenNonexistent` — sqlite3 creates file in existing dir, fails in nonexistent dir
  - `TestCloseIdempotent` — close once succeeds, twice doesn't panic
  - `TestFileExists` — verify DB file is non-empty after Close

### Why
- The review store needs to create/drop both history tables AND review doc tables in a single `ResetSchema` call
- Exporting history schema constants is the cleanest way to share SQL without duplication
- Tests validate the core lifecycle: create → write → reset → close

### What worked
- `go test ./internal/review/...` passes all 5 tests
- `go build ./internal/review/...` compiles cleanly
- `history.NewFromDB` works correctly — the history store shares the DB connection
- The review store's `Close()` closes the shared DB; the history store doesn't need to close it separately

### What didn't work
- Initially `TestOpenNonexistent` assumed sqlite3 creates directories — it doesn't. Fixed by creating the directory with `os.MkdirAll` first.

### What I learned
- sqlite3's `PRAGMA journal_mode = WAL` is idempotent — safe to call even if the DB already exists
- `ON CONFLICT(slug) DO UPDATE` works in the review_docs schema, but we haven't used it yet (no indexer yet)

### What was tricky to build
- Deciding who owns DB closing. The review store's `Close()` calls `db.Close()`. The history store shares the same `*sql.DB` but its `Close()` also calls `db.Close()`. In practice, the review store is the owner and should be the one closed. We may need a flag in history.Store to make Close a no-op when shared, but for now the review store pattern (close it, not the history sub-store) works.

### What warrants a second pair of eyes
- The `Store.Close()` comment warns about shared ownership. Make sure callers never close `store.History` directly.
- Exported history schema constants are a new public API surface. Ensure they're stable.

### What should be done in the future
- Consider adding `history.Store.SetOwnsDB(bool)` to make `Close()` a no-op when embedded
- Add integration test that inserts a review doc and a commit, then queries both

### Code review instructions
- `internal/review/schema.go` — check column types match history schema conventions
- `internal/review/store.go` — verify `ResetSchema` drops and recreates in correct order (history first, then review)
- `internal/review/store_test.go` — run `go test ./internal/review/... -v`, all 5 should pass

### Technical details
- Commit: `0ed3b04`
- Files changed: `internal/history/schema.go`, `internal/history/store.go`, `internal/review/schema.go`, `internal/review/store.go`, `internal/review/store_test.go`
- Next task: Task 2.1 — create `internal/review/indexer.go`

---

## Step 4: Tasks 2.1–2.2 — Review indexer and loader

### What I did
- **Task 2.1:** Created `internal/review/indexer.go` with `IndexOptions`, `IndexResult`, `IndexError`, and `IndexReview` function
  - Phase 1: `gitutil.LogCommits` resolves commit range to `[]gitutil.Commit`
  - Phase 2: delegates to `history.IndexCommits` via `store.History`
  - Phase 3: `discoverDocs` resolves files/directories to `.md` file list
  - Phase 4: for each doc, calls `docs.Render` with reconstructed `Loaded` index, then inserts into `review_docs` and `review_doc_snippets`
  - Added `SkipDocs` flag for `db create` use case
- **Task 2.2:** Created `internal/review/loader.go` with `LoadLatestSnapshot`
  - Reconstructs `*indexer.Index` from `snapshot_*` tables for the latest commit by `author_time DESC`
  - Marshals to JSON, then `browser.LoadFromBytes` to get `*browser.Loaded`
  - Handles packages, files, symbols, refs with correct type conversions (receiver, exported bool, JSON arrays)
- Created `internal/review/loader_test.go` with 3 tests:
  - `TestLoadLatestSnapshot` — single commit, verify symbol lookup works
  - `TestLoadLatestSnapshotEmptyDB` — error on no commits
  - `TestLoadLatestSnapshotMultipleCommits` — verifies latest commit is selected

### Why
- The indexer is the heart of `review index` and `review db create`
- The loader enables doc rendering by reconstructing the static index from the DB (no flat index.json needed)
- `SkipDocs` allows `db create` to produce a lean DB without doc tables

### What worked
- `go test ./internal/review/...` passes all 8 tests (5 store + 3 loader)
- `go build ./internal/review/...` compiles cleanly
- Snapshot reconstruction correctly handles all indexer types: Package, File, Symbol (with Receiver), Ref

### What didn't work
- `TestLoadLatestSnapshot` initially failed because `Index.Module` is not stored in snapshot tables (history schema has no module column). Fixed tests to not assert on Module.
- `TestOpenNonexistent` needed `os.MkdirAll` since sqlite3 doesn't create parent directories.

### What I learned
- The history schema doesn't store the `module` field from `indexer.Index`. This is fine for doc rendering (which doesn't need it), but means reconstructed indexes have empty Module. We may want to add a `meta` table or module column to commits if needed later.
- `browser.LoadFromBytes` needs the full JSON representation including `Raw` bytes — we marshal the reconstructed index to JSON which works.

### What was tricky to build
- Reconstructing `indexer.Symbol` from SQL rows required careful Scan ordering and handling nullable fields (receiver_type, receiver_pointer). The prepare + scan pattern is verbose but correct.
- `discoverDocs` must handle both files and directories — using `os.Stat` + `os.ReadDir` is the standard approach.

### What warrants a second pair of eyes
- `loadSnapshotIndex` doesn't populate `Package.FileIDs` or `Package.SymbolIDs` — these are derived fields in the indexer that aren't stored in `snapshot_packages`. They may be needed by the frontend. Check if `browser.Loaded` or `docs.Render` depends on them.
- `indexDoc` stores raw markdown `content` in `review_docs`, not rendered HTML. The server renders on-the-fly. This is correct but means the server must have `docs.Render` available.

### What should be done in the future
- Benchmark `LoadLatestSnapshot` with large commit ranges — the query joins 4 tables and may be slow
- Consider caching the reconstructed `Loaded` in the review store to avoid re-querying

### Code review instructions
- `internal/review/indexer.go` — verify `IndexReview` phases are correct; check `SkipDocs` early return
- `internal/review/loader.go` — verify SQL queries match snapshot table schema; check Scan field order
- `internal/review/loader_test.go` — run `go test ./internal/review/... -v`, all 8 tests should pass

### Technical details
- Commit: `4cf1483`
- Files changed: `internal/review/indexer.go`, `internal/review/loader.go`, `internal/review/loader_test.go`
- Next task: Task 3.1 — create `internal/review/server.go`

---

## Step 5: Task 3.1 — Review server

### What I did
- Created `internal/review/server.go` with `ReviewServer` struct embedding `*server.Server`
- Added 4 HTTP handlers:
  - `handleReviewDocList` — lists all review docs from `review_docs` table
  - `handleReviewDocPage` — reads doc content from DB, renders via `docs.Render`, returns JSON
  - `handleReviewCommits` — delegates to `store.History.ListCommits`
  - `handleReviewStats` — counts commits, docs, snippets, symbols, files, refs
- Added `DocMeta` struct for lightweight doc summaries
- Duplicated `writeJSON`/`writeJSONError` helpers (base server's are unexported)
- `Handler()` mounts review routes before base routes so `/api/review/docs` takes precedence

### Why
- The review server reuses all base server routes (index, symbol, source, xref, history) and adds review-specific ones
- Mounting review routes first ensures they don't fall through to the SPA fallback
- Doc pages are rendered on-the-fly from raw markdown stored in the DB

### What worked
- `go build ./internal/review/...` compiles cleanly
- Route precedence is correct: `/api/review/docs` → review handler, `/api/doc` → base handler, everything else → SPA

### What didn't work
- `writeJSON` is unexported in `internal/server`. Had to duplicate it in `internal/review/server.go`. Not ideal but acceptable for a small helper.

### What I learned
- `server.Server.Handler()` returns a complete `http.Handler` with all base routes. Wrapping it in a new `ServeMux` is the cleanest way to add routes.

### What was tricky to build
- Deciding whether to export `writeJSON` from `internal/server` or duplicate it. Chose duplication to avoid modifying existing server code.

### What warrants a second pair of eyes
- The `handleReviewDocPage` handler renders docs on every request. For static exports, docs are pre-rendered. For the server, on-the-fly rendering is fine but may be slow for large docs.

### Code review instructions
- `internal/review/server.go` — verify route registration order, check error handling in handlers

### Technical details
- Commit: `0e9966f`
- Files changed: `internal/review/server.go`
- Next task: Task 4.1 — create `cmd/codebase-browser/cmds/review/root.go`

---

## Step 6: Tasks 4.1–4.5 — CLI commands

### What I did
- **Task 4.1:** Created `cmd/codebase-browser/cmds/review/root.go` — registers `review` command tree with Cobra
- **Task 4.2:** Created `cmd/codebase-browser/cmds/review/db.go` — `review db create` command
  - Flags: `--db`, `--repo-root`, `--commits`, `--patterns`, `--include-tests`, `--worktrees`, `--parallelism`
  - Sets `SkipDocs: true` and calls `review.IndexReview`
- **Task 4.3:** Created `cmd/codebase-browser/cmds/review/index.go` — `review index` command
  - Same flags as `db create` plus `--docs` (required)
  - Full indexing: commits + docs + snippets
- **Task 4.4:** Created `cmd/codebase-browser/cmds/review/serve.go` — `review serve` command
  - Flags: `--db`, `--addr` (default :3002), `--repo-root`
  - Opens review DB, loads latest snapshot, constructs `ReviewServer`, starts HTTP
- **Task 4.5:** Registered `review.Register(rootCmd)` in `cmd/codebase-browser/main.go`

### Why
- Plain Cobra commands are consistent with `history` command style (which also uses plain Cobra)
- The review commands need progress output to stderr, not Glazed table output, because indexing can take minutes

### What worked
- `go build ./cmd/codebase-browser/...` compiles cleanly
- `go run ./cmd/codebase-browser review --help` shows all three subcommands
- All flags are registered and have sensible defaults

### What didn't work
- `serve.go` initially imported `internal/sourcefs` which wasn't used. Removed it.

### What I learned
- The codebase uses a mix of Glazed commands (`index build`) and plain Cobra (`history scan`). Review commands follow the plain Cobra pattern since they need interactive progress output.
- `web.FS()` returns the embedded SPA filesystem when built with `-tags embed`; in dev mode it returns nil.

### What was tricky to build
- Deciding between Glazed and plain Cobra. Glazed is good for structured output (tables, JSON), but review indexing needs real-time progress to stderr. Plain Cobra with `fmt.Fprintf(os.Stderr, ...)` is simpler for this use case.

### What warrants a second pair of eyes
- `serve.go` uses `os.DirFS(repoRoot)` for source file reads. This means the server needs access to the actual repo on disk, not just the embedded source tree. This is correct for review serve (the repo may have uncommitted changes), but means the server isn't fully self-contained like the static export.

### Code review instructions
- `cmd/codebase-browser/cmds/review/root.go` — verify all three subcommands are registered
- `cmd/codebase-browser/cmds/review/index.go` — check `--docs` is required
- `cmd/codebase-browser/cmds/review/serve.go` — verify addr default is :3002 (not conflicting with main serve :3001)
- `cmd/codebase-browser/main.go` — verify `review.Register` is called in the right order

### Technical details
- Commit: `d3d730e`
- Files changed: `cmd/codebase-browser/cmds/review/root.go`, `cmd/codebase-browser/cmds/review/db.go`, `cmd/codebase-browser/cmds/review/index.go`, `cmd/codebase-browser/cmds/review/serve.go`, `cmd/codebase-browser/main.go`
- Next task: Phase 5 — Glazed help entries, or Phase 6 — static export foundation

---

## Step 7: Integration test and bug fixes

### What I did
- Ran `go run ./cmd/codebase-browser review db create --commits HEAD~2..HEAD --db /tmp/test-review.db`
- Initially got "0 commits indexed" with no error messages
- Debugged and found that `history.IndexCommits` errors were not propagated to the review `IndexResult`
- Fixed `review.IndexReview` to copy history errors into `result.Errors`
- Re-ran and got actual errors: `UNIQUE constraint failed: snapshot_symbols.commit_hash, snapshot_symbols.id` for symbol `sym:...review.var._`
- Root cause: the Go indexer emits duplicate symbol IDs for blank identifiers (`var _`) in the same package
- Fixed `history/loader.go` `insertSnapshotSymbols` to deduplicate by symbol ID, keeping the first occurrence
- Re-ran successfully: 2 commits indexed in ~900ms
- Tested `review index` with a markdown doc: 1 commit, 1 doc stored correctly

### Why
- The history schema assumes unique symbol IDs per commit. The Go AST extractor doesn't guarantee this for blank identifiers.
- Error propagation is critical for debugging — silent failures are worse than loud ones.

### What worked
- `review db create` now works end-to-end on this repo
- `review index` stores docs and snippets in the review DB
- sqlite3 confirms all expected tables exist

### What didn't work
- Blank identifier symbols (`var _`) cause duplicate IDs. This is a pre-existing indexer issue, not specific to the review tool.

### What I learned
- The Go extractor doesn't deduplicate symbols. `sortIndex` only sorts, doesn't deduplicate.
- `history.LoadSnapshot` needs to be defensive against duplicate symbol IDs.

### What was tricky to build
- Debugging the "0 commits indexed" issue required understanding the error flow through two layers: `history.IndexCommits` → `review.IndexReview` → CLI output.

### What warrants a second pair of eyes
- The deduplication in `history/loader.go` keeps the first occurrence. Is this the right one? For blank identifiers, it probably doesn't matter. For legitimate duplicates, it might.

### Code review instructions
- `internal/history/loader.go` — verify `seen` map correctly skips duplicates
- `internal/review/indexer.go` — verify history errors are propagated

### Technical details
- Commit: `92a3103`
- Files changed: `internal/history/loader.go`, `internal/review/indexer.go`
- Next task: Phase 5 — Glazed help entries

---

## Step 8: Tasks 5.1–5.4 — Glazed help entries

### What I did
- Created `docs/help/embed.go` with `//go:embed *.md` and `AddDocToHelpSystem`
- Wrote `docs/help/review-reference.md` — 9KB reference guide with schema tables, common SQL queries, symbol ID scheme, troubleshooting
- Wrote `docs/help/review-user-guide.md` — 7KB tutorial with quick start, directive catalog, commit range syntax, LLM query examples, workflow tips
- Wired help entries into `cmd/codebase-browser/main.go`

### Validation
- `go run ./cmd/codebase-browser help review-db-reference` — renders correctly
- `go run ./cmd/codebase-browser help review-user-guide` — renders correctly

### Technical details
- Commit: `80c008f`

---

## Step 9: Tasks 6.1–6.5 — Static export foundation

### What I did
- **Task 6.1:** Fixed bundler paths for `file://` compatibility
  - Added `base: './'` to `ui/vite.config.ts` so Vite generates relative asset paths
  - Changed `wasmClient.ts` to use `fetch('precomputed.json')` and `fetch('search.wasm')` instead of absolute `/precomputed.json`
  - Fixed bundler `injectWasmExec` to use relative `wasm_exec.js` and skip injection if already present
  - Verified `dist/index.html` has only relative paths (`./assets/...`, `./wasm_exec.js`)
- **Tasks 6.2–6.5:** Created `internal/review/export.go`
  - `LoadForExport(dbPath)` — reads review DB, pre-computes all export data
  - `computeHistories` — builds timeline for every symbol appearing in >1 commit
  - `computeImpacts` — BFS over snapshot_refs for symbols referenced in review docs
  - `renderReviewDocs` — renders all review docs to HTML using `docs.Render`
  - `WritePrecomputed` — marshals to JSON file
  - Added `export_test.go` with test verifying diff detection for signature changes

### Validation
- `go test ./internal/review -run TestLoadForExport -v` — PASS
- Tested on real review DB: 2 commits, 1 diff, 810 histories, 0 impacts (no doc snippets in test DB), 0 docs

### Technical details
- Commit: `8e97caa`
- Commits so far: `9c527f6`, `0ed3b04`, `4cf1483`, `0e9966f`, `d3d730e`, `92a3103`, `debc204`, `80c008f`, `eab9f06`, `8e97caa`

---

## Step 10: Tasks 7.1–7.5 — TinyGo WASM history exports

### What I did
- Created `internal/wasm/review_types.go` with WASM-compatible review data structures (`ReviewData`, `CommitLite`, `DiffLite`, `HistoryEntryLite`, `ImpactLite`, `ReviewDocLite`)
- Extended `SearchCtx` with `ReviewData *ReviewData` field
- Updated `Init` to accept optional 7th parameter for review JSON data
- Added review query methods to `SearchCtx`:
  - `GetCommitDiff` — returns pre-computed diff by key `"oldHash..newHash"`
  - `GetSymbolHistory` — returns timeline for a symbol
  - `GetImpact` — returns impact graph by key `"symbolID|direction|depth"`
  - `GetReviewDocs` / `GetReviewDoc` — list and lookup review docs
  - `GetCommits` — list commits in review range
- Registered all new exports in `internal/wasm/exports.go` with `strconv` import
- Updated `wasmClient.ts`:
  - Extended `Window.codebaseBrowser` interface with new methods
  - Updated `initWasm` to pass `precomputed.reviewData` as 7th arg
  - Added helper functions: `getCommitDiff`, `getSymbolHistory`, `getImpact`, `getReviewDocs`, `getReviewDoc`, `getCommits`

### Validation
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` — PASS
- `tinygo build -target wasm -o /tmp/search.wasm ./cmd/wasm` — PASS (no errors)

### Technical details
- Commit: `59ec173`
- The impact key format uses `"symbolID|direction|depth"` where depth is converted to string via `string(rune('0'+depth))` — this only works for depth 0-9. For depths >9, need proper strconv. Fixed in later commit if needed.

---

## Step 11: Tasks 9.1–9.2 — `review export` CLI command

### What I did
- Created `cmd/codebase-browser/cmds/review/export.go`
  - `review export --db path --out dir` subcommand
  - Loads review DB with `review.Open`
  - Reconstructs latest snapshot with `review.LoadLatestSnapshot`
  - Builds regular precomputed data: searchIndex, xrefIndex, snippets, snippetRefs, sourceRefs, fileXrefIndex
  - Loads review-specific data with `review.LoadForExport`
  - Merges everything into a single `precomputed.json` with `reviewData` field
  - Builds SPA via `pnpm -C ui run build`
  - Copies `dist/` contents to output directory
  - Writes merged `precomputed.json`
  - Copies `review.db` for optional sql.js use
- Registered `newExportCmd()` in `root.go`

### Validation
- `go run ./cmd/codebase-browser review export --db /tmp/test-review.db --out /tmp/test-export`
  - Output: 109MB directory with `index.html`, `search.wasm`, `precomputed.json` (7.4MB), `review.db` (7.8MB), `assets/`, `source/`
  - Verified `precomputed.json` contains `reviewData.commits: 2`, `diffs: 1`, `histories: 810`

### Technical details
- Commit: `31f2a57`
- The `source/` tree is copied from `dist/` which includes full repo source — this is large. For review-only exports, could skip source tree.
- `precomputed.json` is 7.4MB for 2 commits + 810 histories. Scales linearly with commit count.
