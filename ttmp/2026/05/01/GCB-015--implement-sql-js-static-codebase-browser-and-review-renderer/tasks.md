# Tasks

This ticket is a clean-cut implementation of a static-only runtime:

- Go indexes repositories, review docs, and git history into SQLite.
- Go packages the static frontend and the SQLite DB.
- The browser opens `db/codebase.db` with `sql.js`.
- The frontend has no Go server mode, no `ServerQueryProvider`, and no `/api/*` runtime fallback.

## TODO

### Phase 0 — Ticket setup and implementation bookkeeping

- [x] T00.1 Create and maintain an implementation diary for every substantive change.
- [ ] T00.2 Keep this task list updated as design details become implementation details.
- [ ] T00.3 After each focused implementation slice, run relevant validation and commit code/docs intentionally.
- [ ] T00.4 Keep `docmgr doctor --ticket GCB-015 --stale-after 30` passing.

### Phase 1 — Frontend sql.js dependency and DB bootstrap

- [x] T01.1 Add `sql.js` to `ui/package.json` with pnpm.
- [x] T01.2 Add type coverage for `sql.js` using `@types/sql.js` or a local declaration file if needed.
- [x] T01.3 Ensure `sql-wasm.wasm` is copied into the final static export root or another known browser-loadable path.
- [x] T01.4 Add `ui/src/api/sqljs/sqlJsDb.ts` with a singleton sql.js initializer and `getStaticDb()` loader.
- [x] T01.5 Add `ui/src/api/sqljs/sqlRows.ts` with `queryAll`, `queryOne`, and statement cleanup helpers.
- [x] T01.6 Add BLOB/text helper functions for decoding `file_contents.content` values returned by sql.js.
- [x] T01.7 Add a small smoke helper or test query that can read `SELECT COUNT(*) FROM commits` from `db/codebase.db`.

### Phase 2 — Static export packaging in Go

- [x] T02.1 Create `internal/staticapp` as the new package for static-only sql.js export packaging.
- [x] T02.2 Add `internal/staticapp/manifest.go` with manifest structs and JSON fields.
- [x] T02.3 Add `internal/staticapp/export.go` with `Options` and high-level `Export(ctx, opts)` orchestration.
- [x] T02.4 Make export copy the SQLite DB to `OUT/db/codebase.db`.
- [x] T02.5 Make export write `OUT/manifest.json` with DB path, generated timestamp, feature flags, commit counts, and `hasGoRuntimeServer=false`.
- [x] T02.6 Make export copy/build SPA assets into the output directory.
- [x] T02.7 Make export copy `sql-wasm.wasm` into the output directory.
- [x] T02.8 Decide whether source tree copy remains default or becomes optional; document the decision in the diary.
- [x] T02.9 Refactor `cmd/codebase-browser/cmds/review/export.go` into a thin wrapper around `staticapp.Export`.
- [x] T02.10 Remove old static-runtime `precomputed.json` generation from `review export` once sql.js provider covers equivalent UI functionality.

### Phase 3 — SQLite static-preparation tables

- [ ] T03.1 Add optional `static_export_metadata` table creation on the copied output DB.
- [x] T03.2 Add `static_review_rendered_docs` table creation on the copied output DB.
- [x] T03.3 Render review docs at export time using the existing Go markdown/directive renderer.
- [x] T03.4 Store rendered review HTML, snippets JSON, and errors JSON in `static_review_rendered_docs`.
- [ ] T03.5 Add optional `symbol_search_fts` table creation/population or explicitly defer it with a diary note.
- [x] T03.6 Ensure all static-preparation DB mutations happen on the copied output DB, not the source DB.

### Phase 4 — SqlJsQueryProvider core API

- [ ] T04.1 Add `ui/src/api/queryErrors.ts` with structured `QueryError` codes.
- [ ] T04.2 Add `ui/src/api/queryProvider.ts` with the semantic provider interface and singleton getter.
- [ ] T04.3 Add `ui/src/api/sqlJsQueryProvider.ts` implementing the provider over sql.js.
- [ ] T04.4 Implement `manifest()` by reading `manifest.json`.
- [ ] T04.5 Implement `listCommits()` with SQL over `commits`.
- [ ] T04.6 Implement `resolveCommitRef()` supporting `HEAD`, `HEAD~N`, full hashes, short hashes, and unique prefixes.
- [ ] T04.7 Implement `getCommit()` using `resolveCommitRef()` and SQL.
- [ ] T04.8 Add provider-result normalization helpers for RTK Query `queryFn` endpoints.

### Phase 5 — History and body diff route parity

- [ ] T05.1 Implement `getSymbolHistory(symbolId)` from the `symbol_history` view.
- [ ] T05.2 Implement SQL lookup for symbol body metadata at a commit.
- [ ] T05.3 Implement SQL lookup for file content by `content_hash`.
- [ ] T05.4 Implement byte-offset-safe body extraction from `Uint8Array` content.
- [ ] T05.5 Implement `getSymbolBodyDiff(from, to, symbolId)` with old/new bodies and unified diff text.
- [ ] T05.6 Refactor `ui/src/api/historyApi.ts` history/body-diff endpoints to use provider `queryFn` calls.
- [ ] T05.7 Verify `/history?symbol=sym:...Register` no longer reports `STATIC_NOT_PRECOMPUTED`.

### Phase 6 — Commit diff, refs, and impact from SQL

- [ ] T06.1 Implement `getCommitDiff(from, to)` file diffs using SQLite-compatible `UNION ALL` queries.
- [ ] T06.2 Implement `getCommitDiff(from, to)` symbol diffs using SQLite-compatible `UNION ALL` queries.
- [ ] T06.3 Compute diff stats in TypeScript from SQL rows.
- [ ] T06.4 Implement `getRefsFrom(symbolId, commit)` from `snapshot_refs`.
- [ ] T06.5 Implement `getRefsTo(symbolId, commit)` from `snapshot_refs`.
- [ ] T06.6 Implement `getImpact({ symbolId, direction, depth, commit })` as TypeScript BFS over refs.
- [ ] T06.7 Refactor impact widgets to use SQL provider output only.

### Phase 7 — Generic browser SQL coverage

- [ ] T07.1 Implement package list and package detail queries.
- [ ] T07.2 Implement symbol lookup at commit.
- [ ] T07.3 Implement symbols-at-commit query for browser pages.
- [ ] T07.4 Implement basic symbol search with `LIKE` against name, ID, and signature.
- [ ] T07.5 If FTS was added, switch search to `symbol_search_fts`; otherwise document why basic search is acceptable for now.
- [ ] T07.6 Implement source file lookup from `snapshot_files` + `file_contents`.
- [ ] T07.7 Refactor package, symbol, source, and search UI paths away from TinyGo/static JSON assumptions.

### Phase 8 — Review document SQL coverage

- [ ] T08.1 Implement `listReviewDocs()` from `static_review_rendered_docs`.
- [ ] T08.2 Implement `getReviewDoc(slug)` from `static_review_rendered_docs`.
- [ ] T08.3 Update review doc hydration to use generic `data-codebase-widget` placeholders.
- [ ] T08.4 Add a generic `CodebaseWidget` dispatcher.
- [ ] T08.5 Ensure `codebase-snippet` uses SQL-backed symbol/source queries.
- [ ] T08.6 Ensure `codebase-diff` uses SQL-backed body diffs.
- [ ] T08.7 Ensure `codebase-diff-stats` uses SQL-backed commit diffs.
- [ ] T08.8 Ensure `codebase-symbol-history` uses SQL-backed symbol history.
- [ ] T08.9 Ensure `codebase-impact` uses SQL-backed impact BFS.

### Phase 9 — Remove obsolete runtime paths

- [ ] T09.1 Remove or stop using `ui/src/api/runtimeMode.ts`.
- [ ] T09.2 Remove static/server branching from frontend API files.
- [ ] T09.3 Remove `/api/*` endpoint-string parsing from `historyApi.ts` and `docApi.ts`.
- [ ] T09.4 Remove `reviewData` from WASM initialization.
- [ ] T09.5 Remove or deprecate TinyGo review query exports for commits, histories, diffs, impacts, body diffs, and review docs.
- [ ] T09.6 Remove `PrecomputedReview` as a static runtime data model.
- [ ] T09.7 Decide whether any Go HTTP server code remains in the repository as dead code; if removed, delete it in a focused commit.
- [ ] T09.8 Remove `review serve` from the CLI or clearly mark it deleted as part of the clean cutoff.

### Phase 10 — Tests and validation

- [ ] T10.1 Add Go tests for manifest writing and DB copy layout.
- [ ] T10.2 Add Go tests for `static_review_rendered_docs` generation.
- [ ] T10.3 Add TypeScript tests for `resolveCommitRef()`.
- [ ] T10.4 Add TypeScript tests for sql.js row helpers and BLOB decoding.
- [ ] T10.5 Add TypeScript tests for body extraction by byte offsets.
- [ ] T10.6 Add Playwright regression for static review doc rendering with zero `/api/*` requests.
- [ ] T10.7 Add Playwright regression for direct `/history?symbol=sym:...Register` body diff rendering.
- [ ] T10.8 Validate `pnpm -C ui run typecheck`.
- [ ] T10.9 Validate `go test ./...` or a documented narrower package set if full test suite is too slow/noisy.
- [ ] T10.10 Validate a manual static export served with `python3 -m http.server`.

### Phase 11 — Documentation and delivery

- [ ] T11.1 Update the design doc when implementation discoveries change the plan.
- [ ] T11.2 Update Glazed help entries for sql.js static export behavior.
- [ ] T11.3 Keep the implementation diary current with commands, failures, commits, and review instructions.
- [ ] T11.4 Upload revised implementation notes to reMarkable after major architecture or implementation milestones.
