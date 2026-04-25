# GCB-009 Changelog

## 2026-04-25 — Phases 1-4 complete, Phase 5 in progress

### Phase 1: Foundation — `33d10c6`
- Added `internal/gitutil/` — git CLI wrapper (log, worktree, show)
- Added `internal/history/` — history database (schema, store, loader, scanner, indexer)
- Added `cmd/codebase-browser/cmds/history/` — scan and list CLI commands
- Wired `history` into main.go
- Tests: gitutil (6 tests, all pass), full suite green

### Phase 2: Diff Engine — `40d3455`
- Added `internal/history/diff.go` — FULL OUTER JOIN diff of files and symbols between commits
- Added `internal/history/bodydiff.go` — per-symbol body extraction and unified diff
- Added `internal/history/cache.go` — file content caching for body diffs
- Added `history diff`, `history symbol-diff`, `history symbol-history` CLI commands
- Validated: diff HEAD~3 HEAD shows 3 symbol changes; symbol-diff shows body diff with added line

### Phase 3: History Concepts — `cd577ef`
- Added 6 SQL concept files: commits-timeline, pr-summary, symbol-changes, symbol-history, hotspots, file-changes
- Concepts execute against history.db via `--db` flag (no code changes needed)
- Validated: hotspots shows `main` with 5 distinct body versions across 74 commits

### Phase 4: Server API — `84d95aa`
- Added `internal/server/api_history.go` — 5 HTTP endpoints for history
- Added `History *history.Store` to Server struct
- Added `--history-db` flag to serve command
- Added `writeJSONError` helper to api_index.go

### Phase 5: Web UI (in progress)
- Created `ui/src/api/historyApi.ts` — RTK Query API
- Created `ui/src/features/history/HistoryPage.tsx` — commit timeline + diff viewer
- Wired into store.ts and App.tsx
- Not yet built/tested/committed
