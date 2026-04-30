---
Title: Standalone WASM Export — Browser-Side SQLite Queries for Code Review
Ticket: GCB-013
Status: active
Topics:
    - codebase-browser
    - pr-review
    - code-review
    - sqlite-index
    - wasm
    - static-build
    - browser-side-query
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/wasm/main.go
      Note: |-
        WASM entry point — TinyGo-compiled browser-side module
        WASM entry point — foundation for browser-side query module
    - Path: internal/docs/renderer.go
      Note: Markdown directive pipeline — codebase-* → HTML stubs
    - Path: internal/history/bodydiff.go
      Note: Per-symbol body diff — extracts old/new function bodies
    - Path: internal/history/diff.go
      Note: Symbol-level diff between two commits
    - Path: internal/history/schema.go
      Note: History SQLite schema — commits, snapshot_symbols, snapshot_refs
    - Path: internal/history/store.go
      Note: History store — Open, Create, ResetSchema
    - Path: internal/static/doc_renderer.go
      Note: Pre-renders markdown docs to static HTML at build time
    - Path: internal/static/generate_build.go
      Note: |-
        Build-time pre-computation — search index, xref, snippets, doc HTML
        Pre-computation pipeline — model for review export JSON generation
    - Path: internal/static/search_index.go
      Note: Inverted index builder for substring symbol search
    - Path: internal/static/xref_index.go
      Note: Pre-computed cross-reference data per symbol
    - Path: internal/wasm/exports.go
      Note: JS interop exports — window.codebaseBrowser.findSymbols, getSymbol, etc.
    - Path: internal/wasm/generate_build.go
      Note: Dagger-based TinyGo WASM compiler pipeline
    - Path: internal/wasm/search.go
      Note: |-
        WASM search context — in-memory index with lookup maps
        Browser-side search context — to be extended with history query methods
    - Path: ttmp/2026/04/23/GCB-006--static-wasm-build-pre-render-html-ship-go-search-as-browser-side-module/design-doc/01-wasm-static-html-architecture-design.md
      Note: Predecessor design — static WASM build architecture (GCB-006)
    - Path: ui/src/api/wasmClient.ts
      Note: Frontend WASM client — loads search.wasm and calls exports
    - Path: ui/src/features/doc/DocSnippet.tsx
      Note: Widget hydration dispatcher — mounts React into HTML stubs
ExternalSources: []
Summary: Design exploration for a standalone browser export of the code review tool where all queries run client-side in WASM. Three approaches are compared (pre-computed JSON, sql.js, pure-Go SQLite WASM) and a hybrid recommendation is made that extends the existing GCB-006 static build with history-aware pre-computation.
LastUpdated: 2026-04-30T14:00:00Z
WhatFor: Evaluate and design a serverless browser export for code review guides
WhenToUse: Read when deciding between server-based (review serve) and standalone WASM export modes
---




# Standalone WASM Export — Browser-Side SQLite Queries for Code Review

## 1. Executive summary

The main GCB-013 design proposes `codebase-browser review serve --db file.db` — a Go HTTP server that queries a SQLite database and serves JSON to a React SPA. This document explores an alternative: a **standalone static export** that requires no server at all.

The idea is simple: after running `codebase-browser review index`, instead of (or in addition to) producing a `.db` file for a server, we produce a directory of static files (HTML, JS, WASM, JSON) that can be opened directly in a browser — even over `file://` — with all queries running client-side in WebAssembly.

The codebase-browser **already has most of the infrastructure** for this. GCB-006 built a static WASM search module that compiles the Go indexer logic to TinyGo WASM, pre-computes search/xref/snippet data at build time, and serves it all from a single HTML file with no backend. What is missing is extending that infrastructure to handle **history-aware queries** (commit diffs, symbol history, impact analysis) which are the core of the code review use case.

This document compares three technical approaches, recommends a hybrid, and provides concrete implementation guidance.

## 2. Problem statement

### 2.1 The server dependency problem

The `review serve` approach has these operational costs:
- You need a running process (`codebase-browser review serve --db ...`)
- You need to manage ports, firewalls, and process lifetime
- You cannot email someone a review artifact — you must give them a URL to a running server
- You cannot review code on an airplane or in a locked-down environment without network access

### 2.2 The vision: email a review

A PR author runs:

```bash
codebase-browser review export --commits HEAD~5..HEAD --docs ./reviews/pr-42.md --out ./pr-42-review/
```

This produces:

```
pr-42-review/
├── index.html              # SPA shell
├── search.wasm             # Go search module (TinyGo)
├── wasm_exec.js            # Go WASM runtime glue
├── precomputed.json        # History data + review docs (gzipped)
├── review.db               # SQLite database (optional, for LLM use)
└── assets/
    ├── main-abc123.js      # React bundle
    ├── index-CBAwCAUr.js   # Chunked JS
    └── ...
```

The author zips `pr-42-review/`, emails it to reviewers, or uploads it to S3/Netlify/Vercel as a static site. Reviewers double-click `index.html` and see the full interactive review guide with diff widgets, history, and impact analysis — no server, no install, no network required after the initial load.

### 2.3 Why this is feasible

Every query the review tool makes is against data that is **fully known at build time**:
- Commit metadata, symbol snapshots, cross-references — all in the review DB at `review index` time
- Markdown docs and resolved snippets — all rendered at build time
- Diff computations between commits — deterministic given two commit snapshots
- Impact analysis — BFS over a static graph

The only reason we need a server today is because the data lives in SQLite and the query logic is in Go. If we move both into the browser, the server becomes optional.

## 3. Existing WASM infrastructure (GCB-006)

Before we design the extension, you must understand what already exists. The codebase-browser has a working static WASM build.

### 3.1 The WASM module

File: `cmd/wasm/main.go`

```go
//go:build wasm
package main
import "github.com/wesen/codebase-browser/internal/wasm"
func main() {
    wasm.RegisterExports()
    <-make(chan struct{})  // block forever
}
```

This is compiled with TinyGo to `internal/wasm/embed/search.wasm`. The build is orchestrated by `internal/wasm/generate_build.go` which uses Dagger (Docker) or a local TinyGo binary.

### 3.2 The search context

File: `internal/wasm/search.go`

The `SearchCtx` struct holds:
- `Index` — deserialized index.json (packages, files, symbols, refs)
- `SearchIndex` — inverted index: lowercase name → symbol IDs
- `XrefIndex` — pre-computed usedBy/uses per symbol
- `Snippets` — pre-extracted symbol text (declaration, body, signature)
- `DocHTML` — pre-rendered doc pages
- `DocManifest` — list of doc pages

All data is loaded once at init time from JSON byte slices passed from JS.

### 3.3 The exports

File: `internal/wasm/exports.go`

Registered on `window.codebaseBrowser`:
- `initWasm(jsonIndex, jsonSearchIdx, jsonXrefIdx, jsonSnippets, jsonDocManifest, jsonDocHTML)` — loads all data
- `findSymbols(query, kind)` — substring search over symbol names
- `getSymbol(id)` — lookup by symbol ID
- `getXref(id)` — pre-computed cross-references
- `getSnippet(id, kind)` — pre-extracted text
- `getPackages()` — package list
- `getIndexSummary()` — raw index JSON
- `getDocPages()` / `getDocPage(slug)` — doc pages

### 3.4 The pre-computation pipeline

File: `internal/static/generate_build.go`

Runs at `go generate` time:
1. Loads `index.json`
2. Builds `searchIndex` (inverted index)
3. Builds `xrefIndex` (usedBy/uses per symbol)
4. Extracts `snippets` (text from source files)
5. Extracts `snippetRefs` and `sourceRefs`
6. Pre-renders doc pages to HTML
7. Writes everything to `internal/static/embed/precomputed.json`

### 3.5 The frontend integration

File: `ui/src/api/wasmClient.ts`

The React SPA uses RTK-Query with a custom `baseQuery` that calls WASM exports instead of HTTP endpoints. The API surface (hooks, types) is identical — only the transport changes.

## 4. Three approaches for browser-side history queries

To extend the static WASM build for code review, we need browser-side access to:
- Commit metadata and snapshot symbols across multiple commits
- Diff computations between any two commits
- Symbol history (timeline of changes)
- Impact analysis (BFS over snapshot_refs)

Here are three approaches to make this data queryable in the browser.

### 4.1 Approach A: Pre-compute JSON from SQLite (extend GCB-006)

**Idea:** At `review export` time, query the SQLite database and serialize all needed data into JSON files. The WASM module loads these JSON files into memory and queries them with Go code.

**Data to pre-compute:**

```go
// PrecomputedReviewData is loaded into WASM memory at init time.
type PrecomputedReviewData struct {
    // Commits in the review range
    Commits []CommitRow `json:"commits"`

    // Symbols indexed by (commit_hash, symbol_id)
    // Stored as: map[commitHash]map[symbolID]*SymbolSnapshot
    Snapshots map[string]map[string]*SymbolSnapshot `json:"snapshots"`

    // Pre-computed diffs for every adjacent commit pair
    // Stored as: map["old..new"]*CommitDiff
    Diffs map[string]*CommitDiff `json:"diffs"`

    // Pre-computed symbol history
    // Stored as: map[symbolID][]SymbolHistoryEntry
    Histories map[string][]SymbolHistoryEntry `json:"histories"`

    // Pre-computed impact analysis (usedBy graph)
    // Stored as: map[symbolID][]ImpactNode
    Impacts map[string][]ImpactNode `json:"impacts"`

    // Review docs (pre-rendered HTML + metadata)
    Docs []ReviewDoc `json:"docs"`
}
```

**Pros:**
- Simple — extends existing GCB-006 infrastructure directly
- Fast queries — everything is in-memory Go maps
- No external dependencies — pure Go/TinyGo WASM
- Small WASM binary — only query logic, data is separate JSON

**Cons:**
- Large JSON files — pre-computing all diffs for N commits is O(N²) pairs
- No ad-hoc SQL — LLMs cannot run arbitrary SQL against JSON in the browser
- Memory usage — all data loaded into WASM linear memory at once
- Build time — diff computation at export time can be slow for large ranges

**Best for:** Review guides with known query patterns (diff specific pairs, show specific histories). Not ideal for open-ended LLM exploration.

### 4.2 Approach B: sql.js — SQLite compiled to JavaScript/WASM

**Idea:** Use `sql.js` (SQLite compiled to WASM via Emscripten) to load the `.db` file directly in the browser. The SQLite database is shipped as a binary file. The React frontend calls sql.js to run SQL queries. The Go WASM module is not needed for database access.

**Architecture:**

```
Browser
├── React SPA
│   ├── sql.js (Emscripten SQLite WASM)
│   ├── review.db (SQLite binary, fetched as ArrayBuffer)
│   └── UI components (call sql.js via JS API)
└── No Go WASM needed for DB queries
```

**sql.js API:**

```javascript
// Load sql.js
const SQL = await initSqlJs({ locateFile: file => `/assets/${file}` });

// Load the review database
const response = await fetch('/review.db');
const arrayBuffer = await response.arrayBuffer();
const db = new SQL.Database(new Uint8Array(arrayBuffer));

// Run any SQL query
const result = db.exec(`
  SELECT s.name, s.signature, c.short_hash, c.message
  FROM snapshot_symbols s
  JOIN commits c ON c.hash = s.commit_hash
  WHERE s.id = 'sym:github.com/.../indexer.func.Merge'
  ORDER BY c.author_time DESC
`);
// result = [{ columns: [...], values: [[...], [...]] }]
```

**Pros:**
- Full SQL in the browser — arbitrary queries, perfect for LLM integration
- No pre-computation needed — ship the raw `.db` file
- Small build time — just copy the SQLite file
- sql.js is battle-tested (used by many projects)

**Cons:**
- Large JS bundle — sql.js is ~1MB compressed
- Requires sql.js dependency — another library to manage
- No Go type safety — queries are raw SQL strings, no compile-time checking
- Need to rewrite history query logic in SQL/JS — the Go diff/history/impact code cannot be reused directly
- Performance — sql.js uses a single-threaded WASM SQLite; large DBs may be slow

**Best for:** Maximum flexibility, LLM ad-hoc queries, when the review DB is the primary artifact.

### 4.3 Approach C: Pure-Go SQLite in WASM

**Idea:** Use a pure-Go SQLite implementation (like `modernc.org/sqlite`) compiled to WASM. The Go WASM module includes SQLite and opens the `.db` file from WASM memory. Queries use the standard `database/sql` API.

**Architecture:**

```
Browser
├── React SPA
│   └── Go WASM module (TinyGo/Go)
│       ├── Pure-Go SQLite driver
│       ├── review.db (loaded into WASM memory)
│       └── Query functions (Go code using database/sql)
```

**Pseudocode:**

```go
// WASM-exported function
func QueryHistory(ctx *ReviewCtx, sql string) []byte {
    rows, err := ctx.db.Query(sql)
    // ... scan into structs, marshal to JSON
    return jsonBytes
}
```

**Pros:**
- Full SQL with Go type safety
- Reuses existing Go history query code
- No JavaScript SQL library needed
- Arbitrary queries supported

**Cons:**
- Pure-Go SQLite may not support all SQLite features (FTS5, custom extensions)
- Large WASM binary — SQLite + Go runtime = potentially 5-10MB
- TinyGo compatibility — `modernc.org/sqlite` may not compile with TinyGo
- Complex build — need to embed the DB file into WASM memory or fetch it separately

**Best for:** When you want Go type safety + SQL flexibility, and can tolerate a larger WASM binary.

### 4.4 Comparison table

| Criterion | A: Pre-computed JSON | B: sql.js | C: Pure-Go SQLite |
|---|---|---|---|
| **Binary size** | Small (~200KB WASM + JSON) | Large (~1MB sql.js) | Very large (~5-10MB WASM) |
| **Query flexibility** | Limited (pre-computed only) | Full SQL | Full SQL |
| **LLM-friendly** | No (JSON, not SQL) | Yes (raw SQL) | Yes (raw SQL) |
| **Build time** | Slow (pre-compute diffs) | Fast (copy DB) | Fast (copy DB) |
| **Go code reuse** | High (same structs/maps) | Low (rewrite in JS/SQL) | High (database/sql API) |
| **TinyGo compatible** | Yes | N/A (JS library) | Unknown (likely no) |
| **Frontend changes** | Minimal (extend wasmClient) | Moderate (add sql.js layer) | Minimal (extend wasmClient) |
| **Offline capable** | Yes | Yes | Yes |

## 5. Recommended approach: Hybrid A+B

We recommend a **hybrid** that combines Approach A (pre-computed JSON for UI queries) with Approach B (sql.js for LLM ad-hoc queries).

### 5.1 Rationale

The review tool has two distinct use cases:
1. **Interactive review guides** — reviewers read markdown docs with embedded widgets. The queries are known at build time: "diff commit A vs B for symbol X", "show history of symbol Y", "impact of symbol Z". These are fast, read-only, and map well to pre-computed JSON.
2. **LLM exploration** — an LLM needs to run arbitrary SQL: "which functions changed signatures in this PR?", "find all call sites of deprecated functions". This requires full SQL access.

Pre-computed JSON (Approach A) gives the best experience for interactive widgets: fast, small, pure Go. sql.js (Approach B) gives the LLM full query power. They can coexist in the same export.

### 5.2 Architecture

```
pr-42-review/                    # Standalone export directory
├── index.html                   # SPA shell
├── search.wasm                  # Go WASM (TinyGo) — UI queries
├── wasm_exec.js                 # Go WASM runtime
├── precomputed.json             # Pre-computed history data ( Approach A )
│   ├── commits[]
│   ├── diffs["old..new"]
│   ├── histories[symbolID]
│   ├── impacts[symbolID]
│   └── docs[] (pre-rendered HTML)
├── review.db                    # SQLite database ( Approach B )
├── sql-wasm.wasm                # sql.js SQLite engine
├── sql-wasm.js                  # sql.js JS glue
└── assets/
    ├── main-*.js                # React bundle
    └── ...
```

### 5.3 Data flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        BROWSER (no server)                              │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  React SPA                                                        │  │
│  │                                                                  │  │
│  │  Interactive widgets (diff, history, impact)                     │  │
│  │  → calls Go WASM exports (search.wasm)                           │  │
│  │  → reads precomputed.json (in-memory maps)                        │  │
│  │                                                                  │  │
│  │  LLM query panel / raw SQL explorer                               │  │
│  │  → calls sql.js (sql-wasm.wasm + review.db)                      │  │
│  │  → runs arbitrary SQL in browser                                 │  │
│  │                                                                  │  │
│  │  Doc pages (pre-rendered HTML from precomputed.json)             │  │
│  │  → displayed directly, no query needed                           │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────┐    ┌─────────────────┐    ┌──────────────────────┐   │
│  │ search.wasm  │    │ precomputed.json│    │ sql.js + review.db   │   │
│  │ (TinyGo)     │    │ (history data)  │    │ (arbitrary SQL)      │   │
│  └──────────────┘    └─────────────────┘    └──────────────────────┘   │
│        ▲                      ▲                      ▲                 │
│        │                      │                      │                 │
│        └──────────────────────┴──────────────────────┘                 │
│                              JS fetch()                                │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.4 Pre-computed JSON schema for history data

```go
// internal/review/export.go — build-time pre-computation

type PrecomputedReview struct {
    Version     string              `json:"version"`
    GeneratedAt string              `json:"generatedAt"`
    CommitRange string              `json:"commitRange"`

    // ── Commits ──
    Commits []CommitLite `json:"commits"`

    // ── Diffs ──
    // Key: "oldHash..newHash" (adjacent commits in range)
    Diffs map[string]*CommitDiffLite `json:"diffs"`

    // ── Symbol histories ──
    // Key: symbol ID
    Histories map[string][]HistoryEntryLite `json:"histories"`

    // ── Impact graphs ──
    // Key: symbol ID
    Impacts map[string]*ImpactLite `json:"impacts"`

    // ── Review docs ──
    Docs []ReviewDocLite `json:"docs"`
}

type CommitLite struct {
    Hash       string `json:"hash"`
    ShortHash  string `json:"shortHash"`
    Message    string `json:"message"`
    AuthorName string `json:"authorName"`
    AuthorTime int64  `json:"authorTime"`
}

type CommitDiffLite struct {
    OldHash   string        `json:"oldHash"`
    NewHash   string        `json:"newHash"`
    Stats     DiffStats     `json:"stats"`
    Symbols   []SymbolDiff  `json:"symbols"`
    Files     []FileDiff    `json:"files"`
}

type HistoryEntryLite struct {
    CommitHash string `json:"commitHash"`
    ShortHash  string `json:"shortHash"`
    AuthorTime int64  `json:"authorTime"`
    BodyHash   string `json:"bodyHash"`
    Signature  string `json:"signature"`
    StartLine  int    `json:"startLine"`
    EndLine    int    `json:"endLine"`
}

type ImpactLite struct {
    RootSymbol string       `json:"rootSymbol"`
    Direction  string       `json:"direction"`
    Depth      int          `json:"depth"`
    Nodes      []ImpactNode `json:"nodes"`
}

type ReviewDocLite struct {
    Slug     string       `json:"slug"`
    Title    string       `json:"title"`
    HTML     string       `json:"html"`
    Snippets []SnippetRef `json:"snippets"`
}
```

### 5.5 WASM exports for history queries

Add these to `internal/wasm/exports.go` and `internal/wasm/search.go`:

```go
// GetCommitDiff returns the pre-computed diff between two commits.
func (s *SearchCtx) GetCommitDiff(oldHash, newHash string) []byte {
    key := oldHash + ".." + newHash
    data, _ := json.Marshal(s.ReviewData.Diffs[key])
    return data
}

// GetSymbolHistory returns the pre-computed history for a symbol.
func (s *SearchCtx) GetSymbolHistory(symbolID string) []byte {
    data, _ := json.Marshal(s.ReviewData.Histories[symbolID])
    return data
}

// GetImpact returns the pre-computed impact graph for a symbol.
func (s *SearchCtx) GetImpact(symbolID, direction string, depth int) []byte {
    key := symbolID + "|" + direction + "|" + strconv.Itoa(depth)
    data, _ := json.Marshal(s.ReviewData.Impacts[key])
    return data
}

// GetReviewDocs returns the list of review docs.
func (s *SearchCtx) GetReviewDocs() []byte {
    data, _ := json.Marshal(s.ReviewData.Docs)
    return data
}

// GetReviewDoc returns a single review doc by slug.
func (s *SearchCtx) GetReviewDoc(slug string) []byte {
    for _, doc := range s.ReviewData.Docs {
        if doc.Slug == slug {
            data, _ := json.Marshal(doc)
            return data
        }
    }
    return []byte("null")
}
```

### 5.6 sql.js integration for LLM queries

A separate "SQL Console" component in the React SPA uses sql.js:

```typescript
// ui/src/features/review/SqlConsole.tsx
import initSqlJs from 'sql.js';

let sqlDb: any = null;

async function loadReviewDb() {
    const SQL = await initSqlJs({ locateFile: f => `/assets/${f}` });
    const response = await fetch('/review.db');
    const buffer = await response.arrayBuffer();
    sqlDb = new SQL.Database(new Uint8Array(buffer));
}

export function runQuery(sql: string): { columns: string[], values: any[][] } {
    if (!sqlDb) throw new Error('DB not loaded');
    const result = sqlDb.exec(sql);
    return result[0] ?? { columns: [], values: [] };
}
```

The SQL console is shown only when `review.db` is present in the export. Interactive widgets use the Go WASM path for speed.

## 6. Build pipeline: `review export`

### 6.1 Command

```bash
codebase-browser review export \
  --commits HEAD~10..HEAD \
  --docs ./reviews/pr-42.md \
  --db ./reviews/pr-42.db \
  --out ./pr-42-export/
```

### 6.2 Steps

```
1. review index --commits RANGE --docs DOCS --db DB
   → produces review.db (SQLite with history + docs)

2. Load review.db, query all needed data
   a. SELECT * FROM commits ORDER BY author_time
   b. For each adjacent commit pair (c[i], c[i+1]):
      → run history diff computation
      → store in PrecomputedReview.Diffs
   c. For each symbol that appears in >1 commit:
      → run symbol history query
      → store in PrecomputedReview.Histories
   d. For each symbol referenced in review docs:
      → run impact analysis (BFS over snapshot_refs)
      → store in PrecomputedReview.Impacts
   e. Render all review docs to HTML
      → store in PrecomputedReview.Docs

3. Write precomputed.json (gzip optionally)

4. Copy sql.js assets (sql-wasm.wasm, sql-wasm.js)

5. Build SPA with Vite (ui/)
   → but with wasmClient.ts instead of HTTP client

6. Copy review.db into output directory

7. Emit:
   pr-42-export/
   ├── index.html
   ├── search.wasm
   ├── wasm_exec.js
   ├── precomputed.json
   ├── review.db
   ├── sql-wasm.wasm
   ├── sql-wasm.js
   └── assets/
```

### 6.3 Vite build configuration

The SPA needs to know it's in "static export" mode. We can use an environment variable:

```bash
VITE_STATIC_EXPORT=1 vite build
```

In the frontend:

```typescript
// ui/src/api/store.ts
const isStaticExport = import.meta.env.VITE_STATIC_EXPORT === '1';

export const api = isStaticExport
    ? wasmApi   // calls Go WASM exports
    : httpApi;  // calls /api/* endpoints
```

## 7. File layout for the export feature

### New files

```
cmd/codebase-browser/cmds/review/
├── export.go              # review export subcommand

internal/review/
├── export.go              # ExportOptions, build precomputed.json from review.db
├── export_indexer.go      # Query review.db, compute diffs/histories/impacts
└── export_test.go         # Integration tests

ui/src/features/review/
├── SqlConsole.tsx         # sql.js query UI for LLM exploration
└── ExportNotice.tsx       # Banner showing "static export" mode

ui/src/api/
├── wasmClient.ts          # Extend with history query exports
└── sqlJsClient.ts         # sql.js wrapper for raw SQL queries
```

### Modified files

```
internal/wasm/search.go     # Add ReviewData field + history query methods
internal/wasm/exports.go    # Add JS exports for history queries
internal/static/generate_build.go  # Optionally: integrate review pre-computation
cmd/codebase-browser/cmds/review/root.go  # Register export subcommand
```

## 8. Implementation phases

### Phase 1: Review DB pre-computation (2 days)

**Goal:** Build `internal/review/export.go` that reads a review.db and produces `PrecomputedReview` JSON.

**Tasks:**
1. Define `PrecomputedReview` structs
2. Write `review.LoadForExport(dbPath)` — opens review DB, queries commits, snapshots
3. Write `review.ComputeDiffs(store)` — for each adjacent commit pair, compute symbol diff
4. Write `review.ComputeHistories(store)` — for each symbol with multiple snapshots, build timeline
5. Write `review.ComputeImpacts(store)` — BFS over snapshot_refs for symbols referenced in docs
6. Write `review.RenderDocs(store)` — render markdown docs to HTML
7. Marshal to JSON, write file

**Validation:**
```bash
codebase-browser review index --commits HEAD~3..HEAD --docs ./reviews/ --db /tmp/test.db
codebase-browser review export --db /tmp/test.db --out /tmp/export/
ls -la /tmp/export/precomputed.json
jq '.commits | length' /tmp/export/precomputed.json
jq '.diffs | keys' /tmp/export/precomputed.json
```

### Phase 2: WASM history exports (2 days)

**Goal:** Extend `internal/wasm/search.go` and `exports.go` with history query methods.

**Tasks:**
1. Add `ReviewData *PrecomputedReview` field to `SearchCtx`
2. Add `LoadReviewData(jsonData []byte)` method
3. Implement `GetCommitDiff`, `GetSymbolHistory`, `GetImpact`, `GetReviewDocs`, `GetReviewDoc`
4. Register new exports in `exports.go`
5. Update `initWasm` to accept review data as a 7th parameter

**Validation:**
```bash
# Build WASM
go generate ./internal/wasm
# Test in browser with a small fixture
```

### Phase 3: sql.js integration (1–2 days)

**Goal:** Add sql.js to the export and create a SQL console component.

**Tasks:**
1. Add `sql.js` as a dependency in `ui/package.json`
2. Create `ui/src/features/review/SqlConsole.tsx`
3. Add route `/review/sql` or modal for SQL exploration
4. Ensure `review.db` is copied to the export directory
5. Test with sample queries

**Validation:**
```bash
# In browser, open SQL console
SELECT COUNT(*) FROM commits;
SELECT name, signature FROM snapshot_symbols WHERE commit_hash = '...';
```

### Phase 4: `review export` CLI command (1 day)

**Goal:** Wire everything into a single CLI command.

**Tasks:**
1. Create `cmd/codebase-browser/cmds/review/export.go`
2. Run `review index` internally (or accept existing `--db`)
3. Run `review.LoadForExport`
4. Build SPA with `VITE_STATIC_EXPORT=1`
5. Copy all assets to `--out` directory

**Validation:**
```bash
codebase-browser review export --commits HEAD~3..HEAD --docs ./reviews/ --out /tmp/export/
python3 -m http.server 8080 --directory /tmp/export/
# Open http://localhost:8080, verify widgets and SQL console work
```

### Phase 5: End-to-end testing (2 days)

**Goal:** Verify the full workflow on a real PR range.

**Validation checklist:**
- [ ] `review export` completes without errors
- [ ] Export directory opens in browser (file:// or http-server)
- [ ] Review docs render with correct HTML
- [ ] Diff widgets show correct before/after code
- [ ] History widgets show correct commit timeline
- [ ] Impact widgets show correct caller/callee lists
- [ ] SQL console runs arbitrary queries against review.db
- [ ] Export can be zipped and moved to another machine
- [ ] Export works offline (disconnect network, refresh page)

## 9. Comparison with server-based approach

| Criterion | Server (`review serve`) | Standalone Export (`review export`) |
|---|---|---|
| **Runtime dependency** | Go binary + review.db | Browser only |
| **Portability** | Must keep server running | Zip and email |
| **Offline use** | No | Yes |
| **LLM queries** | Full SQL via sqlite3 CLI | Full SQL via sql.js console |
| **Initial load time** | Fast (incremental HTTP) | Slower (load WASM + JSON + DB) |
| **Memory usage** | Server RAM | Browser RAM |
| **Build complexity** | Simple (just index) | Complex (pre-compute + SPA build) |
| **Concurrent users** | Unlimited | Single user per file |
| **Update workflow** | Restart server | Re-export and redistribute |

**Recommendation:** Support both. `review serve` for development and team-shared reviews. `review export` for final artifacts, email distribution, and offline review.

## 10. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Precomputed.json too large for browser | High | Gzip, lazy-load per-widget, or paginate |
| sql.js 1MB download on slow networks | Medium | Lazy-load sql.js only when SQL console opened |
| TinyGo doesn't compile new history code | High | Test early, fallback to standard Go WASM |
| Review DB >100MB (large commit ranges) | Medium | Offer `--max-commits` flag, or server-only mode |
| Diff computation at export time is slow | Medium | Cache diffs in review.db during `review index` |

## 11. Open questions

1. **Should diffs be cached in the review DB during `review index`?** Currently the GCB-013 design computes diffs on demand in `review serve`. For `review export`, we could pre-compute and store diffs in a `commit_diffs` table during indexing, making export faster.
2. **What is the max practical commit range for export?** 10 commits? 50? We need to benchmark precomputed.json size.
3. **Should we support incremental export?** If the review DB updates, can we re-export only changed data?
4. **Can sql.js handle the review DB size?** sql.js loads the entire DB into WASM memory. For a 50MB DB, this requires 50MB of browser RAM.
5. **Should the Go WASM module use sql.js internally?** Instead of two query paths (Go WASM for UI, sql.js for LLM), could the Go WASM module call sql.js through JS interop? This would unify the query layer but add complexity.

## 12. References

### Key files in this repo

| File | Relevance |
|------|-----------|
| `cmd/wasm/main.go` | WASM entry point |
| `internal/wasm/search.go` | Browser-side search context |
| `internal/wasm/exports.go` | JS interop exports |
| `internal/wasm/generate_build.go` | TinyGo build pipeline |
| `internal/static/generate_build.go` | Pre-computation pipeline |
| `internal/static/search_index.go` | Inverted index builder |
| `internal/static/xref_index.go` | Xref pre-computer |
| `internal/static/doc_renderer.go` | Doc pre-renderer |
| `internal/history/diff.go` | Commit diff logic |
| `internal/history/bodydiff.go` | Body diff logic |
| `internal/history/store.go` | History store |
| `ui/src/api/wasmClient.ts` | Frontend WASM client |
| `ui/src/features/doc/DocSnippet.tsx` | Widget hydration |

### Related tickets

| Ticket | Topic |
|--------|-------|
| GCB-006 | Static WASM build (foundation for this work) |
| GCB-009 | Git-aware indexing (history DB schema) |
| GCB-010 | Embeddable widgets (what the export renders) |
| GCB-013 | Code review tool (main design, server-based) |

### External references

- sql.js: https://sql.js.org/ — SQLite compiled to WASM via Emscripten
- TinyGo WASM: https://tinygo.org/docs/guides/webassembly/
- Go WASM: https://github.com/golang/go/wiki/WebAssembly
- modernc.org/sqlite: Pure-Go SQLite implementation