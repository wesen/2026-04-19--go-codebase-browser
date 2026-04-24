---
Title: Investigation Diary SQLite Index
Ticket: GCB-007
Status: active
Topics:
    - sqlite
    - wasm
    - go
    - search
    - indexing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/sqlite/store.go
      Note: Store creation/opening and DB access for the SQLite index
    - Path: internal/sqlite/schema.go
      Note: Canonical SQLite table/index schema for codebase-browser
    - Path: internal/sqlite/loader.go
      Note: Bulk loader from indexer.Index into SQLite tables
    - Path: internal/sqlite/query.go
      Note: Predicate-based symbol query helpers used by Go-side consumers
    - Path: internal/sqlite/generate_build.go
      Note: go generate entry point that builds codebase.db from the current index
ExternalSources: []
Summary: "Diary for implementing GCB-007 SQLite index support."
LastUpdated: 2026-04-23T23:07:05.676309051-04:00
WhatFor: "Record the implementation steps, failures, validation commands, and review notes for SQLite-backed codebase-browser indexing."
WhenToUse: "Use when continuing GCB-007, reviewing SQLite support, or debugging the codebase.db generation/query path."
---

# Diary

## Goal

Record the implementation journey for GCB-007 as `codebase-browser` moves from JSON/WASM lookup toward SQLite-backed Go-side indexing and CLI querying. Each step captures what changed, how it was validated, and what a reviewer should inspect before the next step builds on it.

## Step 1: Add the SQLite store package

This step introduced the first real Go-side SQLite implementation. The goal was deliberately narrow: create the canonical SQLite schema, load an existing `indexer.Index` into the database, expose a small predicate-based symbol query API, and add `go generate` plumbing that can produce `internal/sqlite/embed/codebase.db` from the current generated index.

This does not yet add the user-facing CLI command. It establishes the library layer the CLI will call in the next step.

### Prompt Context

**User prompt (verbatim):** "alright, go ahead, implement task by task, commit at a ppropriate intervals and keep a diary as you work"

**Assistant interpretation:** Implement the GCB-007 SQLite migration incrementally, commit coherent slices, and update the ticket diary as the work proceeds.

**Inferred user intent:** Move from design/prototype to real Go-side SQLite support while preserving a clear audit trail for review and continuation.

**Commit (code):** d70eaee166094303d93c5a7a076be9aae9a9b4c3 — "Add SQLite index store package"

### What I did

- Created `internal/sqlite/`.
- Added `store.go` with `Open`, `Create`, `DB`, `Close`, and SQLite connection configuration.
- Added `schema.go` with canonical tables: `meta`, `packages`, `files`, `symbols`, and `refs`.
- Added indexes for package, file, symbol, and ref lookup patterns.
- Added `loader.go` to bulk-load `indexer.Index` values into the relational schema.
- Added `query.go` with composable symbol predicates: `ByKind`, `ByPackage`, `NameLike`, `IsExported`, and `Limit`.
- Added optional FTS5 plumbing in `fts5.go` behind the `sqlite_fts5` build tag and a no-op fallback in `fts5_disabled.go`.
- Added `generate.go` and `generate_build.go` so `go generate ./internal/sqlite` can build a database from `internal/indexfs/embed/index.json`.
- Added `store_test.go` to verify row counts and predicate querying on a small in-memory test index.

### Why

The browser-side SQLite prototype proved the idea works, but the base app still needed a real Go package. The CLI should not talk to prototype scripts; it should call a stable internal package that owns schema creation, loading, and query helpers.

The package intentionally treats SQLite as the canonical Go-side index representation. The design no longer includes a backwards-compatible runtime JSON path.

### What worked

- `gofmt -w internal/sqlite` completed successfully.
- `go test ./internal/sqlite -count=1` passed.
- `go test ./... -count=1` passed.
- The first implementation slice was committed as `d70eaee166094303d93c5a7a076be9aae9a9b4c3`.

### What didn't work

- I initially added `//go:embed embed/codebase.db` directly to `internal/sqlite/generate.go`. That would make the package fail to compile before the first generated database exists, and it would also make `go run generate_build.go` import a package that requires the output file it is supposed to create.
- I removed the embed from `generate.go` for now. The first Go-side goal is CLI querying from an explicit DB path; embedded DB shipping can be added later once generation is stable.

### What I learned

The SQLite generation path should not require the generated database to already exist. Embedding `codebase.db` is a packaging concern, not a prerequisite for the first CLI-oriented implementation slice.

### What was tricky to build

The main sharp edge was the relationship between `go generate`, `go:embed`, and package compilation. Because `generate_build.go` imports `internal/sqlite`, anything required by that package must already exist before generation runs. A missing embedded DB would create a bootstrapping problem. The simpler solution is to keep generation and embedding separate until the database file is consistently produced.

### What warrants a second pair of eyes

- The schema column choices in `symbols` and `refs`, especially whether JSON columns for `type_params_json`, `tags_json`, and `build_tags_json` are sufficient for near-term querying.
- The default language normalization (`""` → `"go"`) in the loader.
- Whether `ByPackage` should continue accepting either package ID or import path, or whether the CLI should expose those as separate flags later.

### What should be done in the future

- Add the CLI `query` command that opens a SQLite DB and executes raw SQL or `.sql` files.
- Add real query examples under `queries/`.
- Run `go generate ./internal/sqlite` against the real generated index and verify row counts.
- Decide later whether `codebase.db` should be embedded into a binary, copied into `dist/`, or both.

### Code review instructions

Start with:

- `internal/sqlite/schema.go`
- `internal/sqlite/loader.go`
- `internal/sqlite/query.go`
- `internal/sqlite/store_test.go`

Validate with:

```bash
gofmt -w internal/sqlite
go test ./internal/sqlite -count=1
go test ./... -count=1
```

### Technical details

The first expected smoke query after database generation is:

```bash
codebase-browser query "SELECT COUNT(*) FROM symbols"
```

That command does not exist yet; it is the next implementation slice.

## Step 2: Add the SQLite query CLI

This step made the SQLite database useful from the base app. The new top-level `codebase-browser query` command opens `codebase.db`, executes ad-hoc SQL from either an argument or a `.sql` file, and prints either tab-separated table output or JSON.

The important architectural result is that we can now interact with the codebase index through SQLite without touching the browser. This gives us a fast validation loop for the Go-side schema before any frontend migration work begins.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the GCB-007 implementation task-by-task, adding the next practical slice after the store package.

**Inferred user intent:** Make SQLite observable and useful through the CLI so the base app can query the codebase before browser integration.

**Commit (code):** dc5718614ccfe97b1213317ff73eef930756dc66 — "Add SQLite query CLI"

### What I did

- Added `cmd/codebase-browser/cmds/query/query.go`.
- Registered the new `query` command in `cmd/codebase-browser/main.go`.
- Added `--db`, `--file/-f`, and `--format` flags.
- Supported raw SQL via `codebase-browser query "..."`.
- Supported file-based SQL via `codebase-browser query -f path/to/file.sql`.
- Added JSON output via `--format json`.
- Added reusable query files:
  - `queries/packages/package-counts.sql`
  - `queries/symbols/exported-functions.sql`
  - `queries/symbols/most-referenced.sql`
  - `queries/refs/refs-for-symbol.sql`
- Updated `.gitignore` so generated `internal/sqlite/embed/codebase.db` is not accidentally committed.
- Adjusted the `refs` table schema so `from_symbol_id` and `to_symbol_id` are indexed text columns rather than strict foreign keys.

### Why

The CLI is the safest first consumer for the SQLite backend. It lets us inspect the database, test queries, and evolve the schema before the browser depends on it.

The `refs` schema change was necessary because the real index contains references to external symbols, such as standard-library symbols, that are not present as local rows in `symbols`.

### What worked

The following commands succeeded:

```bash
gofmt -w cmd/codebase-browser/main.go cmd/codebase-browser/cmds/query/query.go
go test ./cmd/codebase-browser ./cmd/codebase-browser/cmds/query ./internal/sqlite -count=1
go generate ./internal/sqlite
go run ./cmd/codebase-browser query "SELECT COUNT(*) AS symbols FROM symbols"
go run ./cmd/codebase-browser query -f queries/packages/package-counts.sql | head -10
go run ./cmd/codebase-browser query --format json "SELECT name, kind FROM symbols ORDER BY name LIMIT 3"
go test ./... -count=1
```

`go generate ./internal/sqlite` produced:

```text
generate_build: wrote /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/sqlite/embed/codebase.db (27 packages, 76 files, 329 symbols, 1005 refs)
```

The count smoke test returned:

```text
symbols
329
```

### What didn't work

The first real `go generate ./internal/sqlite` run failed with:

```text
2026/04/24 13:09:22 load index into sqlite: insert ref sym:github.com/wesen/codebase-browser/cmd/build-ts-index.func.envDefault -> sym:os.func.Getenv: FOREIGN KEY constraint failed
exit status 1
internal/sqlite/generate.go:3: running "go": exit status 1
```

The failure was correct: the real index includes references to external symbols that are not defined in the local codebase index. The schema had incorrectly assumed that every ref endpoint would be a local symbol row.

### What I learned

Cross-reference tables must represent both local and external targets. Strict foreign keys on `refs.from_symbol_id` and `refs.to_symbol_id` are too narrow for a codebase index unless we also introduce an `external_symbols` table or synthesize placeholder symbol rows. For this implementation slice, indexed text columns are the simplest correct representation.

### What was tricky to build

The dynamic SQL output path had to handle unknown result columns because the command accepts arbitrary SQL. The CLI scans rows into `[]any`, normalizes `[]byte` to strings for readability, and renders either tab-separated output or JSON.

The schema issue was trickier because it only appears with real index data, not with the small unit-test fixture. The test fixture used only local refs, while the generated self-index included standard-library targets.

### What warrants a second pair of eyes

- Whether arbitrary SQL should continue using plain tab-separated output or should use Glazed rows for integration with the rest of the CLI output stack.
- Whether external refs should remain plain strings or become explicit rows in a future `external_symbols` table.
- Whether the CLI should treat non-SELECT statements as supported behavior long-term. It currently falls back from `QueryContext` to `ExecContext`.

### What should be done in the future

- Add tests around the CLI command itself, ideally with a temporary database and captured output.
- Verify FTS5 with `go test -tags sqlite_fts5 ./internal/sqlite -count=1` and a real search query.
- Decide whether to expose higher-level commands such as `query symbols --name X` or keep raw SQL as the main CLI interface.

### Code review instructions

Start with:

- `cmd/codebase-browser/cmds/query/query.go`
- `cmd/codebase-browser/main.go`
- `internal/sqlite/schema.go`
- `queries/`

Validate with:

```bash
go generate ./internal/sqlite
go run ./cmd/codebase-browser query "SELECT COUNT(*) AS symbols FROM symbols"
go run ./cmd/codebase-browser query -f queries/packages/package-counts.sql | head -10
go run ./cmd/codebase-browser query --format json "SELECT name, kind FROM symbols ORDER BY name LIMIT 3"
go test ./... -count=1
```

### Technical details

The CLI defaults to:

```text
internal/sqlite/embed/codebase.db
```

That file is generated by `go generate ./internal/sqlite` and intentionally ignored by Git for now.

## Step 3: Verify optional SQLite FTS5 support

This step closed the remaining verification gap around FTS5. The store already had an `EnableFTS5` method behind the `sqlite_fts5` build tag, but it needed a test that actually creates the virtual table and runs a MATCH query.

The result is a small build-tagged test that proves the Go-side SQLite driver and schema can support full-text search when the correct tag is enabled.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing and validating the SQLite Go-side tasks in small committed slices.

**Inferred user intent:** Ensure the SQLite implementation is not just structurally present, but tested for the features the design depends on.

**Commit (code):** 2c5e7525e06ea91a6c78d9254b4d86dc9b355f83 — "Verify SQLite FTS5 setup"

### What I did

- Added `internal/sqlite/fts5_test.go` behind the `sqlite_fts5` build tag.
- Created a tiny index fixture with one symbol whose docs contain searchable text.
- Loaded the fixture into SQLite.
- Ran `EnableFTS5`.
- Queried `symbol_fts` with `MATCH` and verified the expected symbol was returned.

### Why

The SQLite design relies on FTS5 for richer search. We had already seen that FTS5 availability differs by environment, so the Go-side implementation needed an explicit build-tagged validation path.

### What worked

Both tagged and untagged tests passed:

```bash
gofmt -w internal/sqlite/fts5_test.go
go test -tags sqlite_fts5 ./internal/sqlite -count=1
go test ./internal/sqlite -count=1
```

### What didn't work

The code/test work had no failures. While updating docs, I attempted a quick Python one-liner to patch the changelog and introduced a quoted newline syntax error:

```text
  File "<stdin>", line 4
    s=s.replace('Code commit: `dc5718614ccfe97b1213317ff73eef930756dc66`.', 'Code commit: `dc5718614ccfe97b1213317ff73eef930756dc66`.
                                                                            ^
SyntaxError: unterminated string literal (detected at line 4)
```

I replaced that with a direct targeted edit to `changelog.md`.

### What I learned

The existing `github.com/mattn/go-sqlite3` dependency supports this FTS5 path when the `sqlite_fts5` build tag is passed. Keeping the test behind the same tag avoids forcing FTS5 on every normal test run.

### What was tricky to build

The key detail was keeping the test build-tagged. Without the tag, the no-op `EnableFTS5` implementation is compiled, so a MATCH query against `symbol_fts` would be invalid. The test therefore belongs only in the tagged FTS5 compilation mode.

### What warrants a second pair of eyes

- Whether `EnableFTS5` should use triggers for ongoing sync once the DB becomes mutable, or whether build-time bulk creation is sufficient.
- Whether the FTS table should include package import paths through denormalized content rather than only `package_id`.

### What should be done in the future

- Add real search query examples once the CLI grows a search-oriented command or documented FTS query files.
- Decide whether generated production databases should always use `-tags sqlite_fts5`.

### Code review instructions

Review:

- `internal/sqlite/fts5.go`
- `internal/sqlite/fts5_disabled.go`
- `internal/sqlite/fts5_test.go`

Validate with:

```bash
go test -tags sqlite_fts5 ./internal/sqlite -count=1
go test ./internal/sqlite -count=1
```

### Technical details

The FTS query under test is:

```sql
SELECT name
FROM symbol_fts
WHERE symbol_fts MATCH ?
LIMIT 1;
```
