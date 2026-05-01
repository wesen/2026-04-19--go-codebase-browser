# Tasks

## TODO

### Phase 1 — Static bundle contract

- [ ] Add `internal/staticbundle` package with `Options`, manifest types, data shard types, and tests.
- [ ] Add TypeScript static bundle mirror types under `ui/src/api/staticBundleTypes.ts`.
- [ ] Define manifest coverage enums for commit diffs, body diffs, impacts, review docs, source, and LLM DB inclusion.

### Phase 2 — Build-time data loaders

- [ ] Implement SQLite loaders for commits, packages, files, symbols, refs, review docs, and review snippet directives.
- [ ] Add unit tests for loaders using a small review/history database fixture.
- [ ] Preserve `review.db` as optional LLM/query artifact while keeping browser runtime on static data files.

### Phase 3 — Static browser core data

- [ ] Compute symbol histories for all symbols with more than one indexed commit.
- [ ] Precompute body diffs for changed adjacent symbol-history transitions.
- [ ] Precompute adjacent commit diffs and review-requested commit diff pairs.
- [ ] Write `manifest.json` plus `data/*.json` shards instead of the review-centric `precomputed.json` shape.

### Phase 4 — Review layer data

- [ ] Render review markdown docs into `data/review-docs.json`.
- [ ] Rename widget placeholders from snippet-specific markers to generic `data-codebase-widget` markers.
- [ ] Add review-requested coverage for explicit `codebase-diff`, `codebase-impact`, `codebase-symbol-history`, and diff-stat directives.

### Phase 5 — Frontend static query provider

- [ ] Add `CodebaseQueryProvider` interface with semantic methods instead of endpoint-string routing.
- [ ] Implement `StaticQueryProvider` backed by manifest/data shards.
- [ ] Implement `ServerQueryProvider` backed by existing server APIs for dev/server mode.
- [ ] Refactor RTK Query slices to call provider methods via `queryFn`.

### Phase 6 — UI cleanup

- [ ] Remove scattered `isStaticExport()` endpoint interception from API files.
- [ ] Update History page to rely on the generic provider and render capability-aware missing-data messages.
- [ ] Update ReviewDocPage to use a generic widget dispatcher.
- [ ] Ensure review docs can cross-link into `/history?symbol=...`, `/symbol/:id`, `/source/*`, and diff/impact views.

### Phase 7 — CLI cleanup

- [ ] Refactor `review export` into a thin command that calls `staticbundle.Export`.
- [ ] Add explicit flags: `--features`, `--body-diffs`, `--impact`, `--include-db`, `--include-source`, `--repo-root`.
- [ ] Decide whether to add a top-level `export static` command after internals are stable.

### Phase 8 — Regression tests

- [ ] Add Go tests for manifest/data shard writing and coverage selection.
- [ ] Add TypeScript tests for static provider lookup and commit ref resolution.
- [ ] Add Playwright test that opens a static review doc and confirms zero `/api/*` requests.
- [ ] Add Playwright test for `/history?symbol=sym:...Register` to verify history body diffs are available.

### Phase 9 — Documentation and delivery

- [ ] Update Glazed help entries once CLI flags are finalized.
- [ ] Update ticket diary/changelog during implementation.
- [ ] Upload updated design/implementation notes to reMarkable after major revisions.
