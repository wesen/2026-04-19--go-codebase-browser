# GCB-009 Tasks

## Phase 1: Foundation

- [ ] 1.1 Create `internal/gitutil` package (log, worktree, show)
- [ ] 1.2 Create `internal/history` package with schema, store, loader
- [ ] 1.3 Create `history scan` CLI command
- [ ] 1.4 Create `history list` CLI command
- [ ] 1.5 Validate end-to-end: scan 5 commits, list them, query snapshot

## Phase 2: Diff Engine

- [ ] 2.1 Implement `DiffCommits()` — file and symbol level diff
- [ ] 2.2 Implement body hash computation during extraction
- [ ] 2.3 Implement file content cache
- [ ] 2.4 Implement `DiffSymbolBody()` — per-function unified diff
- [ ] 2.5 Create `history diff` CLI command
- [ ] 2.6 Create `history symbol-diff` CLI command
- [ ] 2.7 Validate: diff HEAD~1 vs HEAD, show per-symbol changes

## Phase 3: History Concepts

- [ ] 3.1 Create `concepts/history/pr-summary.sql`
- [ ] 3.2 Create `concepts/history/symbol-history.sql`
- [ ] 3.3 Create `concepts/history/hotspots.sql`
- [ ] 3.4 Create `concepts/history/commits-timeline.sql`
- [ ] 3.5 Create `concepts/history/symbol-changes.sql`
- [ ] 3.6 Create `concepts/history/file-changes.sql`
- [ ] 3.7 Wire history concepts into catalog (dual DB support)
- [ ] 3.8 Validate: run history concepts from CLI

## Phase 4: Server API

- [ ] 4.1 Create `internal/server/api_history.go`
- [ ] 4.2 Add `--history-db` flag to serve command
- [ ] 4.3 Implement history store on Server struct
- [ ] 4.4 Add endpoints: commits, diff, symbol-diff, symbol-history
- [ ] 4.5 Validate: curl endpoints after scanning some commits

## Phase 5: Web UI

- [ ] 5.1 Create `ui/src/api/historyApi.ts` (RTK Query)
- [ ] 5.2 Create `ui/src/features/history/HistoryPage.tsx`
- [ ] 5.3 Create `ui/src/features/history/SymbolDiffPage.tsx`
- [ ] 5.4 Add history links to package and symbol pages
- [ ] 5.5 Validate in browser: timeline, diff, history links

## Phase 6: Polish

- [ ] 6.1 Add parallelism to scan command
- [ ] 6.2 Add incremental scanning support
- [ ] 6.3 Add symbol-history CLI command
- [ ] 6.4 Write implementation diary
- [ ] 6.5 Upload to reMarkable
