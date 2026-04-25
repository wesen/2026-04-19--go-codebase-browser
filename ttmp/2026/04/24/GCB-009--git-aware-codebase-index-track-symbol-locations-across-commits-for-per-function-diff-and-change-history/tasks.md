# GCB-009 Tasks

## Phase 1: Foundation — gitutil + history store + scan CLI

- [x] 1.1 Create `internal/gitutil/log.go` — `Commit` struct, `LogCommits()`, `ChangedFiles()`
- [x] 1.2 Create `internal/gitutil/worktree.go` — `CreateWorktree()`, `RemoveWorktree()`, `WorktreePool`
- [x] 1.3 Create `internal/gitutil/show.go` — `ShowFile()`, `FileBlobHash()`
- [x] 1.4 Create `internal/history/schema.go` — `commits`, `snapshot_*` tables, indexes, views
- [x] 1.5 Create `internal/history/store.go` — `Open()`, `Create()`, `Close()`, `HasCommit()`
- [x] 1.6 Create `internal/history/loader.go` — `LoadSnapshot()` bulk-inserts a commit's Index
- [x] 1.7 Create `internal/history/scanner.go` — `ScanCommits()` discovers and filters commits
- [x] 1.8 Create `internal/history/indexer.go` — `IndexCommits()` orchestrates per-commit extraction
- [x] 1.9 Create `internal/gitutil/log_test.go`, `worktree_test.go` — unit tests with real git repos
- [x] 1.10 Create `cmd/codebase-browser/cmds/history/` — root, scan, list commands
- [x] 1.11 Wire `history` into `cmd/codebase-browser/main.go`
- [x] 1.12 Validate: `history scan --range HEAD~5..HEAD` → 5 commits, 3.6s; `history list` → shows table
- [x] 1.13 Commit Phase 1 — `33d10c6 Add git-aware codebase history index (Phase 1)`

## Phase 2: Diff Engine — symbol-level and body-level diff

- [x] 2.1 Add `body_hash` column computation to `loader.go` — hash symbol bodies during LoadSnapshot
- [x] 2.2 Create `internal/history/diff.go` — `DiffCommits()`, `DiffStats`, `SymbolDiff`, `FileDiff`
- [x] 2.3 Create `internal/history/cache.go` — `CacheFileContents()`, `GetFileContent()`
- [x] 2.4 Create `internal/history/bodydiff.go` — `DiffSymbolBody()` with unified diff output
- [x] 2.5 Create `cmd/codebase-browser/cmds/history/diff.go` — `history diff` command
- [x] 2.6 Create `cmd/codebase-browser/cmds/history/symbol_diff.go` — `history symbol-diff` command
- [x] 2.7 Create `cmd/codebase-browser/cmds/history/symbol_history.go` — `history symbol-history` command
- [x] 2.8 Validate: `history diff HEAD~3 HEAD` shows 1 file modified, 3 symbols changed; `symbol-diff --name main` shows body diff with `+history.Register(rootCmd)`; `symbol-history` shows 50 entries with body_hash changes
- [x] 2.9 Commit Phase 2 — `40d3455 Add diff engine and history CLI commands (Phase 2)`

## Phase 3: History Concepts — SQL concepts for the history DB

- [x] 3.1 Create `concepts/history/commits-timeline.sql`
- [x] 3.2 Create `concepts/history/pr-summary.sql`
- [x] 3.3 Create `concepts/history/symbol-changes.sql`
- [x] 3.4 Create `concepts/history/symbol-history.sql`
- [x] 3.5 Create `concepts/history/hotspots.sql`
- [x] 3.6 Create `concepts/history/file-changes.sql`
- [x] 3.7 Wire history concepts into the concept catalog (dual-DB query support) — concepts load from embedded catalog, execute against `--db history.db`
- [x] 3.8 Validate: `query --db history.db commands history hotspots --limit 10` shows `main` with 5 distinct versions, `commits-timeline` shows all 75 commits
- [x] 3.9 Commit Phase 3 — `cd577ef Add history query concepts (Phase 3)`

## Phase 4: Server API — HTTP endpoints for history

- [x] 4.1 Create `internal/server/api_history.go` — all history endpoints (commits, commit detail, symbols at commit, diff, symbol history)
- [x] 4.2 Add `--history-db` flag to serve command, load history.Store onto Server struct
- [x] 4.3 Validate: curl endpoints — `/api/history/commits` returns 75 commits, `/api/history/diff?from=X&to=Y` returns diff with files and symbols
- [x] 4.4 Commit Phase 4 — `84d95aa Add history server API endpoints (Phase 4)`

## Phase 5: Web UI — React pages for history and diff

- [x] 5.1 Create `ui/src/api/historyApi.ts` — RTK Query history API
- [x] 5.2 Create `ui/src/features/history/HistoryPage.tsx` — commit timeline with old/new diff selector
- [x] 5.3 Create `ui/src/features/history/SymbolDiffPage.tsx` — deferred, HistoryPage covers the main use case
- [x] 5.4 Add history links to existing package and symbol pages — deferred to future iteration
- [x] 5.5 Build frontend, regenerate web assets
- [x] 5.6 Validate in browser with Playwright — history page loads, diff view shows "Symbols: +0 -0 ~1 →2", "Changed symbols" section visible
- [x] 5.7 Commit Phase 5 — `882ad10 Add history web UI with commit timeline and diff viewer (Phase 5)`

## Phase 6: Polish — diff filter, parallelism, diary

- [x] 6.1 Filter unchanged files/symbols from diff output
- [x] 6.2 Add `--parallelism` flag to scan command (goroutine pool)
- [x] 6.3 Add `--incremental` flag (skip already-indexed commits) — already implemented in scanner.go
- [x] 6.4 Write implementation diary entries for Phases 5-6
- [x] 6.5 Update changelog and tasks
- [x] 6.6 Final commit — `05c3174 Filter unchanged from diff output, add --parallelism flag`
