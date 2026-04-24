---
Title: 'Investigation Diary: WASM + Static Refactor'
Ticket: GCB-006
Status: active
Topics:
    - wasm
    - go
    - react-frontend
    - codebase-browser
    - static-build
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: Makefile
      Note: Build targets — template for new build-static target
    - Path: internal/browser/index.go
      Note: Loaded.FindSymbols — the core search logic analyzed for WASM compilation
    - Path: internal/docs/renderer.go
      Note: Render() — doc rendering logic analyzed for pre-rendering
    - Path: internal/indexer/types.go
      Note: Index schema — base for precomputed data design
    - Path: internal/server/server.go
      Note: Handler() — registered all routes that will be replaced
    - Path: internal/web/generate_build.go
      Note: Dagger/Vite build pipeline — understood for adding WASM compilation step
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary: WASM + Static Refactor Investigation

## Goal

Understand why `codebase-browser` needs a backend and design a path to a zero-server static artifact — WASM for search/index lookups, pre-rendered HTML for everything else.

## Step 1: Analyze current server responsibilities

**User prompt (verbatim):** "Why does this software need a backend? Can we create a single page rendered static artifact?"

**Assistant interpretation:** The user asks why the Go server exists and whether the browser can be made fully static (no server, no backend, single HTML file). They want an architecture analysis and a design/implementation plan.

**Inferred user intent:** Evaluate the feasibility of eliminating the backend by pre-computing all runtime operations at build time, using WASM for search, and pre-rendering doc pages.

### What I did
- Read `README.md`, `internal/server/*.go` (server.go, api_index.go, api_source.go, api_doc.go, api_xref.go)
- Read `internal/browser/index.go` (the core search/lookup logic)
- Read `internal/indexer/types.go`, `extractor.go`, `multi.go`
- Read `internal/docs/renderer.go`, `pages.go`, `embed_fs.go`
- Read `internal/web/generate_build.go`, `internal/indexfs/generate_build.go`, `internal/sourcefs/generate_build.go`
- Read `Makefile` and `cmd/codebase-browser/main.go`

### Why
Understanding what each API endpoint does at runtime — and whether it's fundamentally necessary or can be pre-computed at build time.

### What worked
The analysis revealed that every API endpoint operates on data that is already known at build time:
- **Search** (`handleSearch`): substring-match over symbol names → WASM
- **Snippet slicing** (`handleSnippet`): byte-range from file → pre-extract at build time
- **Xref graph walk** (`handleXref`): filter `Refs` slice → pre-compute per-symbol ref lists
- **Doc rendering** (`handleDocPage`): goldmark + directive resolution → pre-render HTML

### What didn't work
N/A (investigation phase)

### What I learned
The backend is essentially a runtime interpreter for the embedded data. All computation is deterministic — there's no live data, no user state, no external queries. This makes it a perfect candidate for pre-computation + WASM.

### What was tricky to build
Mapping every API endpoint to its pre-computation strategy without breaking the RTK-Query API surface (endpoints should stay identical, only the base query changes from HTTP to WASM).

### What warrants a second pair of eyes
- **WASM memory management**: passing JSON strings between JS and Go WASM requires careful pointer/length handling. Verify TinyGo's `syscall/js` interop handles this correctly.
- **Pre-computed xref index size**: for large codebases with many refs, the `xrefIndex` JSON could be large (10MB+). Consider compression or on-demand loading.
- **Search performance**: naive substring match over 50k+ symbols in WASM may be slow. Monitor and add inverted index if needed.

### What should be done in the future
- Profile WASM search performance with a 50k+ symbol codebase before finalizing the search index strategy
- Investigate TinyGo vs. standard Go WASM for this use case (TinyGo has better size, standard Go has better stdlib support)

---

## Step 2: Create ticket workspace and write design document

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** User wants the design documented in a docmgr ticket with a detailed analysis suitable for onboarding a new engineer/intern.

**Inferred user intent:** Create a thorough design document + upload to reMarkable so someone can read it on a tablet.

### What I did
- Created ticket GCB-006 via `docmgr ticket create-ticket`
- Added two docs via `docmgr doc add`: design-doc and reference (diary)
- Read `SKILL.md` files for docmgr and diary
- Read `internal/docs/renderer.go` and `pages.go` in depth to understand directive resolution
- Read `internal/indexer/extractor.go` to understand Go AST extraction
- Read `internal/indexer/multi.go` to understand the merge strategy
- Read `internal/web/generate_build.go` and `internal/indexfs/generate_build.go` to understand the build pipeline
- Wrote a 65,000-word design document at `design-doc/01-wasm-static-html-architecture-design.md`

### Why
A complete design document for an intern/new engineer needs: system overview, why things are the way they are, what changes and why, exact file locations, pseudocode, API references, and an implementation plan. The user specifically asked for "prose paragraphs and bullet points and pseudocode and diagrams and api references and file references" — very detailed, onboarding-oriented.

### What worked
The design document covers:
1. What the system does today (full API reference, build pipeline)
2. Why the backend exists (per-endpoint analysis table)
3. Static WASM architecture (ASCII diagram, build artifact structure)
4. WASM module design (Go code, exports, main entry)
5. Pre-computed data design (search index, xref index, snippet cache)
6. React frontend changes (RTK-Query base query swap)
7. Index schema changes
8. File layout after refactor
9. 6-phase implementation plan
10. Testing strategy
11. Risks and alternatives
12. Full file reference with purposes and key symbols

### What didn't work
N/A

### What I learned
The key insight: the frontend's RTK-Query API surface can stay identical even with the WASM refactor — only the `baseQuery` changes from HTTP fetch to WASM function calls. This dramatically reduces frontend changes.

Also learned: TinyGo's `syscall/js` provides automatic string marshaling between Go and JavaScript, making the WASM JS bridge much simpler than the standard Go WASM approach.

### What was tricky to build
The most complex part is the pre-computation of the search index, xref data, and snippets. The `internal/static/generate_build.go` needs to:
1. Load `index.json` and the source tree
2. Walk every symbol and extract snippets
3. Walk the `Refs` slice and build usedBy/uses maps
4. Walk the symbol list and build a search index
5. Render every doc page and capture HTML

This is a large Go program that runs at build time. It should be well-tested.

### What warrants a second pair of eyes
- **Build-time performance**: the static generator walks the entire source tree and renders all doc pages. On large codebases this could take 30s+. Consider incremental builds (only re-process changed files).
- **Precomputed JSON size**: for a 100k-symbol codebase, the `xrefIndex` alone could be tens of MB. Verify gzip compression in the browser helps, and consider streaming the WASM initialization.

### What should be done in the future
- Add incremental build support to the static generator
- Profile WASM search performance with the existing codebase (run `FindSymbols` in a loop, measure time)
- Consider the single-HTML-file variant (all assets inlined as base64) as an alternative to the directory-of-files approach