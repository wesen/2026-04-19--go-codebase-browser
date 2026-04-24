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

---

## Step 3: Implement Phase 1a — WASM search module

**User prompt (verbatim):** "Add tasks for implementing the static version of this package, and then implement them one by one, committing at appropriate intervals, keeping a diary as you go. Use tinygo."

**Assistant interpretation:** User wants me to implement the static WASM build phase by phase, commit after each phase, keep a diary, and use TinyGo for WASM compilation.

**Inferred user intent:** Produce working code for the static WASM build, not just design docs.

### What I did
- Added 6 implementation tasks to ticket GCB-006
- Created `internal/wasm/` package:
  - `index_types.go`: Lightweight index types (copied from `internal/indexer/types.go` to avoid importing heavy dependencies into TinyGo)
  - `search.go`: `SearchCtx` struct with `FindSymbols`, `GetSymbol`, `GetXref`, `GetSnippet`, `GetPackages`, `GetIndexSummary`, `GetDocPages`, `GetDocPage`
  - `exports.go`: `syscall/js` interop with `RegisterExports()` — registers all functions on `window.codebaseBrowser`
  - `generate_build.go`: Build script (initially for TinyGo, then switched to standard Go)
  - `search_test.go`: Unit tests for search logic
- Created `cmd/wasm/main.go`: WASM entry point with `//go:build wasm`
- Attempted TinyGo compilation but TinyGo 0.28.1 doesn't support Go 1.22
- Switched to standard Go WASM: `GOOS=js GOARCH=wasm go build`
- Built `search.wasm` (3.1MB, 0.9MB gzipped)

### Why
TinyGo was the preferred target (smaller binaries) but Go 1.22 incompatibility forced a fallback to standard Go WASM. The standard Go approach works but produces larger binaries.

### What worked
- `go test ./internal/wasm/...` passes
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` produces a valid WASM binary
- The `syscall/js` API works identically between TinyGo and standard Go for our use case

### What didn't work
- **TinyGo compilation failed**: `error: requires go version 1.18 through 1.20, got go1.22`
- **Initial package structure failed**: Putting `package main` (main.go) and `package wasm` (exports.go) in the same directory caused `found packages wasm and main` error. Fixed by moving main.go to `cmd/wasm/main.go`.

### What I learned
Standard Go WASM is straightforward: compile with `GOOS=js GOARCH=wasm`, include `wasm_exec.js` from `$GOROOT/lib/wasm/`, create a `Go` object, instantiate the WASM, and call `go.run()`. The exported functions appear on `window.codebaseBrowser` after the Go runtime initializes.

### What was tricky to build
The `exports.go` file needs to keep `js.Func` references alive to prevent GC. Used a `keepAlive []js.Func` slice. Also, the Go WASM runtime sets exports asynchronously, so the JS loader needs to poll for `window.codebaseBrowser` to appear.

### What warrants a second pair of eyes
- **WASM binary size**: 3.1MB is large. Consider Binaryen `wasm-opt` or upgrading TinyGo when it supports Go 1.22+.
- **Go WASM runtime dependency**: `wasm_exec.js` is 16KB but tightly coupled to the Go version. Any Go upgrade may require regenerating it.

### Code review instructions
- Start with `internal/wasm/search.go` — verify the SearchCtx API matches the design doc
- Run `go test ./internal/wasm/...` — should pass
- Verify `cmd/wasm/main.go` has the `//go:build wasm` tag
- Check `internal/wasm/generate_build.go` uses `GOOS=js GOARCH=wasm`

### Technical details
```bash
# Build WASM
GOOS=js GOARCH=wasm go build -o internal/wasm/embed/search.wasm ./cmd/wasm

# Size check
ls -lh internal/wasm/embed/search.wasm
gzip -c internal/wasm/embed/search.wasm | wc -c
```

---

## Step 4: Implement Phase 2 — Build-time pre-computation

### What I did
- Created `internal/static/` package:
  - `search_index.go`: `BuildSearchIndexFast()` creates inverted index (full names + prefixes up to 4 chars)
  - `xref_index.go`: `BuildXrefIndex()` pre-computes usedBy/uses per symbol; `BuildFileXrefIndex()` for file-level xref
  - `snippet_extractor.go`: `ExtractSnippets()` reads source files and extracts declaration/body/signature text; `ExtractSnippetRefs()` and `ExtractSourceRefs()` for ref linkification
  - `doc_renderer.go`: `DocRenderer.RenderAll()` pre-renders all doc pages using `docs.Render()`
  - `generate_build.go`: Main generator script that loads index, runs all pre-computation, writes `precomputed.json`
  - `static_test.go`: Unit tests for search index and xref index builders
- Ran generator: produced `precomputed.json` (1.2MB for 329 symbols, 1005 refs, 3 doc pages)

### Why
All runtime computation is moved to build time. The `precomputed.json` file contains everything the WASM module needs: search index, xref data, snippets, doc HTML.

### What worked
- `go test ./internal/static/...` passes
- `go run internal/static/generate_build.go` completes in ~1 second
- `precomputed.json` is 1.2MB — reasonable size for the current codebase

### What didn't work
- **Unexported field access**: `loaded.bySymbolID` is unexported. Fixed by iterating over `loaded.Index.Symbols` instead.
- **Unused import**: `indexer.XrefData` doesn't exist (I defined my own types in the static package). Removed the import.

### What I learned
The `browser.Loaded` struct is the central abstraction for index access. Its public methods (`Symbol()`, `File()`, `Package()`) should be used instead of accessing internal maps directly.

### What was tricky to build
The `BuildXrefIndex` function needs to handle three cases per ref:
1. `usedBy`: ref TO this symbol (from outside)
2. `uses`: ref FROM this symbol (to outside)
3. Neither: refs within the same file/symbol (ignored for file xref)

The deduplication logic for `uses` (grouping by `toSymbolId`) is subtle and needs careful testing.

### Code review instructions
- Run `go test ./internal/static/...` — should pass
- Run `go run internal/static/generate_build.go` — should produce `internal/static/embed/precomputed.json`
- Verify the JSON contains: `searchIndex`, `xrefIndex`, `snippets`, `snippetRefs`, `sourceRefs`, `fileXrefIndex`, `docManifest`, `docHTML`

### Technical details
```bash
# Run pre-computation generator
go run internal/static/generate_build.go

# Check output
ls -lh internal/static/embed/precomputed.json
jq '.searchIndex | keys | length' internal/static/embed/precomputed.json
jq '.xrefIndex | keys | length' internal/static/embed/precomputed.json
jq '.snippets | keys | length' internal/static/embed/precomputed.json
```

---

## Step 5: Implement Phase 3 — Frontend WASM integration

### What I did
- Created `ui/src/api/wasmClient.ts`:
  - `initWasm()`: Loads wasm_exec.js, instantiates WASM, calls `initWasm` with precomputed JSON data
  - `wasmBaseQuery`: RTK-Query baseQuery that routes to WASM functions instead of HTTP
  - `getPrecomputed()`: Caches precomputed.json for direct lookups (snippetRefs, sourceRefs, fileXref)
- Updated `ui/src/api/indexApi.ts`: Swapped `fetchBaseQuery` for `wasmBaseQuery`
- Updated `ui/src/api/sourceApi.ts`: Static file serving for source files, WASM for snippets, precomputed cache for refs/xref
- Updated `ui/src/api/docApi.ts`: `wasmBaseQuery` for doc pages
- Updated `ui/src/api/xrefApi.ts`: `wasmBaseQuery` for xref lookups
- Fixed TypeScript compilation errors (unused params, return type mismatches)

### Why
The frontend's RTK-Query API surface stays identical — only the transport layer changes. This minimizes React component changes.

### What worked
- `pnpm run typecheck` passes cleanly
- The `wasmBaseQuery` correctly maps endpoint strings to WASM function calls:
  - `'index'` → `getIndexSummary()`
  - `'packages'` → `getPackages()`
  - `'symbol:<id>'` → `getSymbol(id)`
  - `'search:<q>|<kind>'` → `findSymbols(q, kind)`
  - `'xref:<id>'` → `getXref(id)`
  - `'docPages'` → `getDocPages()`
  - `'docPage:<slug>'` → `getDocPage(slug)`

### What didn't work
- **TypeScript errors**:
  1. `/// <reference types="../wasm-types" />` — non-existent reference file. Removed.
  2. Unused `api` and `extraOptions` params in `wasmBaseQuery`. Removed.
  3. `getSnippet` return type mismatch — WASM returns `{ text: string }` but component expects `string`. Fixed by extracting `.text` in the queryFn.

### What I learned
Standard Go WASM requires `wasm_exec.js` to be loaded before the WASM binary. The `Go` class is provided by this script. The WASM instantiation pattern is:
```javascript
const go = new Go();
const result = await WebAssembly.instantiateStreaming(fetch('search.wasm'), go.importObject);
go.run(result.instance);
```

### What was tricky to build
The `sourceApi.ts` needs three different data sources:
1. **Source files**: Static fetch (`./source/<path>`)
2. **Snippets**: WASM function call
3. **Refs/Xref**: Precomputed JSON cache

Using `queryFn` instead of `query` in RTK-Query allows custom logic per endpoint. This is cleaner than trying to shoehorn everything into a single baseQuery.

### Code review instructions
- Run `pnpm -C ui run typecheck` — should pass with 0 errors
- Verify `wasmClient.ts` exports: `initWasm`, `wasmBaseQuery`, `getPrecomputed`, `isWasmReady`
- Check that all four API files (`indexApi.ts`, `sourceApi.ts`, `docApi.ts`, `xrefApi.ts`) import from `wasmClient.ts`

### Technical details
```bash
# TypeScript check
cd ui && pnpm run typecheck

# SPA build
cd ui && pnpm run build
```

---

## Step 6: Implement Phase 4 — Static artifact bundler

### What I did
- Created `internal/bundle/generate_build.go`:
  1. Builds SPA (`pnpm -C ui run build`)
  2. Copies SPA assets to `dist/`
  3. Copies `search.wasm`, `precomputed.json`, `source/` tree, `wasm_exec.js` to `dist/`
  4. Injects `wasm_exec.js` into `index.html`
- Switched `App.tsx` from `BrowserRouter` to `HashRouter` (required for `file://` protocol)
- Changed `sourceApi.ts` to use relative paths (`./source/<path>`) instead of absolute (`/source/<path>`)
- Added Makefile targets: `generate-static`, `build-static`
- Added `dist/` to `.gitignore`
- Ran bundler: produced `dist/` artifact (5MB total)
- Verified artifact via HTTP server: index.html, WASM, precomputed.json, source files all accessible

### Why
`BrowserRouter` requires server-side routing (serves index.html for all paths). `HashRouter` uses URL hashes (`/#/symbol/...`) which work with `file://`. Relative paths are required because `file://` doesn't have a concept of "root" like HTTP does.

### What worked
- `make build-static` (via `go run internal/bundle/generate_build.go`) completes successfully
- `dist/` contains all required files: index.html, assets/, search.wasm, precomputed.json, source/, wasm_exec.js
- HTTP server test confirms all files are accessible

### What didn't work
- **Makefile sed command failed**: Using `sed` with tabs in Makefiles is tricky. Used Python instead for the modification.
- **dist/ accidentally staged**: The bundler output was initially included in the git commit. Added `dist/` to `.gitignore` and unstaged.

### What I learned
Vite's production build outputs to `ui/dist/public/`. The bundler copies this directory to `dist/` and adds the WASM-specific files. The `crossorigin` attribute on script tags in Vite's output is fine for `file://` — browsers ignore it for local files.

### What was tricky to build
The `injectWasmExec` function needs to insert `<script src="/wasm_exec.js"></script>` before the first `<script type="module">` tag. The Vite output has exactly one module script, so a simple string replacement works.

### Code review instructions
- Run `make build-static` — should produce `dist/` directory
- Verify `dist/index.html` contains `<script src="/wasm_exec.js">` before the module script
- Start a server in `dist/` and verify: `python3 -m http.server 8767`
- Open `http://localhost:8767/` — SPA should load (WASM initialization happens in background)

### Technical details
```bash
# Build static artifact
make build-static

# Serve and verify
cd dist && python3 -m http.server 8767
curl http://localhost:8767/index.html | head -5
curl http://localhost:8767/search.wasm | wc -c
curl http://localhost:8767/precomputed.json | wc -c
```

### Artifact sizes
- `search.wasm`: 3.1 MB (0.9 MB gzipped)
- `precomputed.json`: 1.2 MB
- SPA JS: 263 KB
- SPA CSS: 10 KB
- `wasm_exec.js`: 17 KB
- Source tree: ~2 MB
- **Total**: ~5 MB