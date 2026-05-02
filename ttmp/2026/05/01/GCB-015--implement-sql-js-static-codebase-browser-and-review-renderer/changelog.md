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

