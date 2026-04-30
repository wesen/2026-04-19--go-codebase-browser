---
Title: TinyGo vs sql.js — Feasibility Assessment for Browser-Side SQLite
Ticket: GCB-013
Status: active
Topics:
    - codebase-browser
    - pr-review
    - code-review
    - wasm
    - sqlite
    - tinygo
    - sql-js
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/wasm/search.go
      Note: Existing TinyGo WASM search module — proven working
    - Path: internal/wasm/generate_build.go
      Note: TinyGo 0.41.1 build pipeline via Dagger
    - Path: cmd/wasm/main.go
      Note: WASM entry point
    - Path: /home/manuel/code/wesen/obsidian-vault/Projects/2026/04/02/ARTICLE - SQLide Browser - Building a Browser SQL IDE with Go Wasm and SQLite.md
      Note: User's own research — definitive investigation of Go WASM + SQLite feasibility
ExternalSources: []
Summary: A concise feasibility assessment answering whether TinyGo WASM can replace sql.js for browser-side SQLite queries. Based on the user's own SQLide research and verified build tests, the conclusion is that TinyGo alone is sufficient for pre-computed interactive review widgets, but sql.js remains necessary for ad-hoc SQL/LLM queries. Pure-Go SQLite in TinyGo WASM is not feasible.
LastUpdated: 2026-04-30T15:00:00Z
WhatFor: Resolve the open question in doc 02 about whether to pursue pure-Go SQLite in TinyGo WASM
WhenToUse: Read when deciding between TinyGo-only and TinyGo+sql.js architectures for the review export
---

# TinyGo vs sql.js — Feasibility Assessment for Browser-Side SQLite

## 1. Executive Summary

**Question:** Can we use TinyGo WASM alone — without sql.js — for all browser-side queries in the code review export, including ad-hoc SQL?

**Answer:** No. But this is not a problem.

TinyGo WASM is **proven and ready** for the interactive review experience (diffs, history, impact, search). We already compile `search.wasm` with TinyGo 0.41.1 via Dagger, and it works. For that use case, we don't need SQLite at all — pre-computed JSON loaded into TinyGo maps/slices is faster, smaller, and simpler.

For **ad-hoc SQL queries** (the LLM use case), sql.js is the only browser-side option that actually works. Pure-Go SQLite in TinyGo WASM is blocked by fundamental constraints: no OS-level VFS, no OPFS access from the main thread, and likely compilation failures with `modernc.org/sqlite`.

This document explains why, referencing the user's own SQLide research and concrete build evidence from this repo.

## 2. What we know works: TinyGo WASM in codebase-browser

### 2.1 Build evidence

The existing `search.wasm` is built with TinyGo via Dagger and works:

```
$ tinygo version
tinygo version 0.41.1 linux/amd64 (using go version go1.25.5 and LLVM version 20.1.1)

$ go generate ./internal/wasm
...
generate_build: wrote .../search.wasm (1170.6 KB, 391.4 KB gzipped) [TinyGo (dagger)]
```

The binary contains TinyGo runtime symbols (`tinygo_getCurrentStackPointer`, `tinygo_unwind`, etc.) and `wasm_exec.js` is TinyGo's modified version. Tests pass: `go test ./internal/wasm/...` — PASS.

### 2.2 What TinyGo WASM currently does

The `SearchCtx` loads pre-computed JSON into memory and provides:
- `FindSymbols` — substring search over symbol names
- `GetSymbol` — ID lookup
- `GetXref` — pre-computed usedBy/uses
- `GetSnippet` — pre-extracted text
- `GetPackages`, `GetIndexSummary`, `GetDocPages`, `GetDocPage`

All data is passed as JSON strings at init time. No file system, no network, no SQLite.

### 2.3 Extending to history queries

The same pattern works for history data:
- `GetCommitDiff(oldHash, newHash)` — lookup in pre-computed `map[string]*CommitDiff`
- `GetSymbolHistory(symbolID)` — lookup in pre-computed `map[string][]HistoryEntry`
- `GetImpact(symbolID, direction, depth)` — lookup in pre-computed `map[string]*Impact`
- `GetReviewDoc(slug)` — lookup in pre-computed doc HTML

**No SQLite engine is needed.** The data is deserialized into Go structs and queried via map lookups. This is what TinyGo WASM is good at.

## 3. What does NOT work: Pure-Go SQLite in TinyGo WASM

### 3.1 The cgo problem

`mattn/go-sqlite3` uses cgo to bind to the C SQLite library. Go's `GOOS=js GOARCH=wasm` target **does not support cgo**. This is a fundamental limitation, not a temporary one. From the SQLide research:

> "Go's `GOOS=js GOARCH=wasm` target does not support cgo. A build that imports `"C"` pulls in `runtime/cgo`, which requires a C toolchain and OS-level threading primitives that do not exist in the browser Wasm environment."

### 3.2 The modernc.org/sqlite ambiguity

`modernc.org/sqlite` is a transpiled pure-Go SQLite with no cgo. In theory, it should compile to `js/wasm`. In practice:

> "The package's own support matrix lists specific OS/arch pairs and does **not** include `js/wasm`. But pkg.go.dev renders the documentation for `js/wasm`, which implies it at least parses. Whether it boots, runs queries, and does not hit OS-level syscall stubs at runtime is a different question."

**We have not tested this in this repo.** The SQLide research concluded it was too ambiguous to pursue. Given that TinyGo has even more limited stdlib support than standard Go, the chances of `modernc.org/sqlite` compiling under TinyGo are very low.

### 3.3 The persistence problem

Even if `modernc.org/sqlite` compiled and booted, SQLite in the browser needs somewhere to persist data between page loads. The browser options are:
- **OPFS** (Origin Private File System) — synchronous, fast, but only available in Web Workers
- **IndexedDB** — async, slow, poor fit for SQLite's synchronous VFS model
- **In-memory** — no persistence

Go's browser Wasm runtime (`GOOS=js GOARCH=wasm`) runs on the **main thread**. It cannot access OPFS, which requires a Web Worker context. Writing a custom VFS that maps to IndexedDB from Go would be substantial custom work.

From the SQLide research:

> "SQLite in a browser needs somewhere to store data between page loads. The official SQLite Wasm project uses OPFS for this, but OPFS is only available from Web Worker contexts, and Go's browser Wasm runtime is single-threaded and runs on the main thread."

### 3.4 The single-threading problem

Go's `js/wasm` target runs all goroutines on a single thread. If you ran SQLite queries in the main-thread Go Wasm module, long queries would freeze the browser tab. From the SQLide research:

> "Go's browser runtime is single-threaded. This means you cannot use Go for long-running computation without blocking the UI. ... Even if Go could run SQLite directly, it would need to be in a worker to avoid blocking the UI. And Go's browser Wasm runtime does not currently support running in a worker context with the same `syscall/js` API."

### 3.5 The size problem

The SQLide project's Go Wasm module was 3.3 MB for 332 lines of Go code. SQLite is a large codebase. A TinyGo Wasm binary containing a full SQLite implementation would likely be 5-10 MB. From the SQLide research:

> "A 3.3 MB download for a SQL statement splitter is hard to defend on performance grounds alone. The justification is architectural — but a production version would need to either give Go substantially more work to do or switch to a lighter Wasm toolchain."

### 3.6 Summary of blockers

| Blocker | Status |
|---------|--------|
| cgo not supported in `js/wasm` | Hard blocker for `mattn/go-sqlite3` |
| `modernc.org/sqlite` untested on `js/wasm` | Likely fails, especially under TinyGo |
| No OPFS access from main thread | Hard blocker for persistence |
| Single-threaded runtime blocks UI | Hard blocker for interactive queries |
| Binary size would be 5-10 MB | Practical blocker |

## 4. What works for ad-hoc SQL: sql.js

The SQLide research settled on the same architecture we proposed in doc 02:

```
Browser Main Thread
├── React SPA (UI)
├── Go/TinyGo Wasm (text processing, search)
└── postMessage → Web Worker
    └── sql.js (official SQLite Wasm from sqlite.org)
        └── OPFS or in-memory storage
```

sql.js is:
- **Proven** — official SQLite project, battle-tested
- **Small enough** — ~1 MB compressed
- **Full SQL** — arbitrary queries, perfect for LLMs
- **Worker-safe** — runs in a Web Worker with OPFS persistence

The tradeoff is that it's a separate WASM module (Emscripten, not TinyGo) and requires JS glue. But it works.

## 5. Recommended architecture: TinyGo for widgets, sql.js optional for SQL console

### 5.1 The default export (no sql.js)

For the vast majority of review use cases, we don't need sql.js at all:

```
pr-42-export/
├── index.html          # SPA shell
├── search.wasm         # TinyGo (~1.2MB, ~400KB gzipped)
├── wasm_exec.js        # TinyGo runtime glue
├── precomputed.json    # History data + review docs (~2-5MB depending on range)
└── assets/             # React bundle
```

This export:
- Opens in any browser, even offline
- Renders review docs with all interactive widgets
- Searches symbols, shows diffs, timelines, impact graphs
- Has **zero external dependencies**

### 5.2 The optional SQL console (with sql.js)

If the user wants the LLM SQL console, add:

```
pr-42-export/
├── review.db           # SQLite database (binary)
├── sql-wasm.wasm       # sql.js engine (~1MB)
├── sql-wasm.js         # sql.js glue
└── ... (same as above)
```

The SQL console is lazy-loaded — it only downloads sql.js assets when the user opens the console panel. The interactive review experience uses the fast TinyGo path.

### 5.3 Why this is the right split

| Capability | TinyGo WASM + JSON | sql.js + review.db |
|---|---|---|
| Symbol search | ✅ Fast map lookup | Overkill |
| Doc rendering | ✅ Pre-computed HTML | Overkill |
| Diff widgets | ✅ Pre-computed diffs | Overkill |
| History timeline | ✅ Pre-computed entries | Overkill |
| Impact analysis | ✅ Pre-computed BFS results | Overkill |
| Ad-hoc SQL for LLMs | ❌ Not possible | ✅ Full SQLite |
| Arbitrary queries | ❌ Not possible | ✅ `SELECT ... JOIN ...` |

**The rule:** If the query pattern is known at export time, pre-compute it in JSON and load it into TinyGo. If the query is unknown (LLM exploration), use sql.js.

## 6. Updating the implementation plan

### Doc 02 proposed: Hybrid A+B (always include sql.js)

This was conservative. We now know:
- **A (TinyGo + JSON) is sufficient** for the core review experience
- **B (sql.js) is optional** — only needed for the SQL console feature

### Revised implementation phases

**Phase 1: Review DB pre-computation** (same as doc 02)
- Build `PrecomputedReview` JSON from review.db
- Pre-compute diffs, histories, impacts, doc HTML

**Phase 2: TinyGo WASM history exports** (same as doc 02)
- Add `GetCommitDiff`, `GetSymbolHistory`, `GetImpact`, `GetReviewDoc` to `SearchCtx`
- Register new exports

**Phase 2b: Fix paths for file:// compatibility** (NEW)
- Change absolute paths (`/search.wasm`, `/precomputed.json`) to relative in the bundler
- Switch `BrowserRouter` to `HashRouter` for file:// support
- Verify the export opens directly in a browser without a server

**Phase 3: sql.js integration** (OPTIONAL — only if SQL console is needed)
- Add sql.js dependency
- Create SQL console component
- Lazy-load sql.js assets

**Phase 4: `review export` CLI command** (same as doc 02)
- Wire everything into a single command
- Build SPA with `VITE_STATIC_EXPORT=1`
- Copy assets to `--out` directory

**Phase 5: End-to-end testing** (same as doc 02)
- Verify offline use, widget hydration, path correctness

## 7. Conclusion

**TinyGo WASM alone is the right default.** It is proven (we already build with it), fast (in-memory Go maps), small (~1.2MB binary + JSON data), and requires no JavaScript SQL libraries.

**sql.js is the right optional add-on** for the LLM SQL console. It is the only browser-side SQLite that actually works, but it adds ~1MB and is unnecessary for the interactive review experience.

**Pure-Go SQLite in TinyGo WASM is not feasible.** It is blocked by cgo limitations, OS-level VFS requirements, single-threading constraints, and likely compilation failures with `modernc.org/sqlite`.

The user's own SQLide research reached the same conclusion:

> "The split architecture is the right shape for this kind of application. Go handles text processing, JavaScript handles the browser, SQLite handles the database. Each layer talks through well-defined data boundaries. No layer tries to do what another layer does better."

For the code review export, the "text processing" layer (diffs, history, impact, search) lives in TinyGo WASM. The "database" layer (ad-hoc SQL) lives in sql.js. The "browser" layer is React. This is the correct architecture.
