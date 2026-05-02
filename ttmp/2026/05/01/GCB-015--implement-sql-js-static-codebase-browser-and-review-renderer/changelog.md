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

