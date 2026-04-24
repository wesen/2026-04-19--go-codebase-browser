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
