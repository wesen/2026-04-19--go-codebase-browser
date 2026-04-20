# Changelog

## 2026-04-20

- Initial workspace created


## 2026-04-20

Created ticket GCB-001 with design-doc + diary; added 8 vocabulary topics (go-ast, codebase-browser, embedded-web, react-frontend, storybook, glazed, rtk-query, documentation-tooling).


## 2026-04-20

Drafted primary analysis and implementation guide: 15 sections covering build-time indexing (go/packages + go/ast + go/types) stable symbol IDs self-embedded source tree Glazed CLI tree REST API React/RTK-Query/Storybook frontend and the MDX-ish live-snippet renderer.


## 2026-04-20

Populated investigation diary with Steps 1-2 (ticket creation + design-doc drafting) including prompt context rationale and follow-up risks for phase-1 implementation.


## 2026-04-20

Uploaded bundled PDF (design doc + diary) to reMarkable at /ai/2026/04/20/GCB-001/'GCB-001 Go Codebase Browser — Design + Diary'. Verified via remarquee cloud ls.


## 2026-04-20

Phase 1 complete: indexer (go/packages+go/ast+go/types) emits deterministic index.json with packages/files/symbols. Glazed commands: index build/stats and symbol show/find. Root main.go wires logging + help system. Unit tests for fixture module and determinism passing.


## 2026-04-20

Phase 2 complete: internal/web + internal/sourcefs + internal/indexfs with //go:build embed + !embed pairs; internal/server with /api/index /api/packages /api/symbol /api/source /api/snippet /api/search endpoints plus SPA fallback. Path whitelist enforced via index; traversal/absolute paths 400; unknown /api 404 rather than falling through to index.html. Live smoke test passing against 12 packages / 22 files / 81 symbols indexed from this repo.

