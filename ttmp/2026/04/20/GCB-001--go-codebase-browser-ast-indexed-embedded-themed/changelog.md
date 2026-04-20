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


## 2026-04-20

Phase 3 complete: Vite+React+TS+Redux Toolkit frontend. RTK-Query slices indexApi + sourceApi (keepUnusedDataFor=3600 since binary is immutable). Routes: / (overview), /packages/:id, /symbol/:id, /source/*. Widget package @codebase-browser/ui with parts.ts + base.css + dark.css theme. go generate ./internal/web runs vite build + copies dist to embed dir. E2E verified: / returns index.html, /assets/*.js served with correct content-type, client-side-routed paths fall through to index.html via the SPA handler.


## 2026-04-20

Phase 4 complete: widgets extracted into ui/src/packages/ui (future @codebase-browser/ui) with parts.ts + theme/base.css + theme/dark.css. Storybook 8 configured with a Theme toolbar (Light/Dark/Unstyled via data-theme on widget root; Unstyled drops base.css). Stories cover SymbolCard (Default/WithSnippet/Method/Struct/Interface/Const/NoDoc/WithNameRenderer/AllKinds), TreeNav, SearchBox, Snippet (incl. JumpLink), SourceView (incl. highlight). Overview.mdx documents the theming contract. build-storybook succeeds.


## 2026-04-20

Phase 5 complete: internal/docs renderer resolves codebase-snippet/-signature/-doc/-file fenced blocks via a goldmark pre-processor. Short-form symbol refs (pkg.Name / pkg.Recv.Method) resolved against the index with ambiguity as a hard error. cmd/codebase-browser doc render + --check gate. /api/doc + /api/doc/{slug} server endpoints. Frontend DocPage + DocList wired via docApi. Two dogfood doc pages (01-overview.md, 02-api.md) embed 6 live snippets from the indexer + server packages; doc render reports 0 errors, 6 snippets across 2 pages.


## 2026-04-20

Completed dark theme via CSS-var override (phase-task 36) and measured real index.json size at ~110 KB for 101 symbols (phase-task 43) — well below design §14.1 threshold so the single-load-on-boot strategy stands.

