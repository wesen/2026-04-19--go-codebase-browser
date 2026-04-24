---
Title: WASM + Static HTML Architecture Design
Ticket: GCB-006
Status: draft
Topics:
    - wasm
    - go
    - react-frontend
    - codebase-browser
    - static-build
    - documentation-tooling
DocType: design-doc
Intent: ""
Owners: []
RelatedFiles:
    - Path: Makefile
      Note: Build targets (generate
    - Path: cmd/codebase-browser/main.go
      Note: main entry point
    - Path: cmd/wasm/main.go
      Note: WASM entry point — registers JS exports and blocks forever
    - Path: internal/browser/index.go
      Note: Loaded struct
    - Path: internal/docs/pages.go
      Note: ListPages() — walks embedded pages FS. Used by static generator
    - Path: internal/docs/renderer.go
      Note: Render() — goldmark markdown + codebase-* directive resolution. Used by the static generator for pre-rendering
    - Path: internal/indexer/extractor.go
      Note: Extract() — Go AST extraction via go/packages. Feeds into index.json build
    - Path: internal/indexer/multi.go
      Note: Merge() — concatenates Go + TS index JSON. Produces index.json
    - Path: internal/indexer/types.go
      Note: Index
    - Path: internal/indexfs/generate_build.go
      Note: Runs codebase-browser index build → embed/index.json. This is the first step in the build pipeline
    - Path: internal/server/api_doc.go
      Note: handleDocList
    - Path: internal/server/api_index.go
      Note: handleSearch — runtime substring match over symbols → compiled to WASM. handleSymbol
    - Path: internal/server/api_source.go
      Note: handleSnippet
    - Path: internal/server/api_xref.go
      Note: handleXref
    - Path: internal/server/server.go
      Note: Handler() registers all HTTP routes — these are the endpoints being replaced by WASM and static files
    - Path: internal/sourcefs/generate_build.go
      Note: Mirrors source tree → embed/source/. Source files served as static assets in the static build
    - Path: internal/static/search_index.go
      Note: BuildSearchIndexFast() — inverted index for symbol name lookup
    - Path: internal/static/snippet_extractor.go
      Note: ExtractSnippets() — pre-extracts declaration/body/signature text
    - Path: internal/static/static_test.go
      Note: Unit tests for pre-computation logic
    - Path: internal/static/xref_index.go
      Note: BuildXrefIndex() — pre-computes usedBy/uses per symbol
    - Path: internal/wasm/exports.go
      Note: JS interop layer using syscall/js — registers functions on window.codebaseBrowser
    - Path: internal/wasm/search_test.go
      Note: Unit tests for WASM search logic
    - Path: internal/web/generate_build.go
      Note: Dagger Vite SPA build → embed/public/. This build produces the SPA shell that gets served alongside WASM
    - Path: ui/src/api/indexApi.ts
      Note: RTK-Query endpoint definitions (getIndex
    - Path: ui/src/api/sourceApi.ts
      Note: 'RTK-Query endpoints for source/snippet/ref/xref. baseQuery changes: source from HTTP'
    - Path: ui/src/app/App.tsx
      Note: |-
        BrowserRouter
        Switched from BrowserRouter to HashRouter for file:// compatibility
    - Path: ui/src/features/doc/DocSnippet.tsx
      Note: React hydration of codebase-* stubs in pre-rendered HTML. Uses useGetSymbolQuery
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# WASM + Static HTML Architecture: A Complete Design & Implementation Guide

**Ticket:** GCB-006  
**Status:** Draft  
**Last Updated:** 2026-04-23  
**Audience:** New engineers / interns starting on this ticket

---

## Executive Summary

This document describes how to transform `codebase-browser` from a Go-based HTTP server (`serve`) into a **zero-server static artifact** — a single HTML file that ships the entire codebase browser. The Go indexer and search logic are compiled to WebAssembly (WASM) and embedded in the page; doc pages are pre-rendered to static HTML at build time.

The result: **no backend, no server, no Docker required at runtime.** Anyone can open the artifact in a browser, over `file://` or via any static CDN, with full symbol search, cross-reference navigation, and doc-page rendering — all client-side.

---

## Table of Contents

1. [What the System Does Today](#1-what-the-system-does-today)
2. [Why the Backend Exists — And Why It Doesn't Have To](#2-why-the-backend-exists--and-why-it-doesnt-have-to)
3. [The Static WASM Architecture](#3-the-static-wasm-architecture)
4. [Build Pipeline: What Changes](#4-build-pipeline-what-changes)
5. [WASM Module Design](#5-wasm-module-design)
6. [Static HTML Pre-Rendering Design](#6-static-html-pre-rendering-design)
7. [JavaScript ↔ WASM Bridge (ABI)](#7-javascript--wasm-bridge-abi)
8. [React Frontend Changes](#8-react-frontend-changes)
9. [Index Schema Changes](#9-index-schema-changes)
10. [File Layout After Refactor](#10-file-layout-after-refactor)
11. [Implementation Phases](#11-implementation-phases)
12. [Testing Strategy](#12-testing-strategy)
13. [Risks, Open Questions, Alternatives](#13-risks-open-questions-alternatives)
14. [Key Files Reference](#14-key-files-reference)

---

## 1. What the System Does Today

`codebase-browser` is a documentation browser for Go + TypeScript codebases. It has two main commands:

### 1.1 `codebase-browser index build` (build-time, Go binary)

This command runs inside `go generate` and performs three steps:

1. **Go AST extraction** (`internal/indexer/extractor.go`): Uses `golang.org/x/tools/go/packages` to walk every Go package, extract top-level declarations (functions, types, methods, constants, vars), record their byte ranges in source files, and collect cross-references from function bodies by visiting every identifier node.

2. **TypeScript extraction** (`tools/ts-indexer/`): Runs as a Node.js process via Dagger (or local `pnpm` fallback). Uses the TypeScript Compiler API (`ts` module) to walk every `.ts` and `.tsx` file under `ui/`, extract declarations, type signatures, and cross-references. Output is written to `internal/indexfs/embed/index-ts.json`.

3. **Merge** (`internal/indexer/multi.go`): Reads both JSON files, concatenates packages/files/symbols/refs, detects duplicate IDs (errors rather than silently dropping), sorts every slice deterministically, and writes `internal/indexfs/embed/index.json`. This is the canonical build artifact.

Concurrently, `internal/sourcefs/generate.go` mirrors the repository source tree into `internal/sourcefs/embed/source/`, excluding build artifacts, caches, and non-source files.

### 1.2 `codebase-browser serve` (runtime, Go HTTP server)

The server is responsible for:

- **Serving the SPA** (React build output from `internal/web/embed/public/`) at `GET /`
- **`GET /api/index`** — Returns the raw `index.json` bytes (cached, one-time)
- **`GET /api/packages`** — Returns a lightweight summary of all packages (id, import path, file count, symbol count)
- **`GET /api/symbol/<id>`** — Returns full symbol metadata (doc, signature, children, range, language)
- **`GET /api/search?q=<q>&kind=<kind>`** — Substring-matches symbol names against a query; returns up to 200 hits
- **`GET /api/source?path=<path>`** — Returns raw bytes of a source file (whitelist-checked against `index.json` Files table)
- **`GET /api/snippet?sym=<id>&kind=<declaration|body|signature>`** — Slices byte ranges from source files using symbol range offsets
- **`GET /api/xref/<id>`** — Walks the `Refs` slice to compute `usedBy` (who calls this symbol) and `uses` (who this symbol calls); bounded to 200 entries
- **`GET /api/snippet-refs?sym=<id>`** — Returns ref entries whose byte ranges fall inside a symbol's declaration (for linkification)
- **`GET /api/source-refs?path=<path>`** — Returns all ref entries in a file with absolute offsets
- **`GET /api/file-xref?path=<path>`** — Aggregates xref data across every symbol in one file
- **`GET /api/doc`** — Lists all doc pages (slug + title)
- **`GET /api/doc/<slug>`** — Renders a markdown page, resolving `codebase-snippet`, `codebase-signature`, `codebase-doc`, and `codebase-file` directives against the index

The server reads from three embedded filesystems (via `go:embed`):
- `internal/indexfs/embed/index.json`
- `internal/sourcefs/embed/source/` (mirrored source tree)
- `internal/web/embed/public/` (React SPA)

### 1.3 The React SPA (`ui/`)

The frontend is a React application using RTK-Query. It talks to the backend via these endpoints:

```typescript
// ui/src/api/indexApi.ts
GET /api/index          → IndexSummary (module name, package count, symbol count, packages list)
GET /api/packages       → PackageLite[] (id, importPath, name, fileCount, symbolCount)
GET /api/symbol/:id     → Symbol (full metadata)
GET /api/search?q=...  → Symbol[]

// ui/src/api/sourceApi.ts
GET /api/source?path=...          → raw text (as string)
GET /api/snippet?sym=...&kind=...  → raw text (declaration/body/signature)
GET /api/snippet-refs?sym=...     → SnippetRefView[]
GET /api/source-refs?path=...     → SourceRefView[]
GET /api/file-xref?path=...       → FileXrefResponse

// ui/src/api/docApi.ts (implicit via doc page component)
GET /api/doc            → PageMeta[]
GET /api/doc/:slug      → Page (slug, title, HTML, snippets[], errors[])
```

The RTK-Query endpoints use `fetchBaseQuery({ baseUrl: '/api' })`, so they only work when the Go server is running. The `BrowserRouter` in `App.tsx` handles client-side routing; any unknown path falls through to `index.html` (the SPA handler).

---

## 2. Why the Backend Exists — And Why It Doesn't Have To

### 2.1 What each API endpoint actually does

| Endpoint | What it computes | Can be pre-computed? |
|---|---|---|
| `/api/index` | Reads raw JSON file | ✅ Already static — just embed in HTML as `<script type="application/json">` |
| `/api/packages` | Derived from `index.json` Packages slice | ✅ Pre-compute at build time |
| `/api/symbol/:id` | Lookup in `index.json` Symbols slice | ✅ Pre-compute: store full symbol JSON in WASM memory |
| `/api/search` | Substring-match symbol names | ✅ Pre-compute: build a search index at build time, ship with WASM |
| `/api/source` | Read file from embedded `source/` tree | ✅ Already static — store alongside HTML |
| `/api/snippet` | Byte-range slice of a file | ✅ Pre-compute: extract and store actual snippet text in index |
| `/api/xref` | Graph walk over `Refs` slice | ✅ Pre-compute: bake `usedBy`/`uses` per symbol at build time |
| `/api/snippet-refs` | Filter Refs inside symbol range | ✅ Pre-compute: store per-symbol ref list in index |
| `/api/source-refs` | Filter Refs by file | ✅ Pre-compute: store per-file ref list in index |
| `/api/file-xref` | Aggregate xref for file | ✅ Pre-compute: store per-file xref summary |
| `/api/doc` | List pages from embedded filesystem | ✅ Pre-compute: generate a manifest JSON at build time |
| `/api/doc/:slug` | Render markdown + resolve directives | ⚠️ Partial: pre-render HTML, embed snippet text statically |

### 2.2 The runtime computation isn't fundamentally necessary

Every piece of runtime logic operates on data that is **already known at build time**:

- **Search**: The index already knows every symbol name. A simple substring-match over 10,000 symbols takes <1ms in Go. The same logic compiled to WASM runs in the browser without any network round-trip.

- **Snippet extraction**: The index already knows byte offsets (`Range.StartOffset`, `Range.EndOffset`) and the source file paths. At build time, we can read those bytes and store the extracted text in the index JSON. No need to slice at runtime.

- **Cross-references**: The `Refs` slice is fully known at build time. At build time, we can pre-compute for each symbol its `usedBy` and `uses` lists. Store them as arrays on each symbol or in a separate `xref-index.json`. At runtime, just look up the array — no graph walk needed.

- **Doc rendering**: The only genuinely dynamic operation is markdown rendering with directive resolution. But the directives resolve against static data (the index + source files). At build time, we can render every doc page to HTML, embedding the resolved snippet text directly. The result is a static HTML string that needs no further processing.

### 2.3 What genuinely requires runtime logic

- **Search** — substring matching over symbol names. This is the primary motivation for WASM.
- **Symbol lookup by ID** — fast map lookup over symbol records.
- **Cross-reference navigation** — reading pre-computed ref lists.

Everything else can be pre-computed and stored as JSON.

### 2.4 What the backend provides that a static build doesn't

| Capability | Backend | Static WASM |
|---|---|---|
| Source file serving | Runtime read + `safePath` check | Pre-embed files alongside HTML |
| Snippet byte-range slicing | Runtime slice | Pre-extract text into index |
| Cross-reference graph walk | Runtime filter over `Refs` | Pre-compute per-symbol ref lists |
| Doc page rendering | Runtime goldmark + directive resolution | Pre-render to HTML with embedded text |
| Symbol search | Runtime substring match | Build-time search index in WASM |

---

## 3. The Static WASM Architecture

### 3.1 High-level diagram

```
BUILD TIME (go generate)
┌─────────────────────────────────────────────────────────┐
│  1. Go AST extraction  →  index-go.json                  │
│  2. TS extraction      →  index-ts.json  (via Dagger)   │
│  3. Merge              →  index.json                    │
│  4. Pre-compute:                                        │
│     - Search index (inverted index: name → symbol IDs)   │
│     - Xref index (per-symbol usedBy/uses lists)         │
│     - Pre-rendered doc pages (HTML strings)             │
│     - Pre-extracted snippets (text, not offsets)         │
│  5. Compile search/wasm package to WASM (TinyGo or Go)   │
│  6. Build SPA (Vite) → static/                          │
│  7. Bundle: index.json + wasm + static/ → output/       │
└─────────────────────────────────────────────────────────┘

RUNTIME (browser, no server)
┌──────────────────────────────────────────────────────────┐
│  index.json (fetched once or embedded as JSON)           │
│                                                         │
│  ┌─────────────────┐     ┌───────────────────────────┐ │
│  │  search.wasm     │ ←── │  React SPA (index.html)   │ │
│  │  (Go compiled)   │     │  calls WASM functions      │ │
│  │                  │     │  via WASM JS interop      │ │
│  │  - FindSymbols() │     └───────────────────────────┘ │
│  │  - GetSymbol()   │                                    │
│  │  - GetXref()     │     ┌───────────────────────────┐ │
│  │  - GetSnippet()  │     │  index.json (searchable)  │ │
│  │  - SearchIndex   │     │  pre-computed xref data   │ │
│  └─────────────────┘     │  pre-rendered doc HTML    │ │
│                           │  pre-extracted snippets    │ │
│                           └───────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### 3.2 Build artifact structure

The final output is a directory containing:

```
output/
├── index.html          # Single HTML file with embedded CSS/JS (or SPA shell)
├── index.json          # Full index with pre-computed data (or embedded in HTML)
├── search.wasm         # Go WASM module for search + lookups
├── search.js           # Generated JS glue (WASM loader + exported fns)
├── source/             # Mirrored source tree (or embedded as base64 or zip)
│   ├── internal/
│   │   ├── browser/index.go
│   │   └── indexer/extractor.go
│   └── ui/src/...
├── docs/               # Pre-rendered doc HTML files (optional)
│   ├── 01-introduction.html
│   └── 02-architecture.html
└── manifest.json       # Index of packages, files, symbols, doc pages
```

The user opens `index.html` and everything works offline with no server.

### 3.3 Why WASM instead of pure JS?

**Go's `FindSymbols` is already written.** The `browser.Loaded.FindSymbols` method (substring-match over symbol names) is a ~30-line Go function. Rewriting it in JavaScript means duplicating logic, introducing inconsistencies, and losing the Go type safety guarantees.

Compiling the existing Go code to WASM means:
- Zero duplication: the search logic is the same code, just compiled.
- Type safety: the same `Loaded` struct with the same lookup maps.
- Future-proof: any Go improvements to the indexer automatically benefit the WASM build.
- Cross-language consistency: Go and TypeScript extractors both feed into the same index format.

The WASM module's job is specifically to expose the `browser.Loaded` lookup operations to JavaScript. It does not need DOM access, does not need file system access (the index is pre-loaded into WASM linear memory), and does not need concurrency.

---

## 4. Build Pipeline: What Changes

### 4.1 Current `go generate` flow

```bash
# Makefile
generate:
    go generate ./cmd/... ./internal/browser ./internal/docs \
                ./internal/indexer ./internal/indexfs \
                ./internal/server ./internal/sourcefs ./internal/web
```

Each `go generate` directive invokes a `generate_build.go` file:

| Package | What it runs | Output |
|---|---|---|
| `internal/indexfs` | `go run ./cmd/codebase-browser index build` | `embed/index.json` |
| `internal/sourcefs` | `go run generate_build.go` (copies source tree) | `embed/source/` |
| `internal/web` | Vite build via Dagger (or local pnpm) | `embed/public/` |
| `internal/docs` | (no generator; pages are embedded directly) | — |

### 4.2 New `go generate` flow

Add a new generator package `internal/wasm/generate.go` that runs **after** `internal/indexfs` (so `index.json` exists):

```bash
# New generate order (in Makefile)
generate:
    go generate ./internal/indexfs          # produces index.json
    go generate ./internal/sourcefs          # produces source/
    go generate ./internal/web              # produces SPA (only needed for dev)
    go generate ./internal/wasm             # NEW: produces search.wasm + search.js
    go generate ./internal/static          # NEW: pre-render docs, extract snippets
```

### 4.3 New generator: `internal/wasm/generate_build.go`

This generator:

1. Reads `index.json` (the freshly-written output from `internal/indexfs`).
2. Compiles the `internal/wasm/` package to WASM using TinyGo or standard Go WASM compiler.
3. Writes `internal/wasm/embed/search.wasm` (picked up by `go:embed`).
4. Generates `search_exports.go` (the JS glue that loads the WASM and exports typed functions).

### 4.4 New generator: `internal/static/generate_build.go`

This generator:

1. Reads `index.json` and the embedded `source/` tree.
2. Pre-computes search index data (inverted index: lowercase name → symbol IDs).
3. Pre-computes xref data per symbol (usedBy/uses arrays, bounded).
4. Pre-renders all doc pages (HTML with embedded snippet text).
5. Pre-extracts snippets (stores actual text, not offsets).
6. Writes everything to `internal/static/embed/` as JSON files.

### 4.5 Build command (TinyGo)

```bash
# Build WASM
tinygo build -target wasm -o internal/wasm/embed/search.wasm \
    -ldflags "-s -w" \
    ./internal/wasm
```

Or with standard Go:

```bash
# Build WASM (requires WASI SDK or GOOS=js GOARCH=wasm)
GOOS=js GOARCH=wasm go build -o internal/wasm/embed/search.wasm \
    -ldflags "-s -w" \
    ./internal/wasm
```

The `GOOS=js GOARCH=wasm` approach uses the standard Go toolchain. TinyGo produces smaller binaries (good for browser loading) but may have some stdlib restrictions.

---

## 5. WASM Module Design

### 5.1 Package layout

```
internal/
├── wasm/
│   ├── generate_build.go       # go:generate runs this
│   ├── generate_exports.go    # generated by generate_build.go
│   ├── search.go               # Go code that gets compiled to WASM
│   ├── search_test.go
│   ├── embed/
│   │   └── .gitkeep
│   └── exports.go              # exports.go: exported WASM functions
```

### 5.2 The WASM-compatible Go code

The core search logic lives in `search.go`. It must be written with WASM constraints in mind:

- **No file system access**: all data is passed in as byte slices.
- **No networking**: WASM has no socket APIs in the browser.
- **No goroutines**: standard Go WASM is single-threaded; use TinyGo for coroutines or compile with `-tags wasm` for threading.
- **Linear memory**: strings are stored in WASM linear memory; the JS glue code must copy data between JS strings and WASM memory using `memory.buffer`.

#### 5.2.1 `search.go`

```go
// Package wasm contains the browser-side search and lookup logic.
// It is compiled to a WASM binary and called from JavaScript.
//
// All exported functions are invoked from JS via the WASM JS interop.
// No goroutines (single-threaded execution).
package wasm

import (
    "encoding/json"
)

// SearchCtx holds the deserialised index + search data structures.
// It is allocated in WASM linear memory and passed around by pointer.
// The JS glue code reads/writes via DataView on memory.buffer.
type SearchCtx struct {
    // Full index (pre-computed at build time)
    Index *Index

    // Pre-computed search index: lowercase name prefix → symbol IDs
    // This is an inverted index built at build time.
    // Format: map[string][]string (JSON-encoded)
    SearchIndex map[string][]string

    // Pre-computed xref data per symbol
    // Format: map[symbolID]XrefData (JSON-encoded)
    XrefIndex map[string]*XrefData

    // Pre-extracted snippets: symbolID → snippet text
    // Format: map[symbolID]string
    SnippetCache map[string]string
}

type XrefData struct {
    UsedBy []RefSummary `json:"usedBy"`
    Uses   []UseTarget  `json:"uses"`
}

type RefSummary struct {
    FromSymbolID string `json:"fromSymbolId"`
    Kind         string `json:"kind"`
    StartLine    int    `json:"startLine"`
    EndLine      int    `json:"endLine"`
}

type UseTarget struct {
    ToSymbolID string `json:"toSymbolId"`
    Kind       string `json:"kind"`
    Count      int    `json:"count"`
}

// Init initialises SearchCtx from pre-loaded JSON data.
// Called once on page load.
func Init(jsonIndex, jsonSearchIdx, jsonXrefIdx, jsonSnippets []byte) (*SearchCtx, error) {
    var idx Index
    if err := json.Unmarshal(jsonIndex, &idx); return err != nil {
        return nil, err
    }
    var searchIdx map[string][]string
    if err := json.Unmarshal(jsonSearchIdx, &searchIdx); err != nil {
        return nil, err
    }
    var xrefIdx map[string]*XrefData
    if err := json.Unmarshal(jsonXrefIdx, &xrefIdx); err != nil {
        return nil, err
    }
    var snippets map[string]string
    if err := json.Unmarshal(jsonSnippets, &snippets); err != nil {
        return nil, err
    }
    return &SearchCtx{
        Index:        &idx,
        SearchIndex:  searchIdx,
        XrefIndex:    xrefIdx,
        SnippetCache: snippets,
    }, nil
}

// FindSymbols performs a substring-match search over symbol names.
// nameQuery is lowercase; if empty, matches everything.
// kind filters by symbol kind (e.g. "func", "type"); empty means all.
// Returns JSON-encoded []*Symbol, capped at 200.
func (s *SearchCtx) FindSymbols(nameQuery, kind string) []byte {
    out := []*Symbol{}
    for i := range s.Index.Symbols {
        sym := &s.Index.Symbols[i]
        if kind != "" && sym.Kind != kind {
            continue
        }
        if nameQuery == "" || containsIgnoreCase(sym.Name, nameQuery) {
            out = append(out, sym)
        }
        if len(out) >= 200 {
            break
        }
    }
    data, _ := json.Marshal(out)
    return data
}

// GetSymbol returns the symbol with the given ID, or nil JSON if not found.
func (s *SearchCtx) GetSymbol(id string) []byte {
    for i := range s.Index.Symbols {
        if s.Index.Symbols[i].ID == id {
            data, _ := json.Marshal(&s.Index.Symbols[i])
            return data
        }
    }
    return []byte("null")
}

// GetXref returns pre-computed cross-reference data for a symbol.
func (s *SearchCtx) GetXref(id string) []byte {
    data, _ := json.Marshal(s.XrefIndex[id])
    return data
}

// GetSnippet returns the pre-extracted snippet text for a symbol.
func (s *SearchCtx) GetSnippet(id, kind string) []byte {
    key := id + ":" + kind
    text, ok := s.SnippetCache[key]
    if !ok {
        // Fallback: try declaration as default
        text, ok = s.SnippetCache[id]
    }
    data, _ := json.Marshal(map[string]string{"text": text})
    return data
}

// GetPackages returns a lightweight package summary.
func (s *SearchCtx) GetPackages() []byte {
    type lite struct {
        ID         string `json:"id"`
        ImportPath string `json:"importPath"`
        Name       string `json:"name"`
        Files      int    `json:"files"`
        Symbols    int    `json:"symbols"`
    }
    out := make([]lite, 0, len(s.Index.Packages))
    for _, p := range s.Index.Packages {
        out = append(out, lite{
            ID:         p.ID,
            ImportPath: p.ImportPath,
            Name:       p.Name,
            Files:      len(p.FileIDs),
            Symbols:    len(p.SymbolIDs),
        })
    }
    data, _ := json.Marshal(out)
    return data
}

// GetDocPages returns the doc page manifest.
func (s *SearchCtx) GetDocPages() []byte {
    data, _ := json.Marshal(s.DocPages)
    return data
}

// GetDocPage returns the pre-rendered HTML for a doc page slug.
func (s *SearchCtx) GetDocPage(slug string) []byte {
    for _, p := range s.DocPages {
        if p.Slug == slug {
            // Load pre-rendered HTML from embedded store
            data, _ := json.Marshal(map[string]interface{}{
                "slug":  p.Slug,
                "title": p.Title,
                "html":  s.DocHTML[slug],
            })
            return data
        }
    }
    return []byte("null")
}

func containsIgnoreCase(s, substr string) bool {
    return contains(strings.ToLower(s), strings.ToLower(substr))
}

func contains(s, substr string) bool {
    return len(substr) == 0 || (len(s) >= len(substr) && findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}
```

#### 5.2.2 `exports.go` (WASM exports)

WASM functions that JS will call. Each function is registered with `wasm.NewFunc`.

```go
//go:build wasm

package wasm

import (
    "syscall/js"
)

// RegisterExports registers all exported WASM functions on the global js.Value.
// Called from main(), which is stubbed out in export-only builds.
func RegisterExports(ctx *SearchCtx) {
    js.Global().Set("codebaseBrowser", js.ValueOf(map[string]interface{}{
        "findSymbols":   js.FuncOf(makeFindSymbols(ctx)),
        "getSymbol":    js.FuncOf(makeGetSymbol(ctx)),
        "getXref":      js.FuncOf(makeGetXref(ctx)),
        "getSnippet":   js.FuncOf(makeGetSnippet(ctx)),
        "getPackages":   js.FuncOf(makeGetPackages(ctx)),
        "getDocPages":   js.FuncOf(makeGetDocPages(ctx)),
        "getDocPage":   js.FuncOf(makeGetDocPage(ctx)),
        "init":         js.FuncOf(makeInit(ctx)),
    }))
}
```

Note: The `go:build wasm` tag is set by the TinyGo/TinyGo compiler. Standard Go uses `GOOS=js GOARCH=wasm`. Use a build tag in `generate_build.go` to pass the right flags.

#### 5.2.3 `main.go` (WASM entry point)

```go
//go:build wasm

package main

import "github.com/wesen/codebase-browser/internal/wasm"

func main() {
    ctx := &wasm.SearchCtx{}
    wasm.RegisterExports(ctx)
    // Keep the WASM alive (prevent GC of exported funcs)
    select {}
}
```

### 5.3 Build-time data pre-computation (`internal/static/generate_build.go`)

Before the WASM is built, the `internal/static` generator produces the data files that the WASM loads at runtime:

```go
//go:build ignore

package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/wesen/codebase-browser/internal/indexer"
    "github.com/wesen/codebase-browser/internal/browser"
    "github.com/wesen/codebase-browser/internal/docs"
)

type PrecomputedIndex struct {
    SearchIndex map[string][]string    `json:"searchIndex"`   // name → symbol IDs
    XrefIndex   map[string]*XrefData   `json:"xrefIndex"`     // symbol ID → xref
    Snippets    map[string]string     `json:"snippets"`      // symID:kind → text
    DocManifest []PageMeta            `json:"docManifest"`   // pages list
    DocHTML     map[string]string     `json:"docHTML"`       // slug → HTML
}

func main() {
    root, _ := findRepoRoot()
    
    // Load index
    idx, err := browser.LoadFromFile(filepath.Join(root, "internal/indexfs/embed/index.json"))
    if err != nil { log.Fatal(err) }
    
    // Build search index (inverted index by name prefixes)
    searchIdx := buildSearchIndex(idx)
    
    // Build xref index (pre-compute usedBy/uses per symbol)
    xrefIdx := buildXrefIndex(idx)
    
    // Extract snippets (pre-compute actual text)
    snippets := extractSnippets(idx, filepath.Join(root, "internal/sourcefs/embed/source"))
    
    // Pre-render doc pages
    docManifest, docHTML := renderDocPages(idx)
    
    out := PrecomputedIndex{
        SearchIndex: searchIdx,
        XrefIndex:   xrefIdx,
        Snippets:    snippets,
        DocManifest: docManifest,
        DocHTML:     docHTML,
    }
    
    data, _ := json.MarshalIndent(out, "", "  ")
    outPath := filepath.Join(root, "internal/static/embed/precomputed.json")
    os.WriteFile(outPath, data, 0644)
    fmt.Println("generated:", outPath)
}
```

**Building the search index**: A naive substring match over all symbols is O(N). For a 10,000-symbol codebase, this is fine. For a 100,000-symbol codebase, build a prefix trie or inverted index at build time.

**Building the xref index**: Walk the `Refs` slice once, group by `FromSymbolID` (for `uses`) and `ToSymbolID` (for `usedBy`). Store as JSON maps.

**Extracting snippets**: For each symbol, read the source file at `Range.StartOffset:Range.EndOffset` and store the text. Also store `kind=body` and `kind=signature` variants.

**Pre-rendering doc pages**: Call `docs.Render()` for each page in the pages FS. Store the resulting `Page.HTML` string. The `Page.Snippets` are already embedded as text in the HTML via `stubHTML()`.

---

## 6. Static HTML Pre-Rendering Design

### 6.1 Doc page pre-rendering

Currently, `GET /api/doc/<slug>` runs `docs.Render()` at request time:

```go
// internal/server/api_doc.go (current)
func (s *Server) handleDocPage(w http.ResponseWriter, r *http.Request) {
    slug := strings.TrimPrefix(r.URL.Path, "/api/doc/")
    // ...
    data, err := fs.ReadFile(docs.PagesFS(), path)
    page, err := docs.Render(slug, data, s.Loaded, s.SourceFS)
    writeJSON(w, page)  // { slug, title, HTML, snippets[], errors[] }
}
```

In the static WASM build, `docs.Render()` runs at build time instead:

```go
// internal/static/doc_renderer.go
type DocRenderer struct {
    Loaded    *browser.Loaded
    SourceFS  fs.FS
    PagesFS   fs.FS
}

func (r *DocRenderer) RenderAll() (manifest []PageMeta, html map[string]string, err error) {
    pages, err := docs.ListPages(r.PagesFS)
    for _, page := range pages {
        data, err := fs.ReadFile(r.PagesFS, page.Path)
        if err != nil {
            err = fmt.Errorf("read %s: %w", page.Path, err)
            return
        }
        rendered, err := docs.Render(page.Slug, data, r.Loaded, r.SourceFS)
        if err != nil {
            err = fmt.Errorf("render %s: %w", page.Slug, err)
            return
        }
        manifest = append(manifest, PageMeta{Slug: page.Slug, Title: rendered.Title})
        html[page.Slug] = rendered.HTML
    }
    return
}
```

The pre-rendered HTML contains **no directives** — everything is resolved. Snippet text is inlined via `stubHTML()` producing static `<div>` elements with the actual source code.

### 6.2 Page structure (before and after)

**Before (runtime rendering):**
```markdown
```codebase-snippet sym=github.com/.../indexer.Merge
```
```
Rendered at request-time: server reads index.json, reads source file, slices bytes, produces HTML
```

**After (pre-rendered):**
```html
<div class="codebase-snippet" data-sym="github.com/.../indexer.Merge" data-directive="codebase-snippet">
  <pre><code class="language-go">// Merge concatenates ...
  func Merge(parts []*Index) (*Index, error) {
      ...
  }</code></pre>
</div>
```

The pre-rendered HTML contains the actual code, pre-highlighted by goldmark (which produces static HTML for the code blocks). The frontend's `DocSnippet` component can still hydrate these stubs for interactivity, but the base content is already there.

---

## 7. JavaScript ↔ WASM Bridge (ABI)

### 7.1 Loading the WASM module

The WASM module is loaded once on page initialization. The generated `search.js` (produced by `generate_build.go`) handles this:

```javascript
// internal/wasm/embed/search.js (generated)
// This file is embedded in the HTML or loaded as a script module.

let wasmModule = null;
let wasmMemory = null;

export async function initSearchEngine(indexJson) {
    // Load the WASM binary
    const response = await fetch('/search.wasm');
    const buffer = await response.arrayBuffer();
    
    const imports = {
        env: {
            // Required by Go WASM runtime: memory, imports
            memory: new WebAssembly.Memory({ initial: 256 }),
            'runtime.ticks': () => 0,
            'runtime.sleep': () => {},
        }
    };
    
    const result = await WebAssembly.instantiate(buffer, imports);
    wasmModule = result.module;
    wasmMemory = result.instance.exports.mem;
    
    // Call WASM init function
    const init = result.instance.exports.init;
    if (init) {
        init();
    }
    
    // Load the pre-computed index data into WASM context
    const loadData = result.instance.exports.loadData;
    if (loadData) {
        // Pass index JSON as a Go []byte (pointer + length)
        const encoder = new TextEncoder();
        const jsonBytes = encoder.encode(JSON.stringify(indexJson));
        const ptr = allocate(jsonBytes.length);
        new Uint8Array(wasmMemory.buffer, ptr, jsonBytes.length).set(jsonBytes);
        loadData(ptr, jsonBytes.length);
    }
    
    return result.instance.exports;
}

function allocate(size) {
    // Use wasmMemory to allocate space in WASM linear memory
    // In Go WASM, use runtime.malloc or similar
    return 0; // placeholder
}

export function findSymbols(query, kind = '') {
    const resultPtr = wasmModule.exports.findSymbols(query, kind);
    return readStringFromWasm(resultPtr);
}

export function getSymbol(id) {
    const resultPtr = wasmModule.exports.getSymbol(id);
    return readStringFromWasm(resultPtr);
}

function readStringFromWasm(ptr) {
    // Read null-terminated string from WASM memory
    const view = new Uint8Array(wasmMemory.buffer, ptr);
    let len = 0;
    while (view[len] !== 0) len++;
    const decoder = new TextDecoder();
    return decoder.decode(view.slice(0, len));
}
```

### 7.2 Alternative: Use `wasm-export` helper library

A simpler approach: instead of hand-rolling the JS glue, use `wa-socket` or similar libraries that handle string passing automatically. Or use TinyGo's built-in WASM JS interop which uses `js.Value` for direct JS object access.

**TinyGo approach** (recommended for simplicity):

```go
// With TinyGo, js.Value provides direct JS interop
// Strings automatically marshal through WASM memory

import "syscall/js"

func makeFindSymbols(ctx *SearchCtx) js.Func {
    return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        query := args[0].String()
        kind := args[1].String()
        results := ctx.FindSymbols(query, kind)
        return js.ValueOf(string(results))  // returns JS string
    })
}
```

This is simpler because TinyGo handles the string conversion automatically via `js.Value.String()` and `js.ValueOf(string)`.

### 7.3 WASM memory management

In Go-compiled WASM, strings are stored in linear memory as `(ptr uint32, len uint32)` pairs. The JS code reads them by accessing `memory.buffer` at the given offset.

**Strategies:**

1. **Return JSON strings** — WASM functions return pointers to JSON bytes in linear memory. JS reads via `memory.buffer`. Simple, works for all results under ~1MB.

2. **Streaming for large results** — For very large symbol lists, use a streaming approach (not needed for typical codebases with <50k symbols).

3. **Zero-copy** — For source file serving, the WASM can expose `getSourcePtr(path)` returning a pointer+length to the file bytes in WASM memory. JS reads via `Uint8Array` view.

---

## 8. React Frontend Changes

### 8.1 Current RTK-Query architecture

```typescript
// Current: all endpoints are HTTP calls to /api/*
export const indexApi = createApi({
  reducerPath: 'indexApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api' }),
  endpoints: (b) => ({
    getIndex: b.query<IndexSummary, void>({ query: () => '/index' }),
    getSymbol: b.query<Symbol, string>({ query: (id) => `/symbol/${id}` }),
    searchSymbols: b.query<Symbol[], { q: string; kind?: string }>({ ... }),
  }),
});
```

### 8.2 New architecture: WASM-backed RTK-Query

The RTK-Query endpoints are replaced with custom `baseQuery` that calls the WASM module instead of HTTP:

```typescript
// ui/src/api/wasmClient.ts

let wasmReady = false;
let wasmExports: typeof import('../../internal/wasm/embed/search') | null = null;

export async function initWasm() {
    if (wasmReady) return;
    const { initSearchEngine } = await import('../../internal/wasm/embed/search');
    // Load pre-computed index from embedded JSON
    const indexRes = await fetch('/precomputed.json');
    const indexJson = await indexRes.json();
    wasmExports = await initSearchEngine(indexJson);
    wasmReady = true;
}

// Base query that calls WASM instead of HTTP
const wasmBaseQuery: BaseQueryFn<string, unknown, unknown, {}> = async (arg) => {
    await initWasm();
    if (!wasmExports) throw new Error('WASM not initialized');
    
    // Parse the endpoint from arg (replaces URL routing)
    const [endpoint, param] = arg.split(':');
    
    switch (endpoint) {
        case 'index': return { data: wasmExports.getIndex() };
        case 'symbol': return { data: JSON.parse(wasmExports.getSymbol(param)) };
        case 'search': {
            const [q, kind] = param.split('|');
            return { data: JSON.parse(wasmExports.findSymbols(q, kind)) };
        }
        case 'xref': return { data: JSON.parse(wasmExports.getXref(param)) };
        case 'snippet': {
            const [sym, kind] = param.split('|');
            return { data: JSON.parse(wasmExports.getSnippet(sym, kind)) };
        }
        case 'packages': return { data: JSON.parse(wasmExports.getPackages()) };
        case 'docList': return { data: wasmExports.getDocPages() };
        case 'docPage': return { data: JSON.parse(wasmExports.getDocPage(param)) };
        default: return { error: 'Unknown endpoint: ' + endpoint };
    }
};

// Re-export the same RTK-Query API (interface unchanged)
export const indexApi = createApi({
    reducerPath: 'indexApi',
    baseQuery: wasmBaseQuery,
    endpoints: (b) => ({
        getIndex: b.query<IndexSummary, void>({ queryFn: async () => ({ data: wasmExports!.getIndex() }) }),
        getSymbol: b.query<Symbol, string>({ queryFn: async (id) => ({ data: JSON.parse(wasmExports!.getSymbol(id)) }) }),
        searchSymbols: b.query<Symbol[], { q: string; kind?: string }>({
            queryFn: async ({ q, kind }) => ({ data: JSON.parse(wasmExports!.findSymbols(q, kind)) }),
        }),
        // ... all other endpoints
    }),
});
```

The **key insight**: the RTK-Query API surface (endpoints, types, React hooks) stays identical. Only the `baseQuery` implementation changes — from HTTP fetch to WASM function calls. This minimises changes to the React components.

### 8.3 Source file serving (static)

Source files are pre-embedded as static assets. In the HTML build, `internal/sourcefs/embed/source/` is copied into the output `source/` directory. The frontend's `SourcePage` component fetches source files via a regular HTTP request:

```typescript
// Source files are static; no WASM needed
export const sourceApi = createApi({
    reducerPath: 'sourceApi',
    baseQuery: fetchBaseQuery({ baseUrl: '/source' }),  // static files
    endpoints: (b) => ({
        getSource: b.query<string, string>({ query: (path) => path }),
        getFileXref: b.query<FileXrefResponse, string>({ query: (path) => `/api/file-xref?path=${path}` }),
        // ...
    }),
});
```

Wait — `getFileXref` still calls an API endpoint. The xref data should also be pre-computed and embedded as JSON, so the frontend can fetch `xref-index.json` statically:

```typescript
// Pre-computed xref data: symbolID → xref JSON
const xrefCache = await fetch('/xref-index.json').then(r => r.json());

// getFileXref looks up in the cache
function getFileXref(path: string): FileXrefResponse {
    const fileId = 'file:' + path;
    return xrefCache[fileId] ?? { path, usedBy: [], uses: [] };
}
```

Similarly, `getSnippet` and `getSnippetRefs` use pre-computed data.

### 8.4 Handling the initial page load

On first load, the SPA needs to:
1. Fetch `precomputed.json` (search index, xref data, snippet cache, doc HTML)
2. Initialize the WASM module with the index data
3. Render the UI

This is handled by the WASM init function above.

---

## 9. Index Schema Changes

### 9.1 Current `index.json` structure

```json
{
  "version": "1",
  "generatedAt": "2026-04-23T...",
  "module": "github.com/wesen/codebase-browser",
  "goVersion": "go1.22...",
  "packages": [{ "id", "importPath", "name", "doc", "fileIds", "symbolIds", "language" }],
  "files": [{ "id", "path", "packageId", "size", "lineCount", "buildTags", "sha256", "language" }],
  "symbols": [{ "id", "kind", "name", "packageId", "fileId", "range", "doc", "signature", "receiver", "typeParams", "exported", "children", "tags", "language" }],
  "refs": [{ "fromSymbolId", "toSymbolId", "kind", "fileId", "range" }]
}
```

### 9.2 New `precomputed.json` structure

The `index.json` stays as-is (no schema change needed). The new `precomputed.json` is an additional build artifact:

```json
{
  "version": "1",
  "searchIndex": {
    "merge": ["sym:github.com/.../indexer.Merge.func", "sym:github.com/.../Merge.go"],
    "extract": ["sym:github.com/.../indexer.Extract.func"],
    "loaded": ["sym:github.com/.../browser.Loaded.type"],
    // ... lowercase name → symbol IDs
    // For fast prefix search: also store "mer" → ["sym:..."], "mer*g" → ["sym:..."]
  },
  "xrefIndex": {
    "sym:github.com/.../indexer.Merge.func": {
      "usedBy": [
        { "fromSymbolId": "sym:...", "kind": "call", "startLine": 42, "endLine": 42 }
      ],
      "uses": [
        { "toSymbolId": "sym:...", "kind": "call", "count": 3, "occurrences": [...] }
      ]
    }
  },
  "snippets": {
    "sym:github.com/.../indexer.Merge.func:declaration": "func Merge(parts []*Index) (*Index, error) {...}",
    "sym:github.com/.../indexer.Merge.func:signature": "func Merge(parts []*Index) (*Index, error)",
    "sym:github.com/.../indexer.Merge.func:body": "{ out := &Index{Version: \"1\"} ... }",
    "sym:github.com/.../Symbol.type:declaration": "type Symbol struct {...}",
    "sym:github.com/.../Symbol.type:signature": "type Symbol struct"
  },
  "docManifest": [
    { "slug": "01-introduction", "title": "Introduction" },
    { "slug": "02-architecture", "title": "Architecture" }
  ],
  "docHTML": {
    "01-introduction": "<h1>Introduction</h1><p>...</p>",
    "02-architecture": "<h1>Architecture</h1>..."
  }
}
```

### 9.3 Xref index format (detailed)

The xref index groups refs by symbol. For each symbol, `usedBy` lists refs TO this symbol (callers), and `uses` lists refs FROM this symbol (callees):

```json
{
  "xrefIndex": {
    "sym:github.com/wesen/codebase-browser/internal/indexer.Merge.func": {
      "usedBy": [
        {
          "fromSymbolId": "sym:github.com/.../indexer.Extract.func",
          "kind": "call",
          "startLine": 28,
          "endLine": 28,
          "startCol": 12,
          "endCol": 17
        }
      ],
      "uses": [
        {
          "toSymbolId": "sym:github.com/.../sortIndex.func",
          "kind": "call",
          "count": 2,
          "occurrences": [
            { "startLine": 89, "endLine": 89 },
            { "startLine": 102, "endLine": 102 }
          ]
        }
      ]
    }
  }
}
```

Bounds: `usedBy` capped at 200 entries; `uses` deduplicated by `toSymbolId`, with up to 5 `occurrences` per target.

### 9.4 Search index format (detailed)

For efficient prefix search:

```json
{
  "searchIndex": {
    // Exact substring matches (lowercase key → symbol IDs)
    "merge": ["sym:.../Merge.func", "sym:.../mergeModuleName.func"],
    "mergei": ["sym:.../Merge.func"],  // case-insensitive root
    "m": ["sym:.../main.func", "sym:.../Merge.func", "sym:.../method..."],
    "me": ["sym:.../Merge.func", "sym:.../method..."],
    "mer": ["sym:.../Merge.func"],
    "merg": ["sym:.../Merge.func", "sym:.../mergeModuleName.func"],
    "mergi": ["sym:.../Merge.func"],
    "merge": ["sym:.../Merge.func", "sym:.../mergeModuleName.func"],
    
    // Multi-word: "find symbols" → split on space, intersection
    "find": ["sym:.../FindSymbols.func", "sym:.../findSubstring.func"],
    "symbols": ["sym:.../FindSymbols.func"],
    
    // Special entries: kind prefixes
    "func:merge": ["sym:.../Merge.func"],
    "type:loaded": ["sym:.../Loaded.type"]
  },
  // Symbol metadata (lightweight, for display)
  "symbolMeta": {
    "sym:github.com/.../Merge.func": {
      "kind": "func",
      "name": "Merge",
      "packageId": "github.com/.../indexer",
      "signature": "func Merge(parts []*Index) (*Index, error)",
      "doc": "Merge concatenates..."
    }
  }
}
```

**Simplified approach for MVP**: Just store full symbol list and do naive substring match at runtime in WASM. For codebases up to ~50k symbols, this is fast enough (sub-millisecond). Optimize with inverted index only if profiling shows search is slow.

---

## 10. File Layout After Refactor

```
codebase-browser/
├── cmd/
│   └── codebase-browser/
│       ├── main.go
│       ├── serve_stub.go              # Empty stub when STATIC_WASM build tag set
│       └── cmds/
│           ├── index/                 # index build command (keep)
│           ├── serve/                 # keep but mark DEPRECATED in docs
│           ├── doc/
│           └── symbol/
├── internal/
│   ├── browser/
│   │   ├── index.go                   # Loaded struct (reused in WASM)
│   │   └── index_test.go
│   ├── indexer/
│   │   ├── types.go                   # Index, Package, File, Symbol, Ref types
│   │   ├── extractor.go              # Go AST extraction
│   │   ├── multi.go                  # Merge function
│   │   ├── xref.go                   # Cross-reference extraction
│   │   ├── id.go                     # ID generation helpers
│   │   └── write.go                  # JSON write helpers
│   ├── docs/
│   │   ├── renderer.go               # docs.Render (used in static generator)
│   │   ├── pages.go                  # docs.ListPages
│   │   ├── embed_fs.go               # pages FS embed
│   │   └── embed_none_fs.go          # noembed stub
│   ├── sourcefs/
│   │   ├── embed.go                  # source tree embed
│   │   ├── embed_none.go             # noembed stub
│   │   └── generate_build.go         # copies source tree
│   ├── wasm/                         # NEW: WASM module
│   │   ├── generate_build.go         # builds WASM, generates exports
│   │   ├── generate_exports.go       # generated JS glue
│   │   ├── search.go                 # search logic (compiled to WASM)
│   │   ├── search_test.go
│   │   ├── exports.go                # WASM export registrations
│   │   ├── main.go                   # WASM entry point (stub main)
│   │   └── embed/
│   │       ├── search.wasm           # compiled WASM (go:embed)
│   │       └── search.js             # generated JS loader
│   ├── static/                       # NEW: static HTML generator
│   │   ├── generate_build.go         # pre-render docs, build precomputed.json
│   │   ├── doc_renderer.go           # doc pre-rendering logic
│   │   ├── search_index.go          # build-time search index builder
│   │   ├── xref_index.go            # build-time xref pre-computer
│   │   └── embed/
│   │       └── precomputed.json      # pre-computed data (go:embed)
│   ├── web/                          # React SPA (keep, but SPA is simplified)
│   │   ├── generate_build.go        # Vite build (only for dev)
│   │   └── embed/
│   │       └── public/               # SPA static output
│   ├── indexfs/                     # Keep as-is
│   │   ├── embed/
│   │   │   └── index.json           # Built by go generate
│   │   └── generate_build.go
│   └── server/                       # DEPRECATED (serve command)
│       ├── server.go
│       ├── api_index.go
│       ├── api_source.go
│       ├── api_doc.go
│       ├── api_xref.go
│       └── spa.go
├── ui/                               # React SPA
│   ├── src/
│   │   ├── app/App.tsx               # No change needed
│   │   ├── api/
│   │   │   ├── wasmClient.ts         # NEW: WASM base query
│   │   │   ├── indexApi.ts          # Modified: WASM-backed
│   │   │   ├── sourceApi.ts         # Modified: static source files
│   │   │   └── docApi.ts            # Modified: pre-rendered HTML
│   │   └── features/
│   │       └── ...
│   └── dist/                         # Vite build output
├── ttmp/
│   └── GCB-006/                     # This ticket
│       └── design-doc/01-wasm-static-html-architecture-design.md
├── Makefile                          # Updated: adds wasm + static build targets
└── go.mod
```

---

## 11. Implementation Phases

### Phase 1: WASM search engine (week 1–2)

**Goal:** Compile the existing `browser.Loaded.FindSymbols` to WASM, verify it works in a browser.

**Steps:**
1. Create `internal/wasm/` package with `search.go` (copy of `browser.Loaded` logic, adapted for WASM).
2. Write `exports.go` with TinyGo `syscall/js` exports.
3. Add `generate_build.go` that compiles to WASM with TinyGo.
4. Create `internal/wasm/embed/search.js` (WASM loader).
5. Build the WASM, open in browser, verify `findSymbols` works.
6. Write `internal/wasm/search_test.go` to test search logic.

**Deliverables:** `search.wasm` + `search.js` that returns symbol search results from the embedded index.

**Files created:**
- `internal/wasm/search.go`
- `internal/wasm/exports.go`
- `internal/wasm/main.go`
- `internal/wasm/generate_build.go`
- `internal/wasm/search_test.go`
- `internal/wasm/embed/search.js` (generated)

### Phase 2: Pre-computed data (week 2–3)

**Goal:** Build `precomputed.json` at build time with search index, xref data, and snippets.

**Steps:**
1. Create `internal/static/generate_build.go`.
2. Implement `buildSearchIndex()` — inverted index of symbol names → IDs.
3. Implement `buildXrefIndex()` — pre-compute usedBy/uses per symbol.
4. Implement `extractSnippets()` — pre-extract declaration/body/signature text.
5. Implement `renderDocPages()` — pre-render all doc pages to HTML.
6. Write `precomputed.json` to `internal/static/embed/`.
7. Update Makefile: add `go generate ./internal/static`.
8. Verify `precomputed.json` is correct by loading in tests.

**Deliverables:** `precomputed.json` containing search index, xref data, snippets, and doc HTML.

**Files created/modified:**
- `internal/static/generate_build.go` (new)
- `internal/static/doc_renderer.go` (new)
- `internal/static/search_index.go` (new)
- `internal/static/xref_index.go` (new)
- `Makefile` (updated)

### Phase 3: WASM integration (week 3–4)

**Goal:** Load `precomputed.json` into WASM at runtime, expose all query APIs.

**Steps:**
1. Update `search.go` to accept precomputed data (search index, xref, snippets).
2. Update `wasmBaseQuery` in `ui/src/api/wasmClient.ts` to call WASM functions.
3. Update RTK-Query endpoints to use `wasmBaseQuery` (all endpoints unchanged types, just baseQuery changed).
4. Test: verify search, symbol lookup, xref navigation all work in browser with no server.
5. Test: verify source file serving (static files via `<img>` or `fetch()` to `file://`).

**Deliverables:** Fully functional SPA with no server dependency.

**Files modified:**
- `internal/wasm/search.go` (updated)
- `ui/src/api/wasmClient.ts` (new)
- `ui/src/api/indexApi.ts` (updated)
- `ui/src/api/sourceApi.ts` (updated)
- `ui/src/api/docApi.ts` (updated)

### Phase 4: Static HTML output (week 4–5)

**Goal:** Generate a single `index.html` file that bundles everything (or a minimal set of files).

**Steps:**
1. Create `internal/bundle/generate_build.go` — generates the final artifact:
   - Copies WASM (`search.wasm`) and JS (`search.js`) to output.
   - Copies source files (`source/`) to output.
   - Copies precomputed data (`precomputed.json`) to output.
   - Generates `index.html` that loads everything.
2. Option A: inline all CSS/JS into `index.html` (single file, ~500KB–2MB depending on codebase).
   Option B: output a small set of files (`index.html`, `search.wasm`, `search.js`, `precomputed.json`, `source/`) for minimal deployment.
3. Update Makefile: add `make build-static` target.
4. Test: open the artifact directly with `file://` — verify everything works offline.

**Deliverables:** A `dist/` directory containing `index.html` + static assets. Openable with `file://`.

**Files created/modified:**
- `internal/bundle/generate_build.go` (new)
- `internal/bundle/templates/index.html` (template)
- `Makefile` (updated)

### Phase 5: Testing and polish (week 5–6)

**Steps:**
1. Write integration tests: build static artifact, open in Playwright, verify search and navigation.
2. Verify all existing tests still pass (`make test`).
3. Add migration notes: explain how to move from `codebase-browser serve` to the static artifact.
4. Update README.md with static build instructions.
5. Verify the artifact works on mobile browsers (Safari, Chrome Android).

---

## 12. Testing Strategy

### 12.1 Unit tests (Go)

```bash
# WASM search logic
go test ./internal/wasm/... -v

# Pre-computation correctness
go test ./internal/static/... -v

# Doc rendering (same as current tests)
go test ./internal/docs/... -v

# Indexer (same as current tests)
go test ./internal/indexer/... -v
```

### 12.2 Integration tests (Playwright)

```typescript
// e2e/static-wasm.test.ts
import { test, expect } from '@playwright/test';

test('static artifact loads without server', async ({ page }) => {
  // Open the static HTML directly (file:// protocol)
  await page.goto('file:///path/to/dist/index.html');
  
  // Verify WASM loaded
  await page.waitForFunction(() => (window as any).codebaseBrowser !== undefined);
  
  // Test search
  const results = await page.evaluate(() => {
    return (window as any).codebaseBrowser.findSymbols('Merge', '');
  });
  expect(results.length).toBeGreaterThan(0);
  
  // Test symbol navigation
  await page.click('text=Merge');
  await expect(page.locator('[data-part="symbol-header"]')).toBeVisible();
  
  // Test xref panel
  await page.click('text=cross-references');
  await expect(page.locator('[data-part="xref-used-by"]')).toBeVisible();
});
```

### 12.3 Smoke test (current, keep)

```bash
# Current smoke test still works (but for dev only now)
make smoke  # builds embedded binary, curls /api/index
```

For the static build:
```bash
make build-static
# Open dist/index.html in Playwright
# Verify all routes work
```

---

## 13. Risks, Open Questions, Alternatives

### 13.1 Risks

| Risk | Severity | Mitigation |
|---|---|---|
| WASM bundle size too large (>5MB) | Medium | Use TinyGo for smaller binaries; consider streaming WASM loading |
| Browser WASM support incomplete (Safari <15) | Low | WASM is widely supported; fallback to Web Worker if needed |
| Go WASM goroutine restrictions | Medium | Use TinyGo (supports coroutines) or standard Go with explicit single-threaded design |
| Pre-computing snippets increases build time | Low | Run in parallel with Dagger; incremental builds if only one file changed |
| Xref index JSON size (large codebases) | Medium | Compress JSON; use LZ-string in browser; or keep xref computation in WASM (not pre-computed) |
| WASM memory limits (browser heap) | Low | Most codebases < 100k symbols fit in 256MB WASM heap |
| Real-time search performance in WASM | Medium | Profile early; add inverted index if substring match is too slow |

### 13.2 Open Questions

1. **How do we handle source file serving in a zero-server model?**
   - Option A: embed source files as base64 in JSON (increases page size significantly)
   - Option B: serve source as static files alongside `index.html` (requires a simple HTTP server or file:// access)
   - Option C: zip the source tree, embed as base64, decompress in WASM on first use

2. **Should the final artifact be a single HTML file or a directory of files?**
   - Single HTML is most portable (email it, share it, open from USB)
   - Directory is easier to build and debug
   - Decision: start with directory (`dist/index.html` + assets), add single-file option later

3. **Should we keep the server for development and only ship static for releases?**
   - Yes — keep `make dev-backend` + `make dev-frontend` for local development
   - The static artifact is the **release** artifact
   - Dev workflow: `make dev` runs the Go server; `make build-static` produces the release artifact

4. **What about hot-reload in development?**
   - The static artifact is not for development
   - Vite's HMR works in dev mode (`make dev-frontend`)
   - Static build is for distribution (shipped with a library, uploaded to a CDN, etc.)

### 13.3 Alternatives considered

**Alternative 1: Pure JS search implementation**
- Rewrite `FindSymbols` in TypeScript
- Pros: no WASM complexity, simpler build pipeline
- Cons: duplicated logic, potential inconsistency with Go indexer, no Go type safety
- **Decision: Use WASM** — the Go code is already correct and well-tested

**Alternative 2: Pre-render every page to static HTML, no WASM**
- Generate HTML files for every package, symbol, and doc page at build time
- Pros: truly static, no client-side computation
- Cons: huge number of files, complex build, can't link between pages dynamically, no search
- **Decision: WASM provides search + dynamic navigation without server**

**Alternative 3: Web Worker for WASM**
- Load WASM in a Web Worker to avoid blocking the main thread
- Pros: better UI responsiveness during search
- Cons: more complex message passing between main thread and worker
- **Decision: Start with main-thread WASM, promote to Web Worker if profiling shows blocking**

**Alternative 4: SQLite in the browser (via WASM)**
- Use `github.com/你的人/wasm-sqlite` to store the index in SQLite
- Pros: mature database, efficient queries, full-text search
- Cons: adds a heavy dependency, over-engineered for our query patterns
- **Decision: Plain JSON + Go search logic is sufficient**

---

## 14. Key Files Reference

This section provides a quick reference for every important file in the system, cross-referenced to the sections above.

### 14.1 Indexer (Go AST extraction)

| File | Purpose | Key Symbols |
|---|---|---|
| `internal/indexer/types.go` | `Index`, `Package`, `File`, `Symbol`, `Ref`, `Range` type definitions. All other packages depend on these. | `type Index struct { Packages, Files, Symbols, Refs }` |
| `internal/indexer/extractor.go` | `Extract()` — uses `go/packages` to walk packages, extract decls, collect xrefs. Core of the build-time pipeline. | `func Extract(ExtractOptions) (*Index, error)` |
| `internal/indexer/multi.go` | `Merge()` — concatenates multiple `Index` outputs (Go + TypeScript), detects duplicates, sorts. | `func Merge([]*Index) (*Index, error)` |
| `internal/indexer/id.go` | `PackageID()`, `FileID()`, `SymbolID()`, `MethodID()` — deterministic ID generation. Stable across file moves. | `func SymbolID(importPath, kind, name, suffix string) string` |
| `internal/indexer/xref.go` | Cross-reference extraction. Walks function bodies and records identifier references. | `func addRefsForFile(...)` |
| `internal/indexer/write.go` | JSON serialisation helpers. | `func WriteIndex(idx *Index, w io.Writer, pretty bool) error` |

### 14.2 Browser (Index loading)

| File | Purpose | Key Symbols |
|---|---|---|
| `internal/browser/index.go` | `Loaded` struct — deserialises index.json, builds lookup maps (`byPackageID`, `byFileID`, `bySymbolID`). Exposes `FindSymbols()`, `Symbol()`, `File()`, `Package()`. **This is the code that goes into WASM.** | `func (l *Loaded) FindSymbols(nameQuery, kind string) []*Symbol` |

### 14.3 Server (HTTP API — to be deprecated)

| File | Endpoint | Notes |
|---|---|---|
| `internal/server/server.go` | `Handler()` — registers all routes | `mux.HandleFunc("/api/index", s.handleIndex)` |
| `internal/server/api_index.go` | `handleIndex`, `handlePackages`, `handleSymbol`, `handleSearch` | RTK-Query `getIndex`, `getPackages`, `getSymbol`, `searchSymbols` |
| `internal/server/api_source.go` | `handleSource`, `handleSnippet` | RTK-Query `getSource`, `getSnippet`. **Snippet slicing → pre-compute** |
| `internal/server/api_xref.go` | `handleXref`, `handleSnippetRefs`, `handleSourceRefs`, `handleFileXref` | RTK-Query `getXref`, `getSnippetRefs`, `getSourceRefs`, `getFileXref`. **Graph walk → pre-compute** |
| `internal/server/api_doc.go` | `handleDocList`, `handleDocPage` | RTK-Query `getDocList`, `getDocPage`. **Markdown render → pre-render** |
| `internal/server/spa.go` | `spaHandler()` — serves React SPA | Falls through to `index.html` for SPA routing |

### 14.4 Docs (Markdown rendering)

| File | Purpose | Key Symbols |
|---|---|---|
| `internal/docs/renderer.go` | `Render()` — goldmark markdown → HTML, resolves `codebase-*` directives. Two-pass: preprocess → render. | `func Render(slug, mdSource, loaded, sourceFS) (*Page, error)` |
| `internal/docs/pages.go` | `ListPages()` — walks embedded pages FS, extracts title from H1 or slug. | `func ListPages(fs.FS) ([]PageMeta, error)` |
| `internal/docs/embed_fs.go` | `//go:build embed` — embeds `embed/pages/` directory. | `var pagesFS embed.FS; func PagesFS() fs.FS` |
| `internal/docs/embed_none_fs.go` | Noembed stub. | `func PagesFS() fs.FS { return os.DirFS("internal/docs/embed/pages") }` |

### 14.5 Embedded filesystems (go:embed)

| File | Embeds | Used By |
|---|---|---|
| `internal/indexfs/embed/index.json` | Merged index JSON | `internal/indexfs/embed.go`, `internal/server/`, `internal/wasm/` |
| `internal/indexfs/embed_fs.go` | index.json | Server + WASM |
| `internal/sourcefs/embed/source/` | Mirrored source tree | `internal/server/api_source.go`, `internal/static/` |
| `internal/web/embed/public/` | Vite SPA build output | `internal/server/spa.go` |
| `internal/docs/embed/pages/` | Markdown doc pages | `internal/docs/pages.go`, `internal/static/` |

### 14.6 Build generators

| File | Invoked By | Produces |
|---|---|---|
| `internal/indexfs/generate_build.go` | `go generate ./internal/indexfs` | `embed/index.json` (via `codebase-browser index build`) |
| `internal/sourcefs/generate_build.go` | `go generate ./internal/sourcefs` | `embed/source/` (mirrored tree) |
| `internal/web/generate_build.go` | `go generate ./internal/web` | `embed/public/` (Vite build, via Dagger or local pnpm) |
| `internal/wasm/generate_build.go` | `go generate ./internal/wasm` | `search.wasm` + `search.js` (NEW) |
| `internal/static/generate_build.go` | `go generate ./internal/static` | `precomputed.json` (NEW) |

### 14.7 Frontend (React)

| File | Purpose | Notes |
|---|---|---|
| `ui/src/app/App.tsx` | Top-level SPA, `BrowserRouter`, layout | Routes: `/`, `/packages/:id`, `/symbol/:id`, `/source/*`, `/doc/:slug` |
| `ui/src/api/indexApi.ts` | RTK-Query: index, packages, symbol, search | **Change baseQuery: HTTP → WASM** |
| `ui/src/api/sourceApi.ts` | RTK-Query: source, snippets, refs, xref | **Change: HTTP → static files + precomputed JSON** |
| `ui/src/api/wasmClient.ts` | NEW: WASM loader + WASM baseQuery | Replaces `fetchBaseQuery` |
| `ui/src/features/tree/SearchPanel.tsx` | Search input widget | Uses `useSearchSymbolsQuery` |
| `ui/src/features/symbol/SymbolPage.tsx` | Symbol detail view | Uses `useGetSymbolQuery`, `useGetXrefQuery` |
| `ui/src/features/symbol/ExpandableSymbol.tsx` | Collapsible symbol body with code | Uses `useGetSnippetQuery` |
| `ui/src/features/doc/DocSnippet.tsx` | Resolves `codebase-*` stubs in doc HTML | Uses `useGetSymbolQuery`, `ExpandableSymbol`, `XrefPanel` |

---

## Appendix A: API Reference (Static WASM Build)

After the refactor, the frontend uses these WASM functions instead of HTTP endpoints:

```typescript
// WASM module interface (JavaScript side)
interface CodebaseBrowser {
    init(jsonIndex: string): void;
    findSymbols(query: string, kind: string): string;  // JSON string
    getSymbol(id: string): string;                       // JSON string
    getXref(id: string): string;                        // JSON string
    getSnippet(id: string, kind: string): string;       // JSON string
    getPackages(): string;                               // JSON string
    getDocPages(): string;                              // JSON string
    getDocPage(slug: string): string;                   // JSON string
}

// Static file served alongside index.html
// GET /precomputed.json
// GET /xref-index.json
// GET /source/<path>  (mirrored source tree)
// GET /docs/<slug>.html  (pre-rendered doc pages)
```

---

## Appendix B: Build Command Reference

```bash
# Build the full index (Go + TS, merge)
go generate ./internal/indexfs

# Mirror source tree
go generate ./internal/sourcefs

# Build WASM (TinyGo)
go generate ./internal/wasm

# Pre-compute search index, xref data, snippets, doc HTML
go generate ./internal/static

# Bundle final artifact
go generate ./internal/bundle

# All-in-one
make build-static
# equivalent to: go generate ./internal/indexfs && go generate ./internal/sourcefs && \
#                 go generate ./internal/wasm && go generate ./internal/static && \
#                 go generate ./internal/bundle

# Development (keep server for hot-reload)
make dev-backend  # go run ./cmd/codebase-browser serve --addr :3001
make dev-frontend # pnpm -C ui run dev
```

---

## Appendix C: Glossary

| Term | Definition |
|---|---|
| **WASM** | WebAssembly — a binary instruction format for a stack-based virtual machine. Go code can be compiled to WASM and run in browsers. |
| **Go:embed** | A Go build directive that embeds files into the binary at compile time. Used for `index.json`, source tree, and SPA assets. |
| **RTK-Query** | Redux Toolkit Query — a data fetching and caching library. The SPA uses RTK-Query endpoints backed by HTTP (current) or WASM (after refactor). |
| **TinyGo** | A Go compiler that targets WASM and microcontrollers. Produces smaller WASM binaries than the standard Go toolchain. |
| **Inverted index** | A search data structure mapping terms to the documents (here: symbol IDs) that contain them. Used for fast prefix search. |
| **Cross-reference (xref)** | A directed edge between two symbols: "symbol A references symbol B at location X". The `Refs` slice contains all xrefs. `usedBy` = in-edges (callers); `uses` = out-edges (callees). |
| **Snippet** | A byte-range slice of a source file, extracted from a symbol's declaration or body. Pre-extracted at build time to avoid slicing at runtime. |
| **Directive** | In doc pages: a fenced code block with info string `codebase-snippet`, `codebase-signature`, `codebase-doc`, or `codebase-file`. Resolved at render time into live source. |
| **Goldmark** | A markdown parser for Go (`github.com/yuin/goldmark`). Used by `docs.Render()` to produce HTML from markdown. |
| **Dagger** | A CI/CD engine that runs pipelines as code. Used here to orchestrate the Node.js TypeScript extractor in a container with a pnpm cache volume. |

---

*This document was produced as part of GCB-006. For questions or clarification, see the investigation diary at `ttmp/2026/04/23/GCB-006--static-wasm-build-pre-render-html-ship-go-search-as-browser-side-module/reference/01-investigation-diary-wasm-static-refactor.md`.*