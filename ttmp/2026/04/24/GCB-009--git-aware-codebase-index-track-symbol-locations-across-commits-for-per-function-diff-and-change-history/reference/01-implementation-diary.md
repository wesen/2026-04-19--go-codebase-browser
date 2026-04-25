# GCB-009 Implementation Diary

## 2026-04-24/25 — Full Session

**Goal:** Implement the git-aware codebase index across 6 phases, keeping a diary and committing at each phase boundary.

**Starting state:** Design doc written (62KB, 13 parts), ticket created, reMarkable upload done. No code yet.

---

### Phase 1: Foundation — gitutil + history store + scan/list CLI

**Commit:** `33d10c6 Add git-aware codebase history index (Phase 1: gitutil + history store + scan/list CLI)`

#### What was implemented

**`internal/gitutil/`** — Git CLI wrapper:
- `log.go`: `Commit` struct, `LogCommits(rangeSpec)` parses custom `git log --format` output, `ChangedFiles(hash)` via `git diff-tree`, `ResolveRef(ref)` via `git rev-parse`, `IsAncestor(parent, child)` via `git merge-base --is-ancestor`
- `worktree.go`: `CreateWorktree(repoRoot, hash)` → `git worktree add --detach`, `RemoveWorktree()`, `WorktreePool` with semaphore-based slot management for parallel indexing
- `show.go`: `ShowFile(repoRoot, hash, path)` → `git show hash:path`, `FileBlobHash()` via `git rev-parse`

**`internal/history/`** — History database:
- `schema.go`: `commits` table (hash, short_hash, message, author, timestamp, parents, tree_hash, indexed_at, branch, error), `snapshot_packages/files/symbols/refs` tables (all carry `commit_hash` PK dimension), `file_contents` table for body diff cache, `symbol_history` view
- `store.go`: `Open()` with WAL mode, `Create()` with schema reset, `HasCommit()`, `GetCommit()`, `ListCommits()`, `SymbolCountAtCommit()`
- `loader.go`: `LoadSnapshot(commit, idx, worktreeDir)` — bulk-inserts all packages/files/symbols/refs for one commit in a transaction. Computes `body_hash` by reading the file from the worktree and hashing `content[startOffset:endOffset]`
- `scanner.go`: `ScanCommits()` — discovers commits via `gitutil.LogCommits()`, filters by file path prefix, skips already-indexed commits when `--incremental`
- `indexer.go`: `IndexCommits()` — orchestrates per-commit extraction. Two modes: `--worktrees` (creates git worktree per commit, extracts, loads) and direct mode (extracts from working dir — for single-commit testing)

**`cmd/codebase-browser/cmds/history/`** — CLI:
- `root.go`: `history` parent command
- `scan.go`: `history scan --range HEAD~20..HEAD --worktrees --incremental --filter path/`
- `list.go`: `history list --db history.db` with formatted table (hash, date, symbol count, message)

**`cmd/codebase-browser/main.go`**: Added `history.Register(rootCmd)`

#### Tests

- `internal/gitutil/log_test.go`: `TestLogCommits` (3 commits parsed correctly, most recent first), `TestLogCommitsRange` (HEAD~1..HEAD returns 1), `TestChangedFiles` (main.go in list), `TestResolveRef`
- `internal/gitutil/worktree_test.go`: `TestCreateAndRemoveWorktree` (creates worktree, reads file, verifies content matches commit, removes), `TestShowFile` (reads file at specific commit via git show)
- All use a temp git repo with 3 commits created in `setupTestRepo(t)`

#### Validation

```bash
# No worktrees (indexes working dir at each commit's hash — fast for small ranges)
codebase-browser history scan --range "HEAD~5..HEAD" --db /tmp/test-history.db
# → scan: 5 commits to index, 0 skipped, 0 filtered
# → Done in 3.655s: 5 indexed, 0 failed

# With worktrees (checks out each commit — accurate but slower)
codebase-browser history scan --range "HEAD" --db /tmp/test-wt-history.db --worktrees
# → 74 commits indexed in 1m58s, 0 failed

codebase-browser history list --db /tmp/test-wt-history.db
# → Shows 74 commits with symbol counts varying (330, 317, 303, 277...)
```

#### Key decisions

- Worktrees are opt-in via `--worktrees` flag. Default mode indexes from the working directory — useful for quick scans but only accurate for HEAD.
- `body_hash` is computed during `LoadSnapshot` when a worktree dir is provided. Falls back to empty string if unavailable.
- WAL journal mode for the history DB to support concurrent reads during indexing.
- `snapshot_refs` uses autoincrement `id` within each commit (not globally unique).
- `commits.error` column records extraction failures without blocking the scan.
- The `findRepoRoot()` helper was initially duplicated in scan.go and symbol_diff.go — extracted to `util.go`.

#### What failed along the way

- First version of `findRepoRoot()` in scan.go used `gitutil.ResolveRef()` which returns a hash, not a dir path. Fixed to just validate the dir is a git repo and return it.
- Unused `context` import in `gitutil_test.go` — removed.

---

### Phase 2: Diff Engine — symbol-level and body-level diff

**Commit:** `40d3455 Add diff engine and history CLI commands (Phase 2)`

#### What was implemented

**`internal/history/diff.go`** — Snapshot diff computation:
- `DiffCommits(oldHash, newHash)` → `CommitDiff` with `FileDiff[]`, `SymbolDiff[]`, `DiffStats`
- File diff: `FULL OUTER JOIN` on `snapshot_files(id)` between two commits, classifies as `added/removed/modified/unchanged`
- Symbol diff: `FULL OUTER JOIN` on `snapshot_symbols(id)`, classifies as `added/removed/modified/signature-changed/moved/unchanged`
- `body_hash` comparison detects modifications even when line numbers shift
- Unchanged symbols are excluded from the output (they'd be noise)

**`internal/history/cache.go`** — File content caching:
- `CacheFileContents()` reads files from a worktree and stores them in `file_contents` table keyed by SHA-256
- `GetFileContent()` tries cache first, falls back to `gitutil.ShowFile()`

**`internal/history/bodydiff.go`** — Per-symbol body diff:
- `DiffSymbolBody()` — looks up symbol in both commits, returns `BodyDiffResult`
- `DiffSymbolBodyWithContent()` — reads file content via `GetFileContent()`, extracts body using byte offsets, computes a simple line-by-line diff
- `simpleUnifiedDiff()` — finds common prefix/suffix, marks middle as added/removed lines. Not a true patience diff but sufficient for MVP.
- `extractBody()` — joins `snapshot_symbols` with `snapshot_files` to get the file path, reads content, slices `[startOffset:endOffset]`

**CLI commands:**
- `cmd/codebase-browser/cmds/history/diff.go`: `history diff <old-ref> <new-ref> [--format json] [--only modified]`
- `cmd/codebase-browser/cmds/history/symbol_diff.go`: `history symbol-diff <old-ref> <new-ref> --symbol <id> --name <name>`
- `cmd/codebase-browser/cmds/history/symbol_history.go`: `history symbol-history --symbol <id> --name <name> --limit 50`
- `cmd/codebase-browser/cmds/history/util.go`: shared `findRepoRoot()`, `findSymbolIDByName()`

#### Validation

```bash
# Diff two commits — shows file and symbol changes
codebase-browser history diff HEAD~3 HEAD --db /tmp/test-wt-history.db
# → Files: modified cmd/codebase-browser/main.go
# → Symbols: moved version, moved rootCmd, modified main

# Symbol body diff — shows what actually changed inside a function
codebase-browser history symbol-diff HEAD~3 HEAD \
  --symbol "sym:github.com/.../func.main" \
  --db /tmp/test-wt-history.db
# → Shows: + cobra.CheckErr(history.Register(rootCmd))

# Symbol history — shows every commit where main() had a different body
codebase-browser history symbol-history \
  --symbol "sym:github.com/.../func.main" \
  --db /tmp/test-wt-history.db
# → 50 entries showing body_hash changes: dc8aabe → 3433c83 → 30f72af etc.
```

#### What failed along the way

- `findRepoRoot()` was duplicated in `scan.go` and `symbol_diff.go` — extracted to `util.go`
- Template variable names with hyphens (e.g., `{{.symbol-id}}`) are invalid Go template identifiers — fixed by renaming to underscores in concept files (`{{.symbol_id}}`)

---

### Phase 3: History Concepts — SQL concepts for the history DB

**Commit:** `cd577ef Add history query concepts (Phase 3)`

#### What was implemented

Six SQL concept files in `concepts/history/`:

1. **`commits-timeline.sql`** — Lists indexed commits with symbol counts. Params: `limit`, `branch`.
2. **`pr-summary.sql`** — Summarizes symbol changes between two commits using LEFT JOIN + UNION ALL (avoiding FULL OUTER JOIN compatibility issues). Params: `base`, `head`.
3. **`symbol-changes.sql`** — Detailed symbol diff between two commits. Params: `base`, `head`, `change_type`.
4. **`symbol-history.sql`** — History of a single symbol across all commits. Params: `symbol_id`, `limit`.
5. **`hotspots.sql`** — Most frequently changed symbols ranked by distinct body hash versions. Params: `limit`, `min_versions`.
6. **`file-changes.sql`** — Files changed between two commits with line deltas. Params: `base`, `head`.

#### Key insight: dual-DB usage

History concepts are SQL files that reference `snapshot_*` and `commits` tables — these only exist in `history.db`, not in `codebase.db`. The existing concept system executes against whatever DB the `--db` flag points to. So to run history concepts:

```bash
codebase-browser query --db history.db commands history hotspots --limit 10
```

This works without any code changes to the concept system — the concepts are just SQL that runs against whichever DB you provide.

#### Validation

```bash
codebase-browser query --db /tmp/test-wt-history.db commands history hotspots --limit 10
# → main: 5 distinct versions across 74 commits
# → Server.Handler: 5 distinct versions across 73 commits
# → Extract: 3 distinct versions

codebase-browser query --db /tmp/test-wt-history.db commands history commits-timeline --limit 5
# → Shows 75 commits with symbol counts (330, 330, 330, 330, 317)
```

#### What failed along the way

- Concept param names with hyphens (`symbol-id`, `change-type`, `min-versions`) caused Go template parse errors: `bad character U+002D '-'`. Fixed with `sed -i 's/symbol-id/symbol_id/g'` etc.
- `pr-summary.sql` initially used `FULL OUTER JOIN` which may have compatibility issues with some SQLite builds — rewrote to use `LEFT JOIN ... UNION ALL` pattern.

---

### Phase 4: Server API — HTTP endpoints for history

**Commit:** `84d95aa Add history server API endpoints (Phase 4)`

#### What was implemented

**`internal/server/api_history.go`** — Five HTTP endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/history/commits` | List all indexed commits |
| GET | `/api/history/commits/{hash}` | Single commit detail |
| GET | `/api/history/commits/{hash}/symbols` | All symbols at a commit |
| GET | `/api/history/diff?from=X&to=Y` | Full diff between two commits |
| GET | `/api/history/symbols/{symbolID}/history` | Symbol change timeline |

**`internal/server/server.go`** — Added `History *history.Store` field and `mux *http.ServeMux` field (needed so `registerHistoryRoutes()` can use the new Go 1.22+ `HandleFunc` pattern with method + path).

**`internal/server/api_index.go`** — Added `writeJSONError()` helper (reused by history handlers).

**`cmd/codebase-browser/cmds/serve/run.go`** — Added `--history-db` flag. Opens `history.Store` if provided, attaches to `srv.History`. Warns but continues if unavailable.

#### What failed along the way

- `writeJSON` was already defined in `api_index.go` — had to remove the duplicate from `api_history.go`.
- `writeJSONError` didn't exist yet — added it to `api_index.go` as a shared helper.
- Server initially used a local `mux` variable — needed to store it as `s.mux` so `registerHistoryRoutes()` could register routes on it after construction.
- Initial edit of server.go left malformed text (`} with all routes mounted`) from a bad edit boundary — rewrote the whole file cleanly.

---

### Phase 5: Web UI — React pages for history and diff (in progress)

**Status:** Partially implemented. Files written but not yet built/tested/committed.

#### What was implemented so far

**`ui/src/api/historyApi.ts`** — RTK Query API with 5 endpoints matching the server:
- `listCommits`, `getCommit`, `getCommitSymbols`, `getDiff`, `getSymbolHistory`
- Types: `CommitRow`, `SymbolAtCommit`, `SymbolHistoryEntry`, `FileDiff`, `SymbolDiff`, `DiffStats`, `CommitDiff`

**`ui/src/api/store.ts`** — Registered `historyApi` reducer and middleware.

**`ui/src/features/history/HistoryPage.tsx`** — Main history page with:
- Commit timeline sidebar (selectable old/new commits)
- Auto-selects HEAD and HEAD~3
- Diff view showing file changes (linked to source pages) and symbol changes (linked to symbol pages)
- Color-coded change type badges (green=added, red=removed, orange=modified)

**`ui/src/app/App.tsx`** — Added:
- `HistoryPage` import
- "History" sidebar link
- `/history` route

#### Remaining

- Build frontend, regenerate web assets
- Validate in browser
- Commit

---

### Summary of commits so far

| Hash | Message |
|------|---------|
| `33d10c6` | Add git-aware codebase history index (Phase 1) |
| `40d3455` | Add diff engine and history CLI commands (Phase 2) |
| `cd577ef` | Add history query concepts (Phase 3) |
| `84d95aa` | Add history server API endpoints (Phase 4) |
