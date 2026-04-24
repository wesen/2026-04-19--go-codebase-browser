# Changelog

## 2026-04-23

- Initial workspace created


## 2026-04-23

Created ticket GCB-006: Static WASM build. Ticket workspace initialised with design-doc (WASM + Static HTML Architecture Design) and reference (Investigation Diary).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/23/GCB-006--static-wasm-build-pre-render-html-ship-go-search-as-browser-side-module/design-doc/01-wasm-static-html-architecture-design.md — 65KB design document with full architecture analysis
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/23/GCB-006--static-wasm-build-pre-render-html-ship-go-search-as-browser-side-module/reference/01-investigation-diary-wasm-static-refactor.md — Two-step diary documenting system analysis and design document writing
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/vocabulary.yaml — Added wasm and static-build topics to vocabulary


## 2026-04-23

Implemented Phases 1-4: WASM module (internal/wasm/), pre-computation (internal/static/), frontend integration (ui/src/api/wasmClient.ts), bundler (internal/bundle/). Static artifact builds successfully via 'make build-static' producing dist/ directory.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/Makefile — Added generate-static and build-static targets
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/bundle/generate_build.go — Bundler assembling dist/ artifact with SPA + WASM + precomputed data
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/static/generate_build.go — Build-time pre-computation producing precomputed.json (1.2MB for 329 symbols)
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/search.go — SearchCtx with FindSymbols
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/wasmClient.ts — WASM loader and RTK-Query baseQuery for static build

