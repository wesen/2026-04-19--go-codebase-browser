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

