# Tasks

## Design phase (complete)

- [x] Create ticket GCB-001 with design-doc and diary scaffolds
- [x] Seed vocabulary topics (go-ast, codebase-browser, embedded-web, react-frontend, storybook, glazed, rtk-query, documentation-tooling)
- [x] Write primary analysis and implementation guide (design-doc/01-...)
- [x] Populate investigation diary (reference/01-...)
- [x] Relate key skill files to both docs
- [x] Update changelog with design-phase entries
- [x] Run docmgr doctor and resolve warnings
- [x] Upload design doc + diary bundle to reMarkable under /ai/2026/04/20/GCB-001

## Phase 0 — Scaffolding

- [x] `go mod init` + directory layout per design §5.2
- [x] Add Makefile targets (dev-backend, dev-frontend, frontend-check, build, generate)
- [x] Scaffold `ui/` with Vite + React + TS + dev proxy per go-web-frontend-embed skill

## Phase 1 — Indexer + CLI

- [x] `internal/indexer/{extractor,id,write}.go` (packages, files, top-level symbols only)
- [x] `cmd/codebase-browser/cmds/index/{build,stats}.go` (Glazed conventions)
- [x] `cmd/codebase-browser/cmds/symbol/{show,find}.go`
- [x] Golden JSON test on fixture module (determinism)
- [x] Wire root main.go with logging + embedded help

## Phase 2 — Server + embed

- [x] `internal/indexfs/`, `internal/sourcefs/`, `internal/web/` with build-tag pairs
- [x] `internal/server/{server,api_index,api_source,api_doc,api_search}.go`
- [x] `cmd/codebase-browser/cmds/serve/run.go`
- [x] Regression tests: SPA fallback, `/api/source` path hygiene

## Phase 3 — Frontend shell

- [ ] RTK-Query slices (indexApi, sourceApi, docApi)
- [ ] Routes: /packages, /packages/:id, /symbol/:id, /source/*
- [ ] Vite build wired into `go generate ./internal/web`

## Phase 4 — Themable widget package + Storybook

- [ ] Extract widgets into `ui/src/packages/ui` (`@codebase-browser/ui`)
- [ ] `parts.ts`, `theme/base.css`, optional theme presets
- [ ] Storybook for TreeNav, SymbolCard, SourceView, Snippet, DocPage, SearchBox
- [ ] MSW-based RTK-Query mock decorator in Storybook

## Phase 5 — Doc renderer with live snippets

- [ ] goldmark custom fenced-block extension (`codebase-snippet`, `codebase-signature`, `codebase-doc`, `codebase-file`)
- [ ] `cmd/codebase-browser/cmds/doc/render.go` (build-time AST emitter)
- [ ] `/api/doc` and `/api/doc/{slug}` endpoints
- [ ] Frontend DocPage + `<Snippet/>` with "jump to source" link
- [ ] Ship 2-3 dogfood doc pages

## Phase 6 — Cross-references (optional)

- [ ] Emit `refs` from `types.Info.Uses`
- [ ] `/api/xref/{id}` endpoint
- [ ] Frontend "Called by" / "Uses" panels

## Phase 7 — Polish

- [ ] Dark theme + one custom theme example
- [ ] Doc page that embeds the doc-renderer source (meta)
- [ ] README with screenshot
- [ ] GitHub Actions: frozen pnpm lockfile → `go generate ./...` → `go build -tags embed ./...`

## Open questions to resolve during phase 1

- [ ] Commit generated `index.json` to repo, or regenerate in CI? (design §14.3 recommends regenerate)
- [ ] Publish `@codebase-browser/ui` to npm, or keep in-tree? (design §14.3 recommends in-tree for phase 1)
- [ ] `<Snippet/>` inline vs lazy-load threshold (design §14.3 recommends ≤ 8 KB inline)
- [ ] Measure real `index.json` size before committing to "load whole index on boot"
