---
title: "GCB-009 Handoff — Session 2 Starting Point"
doc_type: reference
topics: [git, sqlite, history, diff, web-ui]
---

# GCB-009 Handoff — Where We Are and What's Next

**Date:** 2026-04-25, end of session 1
**Status:** 90% complete. Two bugs fixed, one feature half-wired, one feature pending verification.

---

## What Just Happened (last 30 minutes)

Three things were done in rapid succession:

1. **Fixed source page in serve mode** — `ui/src/api/sourceApi.ts` `getSource` query now tries `/api/source?path=...` first (serve mode), then falls back to `./source/${path}` (static bundle mode). Verified: `#/source/cmd/codebase-browser/cmds/index/build.go` now renders 209 lines of Go.

2. **Added `--repo-root` flag to serve** — the server needs a git repo root to read file contents at specific commits for body diffs. New flag in `cmd/codebase-browser/cmds/serve/run.go`. `Server.RepoRoot` field added to `internal/server/server.go`.

3. **Added `/api/history/symbol-body-diff` endpoint** — `internal/server/api_history.go` now has a handler that calls `history.DiffSymbolBodyWithContent()` using the repo root. Returns `BodyDiffResult` JSON with `oldBody`, `newBody`, `unifiedDiff`, `oldRange`, `newRange`.

4. **Rewrote the function history panel** — `ui/src/features/history/HistoryPage.tsx` `SymbolHistoryPanel` now shows:
   - A timeline table with **from/to buttons** per row
   - Auto-selects two commits with different body hashes
   - Below the table, a `SymbolBodyDiffView` component calls `useGetSymbolBodyDiffQuery` and renders:
     - A colored unified diff (green for additions, red for removals)
     - Or a side-by-side view when no unified diff is available

5. **All of this is built and deployed** — frontend built, embedded web regenerated, server running at `:3011` with `--history-db /tmp/test-wt-history.db --repo-root .`.

---

## What's Not Yet Verified

The **body diff in the browser** was not confirmed end-to-end in Playwright. The last test attempt selected `newScanCmd` which had no body changes across the selected range, and when trying to switch to `main` the diff range was too narrow. The feature should work — the API returns data, the UI renders the diff — but it needs a proper Playwright verification selecting:
- old = 30th commit (or later)
- new = HEAD
- click `main` in the symbols table
- verify the function history panel appears
- verify the auto-selected from/to commits show a body diff
- or manually click from/to on commits with orange-highlighted body hashes

---

## Uncommitted Changes

There are **uncommitted changes** on disk right now:

```
 M cmd/codebase-browser/cmds/serve/run.go          (--repo-root flag, RepoRoot on server)
 M internal/server/api_history.go                  (symbol-body-diff endpoint)
 M internal/server/server.go                       (RepoRoot field)
 M ui/src/api/sourceApi.ts                         (dual-mode getSource: API first, static fallback)
 M ui/src/api/historyApi.ts                        (getSymbolBodyDiffQuery + BodyDiffResult type)
 M ui/src/features/history/HistoryPage.tsx          (full rewrite of SymbolHistoryPanel + SymbolBodyDiffView)
```

**Commit these first thing next session** with something like:
```
git add -A
git commit -m "Fix serve-mode source loading, add body diff endpoint and function diff viewer"
```

---

## Git History So Far

```
8374a53 Add clickable symbol history to diff view
743d227 Diary: update tasks — all Phases 1-6 complete
05c3174 Filter unchanged from diff output, add --parallelism flag
882ad10 Add history web UI with commit timeline and diff viewer (Phase 5)
085f9cd Diary: record Phases 1-4 implementation details
84d95aa Add history server API endpoints (Phase 4)
cd577ef Add history query concepts (Phase 3)
40d3455 Add diff engine and history CLI commands (Phase 2)
33d10c6 Add git-aware codebase history index (Phase 1: gitutil + history store + scan/list CLI)
```

---

## Remaining Work

### Must Do (verification)

1. **Verify function body diff in Playwright** — select a wide commit range, click `main`, confirm the diff renders with colored +/- lines.

2. **Test the body diff with actual changes** — the auto-select might pick two commits with the same body. Need to verify that clicking "from" on an orange row and "to" on a different orange row produces a visible diff.

### Nice To Have (polish)

3. **Source page works but could be better** — the fix works for serve mode. Consider making SourcePage detect mode once and cache the strategy instead of trying both fetches every time.

4. **Symbol diff page** — the design doc mentioned a dedicated `/#/history/diff?symbol=...&from=...&to=...` page. The current implementation embeds it in the history panel, which is fine for now but a dedicated page would allow direct linking.

5. **History links on package/symbol pages** — add a "History" link on package and symbol detail pages that jumps to the history page with that symbol pre-selected.

6. **Upload updated docs to reMarkable** — the design doc + diary are on reMarkable from session 1, but the Phase 5-6 additions are not.

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `internal/gitutil/log.go` | Git commit listing, ChangedFiles, ResolveRef |
| `internal/gitutil/worktree.go` | CreateWorktree, RemoveWorktree, WorktreePool |
| `internal/gitutil/show.go` | ShowFile (git show hash:path) |
| `internal/history/schema.go` | commits, snapshot_*, file_contents tables |
| `internal/history/store.go` | Open, Create, HasCommit, ListCommits, DiffCommits |
| `internal/history/loader.go` | LoadSnapshot (bulk insert per commit, body_hash computation) |
| `internal/history/diff.go` | DiffCommits (FULL OUTER JOIN, filters unchanged) |
| `internal/history/bodydiff.go` | DiffSymbolBodyWithContent (extract bodies, unified diff) |
| `internal/history/cache.go` | GetFileContent (cache then git show fallback) |
| `internal/history/scanner.go` | ScanCommits (discover, filter, incremental skip) |
| `internal/history/indexer.go` | IndexCommits (per-commit extraction, parallelism support) |
| `cmd/codebase-browser/cmds/history/scan.go` | CLI: history scan (--range, --worktrees, --parallelism, --incremental) |
| `cmd/codebase-browser/cmds/history/diff.go` | CLI: history diff |
| `cmd/codebase-browser/cmds/history/symbol_diff.go` | CLI: history symbol-diff |
| `cmd/codebase-browser/cmds/history/symbol_history.go` | CLI: history symbol-history |
| `internal/server/api_history.go` | HTTP: 6 endpoints under /api/history/ |
| `ui/src/api/historyApi.ts` | RTK Query: listCommits, getDiff, getSymbolHistory, getSymbolBodyDiff |
| `ui/src/api/sourceApi.ts` | RTK Query: getSource (dual-mode: API first, static fallback) |
| `ui/src/features/history/HistoryPage.tsx` | React: commit timeline, diff view, symbol history, body diff viewer |
| `concepts/history/*.sql` | 6 SQL concepts: hotspots, pr-summary, symbol-history, etc. |

---

## Server Start Command

```bash
# Build + scan + serve (full pipeline)
go generate ./internal/sqlite
go run ./cmd/codebase-browser history scan --range "HEAD" --db history.db --worktrees --incremental
go run ./cmd/codebase-browser serve --addr :3011 --history-db history.db --repo-root .
```

The test DB used this session is at `/tmp/test-wt-history.db` (82 commits indexed with worktrees).

---

## Test Commands

```bash
# CLI smoke tests
go run ./cmd/codebase-browser history list --db history.db
go run ./cmd/codebase-browser history diff HEAD~10 HEAD --db history.db
go run ./cmd/codebase-browser history symbol-history --name main --db history.db
go run ./cmd/codebase-browser query --db history.db commands history hotspots --limit 10

# Full test suite
go test ./... -count=1

# Frontend build
cd ui && pnpm build
BUILD_WEB_LOCAL=1 go generate ./internal/web
```
