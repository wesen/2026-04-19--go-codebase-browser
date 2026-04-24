# GCB-009 Changelog

## 2026-04-24

- Created GCB-009 ticket: Git-Aware Codebase Index
- Wrote comprehensive design doc covering:
  - Current system walkthrough (types, extraction pipeline, ID scheme, SQLite schema, build pipeline, concept system)
  - Per-commit indexing architecture
  - History database schema with snapshot tables
  - Diff engine design (file-level, symbol-level, body-level)
  - File content caching
  - New Go packages (`internal/gitutil`, `internal/history`)
  - CLI commands (`history scan/list/diff/symbol-diff/symbol-history`)
  - History concepts (pr-summary, symbol-history, hotspots, etc.)
  - Server API endpoints
  - Web UI pages and existing-page integration
  - 5-phase implementation plan
  - Performance estimates
  - Risks and open questions
  - Complete file reference map
- Related 6 key source files to the ticket
- Added vocabulary topics: git, diff, history
