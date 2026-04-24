# Tasks

## Goal

Add SQLite support to the Go side of `codebase-browser` first, so the CLI can query the code index directly before any browser-side switch happens.

## Phase 1 — Introduce the SQLite store package

- [ ] Create `internal/sqlite/` as a new package.
- [ ] Add `store.go` with `Store`, `New()`, `Close()`, and `DB()`.
- [ ] Add `schema.go` with the `CREATE TABLE`, `CREATE INDEX`, and trigger statements.
- [ ] Add `loader.go` to bulk-load `index.json` into SQLite.
- [ ] Add `query.go` with a predicate/query-builder API for symbol, package, file, and ref lookups.
- [ ] Add `fts5.go` behind a `sqlite_fts5` build tag.

## Phase 2 — Make SQLite usable from the CLI

- [ ] Add a new `query` sub-command under `cmd/codebase-browser/`.
- [ ] Wire the CLI to open `codebase.db` and execute SQL or predicate-built queries.
- [ ] Add support for `query <sql>` and `query -f <file.sql>`.
- [ ] Add a `queries/` directory with reusable example SQL files.
- [ ] Keep the existing `serve` command working while SQLite is introduced.

## Phase 3 — Build the database at generate time

- [ ] Add `internal/sqlite/generate.go` to embed `codebase.db` when needed.
- [ ] Add `internal/sqlite/generate_build.go` to build `codebase.db` from `index.json`.
- [ ] Add a `go generate ./internal/sqlite` workflow.
- [ ] Ensure the database build matches the current index counts for packages, files, symbols, and refs.

## Phase 4 — Tests and verification

- [ ] Add integration tests for loading the self-index into SQLite.
- [ ] Verify symbol counts and ref counts match the JSON index.
- [ ] Verify FTS5 search works when the build tag is enabled.
- [ ] Verify the CLI can run representative queries from `.sql` files.

## Suggested first implementation slice

1. Build the SQLite schema.
2. Load the index data into SQLite.
3. Make `codebase-browser query "SELECT COUNT(*) FROM symbols"` work.
4. Add one or two real query files under `queries/`.

## Exit criteria for the Go-side phase

- [ ] `go generate ./internal/sqlite` produces a valid `codebase.db`.
- [ ] The CLI can query the DB without the browser.
- [ ] The schema, loader, and query paths are the only supported Go-side index paths.
- [ ] The implementation is documented well enough for the browser-side migration to follow later.
