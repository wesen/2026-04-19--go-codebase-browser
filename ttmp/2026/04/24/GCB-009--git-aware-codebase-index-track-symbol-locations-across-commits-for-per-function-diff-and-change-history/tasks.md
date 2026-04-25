# GCB-009 Tasks

## Phase 1: Foundation — gitutil + history store + scan CLI

- [ ] 1.1 Create `internal/gitutil/log.go` — `Commit` struct, `LogCommits()`, `ChangedFiles()`
- [ ] 1.2 Create `internal/gitutil/worktree.go` — `CreateWorktree()`, `RemoveWorktree()`, `WorktreePool`
- [ ] 1.3 Create `internal/gitutil/show.go` — `ShowFile()`, `FileBlobHash()`
- [ ] 1.4 Create `internal/history/schema.go` — `commits`, `snapshot_*` tables, indexes, views
- [ ] 1.5 Create `internal/history/store.go` — `Open()`, `Create()`, `Close()`, `HasCommit()`
- [ ] 1.6 Create `internal/history/loader.go` — `LoadSnapshot()` bulk-inserts a commit's Index
- [ ] 1.7 Create `internal/history/scanner.go` — `ScanCommits()` discovers and filters commits
- [ ] 1.8 Create `internal/history/indexer.go` — `IndexCommits()` orchestrates per-commit extraction
- [ ] 1.9 Create `internal/history/indexer_test.go` — unit test for per-commit indexing with real git
- [ ] 1.10 Create `cmd/codebase-browser/cmds/history/` — root, scan, list commands
- [ ] 1.11 Wire `history` into `cmd/codebase-browser/main.go`
- [ ] 1.12 Validate: `codebase-browser history scan --range HEAD~5..HEAD` and `history list`
- [ ] 1.13 Commit Phase 1

## Phase 2: Diff Engine — symbol-level and body-level diff

- [ ] 2.1 Add `body_hash` column computation to `loader.go` — hash symbol bodies during LoadSnapshot
- [ ] 2.2 Create `internal/history/diff.go` — `DiffCommits()`, `DiffStats`, `SymbolDiff`, `FileDiff`
- [ ] 2.3 Create `internal/history/cache.go` — `CacheFileContents()`, `GetFileContent()`
- [ ] 2.4 Create `internal/history/bodydiff.go` — `DiffSymbolBody()` with unified diff output
- [ ] 2.5 Create `cmd/codebase-browser/cmds/history/diff.go` — `history diff` command
- [ ] 2.6 Create `cmd/codebase-browser/cmds/history/symbol_diff.go` — `history symbol-diff` command
- [ ] 2.7 Create `cmd/codebase-browser/cmds/history/symbol_history.go` — `history symbol-history` command
- [ ] 2.8 Validate: `history diff HEAD~1 HEAD`, `history symbol-diff`, `history symbol-history`
- [ ] 2.9 Commit Phase 2

## Phase 3: History Concepts — SQL concepts for the history DB

- [ ] 3.1 Create `concepts/history/commits-timeline.sql`
- [ ] 3.2 Create `concepts/history/pr-summary.sql`
- [ ] 3.3 Create `concepts/history/symbol-changes.sql`
- [ ] 3.4 Create `concepts/history/symbol-history.sql`
- [ ] 3.5 Create `concepts/history/hotspots.sql`
- [ ] 3.6 Create `concepts/history/file-changes.sql`
- [ ] 3.7 Wire history concepts into the concept catalog (dual-DB query support)
- [ ] 3.8 Validate: run history concepts from CLI
- [ ] 3.9 Commit Phase 3

## Phase 4: Server API — HTTP endpoints for history

- [ ] 4.1 Create `internal/server/api_history.go` — all history endpoints
- [ ] 4.2 Add `--history-db` flag to serve command, load history.Store
- [ ] 4.3 Validate: curl endpoints after scanning commits
- [ ] 4.4 Commit Phase 4

## Phase 5: Web UI — React pages for history and diff

- [ ] 5.1 Create `ui/src/api/historyApi.ts` — RTK Query history API
- [ ] 5.2 Create `ui/src/features/history/HistoryPage.tsx` — commit timeline
- [ ] 5.3 Create `ui/src/features/history/SymbolDiffPage.tsx` — per-function diff viewer
- [ ] 5.4 Add history links to existing package and symbol pages
- [ ] 5.5 Build frontend, regenerate web assets
- [ ] 5.6 Validate in browser with Playwright
- [ ] 5.7 Commit Phase 5

## Phase 6: Polish — parallelism, incremental, diary

- [ ] 6.1 Add `--parallelism` flag to scan command (goroutine pool)
- [ ] 6.2 Add `--incremental` flag (skip already-indexed commits)
- [ ] 6.3 Write implementation diary entries for each phase
- [ ] 6.4 Update changelog and tasks
- [ ] 6.5 Final commit
