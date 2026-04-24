# Tasks

## Goal

Add SQLite support to the Go side of `codebase-browser` first, so the CLI can query the code index directly before any browser-side switch happens.

## Phase 1 — Introduce the SQLite store package

- [x] Create `internal/sqlite/` as a new package.
- [x] Add `store.go` with `Store`, `Open()`, `Create()`, `Close()`, and `DB()`.
- [x] Add `schema.go` with the `CREATE TABLE` and `CREATE INDEX` statements.
- [x] Add `loader.go` to bulk-load index data into SQLite.
- [ ] Add `query.go` with a predicate/query-builder API for symbol, package, file, and ref lookups. _(Symbol predicates are implemented; package/file/ref helpers still need follow-up if desired.)_
- [x] Add `fts5.go` behind a `sqlite_fts5` build tag.

## Phase 2 — Make SQLite usable from the CLI

- [x] Add a new `query` sub-command under `cmd/codebase-browser/`.
- [x] Wire the CLI to open `codebase.db` and execute SQL queries.
- [x] Add support for `query <sql>` and `query -f <file.sql>`.
- [x] Add a `queries/` directory with reusable example SQL files.
- [x] Keep the existing `serve` command working while SQLite is introduced.

## Phase 3 — Build the database at generate time

- [x] Add `internal/sqlite/generate.go` with `go generate` wiring. _(Embedding is deferred until packaging needs it.)_
- [x] Add `internal/sqlite/generate_build.go` to build `codebase.db` from the generated index.
- [x] Add a `go generate ./internal/sqlite` workflow.
- [x] Ensure the database build matches the current index counts for packages, files, symbols, and refs.

## Phase 4 — Tests and verification

- [x] Add integration tests for loading an index into SQLite.
- [x] Verify symbol counts and ref counts match the generated index during `go generate ./internal/sqlite` smoke testing.
- [x] Verify FTS5 search works when the build tag is enabled.
- [x] Verify the CLI can run representative queries from `.sql` files.

## Suggested first implementation slice

1. Build the SQLite schema.
2. Load the index data into SQLite.
3. Make `codebase-browser query "SELECT COUNT(*) FROM symbols"` work.
4. Add one or two real query files under `queries/`.

## Exit criteria for the Go-side phase

- [x] `go generate ./internal/sqlite` produces a valid `codebase.db`.
- [x] The CLI can query the DB without the browser.
- [x] The schema, loader, and query paths are the only supported Go-side index paths.
- [ ] The implementation is documented well enough for the browser-side migration to follow later.
