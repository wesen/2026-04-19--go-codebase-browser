---
Title: Sql.js static frontend architecture and implementation guide
Ticket: GCB-015
Status: active
Topics:
    - codebase-browser
    - static-export
    - sqlite
    - react-frontend
    - review-docs
    - architecture
    - history
    - markdown-directives
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/review/export.go
      Note: Current export command to replace with sql.js static packaging
    - Path: internal/history/bodydiff.go
      Note: Reference implementation for symbol body diff extraction to port to sql.js queries
    - Path: internal/history/diff.go
      Note: Reference implementation for commit diff semantics to port to SQLite-compatible frontend SQL
    - Path: internal/history/schema.go
      Note: Defines the SQLite tables/views that sql.js will query in the static browser
    - Path: internal/review/schema.go
      Note: Defines review document tables used by the SQL-backed review renderer
    - Path: internal/review/server.go
      Note: Reference for current review API behavior to expose through the query provider
    - Path: ui/package.json
      Note: Frontend dependencies need sql.js and possibly sql.js type declarations
    - Path: ui/src/api/historyApi.ts
      Note: Current endpoint/static transport to refactor into provider query functions
    - Path: ui/src/api/wasmClient.ts
      Note: Current TinyGo WASM client to remove or reduce for SQL-backed static runtime
    - Path: ui/src/features/history/HistoryPage.tsx
      Note: Primary browser route that must use SQL-backed history and body diffs
    - Path: ui/src/features/review/ReviewDocPage.tsx
      Note: Review document renderer to hydrate generic provider-backed widgets
ExternalSources: []
Summary: 'Design and implementation guide for a clean sql.js-backed static frontend: Go performs indexing/export, the browser opens the exported SQLite DB with sql.js, and both the generic codebase browser and rich review markdown renderer use the same SQL query provider.'
LastUpdated: 2026-05-01T19:40:00-04:00
WhatFor: Use this as the primary implementation guide for replacing the current TinyGo/precomputed-review static path with a sql.js static codebase browser and review renderer.
WhenToUse: Use before implementing static export, static frontend data access, sql.js integration, review widgets, history pages, or browser-side SQLite queries.
---


# Sql.js static frontend architecture and implementation guide

## Executive Summary

This ticket replaces the previous static export direction with a cleaner architecture:

> Go indexes the repository and writes a SQLite database. The static frontend ships that SQLite database and opens it in the browser with `sql.js`. Both the generic codebase browser and the rich review markdown renderer query that same database through a TypeScript `CodebaseQueryProvider`.

This is a deliberate clean cutoff. The app is not used externally, so we do not need backward compatibility with the current `precomputed.json` / TinyGo review-data transport. We should remove the prototype-specific static path rather than wrap it.

The previous direction tried to convert the SQLite database into a parallel universe of precomputed JSON maps:

```text
SQLite DB
  -> Go export precomputes selected JSON payloads
  -> TinyGo WASM receives reviewData
  -> React asks TinyGo for commits, histories, diffs, impacts, review docs
```

That worked for curated review widgets but failed as soon as the generic browser needed open-ended data. The example failure was:

```text
Failed to load body diff: STATIC_NOT_PRECOMPUTED
symbol body diff not precomputed: sym:...review.func.Register
```

The failure happened because a generic history route, `/history?symbol=...`, asked for a symbol body diff that the review-focused export did not precompute. In a real static codebase browser, this should not be a precompute coverage problem. The browser should query the database for the two symbol ranges and file contents, then compute the diff on demand.

The new architecture is:

```text
+---------------------------------------------------------------+
| Go indexing/export                                             |
|                                                               |
|  git commit range -> SQLite DB -> optional static metadata     |
+-------------------------------+-------------------------------+
                                |
                                v
+---------------------------------------------------------------+
| Static export directory                                        |
|                                                               |
|  index.html                                                    |
|  assets/                                                       |
|  manifest.json                                                 |
|  db/codebase.db                                                |
|  sql-wasm.wasm / sql.js assets                                 |
|  source/ optional                                               |
+-------------------------------+-------------------------------+
                                |
                                v
+---------------------------------------------------------------+
| Browser runtime                                                |
|                                                               |
|  React UI                                                      |
|  SqlJsQueryProvider                                            |
|  sql.js opens db/codebase.db                                   |
|  widgets and pages call semantic provider methods              |
+---------------------------------------------------------------+
```

The two main product features are separate but share the same provider:

1. **Generic static codebase browser**
   - packages, symbols, source, search, xrefs;
   - commits, diffs, symbol history, body diffs, impact graphs;
   - runs without a Go server.

2. **Static rich review markdown renderer**
   - rendered markdown documents;
   - inline React widgets from markdown directives;
   - cross-links into the generic browser;
   - runs without a Go server.

The SQLite database also remains the **LLM/query artifact**. The same file can be inspected by humans, scripts, and LLMs with SQL. That is a strength of this design: the static browser runtime and the LLM artifact converge on one source of truth.

## Problem Statement

### What we built before

The current GCB-013 prototype created a review command tree and a static export path. It includes:

- `review db create`
- `review index`
- `review serve`
- `review export`

The database contains both history tables and review document tables:

```text
commits
snapshot_packages
snapshot_files
snapshot_symbols
snapshot_refs
file_contents
review_docs
review_doc_snippets
```

The current static export then computes a `PrecomputedReview` object in Go:

```go
type PrecomputedReview struct {
    Version     string
    GeneratedAt string
    CommitRange string
    Commits     []CommitLite
    Diffs       map[string]*DiffLite
    Histories   map[string][]HistoryEntryLite
    Impacts     map[string]*ImpactLite
    BodyDiffs   map[string]*history.BodyDiffResult
    Docs        []ReviewDocLite
}
```

That object is serialized into `precomputed.json`, loaded into TinyGo WASM, and exposed to React through functions such as:

```go
GetCommitDiff(oldHash, newHash string)
GetSymbolHistory(symbolID string)
GetImpact(symbolID, direction string, depth int, commit string)
GetSymbolBodyDiff(oldHash, newHash, symbolID string)
GetReviewDocs()
GetReviewDoc(slug string)
GetCommits()
```

The frontend then has static-specific logic in files such as:

```text
ui/src/api/historyApi.ts
ui/src/api/wasmClient.ts
ui/src/api/docApi.ts
ui/src/api/runtimeMode.ts
```

This was a reasonable prototype, but it is the wrong foundation for a full static browser.

### Why the prototype breaks down

The prototype precomputes data for known review widgets. But the generic browser asks open-ended questions:

- show me history for any symbol;
- show me a body diff for any adjacent history transition;
- find references to any symbol;
- compute callers/callees for any symbol at a commit;
- search symbols;
- inspect files and source.

A static browser cannot rely on precomputing every possible answer as JSON. The natural query engine for these questions already exists: SQLite.

The current system fights its own data model:

```text
Useful relational DB exists
  -> Go extracts selected denormalized JSON
  -> Browser can only ask questions represented in that JSON
  -> Missing question produces STATIC_NOT_PRECOMPUTED
```

The better model is:

```text
Useful relational DB exists
  -> Browser opens the DB with sql.js
  -> Browser asks SQL questions directly
  -> Missing precompute maps are unnecessary for common navigation
```

### Why TinyGo WASM is not the right query engine here

TinyGo WASM is not useless, but it is not solving the core problem. The hope was that we could run Go SQLite code in the browser. That is not practical for this app:

- the existing Go SQLite driver depends on cgo or native SQLite bindings;
- TinyGo cannot simply run that server-side SQLite stack in the browser;
- file access, OPFS, threading, and driver support are all complicated;
- we ended up writing a separate WASM lookup layer, not reusing the Go DB code.

`sql.js` is already SQLite compiled to WebAssembly with a JavaScript API. It is the straightforward tool for opening a `.db` file in the browser.

So the revised decision is:

> Do not use TinyGo WASM as the static review/browser query layer. Use `sql.js` over the exported SQLite DB.

## Goals

### Product goals

- A static export can be hosted by any ordinary static file server.
- The generic codebase browser works without a Go backend.
- The rich review markdown renderer works without a Go backend.
- Review docs can link to browser pages such as `/history?symbol=...`.
- The same SQLite DB can be used by the browser and by LLM/script workflows.
- Static mode makes zero `/api/*` requests.

### Engineering goals

- Keep Go responsible for indexing, schema creation, database population, and static export packaging.
- Keep the browser responsible for interactive navigation and SQL queries through `sql.js`.
- Create one semantic frontend query provider used by all pages and widgets.
- Avoid static/server branching inside React components.
- Remove the TinyGo review-data transport and `reviewData` blob.
- Use SQL queries that mirror or replace existing server handlers.
- Add explicit tests for static browser routes and review widgets.

### Non-goals for the first implementation

- Do not implement `sql.js-httpvfs` immediately.
- Do not make `file://` the primary supported mode. Static HTTP serving is the baseline.
- Do not preserve old `precomputed.json` compatibility.
- Do not preserve current TinyGo review WASM APIs.
- Do not implement a perfect global search engine before the core SQL provider works.
- Do not optimize huge DB performance before the baseline architecture is correct.

## Key Concepts for a New Intern

### The repository index

The app indexes a Go repository across one or more git commits. For each commit, it extracts:

- packages;
- files;
- symbols such as functions, methods, structs, interfaces, variables;
- references between symbols;
- file contents.

The result is stored in SQLite. Think of each commit as a snapshot:

```text
commit A
  packages
  files
  symbols
  refs

commit B
  packages
  files
  symbols
  refs
```

When the user asks for history, we compare the same symbol ID across snapshots.

### The SQLite DB

The DB is the source of truth. It is created by Go. It is read by:

- Go server mode;
- static browser mode through `sql.js`;
- LLMs and scripts through ordinary SQLite tools.

Important file references:

```text
internal/history/schema.go
internal/review/schema.go
internal/history/diff.go
internal/history/bodydiff.go
internal/review/indexer.go
```

### The generic browser

The generic browser is the exploratory UI:

- package list;
- symbol pages;
- source pages;
- search;
- commit timeline;
- symbol history;
- body diff;
- impact/call graph.

It should not depend on review documents.

### The review renderer

The review renderer is an authored guide:

```markdown
# Review guide

This function changed:

```codebase-diff sym=... from=HEAD~1 to=HEAD
```

The impact is:

```codebase-impact sym=... dir=uses depth=2
```
```

At export or serve time, markdown is rendered into HTML with widget placeholders. React hydrates those placeholders into actual widgets. The widgets use the same query provider as the generic browser.

### sql.js

`sql.js` is SQLite compiled to WebAssembly and exposed to JavaScript. It lets the browser load a `.db` file:

```ts
const SQL = await initSqlJs({ locateFile: file => `/assets/${file}` });
const bytes = new Uint8Array(await fetch('db/codebase.db').then(r => r.arrayBuffer()));
const db = new SQL.Database(bytes);
```

Then we can run SQL:

```ts
const rows = queryAll(db, `SELECT * FROM commits ORDER BY author_time DESC`);
```

## Proposed Architecture

### Runtime diagram

```text
+---------------------------------------------------------------+
| React app                                                     |
|                                                               |
|  HomePage                                                     |
|  PackagePage                                                  |
|  SymbolPage                                                   |
|  SourcePage                                                   |
|  HistoryPage                                                  |
|  ReviewDocPage                                                |
|  DocSnippet / widgets                                         |
+----------------------------+----------------------------------+
                             |
                             v
+---------------------------------------------------------------+
| CodebaseQueryProvider                                         |
|                                                               |
|  listCommits()                                                |
|  getSymbolHistory(symbolId)                                   |
|  getSymbolBodyDiff(from, to, symbolId)                        |
|  getImpact(symbolId, dir, depth, commit)                      |
|  getReviewDoc(slug)                                           |
|  ...                                                          |
+----------------------------+----------------------------------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
+---------------------------+   +-------------------------------+
| ServerQueryProvider       |   | SqlJsQueryProvider            |
|                           |   |                               |
| fetch('/api/...')         |   | db.exec / db.prepare          |
| used in dev/server mode   |   | used in static export mode    |
+---------------------------+   +-------------------------------+
                                            |
                                            v
                                  +-------------------+
                                  | db/codebase.db    |
                                  | opened by sql.js  |
                                  +-------------------+
```

### Export directory layout

Target static export:

```text
export/
  index.html
  assets/
    index-....js
    index-....css
    sql-wasm.wasm or equivalent sql.js wasm asset

  manifest.json

  db/
    codebase.db

  source/                    # optional initially; may be removed later
```

`codebase.db` is the unified SQLite database. It includes history tables and review doc tables. We should not copy two databases unless there is a clear reason.

### Manifest

Even though SQLite is the runtime data source, we still need a small manifest. It tells the app what kind of bundle this is.

```json
{
  "schemaVersion": 1,
  "kind": "codebase-browser-sqljs-static-export",
  "generatedAt": "2026-05-01T23:00:00Z",
  "db": {
    "path": "db/codebase.db",
    "sizeBytes": 12345678,
    "schemaVersion": 1
  },
  "features": {
    "codebaseBrowser": true,
    "reviewDocs": true,
    "llmDatabase": true,
    "sourceTree": true
  },
  "repo": {
    "module": "github.com/wesen/codebase-browser",
    "rootLabel": "codebase-browser"
  },
  "commitRange": "HEAD~20..HEAD",
  "commits": {
    "count": 21,
    "oldest": "abc...",
    "newest": "def..."
  },
  "runtime": {
    "queryEngine": "sql.js",
    "requiresHttpServer": true
  }
}
```

The manifest is not a replacement for SQL. It is boot metadata.

## Build-time Responsibilities in Go

Go remains essential. The browser should not index source code. Go does the hard indexing work before export.

### Go indexing flow

```text
git commit range
  -> checkout/extract each commit
  -> Go AST/indexer extracts packages/files/symbols/refs
  -> history tables populated
  -> review markdown docs indexed into review tables
  -> static export copies DB and SPA assets
```

Current relevant files:

```text
internal/history/schema.go
internal/history/store.go
internal/history/loader.go
internal/review/schema.go
internal/review/store.go
internal/review/indexer.go
cmd/codebase-browser/cmds/review/index.go
cmd/codebase-browser/cmds/review/db.go
cmd/codebase-browser/cmds/review/export.go
```

### What Go should no longer do for static runtime

Go should no longer precompute a parallel `reviewData` blob for runtime browser navigation:

- no `BodyDiffs` map needed for normal body diffing;
- no `Impacts` map needed for normal impact queries;
- no `Histories` map needed for normal history pages;
- no `Diffs` map needed for normal commit diff pages.

Those can be SQL queries in the browser.

### What Go may still precompute into SQLite

Some derived data may be useful as DB tables or views:

- rendered review document HTML;
- FTS search tables;
- optional commit diff cache tables;
- metadata table for export information.

The key rule:

> If we precompute, precompute into SQLite tables/views, not into a separate opaque JSON universe.

## SQLite Schema Reference

### History tables

From `internal/history/schema.go`.

#### `commits`

One row per indexed commit.

Important columns:

```text
hash TEXT PRIMARY KEY
short_hash TEXT
message TEXT
author_name TEXT
author_email TEXT
author_time INTEGER
parent_hashes TEXT
indexed_at INTEGER
branch TEXT
error TEXT
```

Common query:

```sql
SELECT hash, short_hash, message, author_name, author_email,
       author_time, indexed_at, branch, error
FROM commits
WHERE error = ''
ORDER BY author_time DESC;
```

#### `snapshot_files`

One row per file per commit.

Important columns:

```text
commit_hash
id
path
package_id
size
line_count
sha256
language
content_hash
```

Common query:

```sql
SELECT *
FROM snapshot_files
WHERE commit_hash = ?
ORDER BY path;
```

#### `snapshot_symbols`

One row per symbol per commit.

Important columns:

```text
commit_hash
id
kind
name
package_id
file_id
start_line
end_line
start_offset
end_offset
signature
body_hash
exported
language
```

Common symbol lookup:

```sql
SELECT *
FROM snapshot_symbols
WHERE commit_hash = ? AND id = ?;
```

#### `snapshot_refs`

One row per reference edge per commit.

Important columns:

```text
commit_hash
from_symbol_id
to_symbol_id
kind
file_id
start_line
start_col
end_line
end_col
```

References from a symbol:

```sql
SELECT *
FROM snapshot_refs
WHERE commit_hash = ? AND from_symbol_id = ?
ORDER BY to_symbol_id, kind;
```

References to a symbol:

```sql
SELECT *
FROM snapshot_refs
WHERE commit_hash = ? AND to_symbol_id = ?
ORDER BY from_symbol_id, kind;
```

#### `file_contents`

Deduplicated file contents.

```text
content_hash TEXT PRIMARY KEY
content BLOB NOT NULL
```

Common query:

```sql
SELECT content
FROM file_contents
WHERE content_hash = ?;
```

#### `symbol_history` view

Already defined in `internal/history/schema.go`.

Useful query:

```sql
SELECT symbol_id, name, kind, package_id, commit_hash, short_hash,
       commit_message, author_time, body_hash, start_line, end_line,
       signature, file_id
FROM symbol_history
WHERE symbol_id = ?
ORDER BY author_time DESC;
```

### Review tables

From `internal/review/schema.go`.

#### `review_docs`

Raw review markdown documents.

```text
id
slug
title
path
content
frontmatter_json
indexed_at
```

List docs:

```sql
SELECT slug, title, path, indexed_at
FROM review_docs
ORDER BY slug;
```

Fetch raw doc:

```sql
SELECT slug, title, content, frontmatter_json
FROM review_docs
WHERE slug = ?;
```

#### `review_doc_snippets`

Resolved directive/widget references from review docs.

```text
doc_id
stub_id
directive
symbol_id
file_path
kind
language
text
params_json
start_line
end_line
commit_hash
```

Fetch snippets for doc:

```sql
SELECT stub_id, directive, symbol_id, file_path, kind, language,
       text, params_json, start_line, end_line, commit_hash
FROM review_doc_snippets
WHERE doc_id = ?
ORDER BY id;
```

## Suggested Additional SQLite Tables

### `static_export_metadata`

Use this to store export metadata inside the DB.

```sql
CREATE TABLE IF NOT EXISTS static_export_metadata (
  key TEXT PRIMARY KEY,
  value_json TEXT NOT NULL
);
```

Example rows:

```text
manifest
commitRange
exportOptions
```

### `static_review_rendered_docs`

Recommended for the first implementation. It lets Go keep markdown parsing/directive rendering responsibilities, while the browser simply displays rendered HTML and hydrates widgets.

```sql
CREATE TABLE IF NOT EXISTS static_review_rendered_docs (
  slug TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  html TEXT NOT NULL,
  snippets_json TEXT NOT NULL DEFAULT '[]',
  errors_json TEXT NOT NULL DEFAULT '[]',
  rendered_at INTEGER NOT NULL DEFAULT 0
);
```

Queries:

```sql
SELECT slug, title
FROM static_review_rendered_docs
ORDER BY slug;
```

```sql
SELECT slug, title, html, snippets_json, errors_json
FROM static_review_rendered_docs
WHERE slug = ?;
```

Why pre-render review docs?

- The Go renderer already knows how to resolve directives against `browser.Loaded`.
- It avoids porting the markdown/directive renderer to TypeScript immediately.
- It keeps review HTML deterministic at export time.

### `symbol_search_fts`

Optional but recommended after the first provider works.

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS symbol_search_fts USING fts5(
  commit_hash,
  symbol_id,
  name,
  kind,
  package_id,
  file_path,
  signature,
  doc
);
```

Populate from `snapshot_symbols` joined to `snapshot_files`:

```sql
INSERT INTO symbol_search_fts(commit_hash, symbol_id, name, kind, package_id, file_path, signature, doc)
SELECT s.commit_hash, s.id, s.name, s.kind, s.package_id, f.path, s.signature, s.doc
FROM snapshot_symbols s
LEFT JOIN snapshot_files f
  ON f.commit_hash = s.commit_hash
 AND f.id = s.file_id;
```

Search:

```sql
SELECT s.*
FROM symbol_search_fts fts
JOIN snapshot_symbols s
  ON s.commit_hash = fts.commit_hash
 AND s.id = fts.symbol_id
WHERE symbol_search_fts MATCH ?
  AND s.commit_hash = ?
LIMIT 100;
```

## Frontend Package Changes

Current `ui/package.json` does not include `sql.js`.

Add:

```bash
pnpm -C ui add sql.js
pnpm -C ui add -D @types/sql.js
```

If types are not sufficient or package exports differ, add a local declaration file:

```text
ui/src/types/sql.js.d.ts
```

The Vite build must copy `sql-wasm.wasm` to the output. There are two common approaches:

1. Use `locateFile` to point to a copied public asset.
2. Import/copy through Vite asset handling.

Recommended first pass:

- copy `node_modules/sql.js/dist/sql-wasm.wasm` into `ui/public/sql-wasm.wasm` or static export output;
- initialize with `locateFile: () => 'sql-wasm.wasm'`.

Example:

```ts
const SQL = await initSqlJs({
  locateFile: (file) => file === 'sql-wasm.wasm' ? 'sql-wasm.wasm' : file,
});
```

## TypeScript SQL Utility Layer

Create:

```text
ui/src/api/sqljs/sqlJsDb.ts
ui/src/api/sqljs/sqlRows.ts
```

### `sqlJsDb.ts`

Responsibilities:

- initialize sql.js once;
- fetch `manifest.json`;
- fetch `db/codebase.db`;
- create `SQL.Database`;
- expose a singleton promise.

Pseudocode:

```ts
import initSqlJs, { type Database, type SqlJsStatic } from 'sql.js';

let sqlPromise: Promise<SqlJsStatic> | null = null;
let dbPromise: Promise<Database> | null = null;

export async function getSqlJs(): Promise<SqlJsStatic> {
  if (!sqlPromise) {
    sqlPromise = initSqlJs({
      locateFile: (file) => file === 'sql-wasm.wasm' ? 'sql-wasm.wasm' : file,
    });
  }
  return sqlPromise;
}

export async function getStaticDb(): Promise<Database> {
  if (!dbPromise) {
    dbPromise = (async () => {
      const SQL = await getSqlJs();
      const manifest = await fetch('manifest.json').then(r => r.json());
      const dbPath = manifest.db?.path ?? 'db/codebase.db';
      const response = await fetch(dbPath);
      if (!response.ok) {
        throw new Error(`failed to fetch SQLite DB: ${response.status} ${response.statusText}`);
      }
      const bytes = new Uint8Array(await response.arrayBuffer());
      return new SQL.Database(bytes);
    })();
  }
  return dbPromise;
}
```

### `sqlRows.ts`

Responsibilities:

- make prepared statements easy;
- convert rows to objects;
- always free statements.

Pseudocode:

```ts
import type { Database, SqlValue } from 'sql.js';

export function queryAll<T extends Record<string, unknown>>(
  db: Database,
  sql: string,
  params: SqlValue[] = [],
): T[] {
  const stmt = db.prepare(sql);
  try {
    stmt.bind(params);
    const rows: T[] = [];
    while (stmt.step()) {
      rows.push(stmt.getAsObject() as T);
    }
    return rows;
  } finally {
    stmt.free();
  }
}

export function queryOne<T extends Record<string, unknown>>(
  db: Database,
  sql: string,
  params: SqlValue[] = [],
): T | null {
  return queryAll<T>(db, sql, params)[0] ?? null;
}
```

### Blob/text decoding

`file_contents.content` is a BLOB. `sql.js` may return it as `Uint8Array`.

Utility:

```ts
const decoder = new TextDecoder('utf-8');

export function blobToText(value: unknown): string {
  if (value instanceof Uint8Array) return decoder.decode(value);
  if (typeof value === 'string') return value;
  if (Array.isArray(value)) return decoder.decode(new Uint8Array(value));
  return '';
}
```

## CodebaseQueryProvider

Create:

```text
ui/src/api/queryProvider.ts
ui/src/api/sqlJsQueryProvider.ts
ui/src/api/serverQueryProvider.ts
ui/src/api/queryErrors.ts
```

### Interface

```ts
export interface CodebaseQueryProvider {
  manifest(): Promise<StaticManifest>;

  listCommits(): Promise<CommitRow[]>;
  getCommit(ref: string): Promise<CommitRow>;
  resolveCommitRef(ref: string): Promise<string>;

  listPackages(options?: { commit?: string }): Promise<PackageRow[]>;
  getPackage(id: string, options?: { commit?: string }): Promise<PackageRow>;

  searchSymbols(query: string, options?: { kind?: string; commit?: string }): Promise<SymbolAtCommit[]>;
  getSymbol(id: string, options?: { commit?: string }): Promise<SymbolAtCommit>;
  getSymbolsAtCommit(commit?: string): Promise<SymbolAtCommit[]>;

  getRefsFrom(symbolId: string, options?: { commit?: string }): Promise<ImpactEdge[]>;
  getRefsTo(symbolId: string, options?: { commit?: string }): Promise<ImpactEdge[]>;

  getSymbolHistory(symbolId: string): Promise<SymbolHistoryEntry[]>;

  getCommitDiff(from: string, to: string): Promise<CommitDiff>;
  getSymbolBodyDiff(from: string, to: string, symbolId: string): Promise<BodyDiffResult>;

  getImpact(options: {
    symbolId: string;
    direction: 'usedby' | 'uses';
    depth: number;
    commit?: string;
  }): Promise<ImpactResponse>;

  listReviewDocs(): Promise<ReviewDocSummary[]>;
  getReviewDoc(slug: string): Promise<ReviewDoc>;
}
```

### Provider factory

```ts
let provider: CodebaseQueryProvider | null = null;

export function getQueryProvider(): CodebaseQueryProvider {
  if (provider) return provider;

  if (import.meta.env.VITE_STATIC_EXPORT === '1') {
    provider = new SqlJsQueryProvider();
  } else {
    provider = new ServerQueryProvider('/api');
  }

  return provider;
}
```

### Error type

```ts
export class QueryError extends Error {
  constructor(
    public code: 'NOT_FOUND' | 'AMBIGUOUS_REF' | 'SQL_ERROR' | 'DB_LOAD_ERROR' | 'FEATURE_UNAVAILABLE',
    message: string,
    public details?: Record<string, unknown>,
  ) {
    super(message);
  }
}
```

## SQL Queries for Provider Methods

### `listCommits`

```sql
SELECT hash AS Hash,
       short_hash AS ShortHash,
       message AS Message,
       author_name AS AuthorName,
       author_email AS AuthorEmail,
       author_time AS AuthorTime,
       indexed_at AS IndexedAt,
       branch AS Branch,
       error AS Error
FROM commits
WHERE error = ''
ORDER BY author_time DESC;
```

### `resolveCommitRef`

Implement in TypeScript using `listCommits()`.

Pseudocode:

```ts
async resolveCommitRef(ref: string): Promise<string> {
  const commits = await this.listCommits();
  const ordered = [...commits].sort((a, b) => a.AuthorTime - b.AuthorTime);
  const newestIndex = ordered.length - 1;

  if (!ref || ref === 'HEAD') return ordered[newestIndex].Hash;

  const m = ref.match(/^HEAD~(\d+)$/);
  if (m) {
    const idx = newestIndex - Number.parseInt(m[1], 10);
    const commit = ordered[idx];
    if (!commit) throw new QueryError('NOT_FOUND', `commit ref not found: ${ref}`);
    return commit.Hash;
  }

  const exact = ordered.find(c => c.Hash === ref || c.ShortHash === ref);
  if (exact) return exact.Hash;

  const prefix = ordered.filter(c => c.Hash.startsWith(ref));
  if (prefix.length === 1) return prefix[0].Hash;
  if (prefix.length > 1) throw new QueryError('AMBIGUOUS_REF', `ambiguous commit ref: ${ref}`);

  throw new QueryError('NOT_FOUND', `commit ref not found: ${ref}`);
}
```

### `getSymbolsAtCommit`

```sql
SELECT id,
       kind,
       name,
       package_id AS packageId,
       file_id AS fileId,
       start_line AS startLine,
       end_line AS endLine,
       signature,
       exported,
       body_hash AS bodyHash,
       language
FROM snapshot_symbols
WHERE commit_hash = ?
ORDER BY package_id, file_id, start_line, name;
```

### `getSymbol`

```sql
SELECT id,
       kind,
       name,
       package_id AS packageId,
       file_id AS fileId,
       start_line AS startLine,
       end_line AS endLine,
       start_offset AS startOffset,
       end_offset AS endOffset,
       signature,
       exported,
       body_hash AS bodyHash,
       language
FROM snapshot_symbols
WHERE commit_hash = ? AND id = ?;
```

If the symbol is not found at the requested commit, decide based on route context:

- symbol page at latest commit: show not found;
- history page: maybe use latest known entry;
- impact graph: mark external.

### `searchSymbols`

First-pass simple search:

```sql
SELECT id,
       kind,
       name,
       package_id AS packageId,
       file_id AS fileId,
       start_line AS startLine,
       end_line AS endLine,
       signature,
       exported,
       body_hash AS bodyHash,
       language
FROM snapshot_symbols
WHERE commit_hash = ?
  AND (? = '' OR kind = ?)
  AND (
    name LIKE ?
    OR id LIKE ?
    OR signature LIKE ?
  )
ORDER BY name
LIMIT 100;
```

Parameters:

```ts
const like = `%${query}%`;
[commit, kind ?? '', kind ?? '', like, like, like]
```

Later replace with FTS.

### `getSymbolHistory`

```sql
SELECT symbol_id AS symbolId,
       name,
       kind,
       package_id AS packageId,
       commit_hash AS commitHash,
       short_hash AS shortHash,
       commit_message AS message,
       author_time AS authorTime,
       body_hash AS bodyHash,
       start_line AS startLine,
       end_line AS endLine,
       signature,
       file_id AS fileId
FROM symbol_history
WHERE symbol_id = ?
ORDER BY author_time DESC;
```

### `getRefsFrom`

```sql
SELECT from_symbol_id AS fromSymbolId,
       to_symbol_id AS toSymbolId,
       kind,
       file_id AS fileId,
       start_line AS startLine,
       start_col AS startCol,
       end_line AS endLine,
       end_col AS endCol
FROM snapshot_refs
WHERE commit_hash = ? AND from_symbol_id = ?
ORDER BY to_symbol_id, kind;
```

### `getRefsTo`

```sql
SELECT from_symbol_id AS fromSymbolId,
       to_symbol_id AS toSymbolId,
       kind,
       file_id AS fileId,
       start_line AS startLine,
       start_col AS startCol,
       end_line AS endLine,
       end_col AS endCol
FROM snapshot_refs
WHERE commit_hash = ? AND to_symbol_id = ?
ORDER BY from_symbol_id, kind;
```

### `getCommitDiff`

The existing Go implementation in `internal/history/diff.go` uses `FULL OUTER JOIN`. SQLite does not support `FULL OUTER JOIN`. In the browser provider, use `UNION ALL` queries instead.

#### File diffs

```sql
SELECT b.id AS fileId,
       b.path AS path,
       'added' AS changeType,
       '' AS oldSha256,
       b.sha256 AS newSha256
FROM snapshot_files b
LEFT JOIN snapshot_files a
  ON a.commit_hash = ? AND a.id = b.id
WHERE b.commit_hash = ? AND a.id IS NULL

UNION ALL

SELECT a.id AS fileId,
       a.path AS path,
       'removed' AS changeType,
       a.sha256 AS oldSha256,
       '' AS newSha256
FROM snapshot_files a
LEFT JOIN snapshot_files b
  ON b.commit_hash = ? AND b.id = a.id
WHERE a.commit_hash = ? AND b.id IS NULL

UNION ALL

SELECT b.id AS fileId,
       b.path AS path,
       'modified' AS changeType,
       a.sha256 AS oldSha256,
       b.sha256 AS newSha256
FROM snapshot_files a
JOIN snapshot_files b
  ON b.id = a.id
WHERE a.commit_hash = ?
  AND b.commit_hash = ?
  AND a.sha256 != b.sha256
ORDER BY path;
```

Bind parameters in pairs:

```ts
[oldHash, newHash, newHash, oldHash, oldHash, newHash]
```

#### Symbol diffs

Use the same three-way union: added, removed, modified/moved/signature-changed.

```sql
SELECT b.id AS symbolId,
       b.name AS name,
       b.kind AS kind,
       b.package_id AS packageId,
       'added' AS changeType,
       0 AS oldStartLine,
       0 AS oldEndLine,
       b.start_line AS newStartLine,
       b.end_line AS newEndLine,
       '' AS oldSignature,
       b.signature AS newSignature,
       '' AS oldBodyHash,
       b.body_hash AS newBodyHash
FROM snapshot_symbols b
LEFT JOIN snapshot_symbols a
  ON a.commit_hash = ? AND a.id = b.id
WHERE b.commit_hash = ? AND a.id IS NULL

UNION ALL

SELECT a.id AS symbolId,
       a.name AS name,
       a.kind AS kind,
       a.package_id AS packageId,
       'removed' AS changeType,
       a.start_line AS oldStartLine,
       a.end_line AS oldEndLine,
       0 AS newStartLine,
       0 AS newEndLine,
       a.signature AS oldSignature,
       '' AS newSignature,
       a.body_hash AS oldBodyHash,
       '' AS newBodyHash
FROM snapshot_symbols a
LEFT JOIN snapshot_symbols b
  ON b.commit_hash = ? AND b.id = a.id
WHERE a.commit_hash = ? AND b.id IS NULL

UNION ALL

SELECT b.id AS symbolId,
       b.name AS name,
       b.kind AS kind,
       b.package_id AS packageId,
       CASE
         WHEN a.body_hash != b.body_hash AND a.body_hash != '' AND b.body_hash != '' THEN 'modified'
         WHEN a.signature != b.signature THEN 'signature-changed'
         WHEN a.start_line != b.start_line OR a.end_line != b.end_line THEN 'moved'
         ELSE 'unchanged'
       END AS changeType,
       a.start_line AS oldStartLine,
       a.end_line AS oldEndLine,
       b.start_line AS newStartLine,
       b.end_line AS newEndLine,
       a.signature AS oldSignature,
       b.signature AS newSignature,
       a.body_hash AS oldBodyHash,
       b.body_hash AS newBodyHash
FROM snapshot_symbols a
JOIN snapshot_symbols b
  ON b.id = a.id
WHERE a.commit_hash = ?
  AND b.commit_hash = ?
  AND (
    a.body_hash != b.body_hash
    OR a.signature != b.signature
    OR a.start_line != b.start_line
    OR a.end_line != b.end_line
  )
ORDER BY name;
```

Then compute stats in TypeScript by iterating rows.

### `getSymbolBodyDiff`

This is the most important operation that sql.js unlocks.

Step 1: look up old symbol and file.

```sql
SELECT s.id AS symbolId,
       s.name AS name,
       s.start_offset AS startOffset,
       s.end_offset AS endOffset,
       s.start_line AS startLine,
       s.end_line AS endLine,
       f.path AS filePath,
       f.content_hash AS contentHash,
       f.sha256 AS sha256
FROM snapshot_symbols s
JOIN snapshot_files f
  ON f.commit_hash = s.commit_hash
 AND f.id = s.file_id
WHERE s.commit_hash = ? AND s.id = ?;
```

Step 2: look up new symbol and file with the same query.

Step 3: read old and new file contents.

```sql
SELECT content
FROM file_contents
WHERE content_hash = ?;
```

Step 4: extract bodies by byte offsets.

```ts
function extractBody(content: string, startOffset: number, endOffset: number): string {
  return content.slice(startOffset, endOffset);
}
```

Caution: Go byte offsets and JavaScript string offsets are not always identical for non-ASCII source. Safer first pass:

- decode as UTF-8 string;
- use offsets if source is mostly ASCII Go code;
- add tests with non-ASCII later;
- if needed, slice the `Uint8Array` before decoding:

```ts
const bodyBytes = bytes.slice(startOffset, endOffset);
const body = decoder.decode(bodyBytes);
```

Recommended implementation:

```ts
function extractBodyBytes(content: Uint8Array, startOffset: number, endOffset: number): string {
  if (startOffset < 0 || endOffset > content.length || startOffset > endOffset) {
    throw new QueryError('SQL_ERROR', `invalid symbol byte range ${startOffset}-${endOffset}`);
  }
  return new TextDecoder('utf-8').decode(content.slice(startOffset, endOffset));
}
```

Step 5: compute diff.

For initial implementation, we can generate a simple unified diff or feed `oldBody` and `newBody` directly to existing Diffs UI.

Result shape:

```ts
interface BodyDiffResult {
  symbolId: string;
  name: string;
  oldCommit: string;
  newCommit: string;
  oldBody: string;
  newBody: string;
  unifiedDiff: string;
  oldRange: string;
  newRange: string;
}
```

Pseudocode:

```ts
async getSymbolBodyDiff(from, to, symbolId) {
  const oldHash = await this.resolveCommitRef(from);
  const newHash = await this.resolveCommitRef(to);

  const oldMeta = await this.getSymbolBodyMeta(oldHash, symbolId);
  const newMeta = await this.getSymbolBodyMeta(newHash, symbolId);

  const oldBytes = await this.getContentBytes(oldMeta.contentHash);
  const newBytes = await this.getContentBytes(newMeta.contentHash);

  const oldBody = extractBodyBytes(oldBytes, oldMeta.startOffset, oldMeta.endOffset);
  const newBody = extractBodyBytes(newBytes, newMeta.startOffset, newMeta.endOffset);

  return {
    symbolId,
    name: newMeta.name || oldMeta.name,
    oldCommit: oldHash,
    newCommit: newHash,
    oldBody,
    newBody,
    unifiedDiff: simpleUnifiedDiff(oldBody, newBody),
    oldRange: `lines ${oldMeta.startLine}-${oldMeta.endLine}`,
    newRange: `lines ${newMeta.startLine}-${newMeta.endLine}`,
  };
}
```

### `getImpact`

Implement impact as JavaScript BFS over SQL refs.

Pseudocode:

```ts
async getImpact({ symbolId, direction, depth, commit }) {
  const commitHash = await this.resolveCommitRef(commit || 'HEAD');
  const visited = new Set<string>([symbolId]);
  const nodeById = new Map<string, ImpactNode>();
  const queue = [{ symbolId, depth: 0 }];

  while (queue.length > 0) {
    const item = queue.shift()!;
    if (item.depth >= depth) continue;

    const edges = direction === 'uses'
      ? await this.getRefsFrom(item.symbolId, { commit: commitHash })
      : await this.getRefsTo(item.symbolId, { commit: commitHash });

    for (const edge of edges) {
      const nextId = direction === 'uses' ? edge.toSymbolId : edge.fromSymbolId;
      const nextDepth = item.depth + 1;

      let node = nodeById.get(nextId);
      if (!node) {
        const meta = await this.getSymbol(nextId, { commit: commitHash }).catch(() => null);
        node = {
          symbolId: nextId,
          name: meta?.name ?? fallbackName(nextId),
          kind: meta?.kind ?? 'external',
          depth: nextDepth,
          edges: [],
          compatibility: 'unknown',
          local: !!meta,
        };
        nodeById.set(nextId, node);
      }

      node.edges.push(edge);

      if (!visited.has(nextId)) {
        visited.add(nextId);
        queue.push({ symbolId: nextId, depth: nextDepth });
      }
    }
  }

  return {
    root: symbolId,
    direction,
    depth,
    commit: commitHash,
    nodes: [...nodeById.values()],
  };
}
```

### `getReviewDoc`

Preferred query if using pre-render table:

```sql
SELECT slug, title, html, snippets_json AS snippetsJson, errors_json AS errorsJson
FROM static_review_rendered_docs
WHERE slug = ?;
```

If that table does not exist yet, first pass can fall back to raw `review_docs` and render client-side later, but pre-rendering in Go is recommended.

## Review Markdown Renderer Design

### Current behavior

`ui/src/features/review/ReviewDocPage.tsx` fetches a review doc and hydrates placeholders. It currently looks for snippet-specific markers:

```ts
querySelectorAll('[data-codebase-snippet]')
```

But review docs now support many widget types:

- `codebase-snippet`
- `codebase-diff`
- `codebase-diff-stats`
- `codebase-symbol-history`
- `codebase-impact`
- `codebase-changed-files`

### Target behavior

Use a generic marker:

```html
<div
  data-codebase-widget="true"
  data-directive="codebase-impact"
  data-sym="sym:..."
  data-params='{"dir":"uses","depth":"2"}'
></div>
```

Frontend:

```tsx
function ReviewDocPage() {
  // find [data-codebase-widget]
  // parse directive, sym, kind, lang, commit, params
  // createPortal(<CodebaseWidget ... />, element)
}
```

Widget dispatcher:

```tsx
function CodebaseWidget(props: WidgetProps) {
  switch (props.directive) {
    case 'codebase-snippet':
      return <SnippetWidget {...props} />;
    case 'codebase-diff':
      return <BodyDiffWidget {...props} />;
    case 'codebase-diff-stats':
      return <DiffStatsWidget {...props} />;
    case 'codebase-symbol-history':
      return <SymbolHistoryWidget {...props} />;
    case 'codebase-impact':
      return <ImpactInlineWidget {...props} />;
    default:
      return <UnsupportedWidget directive={props.directive} />;
  }
}
```

The widgets call provider-backed hooks. They do not know if data came from HTTP or sql.js.

## Server Mode vs Static Mode

We still keep server mode for development and live serving. But server mode and static mode should be implementation details behind the provider.

```text
React component
  -> useGetSymbolHistoryQuery
  -> queryFn calls getQueryProvider().getSymbolHistory
  -> ServerQueryProvider or SqlJsQueryProvider
```

### ServerQueryProvider

Uses existing APIs:

```text
/api/history/commits
/api/history/diff
/api/history/symbols/:id/history
/api/history/symbol-body-diff
/api/history/impact
/api/review/docs
```

### SqlJsQueryProvider

Uses SQL against `db/codebase.db`.

This means old static endpoint parsing in `historyApi.ts` should be removed:

```ts
// remove this style
if (arg.startsWith('/symbol-body-diff?')) { ... }
```

Replace with semantic `queryFn` calls.

## Export Command Design

The static export command should become mostly packaging:

1. ensure DB exists;
2. optionally add static-only tables such as rendered review docs and FTS;
3. build SPA with `VITE_STATIC_EXPORT=1`;
4. copy SPA output;
5. copy SQLite DB to `db/codebase.db`;
6. copy sql.js wasm asset;
7. write `manifest.json`;
8. optionally copy source tree.

Suggested flags:

```text
--db string
    Path to SQLite DB produced by review index or review db create.

--out string
    Output directory.

--repo-root string
    Repository root, used for optional source copy or validation.

--include-source bool
    Copy source tree into export. Optional; DB-backed source should make this less necessary.

--include-db bool
    Always true for sql.js static browser. Keep only if we later support review-only HTML exports.

--render-review-docs bool
    Pre-render review docs into static_review_rendered_docs.

--build-search-fts bool
    Add/populate symbol_search_fts.
```

Since this ticket is a clean cutoff, it is acceptable to remove old flags or old behavior.

## Go Export Implementation Plan

### New package

Create:

```text
internal/staticapp
```

Possible files:

```text
internal/staticapp/export.go
internal/staticapp/manifest.go
internal/staticapp/reviewdocs.go
internal/staticapp/fts.go
internal/staticapp/assets.go
```

### Manifest types

```go
type Manifest struct {
    SchemaVersion int               `json:"schemaVersion"`
    Kind          string            `json:"kind"`
    GeneratedAt   string            `json:"generatedAt"`
    DB            DBManifest        `json:"db"`
    Features      FeatureManifest   `json:"features"`
    Repo          RepoManifest      `json:"repo"`
    CommitRange   string            `json:"commitRange"`
    Commits       CommitManifest    `json:"commits"`
    Runtime       RuntimeManifest   `json:"runtime"`
}

type DBManifest struct {
    Path          string `json:"path"`
    SizeBytes     int64  `json:"sizeBytes"`
    SchemaVersion int    `json:"schemaVersion"`
}
```

### Export entrypoint

```go
type Options struct {
    DBPath string
    OutDir string
    RepoRoot string
    IncludeSource bool
    RenderReviewDocs bool
    BuildSearchFTS bool
}

func Export(ctx context.Context, opts Options) error {
    if opts.DBPath == "" { return errors.New("DBPath required") }
    if opts.OutDir == "" { return errors.New("OutDir required") }

    workDB := opts.DBPath

    if opts.RenderReviewDocs || opts.BuildSearchFTS {
        // Either mutate a copy or mutate the input DB intentionally.
        // Recommended: copy to output first, then add static tables to output DB.
    }

    if err := buildSPA(ctx); err != nil { return err }
    if err := copySPA(opts.OutDir); err != nil { return err }
    if err := copyDB(opts.DBPath, filepath.Join(opts.OutDir, "db", "codebase.db")); err != nil { return err }

    outDB := filepath.Join(opts.OutDir, "db", "codebase.db")
    if opts.RenderReviewDocs { addRenderedReviewDocs(ctx, outDB, opts) }
    if opts.BuildSearchFTS { addSearchFTS(ctx, outDB) }

    manifest := buildManifest(ctx, outDB, opts)
    writeManifest(opts.OutDir, manifest)

    copySqlJsWasm(opts.OutDir)

    if opts.IncludeSource { copySource(opts.RepoRoot, filepath.Join(opts.OutDir, "source")) }

    return nil
}
```

### Rendered review docs

Use existing renderer:

```text
internal/docs/renderer.go
```

Existing server path renders docs on demand in:

```text
internal/review/server.go
```

For static export, do it once and store in DB.

Pseudocode:

```go
func AddRenderedReviewDocs(ctx context.Context, dbPath string, repoRoot string) error {
    store, err := review.Open(dbPath)
    if err != nil { return err }
    defer store.Close()

    loaded, err := review.LoadLatestSnapshot(ctx, store)
    if err != nil { return err }

    _, err = store.DB().ExecContext(ctx, createRenderedDocsTableSQL)
    if err != nil { return err }

    rows, err := store.DB().QueryContext(ctx, `SELECT slug, title, content FROM review_docs ORDER BY slug`)
    if err != nil { return err }
    defer rows.Close()

    for rows.Next() {
        var slug, title, content string
        rows.Scan(&slug, &title, &content)

        page, err := docs.Render(slug, []byte(content), loaded, os.DirFS(repoRoot))
        errorsJSON := "[]"
        html := ""
        snippetsJSON := "[]"
        if err != nil {
            errorsJSON = marshal([]string{err.Error()})
        } else {
            html = page.HTML
            snippetsJSON = marshal(page.Snippets)
        }

        upsert rendered row
    }
}
```

If the current `docs.Page` type fields differ, adapt to the actual type. The design intent is stable: store pre-rendered HTML in SQLite.

## Frontend Refactor Plan

### Phase 1: Install sql.js and load DB

- Add `sql.js` dependency.
- Ensure `sql-wasm.wasm` is copied to static output.
- Add `getStaticDb()`.
- Add a simple debug page or console test that runs:

```sql
SELECT COUNT(*) FROM commits;
```

Acceptance:

- static export loads DB in browser;
- no `/api/*` request is made for the DB;
- console can print commit count.

### Phase 2: Implement `SqlJsQueryProvider` commits/history/body diff

Implement first:

- `listCommits`
- `resolveCommitRef`
- `getSymbolHistory`
- `getSymbolBodyDiff`

Why these first?

They directly fix the `/history?symbol=...Register` class of problem.

Acceptance:

- `/history?symbol=sym:...Register` works in static export;
- body diff is computed on demand from DB content;
- no `STATIC_NOT_PRECOMPUTED` remains.

### Phase 3: Replace `historyApi.ts` static branch

Refactor RTK Query endpoints to use provider `queryFn`.

Current style:

```ts
baseQuery: historyBaseQuery
query: ({ from, to }) => `/diff?from=${...}`
```

Target style:

```ts
getDiff: builder.query<CommitDiff, { from: string; to: string }>({
  queryFn: async ({ from, to }) => providerResult(() => getQueryProvider().getCommitDiff(from, to)),
})
```

`providerResult` converts exceptions to RTK Query errors:

```ts
async function providerResult<T>(fn: () => Promise<T>) {
  try {
    return { data: await fn() };
  } catch (err) {
    return { error: normalizeQueryError(err) };
  }
}
```

### Phase 4: Implement commit diff and impact

Add:

- `getCommitDiff`
- `getRefsFrom`
- `getRefsTo`
- `getImpact`

Acceptance:

- commit diff view works;
- impact widget works from SQL refs;
- review doc impact widget no longer needs precomputed impacts.

### Phase 5: Implement review docs through SQL

Add:

- `listReviewDocs`
- `getReviewDoc`

Preferred source:

```text
static_review_rendered_docs
```

Fallback source if needed:

```text
review_docs
```

Acceptance:

- `/review/:slug` loads from sql.js provider;
- widget stubs hydrate;
- widgets use SQL provider.

### Phase 6: Remove TinyGo reviewData path

Delete or stop using:

```text
internal/wasm/review_types.go
GetCommitDiff/GetSymbolHistory/GetImpact/GetSymbolBodyDiff/GetReviewDocs/GetReviewDoc in internal/wasm/search.go
reviewData argument in WASM init
reviewData object in precomputed.json
```

Keep TinyGo WASM only if there is another clear value, such as specialized search. If SQL/FTS covers search, consider removing it from static mode entirely.

## UI Route Expectations

### Must work in static sql.js export

```text
/
/packages/:id
/symbol/:id
/source/*
/history
/history?symbol=sym:...
/review/:slug
```

### Should work after provider completeness

```text
/queries
/doc/:slug if docs are still included
```

### Review doc cross-links

Review docs should link into browser routes:

```text
#/symbol/:encodedSymbolId
#/history?symbol=:encodedSymbolId
#/source/:path
```

## Handling Source Content

There are two possible source paths:

1. Fetch copied source files from `source/`.
2. Read source from `file_contents` through sql.js.

For static browser consistency, prefer DB-backed source content for symbol bodies and snippets. The DB already contains file contents keyed by content hash.

For full source pages, query file metadata then file content:

```sql
SELECT f.path, f.content_hash, fc.content
FROM snapshot_files f
JOIN file_contents fc ON fc.content_hash = f.content_hash
WHERE f.commit_hash = ? AND f.path = ?;
```

If source pages currently rely heavily on `source/`, keep `source/` initially, but the long-term goal is DB-backed source reading.

## Performance Considerations

### Plain sql.js loads the whole DB

First implementation should use plain sql.js. It fetches the whole DB and loads it into memory.

Good:

- simple;
- works with ordinary static servers;
- easy to debug;
- no range request complexity.

Bad:

- initial load can be slow for large DBs;
- memory usage roughly tracks DB size;
- queries run on main thread unless moved to a worker.

### Worker plan

If UI blocking becomes noticeable, move SQL to a Web Worker.

Architecture:

```text
React main thread
  -> QueryProvider proxy
  -> Worker postMessage
  -> sql.js DB in worker
  -> rows returned
```

Do not implement worker first unless necessary. It increases complexity.

### sql.js-httpvfs later

If DBs become too large, consider `sql.js-httpvfs`.

Pros:

- lazy page loading;
- lower initial download;
- better for hosted large DBs.

Cons:

- static server needs range requests;
- more moving parts;
- not a good first milestone.

## Testing Strategy

### Go tests

Add tests for:

- manifest writing;
- static export copies DB to `db/codebase.db`;
- rendered review docs table creation;
- FTS table creation if implemented;
- output directory contains required assets.

### TypeScript tests

Add tests for provider SQL methods. Use a tiny fixture DB if possible.

Test cases:

- `resolveCommitRef('HEAD')`;
- `resolveCommitRef('HEAD~1')`;
- `listCommits()` order;
- `getSymbolHistory(symbol)`;
- `getSymbolBodyDiff(old, new, symbol)` extracts expected text;
- `getImpact(...dir='uses')` returns edges.

### Browser tests

Use Playwright against a real static export.

Smoke review doc should contain:

```markdown
# Static SQL Smoke Review

```codebase-snippet sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd
```

```codebase-diff-stats from=HEAD~1 to=HEAD
```

```codebase-diff sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd from=HEAD~1 to=HEAD
```

```codebase-symbol-history sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.Register
```

```codebase-impact sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd dir=uses depth=2
```
```

Assertions:

```ts
expect(apiRequests).toHaveLength(0);
await expect(page.getByText('Static SQL Smoke Review')).toBeVisible();
await expect(page.getByText('Diff: newExportCmd')).toBeVisible();
await expect(page.getByText('Impact: newExportCmd')).toBeVisible();
```

Also open directly:

```text
/#/history?symbol=sym:github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.Register
```

Assert:

- history entries render;
- selecting a changed transition renders body diff;
- no `STATIC_NOT_PRECOMPUTED` appears;
- no `/api/*` requests.

## Implementation Checklist

### Phase 1 — Ticket setup and dependencies

- [ ] Add `sql.js` dependency to `ui/package.json`.
- [ ] Ensure `sql-wasm.wasm` is copied into static output.
- [ ] Add `manifest.json` writing in export.
- [ ] Change export to copy DB to `db/codebase.db`.

### Phase 2 — DB loader in frontend

- [ ] Add `ui/src/api/sqljs/sqlJsDb.ts`.
- [ ] Add `ui/src/api/sqljs/sqlRows.ts`.
- [ ] Add debug/test query for commit count.

### Phase 3 — Provider interface

- [ ] Add `CodebaseQueryProvider` interface.
- [ ] Add `SqlJsQueryProvider`.
- [ ] Add `ServerQueryProvider` or adapt existing API calls behind provider.
- [ ] Add provider factory based on `VITE_STATIC_EXPORT`.

### Phase 4 — Core history support

- [ ] Implement `listCommits`.
- [ ] Implement `resolveCommitRef`.
- [ ] Implement `getSymbolHistory`.
- [ ] Implement `getSymbolBodyDiff` from SQL file contents.
- [ ] Refactor History page API usage to provider-backed query functions.

### Phase 5 — Browser completeness

- [ ] Implement symbol lookup/search.
- [ ] Implement package/file/source queries.
- [ ] Implement commit diff in SQL without `FULL OUTER JOIN`.
- [ ] Implement refs/xrefs.
- [ ] Implement impact BFS in TypeScript over SQL refs.

### Phase 6 — Review renderer

- [ ] Add `static_review_rendered_docs` table during export.
- [ ] Implement `listReviewDocs` and `getReviewDoc` with SQL.
- [ ] Rename widget placeholder marker to `data-codebase-widget`.
- [ ] Add generic widget dispatcher.
- [ ] Ensure review widgets use provider methods.

### Phase 7 — Remove old static WASM review path

- [ ] Remove `reviewData` from WASM init.
- [ ] Remove WASM review exports that duplicate SQL provider methods.
- [ ] Remove static `historyApi.ts` endpoint parsing.
- [ ] Remove `PrecomputedReview` runtime dependency.

### Phase 8 — Tests and docs

- [ ] Add Go export tests.
- [ ] Add TypeScript provider tests.
- [ ] Add Playwright static export regression.
- [ ] Update help docs for new static sql.js export behavior.

## Design Decisions

### Decision 1: SQLite is both build-time and static runtime source

Rationale:

- The DB already contains the relational data needed by the frontend.
- SQL naturally expresses the frontend's queries.
- The LLM artifact and static browser artifact become the same file.

### Decision 2: sql.js replaces TinyGo reviewData transport

Rationale:

- TinyGo does not let us reuse the existing Go SQLite server code in the browser.
- sql.js is purpose-built for browser SQLite.
- It avoids precompute coverage gaps for normal navigation.

### Decision 3: Go still owns indexing

Rationale:

- Go has the AST/indexing implementation.
- The browser should not parse source or run git operations.
- Static export is a packaging step, not an indexing engine.

### Decision 4: Review docs are layered on the browser provider

Rationale:

- Review widgets are just authored uses of browser queries.
- Cross-links should work because the generic browser works.
- Review docs should not own a separate data model.

### Decision 5: Precompute into SQLite tables, not JSON blobs

Rationale:

- SQLite keeps the data inspectable and queryable.
- Derived tables such as rendered review docs or FTS are still part of one artifact.
- Avoids drift between DB and JSON data.

## Alternatives Considered

### Keep precomputed JSON plus TinyGo WASM

Rejected.

It causes coverage gaps and duplicates the database in a less flexible form.

### Use sql.js-httpvfs immediately

Deferred.

It may be useful for large DBs, but plain sql.js is simpler and better for proving the architecture.

### Render markdown entirely in the browser

Deferred.

Go already has a directive-aware renderer. Pre-rendered HTML in SQLite is simpler for the first implementation.

### Keep review-only static export

Rejected.

The desired product has a generic static browser as the foundation and review docs as a layer on top.

## Acceptance Criteria

The ticket is complete when:

1. Static export ships `db/codebase.db` and opens it with sql.js.
2. Static frontend uses `SqlJsQueryProvider` for browser and review widgets.
3. Static mode makes zero `/api/*` requests.
4. `/history?symbol=sym:...Register` works and computes body diffs on demand.
5. Review docs render from static SQL-backed data.
6. `codebase-diff`, `codebase-diff-stats`, `codebase-symbol-history`, and `codebase-impact` widgets work in static mode.
7. TinyGo reviewData transport is removed or unused.
8. The SQLite DB remains usable as an LLM/query artifact.
9. Browser tests cover the static review doc and direct history route.

## File Reference Map

### Go indexing and database

- `internal/history/schema.go` — history tables and `symbol_history` view.
- `internal/history/store.go` — database store wrapper.
- `internal/history/loader.go` — commit indexing into history tables.
- `internal/history/diff.go` — current Go commit diff logic; useful reference for SQL provider.
- `internal/history/bodydiff.go` — current body diff logic; useful reference for SQL provider.
- `internal/review/schema.go` — review document tables.
- `internal/review/indexer.go` — review indexing flow.
- `internal/review/server.go` — server review routes; useful reference for provider methods.
- `cmd/codebase-browser/cmds/review/export.go` — current export command to replace/simplify.

### Current static/TinyGo path to remove or simplify

- `internal/review/export.go` — current `PrecomputedReview` builder.
- `internal/wasm/review_types.go` — reviewData WASM types.
- `internal/wasm/search.go` — current review query exports.
- `internal/wasm/exports.go` — JS exports for TinyGo WASM.
- `ui/src/api/wasmClient.ts` — current WASM client.

### Frontend APIs and UI

- `ui/src/api/historyApi.ts` — refactor to provider query functions.
- `ui/src/api/docApi.ts` — refactor review docs to provider methods.
- `ui/src/api/runtimeMode.ts` — likely replaced by provider factory usage.
- `ui/src/app/App.tsx` — routes for browser and review renderer.
- `ui/src/features/history/HistoryPage.tsx` — direct target for SQL-backed history/body diff.
- `ui/src/features/review/ReviewDocPage.tsx` — review HTML hydration.
- `ui/src/features/doc/DocSnippet.tsx` — widget dispatcher and inline widgets.
- `ui/src/features/diff/DiffsUnifiedDiff.tsx` — body diff rendering.

## Final Guidance for the Intern

Work from the inside out:

1. First make the browser open the SQLite DB.
2. Then make one important route work: `/history?symbol=...`.
3. Then replace one widget: `codebase-diff` using SQL body diffs.
4. Then expand to commit diff and impact.
5. Only then remove the old TinyGo reviewData path.

Do not start by optimizing performance or redesigning every UI component. The important architectural proof is:

```text
static React page -> SqlJsQueryProvider -> db/codebase.db -> real result
```

Once that works, the rest is systematic query coverage.
