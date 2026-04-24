---
Title: SQLite Index Architecture and Design
Ticket: GCB-007
Status: active
Topics:
    - sqlite
    - wasm
    - go
    - search
    - indexing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/main.go
      Note: CLI entry point where query sub-command will be added
    - Path: internal/bundle/generate_build.go
      Note: Bundler that will ship codebase.db instead of search.wasm
    - Path: internal/indexer/types.go
      Note: The Index/Package/File/Symbol/Ref types that map to the SQLite schema tables
    - Path: internal/server/api_index.go
      Note: Server endpoints that can query SQLite directly
    - Path: internal/static/generate_build.go
      Note: Pre-computation step that SQLite generation replaces
    - Path: internal/static/search_index.go
      Note: Custom inverted index that FTS5 replaces
    - Path: internal/static/xref_index.go
      Note: Custom xref maps that SQL refs table replaces
    - Path: internal/wasm/search.go
      Note: The WASM SearchCtx that SQLite+sql.js will replace
    - Path: ui/src/api/indexApi.ts
      Note: RTK-Query API that switches baseQuery to SQLite
    - Path: ui/src/api/wasmClient.ts
      Note: WASM client that dbClient.ts will replace
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# GCB-007: SQLite Codebase Index

## Executive Summary

This document describes a plan to replace the current `index.json` +
`precomputed.json` + `search.wasm` data pipeline in `codebase-browser` with a
single **SQLite database file** called `codebase.db`.

SQLite is a lightweight, serverless, zero-configuration SQL database engine
that stores an entire database in a single file. It is the most widely deployed
database engine in the world — it runs inside every smartphone, every browser,
every copy of macOS and Windows, and most embedded devices. Unlike PostgreSQL
or MySQL, SQLite doesn't require a running server process. You open a file, run
SQL queries against it, and close it. The entire database is a single
cross-platform file that you can copy, email, or embed in a binary.

The key idea is this: a codebase index is **inherently relational**. Symbols
live inside packages. Files belong to packages. Cross-references connect one
symbol to another. Today we flatten all these relationships into JSON arrays,
and then the consumer (a WASM module in the browser, or the Go server) has to
rebuild the relationships by scanning and indexing the arrays by hand. SQLite
gives us proper tables, indexes, joins, and full-text search out of the box —
in a file format that both Go programs and web browsers can read natively.

The database will be built once at `go generate` time (the same build step
that currently produces `index.json`), and then consumed in three contexts:

- **The Go server** opens `codebase.db` directly when you run
  `codebase-browser serve`.
- **The CLI** runs ad-hoc SQL queries via a new `codebase-browser query`
  sub-command.
- **The browser** loads `codebase.db` via `sql.js` (a JavaScript library that
  embeds a complete SQLite engine compiled to WebAssembly) and runs the same
  SQL queries client-side with zero server dependency.

This design draws on two existing codebases in the wesen/corporate-headquarters
monorepo that you should study as reference:

1. **go-minitrace** — a tool that stores transcript sessions (recordings of AI
   coding agent conversations) in SQLite and DuckDB. It has a `queries/`
   directory full of `.sql` files that you can run from the command line.
   You'll find it at `~/code/wesen/corporate-headquarters/go-minitrace/`.

2. **glazed help system** — a SQLite-backed help documentation system with
   composable query predicates and FTS5 full-text search. You'll find it at
   `~/code/wesen/corporate-headquarters/glazed/pkg/help/store/`.

---

## Background: What codebase-browser Does Today

Before we get into the design, you need to understand what the current system
looks like. This section walks through every piece of the existing pipeline.

### The indexer

The whole system starts with the **indexer** — a Go package at
`internal/indexer/` that parses Go source code using the `go/packages` and
`go/ast` standard library packages. It walks every `.go` file in the repo,
extracts every function, method, type, interface, constant, and variable
declaration, and writes out a big JSON file called `index.json`.

The indexer also handles TypeScript code: there's a separate Node.js-based
extractor at `tools/ts-indexer/` that parses TypeScript files using the
TypeScript Compiler API and produces the same JSON shape. Both extractors emit
records into the same `index.json`.

The JSON file has this top-level shape (see `internal/indexer/types.go`):

- **`packages[]`** — each Go or TypeScript package. A package has an import
  path (like `github.com/wesen/codebase-browser/internal/server`), a name
  (like `server`), and lists of the file IDs and symbol IDs it contains.

- **`files[]`** — each source file. A file has a path, a parent package ID,
  its size in bytes, its line count, and a SHA256 hash.

- **`symbols[]`** — each named declaration: functions, methods, types,
  interfaces, constants, variables. A symbol has a kind, a name, the package
  and file it lives in, byte-range coordinates (start line/col, end line/col),
  a doc comment, a function signature, and whether it's exported (public).

- **`refs[]`** — cross-references between symbols. Each ref says "symbol A
  references symbol B" with the kind of reference (call, embed, implements,
  etc.) and the file/line where the reference occurs.

When you run `go generate ./internal/indexfs`, the indexer reads all the Go
source code in the repo and writes `internal/indexfs/embed/index.json`. This
file is about 705 KB when the tool indexes itself.

### The pre-computation step

A second build step at `internal/static/generate_build.go` reads `index.json`
and produces `precomputed.json` (about 1.9 MB). This file contains:

- **`searchIndex`** — a map from lowercase name substrings to symbol IDs.
  This is a simple inverted index that makes substring search fast without
  scanning every symbol.

- **`xrefIndex`** — a map from symbol ID to its "used by" and "uses" lists.
  Instead of scanning all refs at query time, we pre-compute per-symbol
  reference lists.

- **`snippets`** — extracted source text for each symbol (its declaration,
  body, and signature). The browser needs these for displaying code snippets
  in symbol cards and cross-reference panels.

- **`docHTML`** — pre-rendered HTML for documentation pages (written in
  Markdown with custom directives for embedding symbol cards).

- **`indexJSON`** — the raw `index.json` content embedded verbatim, because
  the frontend needs the full package/symbol arrays.

The reason we pre-compute all this is that the browser-side WASM module (see
below) is slow at JSON parsing and doesn't have access to the source files.
So we do all the expensive work at build time and ship the results.

### The WASM module

A Go program at `cmd/wasm/main.go` is compiled to WebAssembly (WASM) using
TinyGo via a Dagger container. The resulting `search.wasm` file is about 1.2
MB. When loaded in the browser, it:

1. Parses six JSON strings (the index, search index, xref index, snippets,
   doc manifest, and doc HTML).
2. Builds in-memory lookup maps (symbol ID → symbol, package ID → package,
   etc.).
3. Exposes JavaScript-callable functions like `findSymbols(query, kind)`,
   `getSymbol(id)`, `getXref(id)`, and `getDocPage(slug)`.

The React frontend uses RTK-Query (a data fetching library for Redux) with a
custom `wasmBaseQuery` that routes API calls to these WASM functions instead
of making HTTP requests to a server.

This approach works — we verified it with Playwright end-to-end tests in
ticket GCB-006 — but it's heavy. The WASM module is a 1.2 MB program whose
entire job is parsing JSON and serving map lookups. JavaScript can do map
lookups natively.

### The static artifact

The bundler at `internal/bundle/generate_build.go` assembles everything into
a `dist/` directory:

- The Vite-built SPA (HTML, JS, CSS)
- `search.wasm` (1.2 MB)
- `wasm_exec.js` (16 KB — the TinyGo runtime)
- `precomputed.json` (1.9 MB)
- `source/` directory tree (full source files for viewing)

The total is about 3.8 MB. You can open `dist/index.html` in any browser,
even from `file://`, and the full codebase browser works with no server.

---

## Problem Statement

Now that you understand the current system, here's what's wrong with it.

### The data is relational but stored flat

The most fundamental problem is that the data model is relational — symbols
belong to packages, files belong to packages, refs connect symbols — but
we store it as flat JSON arrays. Every consumer has to rebuild the
relationships by scanning the arrays and building lookup maps.

Think about what happens when the browser needs to show a symbol's page. It
needs:

1. The symbol itself (look up by ID in the symbols array)
2. The symbol's package (scan packages array for matching packageId)
3. The symbol's file (scan files array for matching fileId)
4. All refs TO this symbol (scan refs array for matching toId)
5. All refs FROM this symbol (scan refs array for matching fromId)

Each of these is a linear scan or a pre-built map. In SQL, each would be a
single indexed query:

```sql
SELECT * FROM symbols WHERE id = ?
SELECT * FROM packages WHERE id = ?
SELECT * FROM refs WHERE to_id = ?
```

### No full-text search

Symbol search is pure substring matching: `strings.Contains(name, query)`.
There's no ranking, no tokenization, no stemming, no searching across doc
comments or file paths. If you search for "handle" you get every symbol whose
name contains "handle" in any position, in no particular order. SQLite's FTS5
(Full-Text Search 5) extension gives us proper tokenization, ranking with
BM25, phrase matching, and NEAR queries.

### Redundant data in precomputed.json

The `precomputed.json` file contains a lot of duplicated data. The raw
`indexJSON` is embedded verbatim, and then the search index, xref index, and
snippets are all denormalized copies of data already in the index. For the
self-indexing case (the tool browsing its own source), `precomputed.json` is
1.9 MB — almost three times larger than the original `index.json` (705 KB).

### The WASM module is expensive overhead for simple lookups

The WASM approach was an interesting experiment (documented in GCB-006), but
it adds complexity without proportional benefit. The module is a 1.2 MB Go
program that:

- Requires TinyGo (a specialized Go compiler for WebAssembly) or Dagger
  containers to build
- Needs a separate `wasm_exec.js` runtime file (and the correct version —
  Go's and TinyGo's are different)
- Has an asynchronous initialization ceremony (load WASM, wait for exports to
  appear on `window`, then call initWasm with six JSON strings)
- Ultimately just parses JSON into Go maps and serves lookups

JavaScript handles maps and substring search natively. With SQLite in the
browser, we can replace the entire WASM module with `sql.js` — a well-tested
library that embeds the full SQLite engine as an 800 KB WASM binary.

### No CLI query interface

If you want to ask a question about the codebase from the command line —
"show me all exported functions with no doc comments" — you currently have
two options: open the browser, or write a one-off Go program that imports
`internal/browser` and walks the index data structures. There's no
`codebase-browser query "SELECT ..."` command. With a SQLite database,
this becomes trivial.

---

## Reference System 1: go-minitrace

go-minitrace is a tool for analyzing transcripts (recordings) of AI coding
agent sessions. It stores session data in SQLite and DuckDB databases, and
provides a library of pre-written SQL queries for common analyses.

### How it stores data

The key file to study is `pkg/adapters/turnsdb/convert.go`. Each data source
(an AI coding agent like Codex, Claude, or Pi) has an adapter that reads its
native format and writes into a canonical SQLite schema. The adapter pattern
means you can add support for a new agent by writing a single file.

### The queries/ directory

go-minitrace has a `queries/` directory tree like this:

```
queries/
├── tools/
│   ├── tool-failures.sql
│   ├── tool-operation-breakdown.sql
│   └── exit-codes.sql
├── timing/
│   └── timing-analysis.sql
├── framework-metadata/
│   └── pi-edit-diffs.sql
└── my-queries/
    └── .gitignore
```

Each `.sql` file is self-contained. It has a comment header explaining its
purpose, and a SELECT statement that queries the canonical schema. Users run
them with:

```bash
duckdb analysis.duckdb -f queries/tools/tool-failures.sql
```

The `my-queries/` directory is gitignored — users drop their own ad-hoc
queries there without polluting the git history. This is a pattern we should
adopt for codebase-browser.

### What we borrow

- The **adapter pattern** for data sources (Go indexer, TypeScript indexer)
- The **queries/ directory tree** with self-documenting `.sql` files
- The **my-queries/ gitignored sandbox** for ad-hoc exploration

---

## Reference System 2: Glazed Help Store

The glazed help system is a SQLite-backed documentation store used by the
Glazed CLI framework. The relevant code is at
`~/code/wesen/corporate-headquarters/glazed/pkg/help/store/`.

### Architecture

The store has a clean separation of concerns:

- **`model/section.go`** — domain types (Section, SectionType enum)
- **`store.go`** — the Store struct with `New()`, `NewInMemory()`, `Close()`,
  and CRUD methods. It opens a SQLite database and creates tables on first use.
- **`schema creation`** — CREATE TABLE and CREATE INDEX statements are
  embedded in the store initialization. FTS5 virtual tables are behind a
  build tag (`//go:build sqlite_fts5`).
- **`query.go`** — a composable predicate system. Each predicate is a
  function that modifies a QueryBuilder (adds WHERE clauses, JOINs, ORDER BY).
  Predicates compose: `And(HasTopic("x"), IsExample(), Limit(10))`.
- **`loader.go`** — reads markdown files from an `embed.FS` and inserts them
  into the store.

### The predicate pattern

This is the most important pattern to understand. Instead of writing raw SQL
everywhere, you build queries from composable building blocks:

```go
// Each predicate is a function that adds SQL clauses to a builder
type Predicate func(*QueryBuilder)

// "Find all examples about configuration, limited to 10"
results, _ := store.Find(
    store.And(
        store.IsExample(),
        store.HasTopic("configuration"),
        store.Limit(10),
    ),
)
```

Internally, each predicate mutates a `QueryBuilder`:

```go
func HasTopic(topic string) Predicate {
    return func(qb *QueryBuilder) {
        qb.AddWhere("topics LIKE ?", "%"+topic+"%")
    }
}
```

The builder then compiles all the accumulated WHERE clauses, JOINs, and
parameters into a single SQL statement. This gives you the expressiveness of
SQL without the risk of SQL injection or string concatenation errors.

### What we borrow

- The **Store struct pattern** (`New()`, `NewInMemory()`, CRUD methods)
- The **composable predicate system** for building queries programmatically
- The **FTS5 integration** with automatic triggers to keep the search index
  in sync with the data tables
- The **in-memory mode** (`:memory:`) for embedded use in the Go server

---

## Database Schema Design

This section describes every table in the SQLite database, what each column
means, and why the indexes exist. If you're new to SQL, think of a table as a
spreadsheet with rows and columns, an index as a sorted lookup table that makes
finding rows fast (like the index at the back of a textbook), and a foreign key
as a link from one row to another row in a different table.

### Entity-relationship overview

The database has five main tables plus one virtual table for full-text search.
The relationships form a star around the `symbols` table:

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│  packages    │       │    files     │       │   symbols    │
├──────────────┤       ├──────────────┤       ├──────────────┤
│ id PK        │──┐    │ id PK        │──┐    │ id PK        │
│ import_path  │  │    │ path         │  │    │ name         │
│ name         │  │    │ package_id FK│──┘    │ kind         │
│ doc          │  │    │ language     │       │ package_id FK│──┐
│ language     │  │    │ size         │       │ file_id FK   │  │
│              │  │    │ line_count   │       │ start_line   │  │
│              │  │    │ sha256       │       │ signature    │  │
└──────────────┘  │    └──────────────┘       │ doc          │  │
                  │                           │ exported     │  │
                  │    ┌──────────────┐       └──────────────┘  │
                  │    │    refs      │                          │
                  │    ├──────────────┤                          │
                  └───>│ from_id FK   │──┐    ┌──────────────────┐
                       │ to_id FK     │──┘    │ index_meta       │
                       │ kind         │       ├──────────────────┤
                       │ file_id FK   │──────>│ version          │
                       │ start_line   │       │ generated_at     │
                       └──────────────┘       │ module           │
                                              └──────────────────┘
```

Each arrow represents a foreign key relationship: the `package_id` column in
`files` points to the `id` column in `packages`, and so on.

### Table: `packages`

One row per Go or TypeScript package. A package is the unit of organization
in Go — every `.go` file lives in exactly one package, and a package's name
is the `package` declaration at the top of each file. In TypeScript, we treat
a directory as a package (since TypeScript doesn't have Go's strict package
system).

The `id` column uses the format `pkg:<import_path>` (e.g.,
`pkg:github.com/wesen/codebase-browser/internal/server`). The `import_path`
is how Go identifies packages uniquely — it includes the module path and the
relative directory path. The `doc` column holds the package-level doc comment
(the comment block immediately above the `package` declaration). The
`language` column is `"go"` or `"ts"` so we can filter by language.

```sql
CREATE TABLE IF NOT EXISTS packages (
    id          TEXT PRIMARY KEY,
    import_path TEXT NOT NULL,
    name        TEXT NOT NULL,
    doc         TEXT    DEFAULT '',
    language    TEXT    DEFAULT 'go'
);

CREATE INDEX IF NOT EXISTS idx_packages_import_path ON packages(import_path);
CREATE INDEX IF NOT EXISTS idx_packages_language    ON packages(language);
```

**Why the indexes?** We query packages by import path frequently (e.g., "show
me all symbols in `internal/server`"), so an index on `import_path` makes that
lookup O(log N) instead of O(N). The language index supports filtering by
language.

### Table: `files`

One row per source file. The `path` is relative to the repo root (e.g.,
`internal/server/server.go`). The `package_id` is a foreign key pointing to
the `packages` table — every file belongs to exactly one package. We store
`size` (bytes) and `line_count` for summary statistics, and `sha256` for
integrity checking (to know if a file has changed since the last index).

```sql
CREATE TABLE IF NOT EXISTS files (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL,
    package_id  TEXT NOT NULL REFERENCES packages(id),
    language    TEXT    DEFAULT 'go',
    size        INTEGER NOT NULL DEFAULT 0,
    line_count  INTEGER NOT NULL DEFAULT 0,
    sha256      TEXT    DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_files_package_id ON files(package_id);
CREATE INDEX IF NOT EXISTS idx_files_path       ON files(path);
```

**Why `REFERENCES packages(id)`?** This is a foreign key constraint. It tells
SQLite that every `package_id` value must exist as an `id` in the `packages`
table. SQLite will enforce this — you can't insert a file with a non-existent
package. This prevents data corruption.

### Table: `symbols`

This is the most important table. One row per named declaration: every
function, method, type, interface, constant, and variable gets a row.

The `kind` column uses a controlled vocabulary: `"func"`, `"method"`,
`"type"`, `"interface"`, `"const"`, `"var"`, and TypeScript-specific kinds
like `"class"`, `"arrow_func"`, `"enum"`.

The `receiver` column is specific to Go methods. In Go, a method is a function
with a "receiver" — the type it operates on. For example, `func (s *Server)
Start()` has receiver `Server` with `receiver_pointer = true`. For regular
functions, `receiver` is NULL.

The `exported` column is a boolean: `true` if the symbol name starts with an
uppercase letter (Go's convention for public visibility) or is marked
`export` in TypeScript.

The `start_line`/`start_col`/`end_line`/`end_col` columns record where the
symbol's declaration appears in its source file. These are used by the browser
to highlight the right lines when you view a symbol.

```sql
CREATE TABLE IF NOT EXISTS symbols (
    id          TEXT PRIMARY KEY,
    kind        TEXT NOT NULL,
    name        TEXT NOT NULL,
    package_id  TEXT NOT NULL REFERENCES packages(id),
    file_id     TEXT NOT NULL REFERENCES files(id),
    start_line  INTEGER NOT NULL,
    start_col   INTEGER NOT NULL DEFAULT 0,
    end_line    INTEGER NOT NULL,
    end_col     INTEGER NOT NULL DEFAULT 0,
    signature   TEXT    DEFAULT '',
    doc         TEXT    DEFAULT '',
    receiver    TEXT    DEFAULT NULL,
    receiver_pointer BOOLEAN DEFAULT FALSE,
    exported    BOOLEAN NOT NULL DEFAULT FALSE,
    language    TEXT    DEFAULT 'go'
);

CREATE INDEX IF NOT EXISTS idx_symbols_name       ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind        ON symbols(kind);
CREATE INDEX IF NOT EXISTS idx_symbols_package_id  ON symbols(package_id);
CREATE INDEX IF NOT EXISTS idx_symbols_file_id     ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_exported    ON symbols(exported);
CREATE INDEX IF NOT EXISTS idx_symbols_receiver    ON symbols(receiver);
```

**Why so many indexes?** Each index optimizes a common query pattern:
- `idx_symbols_name` — search for a symbol by name
- `idx_symbols_kind` — filter by kind ("show me all functions")
- `idx_symbols_package_id` — list all symbols in a package
- `idx_symbols_file_id` — list all symbols in a file
- `idx_symbols_exported` — filter by visibility
- `idx_symbols_receiver` — find all methods on a type

Without indexes, every query would scan the entire table (a "full table scan").
With indexes, SQLite can jump directly to the matching rows.

### Table: `refs`

Cross-references between symbols. Each row says "symbol `from_id` references
symbol `to_id`" at a specific location in a file. The `kind` describes the
nature of the reference: `"call"` (function call), `"embed"` (type embedding),
`"implements"` (interface implementation), `"use"` (variable access), etc.

This table is the foundation of the cross-reference UI: when you view a
symbol, the browser queries `SELECT * FROM refs WHERE to_id = ?` to find who
calls this function, and `SELECT * FROM refs WHERE from_id = ?` to find what
this function calls.

```sql
CREATE TABLE IF NOT EXISTS refs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id      TEXT NOT NULL REFERENCES symbols(id),
    to_id        TEXT NOT NULL REFERENCES symbols(id),
    kind         TEXT NOT NULL DEFAULT '',
    file_id      TEXT NOT NULL REFERENCES files(id),
    start_line   INTEGER NOT NULL,
    start_col    INTEGER NOT NULL DEFAULT 0,
    end_line     INTEGER NOT NULL,
    end_col      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_refs_from_id  ON refs(from_id);
CREATE INDEX IF NOT EXISTS idx_refs_to_id    ON refs(to_id);
CREATE INDEX IF NOT EXISTS idx_refs_kind     ON refs(kind);
CREATE INDEX IF NOT EXISTS idx_refs_file_id  ON refs(file_id);
```

**Why AUTOINCREMENT for refs but not other tables?** The other tables use
textual IDs (like `pkg:...` or `sym:...`) that are globally unique and stable
across re-indexes. Refs don't have a natural unique key — two refs might have
identical fields but represent different occurrences. So we use SQLite's
auto-incrementing integer as the primary key.

### Table: `index_meta`

A single-row table storing metadata about the index: when it was generated,
what module it covers, what Go version was used. This replaces the top-level
fields in the current `index.json`.

```sql
CREATE TABLE IF NOT EXISTS index_meta (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    version      TEXT    NOT NULL DEFAULT '1',
    generated_at TEXT    NOT NULL,
    module       TEXT    NOT NULL DEFAULT '',
    go_version   TEXT    NOT NULL DEFAULT '',
    tool_version TEXT    NOT NULL DEFAULT ''
);
```

The `CHECK (id = 1)` constraint enforces that this table can only ever have
one row. Every insert must use `id = 1`.

### FTS5 Virtual Table: `symbols_fts`

This is the most powerful part of the schema. FTS5 (Full-Text Search 5) is a
SQLite extension that creates a **virtual table** — a table that looks like a
regular table but is actually backed by a specialized data structure (an
inverted index) optimized for text search.

We create an FTS5 table over the symbol's name, doc comment, signature, kind,
and its package's import path. This lets us search across all these fields
simultaneously with ranking:

```sql
-- "Find symbols related to 'search', ranked by relevance"
SELECT s.name, s.kind, s.signature, p.import_path,
       bm25(symbols_fts) AS rank
FROM symbols_fts fts
JOIN symbols s ON s.rowid = fts.rowid
JOIN packages p ON p.id = s.package_id
WHERE symbols_fts MATCH 'search'
ORDER BY rank
LIMIT 20;
```

The `bm25()` function returns a relevance score based on term frequency and
document length — the same algorithm used by search engines. Results with the
best match appear first.

The FTS5 table is kept in sync with the `symbols` table via **triggers** —
small pieces of SQL that automatically run whenever a row is inserted, updated,
or deleted in the `symbols` table:

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
    name, signature, doc, kind, import_path,
    content='symbols',
    content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS symbols_fts_insert AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, name, signature, doc, kind, import_path)
    SELECT new.rowid, new.name, new.signature, new.doc, new.kind, p.import_path
    FROM packages p WHERE p.id = new.package_id;
END;

CREATE TRIGGER IF NOT EXISTS symbols_fts_delete AFTER DELETE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid) VALUES ('delete', old.rowid);
END;

CREATE TRIGGER IF NOT EXISTS symbols_fts_update AFTER UPDATE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid) VALUES ('delete', old.rowid);
    INSERT INTO symbols_fts(rowid, name, signature, doc, kind, import_path)
    SELECT new.rowid, new.name, new.signature, new.doc, new.kind, p.import_path
    FROM packages p WHERE p.id = new.package_id;
END;
```

Notice that the `import_path` is denormalized into the FTS table via a join
in the trigger. This is necessary because FTS5 can only index columns from a
single table — it can't do joins at query time. By copying the import path
into the FTS table, we can search across symbol names AND package paths with a
single MATCH expression.

### Example queries

Here are some queries that the schema enables. Each one would be difficult or
impossible with the current JSON-based approach:

**Exported functions with no doc comments** — a code quality audit:
```sql
SELECT s.name, p.import_path, f.path, s.start_line
FROM symbols s
JOIN packages p ON p.id = s.package_id
JOIN files f ON f.id = s.file_id
WHERE s.kind = 'func' AND s.exported = 1
  AND (s.doc IS NULL OR s.doc = '')
ORDER BY p.import_path, s.name;
```

**Most-referenced symbols** — what does the codebase depend on most?
```sql
SELECT s.name, s.kind, p.import_path, COUNT(*) AS ref_count
FROM refs r
JOIN symbols s ON s.id = r.to_id
JOIN packages p ON p.id = s.package_id
GROUP BY s.id ORDER BY ref_count DESC LIMIT 20;
```

**Package dependency graph** — which packages reference which other packages:
```sql
SELECT p1.import_path AS from_pkg,
       p2.import_path AS to_pkg,
       COUNT(*) AS ref_count
FROM refs r
JOIN symbols s1 ON s1.id = r.from_id
JOIN symbols s2 ON s2.id = r.to_id
JOIN packages p1 ON p1.id = s1.package_id
JOIN packages p2 ON p2.id = s2.package_id
WHERE p1.id != p2.id
GROUP BY p1.id, p2.id ORDER BY ref_count DESC;
```

---

## Go Package Design

The SQLite store lives in a new package at `internal/sqlite/`. This section
describes every file, what it does, and why it's structured that way.

### File layout

```
internal/sqlite/
├── store.go           # The Store struct — opens DB, runs queries, closes
├── schema.go          # CREATE TABLE statements, indexes, triggers
├── loader.go          # Bulk import from indexer.Index into the database
├── query.go           # Composable predicate system (like glazed help store)
├── fts5.go            # FTS5 virtual table creation (behind build tag)
├── export.go          # Export to JSON for backward compatibility
├── generate_build.go  # Build script: reads index.json → writes codebase.db
├── generate.go        # //go:embed directive for codebase.db
├── embed/             # Where the generated database file lives
│   └── .gitkeep
└── store_test.go      # Integration tests

queries/               # SQL query files (like go-minitrace)
├── symbols/
│   ├── exported-undocumented.sql
│   ├── most-referenced.sql
│   └── largest-symbols.sql
├── packages/
│   ├── dependency-graph.sql
│   └── per-package-stats.sql
├── search/
│   ├── fts-search.sql
│   └── name-prefix-search.sql
└── my-queries/
    └── .gitignore     # Ad-hoc queries, not tracked in git
```

### `store.go` — Opening and closing the database

The `Store` struct is the main entry point. It wraps a `*sql.DB` (Go's
standard database interface) and provides methods for all operations. You
create one by calling `New("path/to/codebase.db")` for a file-backed
database or `NewInMemory()` for a transient in-memory database.

Why would you want in-memory? Two reasons:

1. **Testing** — tests create an in-memory database, load test data, run
   queries, and it vanishes when the test ends. No temp files to clean up.
2. **The Go server** — `codebase-browser serve` can load the embedded
   `codebase.db` into memory for faster queries (no disk I/O).

```go
package sqlite

import "database/sql"

type Store struct {
    db *sql.DB
}

func New(dbPath string) (*Store, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil { return nil, err }
    s := &Store{db: db}
    if err := s.createSchema(); err != nil {
        db.Close()
        return nil, err
    }
    return s, nil
}

func NewInMemory() (*Store, error) { return New(":memory:") }
func (s *Store) Close() error      { return s.db.Close() }
func (s *Store) DB() *sql.DB       { return s.db }
```

The `DB()` method exposes the raw `*sql.DB` for the CLI query command and for
ad-hoc SQL that doesn't need the predicate system.

### `loader.go` — Bulk import from the indexer

The `LoadFromIndex` method is the bridge between the existing `indexer.Index`
struct (which is what `index.json` deserializes into) and the SQLite database.
It runs inside a transaction so that the database is never in a half-loaded
state.

Why a transaction? SQLite transactions are atomic — either all the inserts
succeed, or none of them do. If the loader crashes halfway through (e.g., due
to a foreign key violation), the database rolls back to its previous state.
The caller never sees a partially-loaded database.

The method first clears all existing data (so it's idempotent — you can run
it multiple times safely), then inserts packages, files, symbols, and refs
in that order. The order matters because of foreign key constraints: you can't
insert a symbol that references a package that doesn't exist yet.

### `query.go` — The predicate system

This is the most intellectually interesting part of the design. It's modeled
directly on the glazed help store's predicate system.

The idea is simple: instead of writing raw SQL strings everywhere, you build
queries from small, composable functions called **predicates**. Each predicate
adds a WHERE clause (or a JOIN, or an ORDER BY) to a query builder. You
compose predicates with `And()`, `Or()`, and `Not()`.

Here's what that looks like in practice:

```go
// Find all exported functions in the server package
syms, _ := store.FindSymbols(
    sqlite.ByKind("func"),
    sqlite.ByPackage("github.com/wesen/codebase-browser/internal/server"),
    sqlite.IsExported(),
)

// Full-text search for "search" across names, docs, signatures
results, _ := store.FindSymbols(
    sqlite.TextSearch("search"),
    sqlite.Limit(20),
)
```

Each predicate is a function that takes a `*QueryBuilder` and modifies it:

```go
type Predicate func(*QueryBuilder)

func ByKind(kind string) Predicate {
    return func(qb *QueryBuilder) {
        qb.where = append(qb.where, "s.kind = ?")
        qb.args = append(qb.args, kind)
    }
}
```

The `?` placeholder is a **parameterized query** — SQLite will substitute the
actual value safely, preventing SQL injection. You never concatenate user
input into SQL strings.

Available predicates:

- **`ByKind(kind)`** — filter by symbol kind (func, type, method, etc.)
- **`ByPackage(importPath)`** — filter by package import path
- **`ByFile(path)`** — filter by source file path
- **`IsExported()`** — only exported (public) symbols
- **`NameContains(substr)`** — substring match on name (uses `LIKE`)
- **`TextSearch(query)`** — FTS5 full-text search with BM25 ranking
- **`HasDoc()`** — only symbols with doc comments
- **`And(preds...)`** — all predicates must match
- **`Or(preds...)`** — any predicate must match
- **`Not(pred)`** — predicate must not match
- **`Limit(n)`**, **`Offset(n)`** — pagination

---

## CLI Integration: The `query` Command

One of the most valuable features of a SQLite-based index is the ability to
run ad-hoc SQL queries from the command line. This follows the go-minitrace
pattern where you run `.sql` files against the database.

### Usage

```bash
# Run an inline SQL query
codebase-browser query "SELECT name, kind FROM symbols WHERE exported=1 LIMIT 5"

# Run a pre-built query file
codebase-browser query -f queries/symbols/most-referenced.sql

# List available query files
codebase-browser query --list

# Output as JSON (for piping to jq)
codebase-browser query --format json "SELECT * FROM packages"
```

### Pre-built query files

Each `.sql` file in `queries/` has a comment header explaining its purpose,
just like in go-minitrace:

```sql
-- exported-undocumented: Exported symbols missing doc comments
-- Useful for code quality audits
SELECT
    s.name,
    s.kind,
    p.import_path AS package,
    f.path         AS file,
    s.start_line   AS line
FROM symbols s
JOIN packages p ON p.id = s.package_id
JOIN files f     ON f.id = s.file_id
WHERE s.exported = 1
  AND (s.doc IS NULL OR s.doc = '')
ORDER BY p.import_path, s.name;
```

---

## Browser-Side SQL: Replacing WASM with sql.js

This is where the design gets exciting for the static build. Instead of
shipping a 1.2 MB Go-compiled WASM module that parses JSON, we ship an 800 KB
SQLite WASM engine (`sql.js`) and the database file directly.

### What is sql.js?

`sql.js` is a JavaScript library that bundles the complete SQLite C library,
compiled to WebAssembly using Emscripten. It gives you a full SQL database
engine that runs entirely in the browser — no server, no network requests.

You use it like this:

```typescript
import initSqlJs from 'sql.js';

// Load the WASM engine
const SQL = await initSqlJs({ locateFile: f => `/${f}` });

// Fetch the database file
const response = await fetch('/codebase.db');
const buffer = await response.arrayBuffer();

// Open the database
const db = new SQL.Database(new Uint8Array(buffer));

// Run queries
const results = db.exec("SELECT name FROM symbols WHERE kind='func' LIMIT 5");
```

### Size comparison

| Component | Current (WASM) | Proposed (SQLite) |
|-----------|---------------|-------------------|
| Engine | `search.wasm` (1,200 KB) | `sql-wasm.wasm` (800 KB) |
| Runtime | `wasm_exec.js` (16 KB) | `sql-wasm.js` (150 KB) |
| Data | `precomputed.json` (1,900 KB) | `codebase.db` (~500 KB est.) |
| **Total** | **3,116 KB** | **~1,450 KB** |

The SQLite approach is less than half the size, and provides enormously more
query power.

### `ui/src/api/dbClient.ts`

The browser-side database client is much simpler than the current WASM client.
No async initialization ceremony, no polling for `window.codebaseBrowser`:

```typescript
import initSqlJs, { Database } from 'sql.js';

let db: Database | null = null;

export async function initDb(): Promise<Database> {
  if (db) return db;
  const SQL = await initSqlJs({ locateFile: (f: string) => `/${f}` });
  const response = await fetch('/codebase.db');
  const buffer = await response.arrayBuffer();
  db = new SQL.Database(new Uint8Array(buffer));
  return db;
}

export async function queryDb(sql: string, params: unknown[] = []): Promise<unknown[]> {
  const database = await initDb();
  const stmt = database.prepare(sql);
  stmt.bind(params);
  const results: unknown[] = [];
  while (stmt.step()) { results.push(stmt.getAsObject()); }
  stmt.free();
  return results;
}
```

For performance-sensitive operations like autocomplete search, we prepare
statements once and reuse them:

```typescript
// Prepared statement — compiled once, reused for every keystroke
const searchStmt = db.prepare(`
  SELECT s.*, p.import_path, bm25(symbols_fts) AS rank
  FROM symbols_fts fts
  JOIN symbols s ON s.rowid = fts.rowid
  JOIN packages p ON p.id = s.package_id
  WHERE symbols_fts MATCH ?
  ORDER BY rank LIMIT 50
`);

function searchSymbols(query: string): Symbol[] {
  searchStmt.bind([query]);
  const results = [];
  while (searchStmt.step()) { results.push(searchStmt.getAsObject()); }
  searchStmt.reset();  // ready for next call
  return results;
}
```

### Updating the React API layer

The existing RTK-Query endpoints (`indexApi`, `sourceApi`, `docApi`, `xrefApi`)
remain identical — only the `baseQuery` implementation changes from calling
WASM functions to running SQL. This is a one-file change:

```typescript
// Before: wasmBaseQuery called window.codebaseBrowser.findSymbols(...)
// After: dbBaseQuery runs SQL queries against the in-browser SQLite
import { dbBaseQuery } from './dbClient';

export const indexApi = createApi({
    reducerPath: 'indexApi',
    baseQuery: dbBaseQuery,  // <-- only change
    ...
});
```

Source files remain as static files in `dist/source/` — they're too large to
store as BLOBs in SQLite and the browser needs them as text for syntax
highlighting.

---

## Build Pipeline Changes

### Current pipeline (what happens when you run `make generate`)

The build pipeline is a series of `go generate` commands, each one producing
a different artifact:

1. **`go generate ./internal/indexfs`** — the Go indexer reads all `.go`
   source files, extracts symbols/refs, and writes `index.json` (705 KB).

2. **`go generate ./internal/sourcefs`** — copies the entire source tree into
   an embeddable directory so the Go server can serve source files.

3. **`go generate ./internal/web`** — uses Dagger to run Vite (a JavaScript
   build tool) inside a Node.js container, producing the production SPA.

4. **`go generate ./internal/static`** — reads `index.json`, pre-computes
   search indexes, xref maps, snippets, and doc HTML, writes
   `precomputed.json` (1.9 MB).

5. **`go generate ./internal/wasm`** — uses Dagger to compile `cmd/wasm/`
   with TinyGo to produce `search.wasm` (1.2 MB).

6. **`go generate ./internal/bundle`** — assembles everything into `dist/`.

### New pipeline

The SQLite approach eliminates steps 4 and 5 entirely, replacing them with
a single simpler step:

1. **`go generate ./internal/indexfs`** — unchanged
2. **`go generate ./internal/sourcefs`** — unchanged
3. **`go generate ./internal/web`** — unchanged
4. **`go generate ./internal/sqlite`** — reads `index.json`, bulk-loads into
   SQLite, writes `codebase.db` (~500 KB). **This replaces steps 4+5.**
5. **`go generate ./internal/bundle`** — updated: ships `codebase.db` and
   `sql.js` instead of `search.wasm` and `precomputed.json`.

### What gets removed

| Old package | Old output | Why it's no longer needed |
|-------------|-----------|--------------------------|
| `internal/static/` | `precomputed.json` | SQLite replaces all pre-computed data |
| `internal/wasm/` | `search.wasm` | sql.js replaces Go WASM module |
| `cmd/wasm/` | WASM entry point | No Go code compiled to WASM |

That's roughly 1,100 lines of Go code and an entire Dagger + TinyGo build
pipeline that goes away.

### Static artifact layout (after)

```
dist/
├── index.html              # SPA entry point
├── codebase.db             # The SQLite database (~500 KB)
├── sql-wasm.wasm           # SQLite engine for browser (~800 KB)
├── sql-wasm.js             # SQLite JavaScript wrapper (~150 KB)
├── assets/
│   ├── index-xxxxx.js      # Vite bundle
│   └── index-xxxxx.css
└── source/                 # Source files (not in DB — too large)
    ├── internal/
    ├── cmd/
    └── ui/
```

---

## Migration Strategy: How to Get There in Safe Steps

We don't rip out the old system and replace it in one shot. That would break
everything at once. Instead, we add the SQLite system alongside the existing
one, verify it works, and then switch over in stages.

### Phase 1: Add SQLite package (non-breaking, no frontend changes)

Create the SQLite store and build pipeline alongside the existing one. Both
systems produce output; nothing changes for users.

**What to do:**
- Create `internal/sqlite/` package (store, schema, loader, predicates)
- Create `generate_build.go` that reads `index.json` and writes `codebase.db`
- Add the `query` sub-command to the CLI
- Add `queries/` directory with example SQL files
- Write tests: load the self-index into SQLite, verify row counts match
  `index.json`

**How to verify:** Run `go generate ./internal/sqlite` and then:
```bash
codebase-browser query "SELECT COUNT(*) FROM symbols"
# Should return 329 (same as the JSON index)
```

### Phase 2: Browser integration (non-breaking, feature-flagged)

Add `sql.js` to the frontend and create a `dbBaseQuery` alongside the existing
`wasmBaseQuery`. Both paths work; an environment variable switches between them.

**What to do:**
- Install `sql.js` in `ui/` (`pnpm -C ui add sql.js`)
- Create `ui/src/api/dbClient.ts` with `initDb()`, `queryDb()`, `dbBaseQuery`
- Update the four API files to use `dbBaseQuery` when `VITE_USE_SQLITE=1`
- Update the bundler to ship `codebase.db` and `sql.js` assets

**How to verify:** `VITE_USE_SQLITE=1 make build-static` produces a working
static artifact. Open in browser, search and navigate work.

### Phase 3: Switch default (breaking for static build only)

Make SQLite the default for the static build. The WASM path still works but
is no longer the default.

### Phase 4: Clean up (breaking)

Remove `precomputed.json` generation, mark old packages as deprecated, update
README.

---

## Risks and Open Questions

### Risk: sql.js binary size

`sql-wasm.wasm` is about 800 KB (300 KB gzipped). If this is too large for
some use case, alternatives include:
- **wa-sqlite**: Google's WASM SQLite with Origin Private File System (OPFS)
  support for persistence and streaming
- **Keep the WASM module** for simple lookups and add SQLite only for the CLI
- **Lazy-load**: only fetch the SQLite WASM when the user opens a query panel

### Risk: FTS5 in browser vs Go

`sql.js` is compiled with all SQLite features including FTS5, so full-text
search works in the browser. On the Go side, `mattn/go-sqlite3` (the CGo
driver already in `go.mod`) also supports FTS5. The alternative pure-Go
driver (`modernc.org/sqlite`) may need build tags. We should standardize on
one driver.

### Open question: Doc page storage

Doc pages are currently pre-rendered HTML in `precomputed.json`. We could:
1. Add a `doc_pages` table (slug, title, HTML)
2. Keep them as static files in the bundle
3. Render on-the-fly from the `symbols.doc` column

Option 1 is recommended for consistency — everything in one database.

### Open question: Source files

Source files (~2 MB of text) are currently a directory tree. Should they go
into SQLite? Arguments for: single-file artifact. Arguments against: the
browser needs the full text for syntax highlighting, and fetching from SQLite
isn't faster than a static HTTP request. Recommendation: keep source files as
static files outside the database.

### Open question: Backward compatibility

The Go server currently loads `index.json` via `internal/browser/loaded.go`.
We should add a `LoadFromSQLite(dbPath string)` method alongside the existing
`LoadFromFile(path string)`. Over time, the SQLite path becomes the default
and JSON becomes optional.

---

## File Reference

Every source file that will be created, modified, or deprecated by this
ticket:

### New files

| File | Purpose |
|------|---------|
| `internal/sqlite/store.go` | Store struct, New(), Close(), DB() |
| `internal/sqlite/schema.go` | CREATE TABLE/INDEX/TRIGGER statements |
| `internal/sqlite/loader.go` | LoadFromIndex() bulk import |
| `internal/sqlite/query.go` | Predicate system (ByKind, TextSearch, And, Or...) |
| `internal/sqlite/fts5.go` | FTS5 setup (behind `sqlite_fts5` build tag) |
| `internal/sqlite/export.go` | ExportToJSON() for backward compat |
| `internal/sqlite/generate.go` | `//go:embed codebase.db` |
| `internal/sqlite/generate_build.go` | Build script: index.json → codebase.db |
| `internal/sqlite/store_test.go` | Integration tests |
| `cmd/codebase-browser/cmds/query/query.go` | CLI `query` sub-command |
| `queries/symbols/*.sql` | Pre-built symbol queries |
| `queries/packages/*.sql` | Pre-built package queries |
| `queries/search/*.sql` | Search query examples |
| `ui/src/api/dbClient.ts` | Browser-side SQLite client |

### Modified files

| File | Change |
|------|--------|
| `internal/bundle/generate_build.go` | Ship codebase.db + sql.js instead of search.wasm + precomputed.json |
| `ui/src/api/indexApi.ts` | Switch baseQuery to dbBaseQuery (feature-flagged) |
| `ui/src/api/sourceApi.ts` | Same |
| `ui/src/api/docApi.ts` | Same |
| `ui/src/api/xrefApi.ts` | Same |
| `ui/package.json` | Add sql.js dependency |
| `Makefile` | Add generate-sqlite target, update build-static |

### Deprecated (kept for reference, eventually removed)

| Package | Reason |
|---------|--------|
| `internal/wasm/` | Replaced by sql.js in browser |
| `internal/static/` | Replaced by SQLite tables |
| `cmd/wasm/` | No Go code compiled to WASM |
| `ui/src/api/wasmClient.ts` | Replaced by dbClient.ts |
