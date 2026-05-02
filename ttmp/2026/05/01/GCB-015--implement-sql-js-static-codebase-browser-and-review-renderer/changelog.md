# Changelog

## 2026-05-01

- Initial workspace created


## 2026-05-01

Created sql.js static frontend design and implementation guide. The new architecture uses Go for indexing/export, ships db/codebase.db, opens it with sql.js in the browser, and routes both generic browser pages and review markdown widgets through one SQL-backed CodebaseQueryProvider.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/design-doc/01-sql-js-static-frontend-architecture-and-implementation-guide.md — Primary sql.js static architecture and intern implementation guide
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Phase task list for sql.js static frontend implementation


## 2026-05-01

Updated sql.js static frontend design to v2: removed Go server mode from the target architecture. Go is now only an offline indexer/exporter, the browser always uses SqlJsQueryProvider against db/codebase.db, and there is no ServerQueryProvider or /api runtime fallback.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/design-doc/01-sql-js-static-frontend-architecture-and-implementation-guide.md — v2 static-only runtime update
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Tasks updated to remove ServerQueryProvider/runtime-mode branching


## 2026-05-01

Expanded detailed implementation tasks and created the implementation diary for GCB-015.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Implementation diary initialized
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Detailed phased task checklist


## 2026-05-01

Step 1: Added sql.js dependency, sql-wasm public asset, DB bootstrap helpers, SQL row helpers, and BLOB byte-range utilities.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 1
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked Phase 1 bootstrap tasks complete
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/package.json — Added sql.js dependency and type package
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/pnpm-lock.yaml — Updated dependency lockfile
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/public/sql-wasm.wasm — Browser-loadable sql.js WASM runtime
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqljs/sqlJsDb.ts — Static SQLite DB loader
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqljs/sqlRows.ts — Prepared statement and BLOB helpers


## 2026-05-01

Step 2: Added staticapp package and refactored review export to write a static-only sql.js bundle with manifest.json and db/codebase.db.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/export.go — Thin CLI wrapper around staticapp.Export
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/staticapp/export.go — Static-only sql.js export packaging
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/staticapp/manifest.go — Static export manifest types
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 2
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked static export packaging tasks complete


## 2026-05-01

Step 3: Added export-time review doc rendering into static_review_rendered_docs inside the copied SQLite DB.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/export.go — Enables RenderReviewDocs in staticapp options
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/staticapp/export.go — Runs AddRenderedReviewDocs on the copied output DB
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/staticapp/reviewdocs.go — Rendered review docs table and export-time renderer
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 3
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked rendered review doc tasks complete


## 2026-05-01

Step 4: Added SqlJsQueryProvider skeleton, SQL-backed commit/history/body-diff methods, and rewrote historyApi to provider queryFn calls without /api endpoint parsing.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 4
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked provider and core history tasks complete
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/historyApi.ts — RTK Query endpoints now call provider methods
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/queryErrors.ts — Structured provider errors
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/queryProvider.ts — Static-only provider singleton
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqlJsQueryProvider.ts — Initial sql.js provider with commits


## 2026-05-01

Step 5: Implemented SQL-backed commit diffs, refs, and TypeScript impact BFS in SqlJsQueryProvider.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 5
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked Phase 6 SQL diff/impact tasks complete
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqlJsQueryProvider.ts — Commit diff SQL


## 2026-05-01

Step 6: Removed provider wrapper/runtime-mode helper and routed review docs through SqlJsQueryProvider only, with no server or WASM fallback.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 6
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked clean-cut frontend API tasks complete
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/docApi.ts — Review doc APIs now use sql.js provider only
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/historyApi.ts — History APIs call getSqlJsProvider directly
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/queryProvider.ts — Removed wrapper provider layer
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/runtimeMode.ts — Removed obsolete runtime-mode helper
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqlJsQueryProvider.ts — Owns the only provider singleton and review doc SQL methods


## 2026-05-01

Step 7: Moved index/package/symbol/search APIs from TinyGo wasmBaseQuery to SqlJsQueryProvider.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 7
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Marked generic browser SQL index/search tasks complete
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/indexApi.ts — Index APIs now use SQL provider queryFns
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqlJsQueryProvider.ts — Added latest package/file/symbol/index/search SQL methods


## 2026-05-01

Step 8: Deleted wasmClient and moved source/snippet/xref frontend APIs to SqlJsQueryProvider, with no /api or precomputed JSON fallback.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 8
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/conceptsApi.ts — Removed /api concept fetching from static-only runtime
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sourceApi.ts — Source/snippet APIs now use SQL provider
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqlJsQueryProvider.ts — Added source/snippet/xref SQL methods
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/wasmClient.ts — Removed old TinyGo/precomputed client
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/xrefApi.ts — Xref API now uses SQL provider
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocSnippet.tsx — Removed /api snippet fetch
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/AnnotationWidget.tsx — Removed /api snippet fetch


## 2026-05-01

End-of-day handoff: documented current GCB-015 sql.js cutover status, validation failure on missing sql-wasm-browser.wasm, files to read tomorrow, and next implementation tasks.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — End-of-day handoff Step 9
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/public/sql-wasm-browser.wasm — Added after browser requested this sql.js WASM asset


## 2026-05-01

Step 10: Validated sql.js static smoke with zero /api requests, verified direct history body diff no longer reports STATIC_NOT_PRECOMPUTED, and removed review serve plus the review HTTP server wrapper (commit 45de723).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/patterns.go — Preserved defaultPatterns for index/db commands
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/root.go — Removed review serve registration
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/serve.go — Deleted deprecated review runtime command
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/docs/help/review-reference.md — Updated static export schema help
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/docs/help/review-user-guide.md — Updated static export workflow help
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/review/server.go — Deleted deprecated review HTTP server wrapper
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 10


## 2026-05-01

Step 11: Removed PrecomputedReview, WASM ReviewData, review-specific TinyGo exports, and the frontend wasm_exec.js runtime artifact; static sql.js smoke still loads with zero /api requests (commit 12d31ec).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/review/export.go — Deleted old PrecomputedReview builder
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/exports.go — Removed review JS exports
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/review_types.go — Deleted WASM ReviewData model
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/search.go — Removed reviewData runtime state and review query methods
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 11
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/index.html — Removed Go WASM loader script tag
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/public/wasm_exec.js — Deleted unused Go WASM loader asset


## 2026-05-01

Step 12: Added Go tests for static export manifest/layout, absence of legacy runtime files, and static_review_rendered_docs generation (commit 35045da).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/staticapp/export_test.go — Tests staticapp export layout and rendered review doc table
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 12


## 2026-05-01

Step 13: Added route-change scroll reset and replaced the flat package sidebar with a collapsible import-path package tree for the static browser shell (commit cc18c22).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 13
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/app/App.tsx — Route scroll reset and package tree navigation
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/packages/ui/src/theme/base.css — Tree active/root/nesting styles


## 2026-05-01

Step 14: Investigated no-diff history smoke; found non-worktree multi-commit indexing reuses current checkout snapshots, while a 20-commit --worktrees export shows real body hash changes for review.Register and newExportCmd.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 14 validation finding


## 2026-05-01

Step 15: Removed manual review --worktrees flags and made review indexing automatically use worktrees for multi-commit ranges while keeping direct indexing for single-commit snapshots (commit c8ee93d).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/db.go — Removed --worktrees flag and updated help
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/index.go — Removed --worktrees flag and updated help
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/docs/help/review-user-guide.md — Documented automatic multi-commit worktrees
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/review/indexer.go — Automatic worktree mode based on resolved commit count
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 15


## 2026-05-01

Step 16: Completed SQL-backed signature snippets, snippet refs, source refs, and file xrefs, and removed stale server-backed UI error copy (commit 5a0e840).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 16
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sourceApi.ts — Provider-backed ref and file xref endpoints
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqlJsQueryProvider.ts — SQL source/snippet/file xref provider methods
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/history/HistoryPage.tsx — Static SQLite error copy
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/query/QueryConceptsPage.tsx — Static concept-unavailable copy


## 2026-05-01

Step 17: Removed the unpackaged structured query concepts UI/API slice from the static browser instead of keeping a dead unavailable route (commit 714708c).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 17
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/conceptsApi.ts — Deleted
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/store.ts — Removed conceptsApi wiring
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/app/App.tsx — Removed /queries routes and sidebar link
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/query/QueryConceptsPage.tsx — Deleted


## 2026-05-01

Step 18: Deleted obsolete TinyGo/precomputed static runtime packages, generated WASM/precomputed artifacts, and old bundle targets; web generation no longer copies search.wasm, wasm_exec.js, or precomputed.json (commit e0e7e60).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/Makefile — Removed WASM static build targets
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/wasm/main.go — Deleted obsolete WASM entrypoint
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/bundle — Deleted obsolete bundle generator
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/static — Deleted obsolete precomputed data package
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm — Deleted obsolete TinyGo runtime package
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/web/generate_build.go — Removed old runtime asset copy/injection
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 18


## 2026-05-01

Step 19: Deleted obsolete Go serve command, internal/server HTTP API runtime, internal/web embed package, and stale server docs/proxy config; README/Makefile now point to review export plus static file serving (commit 05f3ffe).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/Makefile — Removed server build/dev paths
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/README.md — Updated static export quick start
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/main.go — Removed serve registration
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server — Deleted old HTTP runtime
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/web — Deleted old web embed runtime
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 19
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/vite.config.ts — Removed /api proxy


## 2026-05-01

Step 20: Added UI Vitest runner and sql.js row/BLOB/byte-offset tests; converted tokenizer smoke scripts into real Vitest suites.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md — Recorded Step 20
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md — Updated test task status
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/package.json — Added test script and Vitest dependency
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/sqljs/sqlRows.test.ts — Added sql.js row helper tests
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/packages/ui/src/highlight/go.test.ts — Converted to Vitest
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/packages/ui/src/highlight/ts.test.ts — Converted to Vitest

