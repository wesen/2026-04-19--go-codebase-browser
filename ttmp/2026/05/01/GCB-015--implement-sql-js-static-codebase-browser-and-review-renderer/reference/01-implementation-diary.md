---
Title: Implementation diary
Ticket: GCB-015
Status: active
Topics:
    - codebase-browser
    - static-export
    - sqlite
    - react-frontend
    - review-docs
    - architecture
    - history
    - markdown-directives
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/review/export.go
      Note: Step 2 CLI wrapper around staticapp.Export
    - Path: internal/staticapp/export.go
      Note: Step 2 static-only export packaging
    - Path: internal/staticapp/manifest.go
      Note: Step 2 manifest schema
    - Path: internal/staticapp/reviewdocs.go
      Note: Step 3 rendered review docs into SQLite
    - Path: ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/design-doc/01-sql-js-static-frontend-architecture-and-implementation-guide.md
      Note: Architecture source for implementation decisions
    - Path: ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md
      Note: Task checklist that drives diary steps
    - Path: ui/package.json
      Note: Step 1 added sql.js dependencies
    - Path: ui/pnpm-lock.yaml
      Note: Step 1 dependency lock updates
    - Path: ui/public/sql-wasm.wasm
      Note: Step 1 browser sql.js WASM runtime asset
    - Path: ui/src/api/historyApi.ts
      Note: Step 4 provider-backed history RTK endpoints
    - Path: ui/src/api/queryErrors.ts
      Note: Step 4 provider error normalization
    - Path: ui/src/api/queryProvider.ts
      Note: Step 4 static-only provider singleton
    - Path: ui/src/api/sqlJsQueryProvider.ts
      Note: Step 4 initial SQL provider
    - Path: ui/src/api/sqljs/sqlJsDb.ts
      Note: Step 1 sql.js and DB singleton loader
    - Path: ui/src/api/sqljs/sqlRows.ts
      Note: Step 1 prepared statement and BLOB utilities
ExternalSources: []
Summary: Chronological implementation diary for the static-only sql.js frontend cleanup.
LastUpdated: 2026-05-01T20:15:00-04:00
WhatFor: Use this diary to resume or review GCB-015 implementation work, including what changed, why, commands run, failures, commits, and validation notes.
WhenToUse: Read before continuing GCB-015 implementation or reviewing commits from this ticket.
---






# Implementation diary

## Goal

This diary records the implementation of GCB-015: replacing the prototype Go-server/TinyGo/precomputed-review static runtime paths with a static-only browser runtime backed by `sql.js` over `db/codebase.db`.

The intended final shape is simple: Go creates and packages the SQLite database, then exits. React opens that database in the browser with `sql.js`. There is no Go runtime server, no `ServerQueryProvider`, no `/api/*` fallback, and no dynamic reload path other than rerunning the index/export command and refreshing the browser.

## Step 1: Add sql.js dependency and browser DB bootstrap helpers

This step starts the static-only runtime implementation from the browser side. I added `sql.js`, copied its WASM runtime into the frontend public assets, and introduced small TypeScript utilities for opening `db/codebase.db` and running prepared SQL statements.

The intent is to establish the first working seam for the new architecture: React should be able to load a static SQLite database directly, without Go HTTP routes, TinyGo review data, or `/api/*` fallbacks.

### Prompt Context

**User prompt (verbatim):** "add detailed tasks to the ticket. then start implementing them one by one, keeping a detailed diary as you go, and commit at approprite intervals."

**Assistant interpretation:** Add a more granular GCB-015 task plan, create a diary, then begin the sql.js static frontend implementation in focused commit-sized steps.

**Inferred user intent:** Move from design into implementation while leaving enough documentation for review, handoff, and continuation.

**Commit (code):** pending — dependency/bootstrap slice not committed yet

### What I did

- Expanded `tasks.md` into detailed task IDs grouped by implementation phase.
- Created `reference/01-implementation-diary.md`.
- Ran:
  - `pnpm -C ui add sql.js`
  - `pnpm -C ui add -D @types/sql.js`
- Copied `ui/node_modules/sql.js/dist/sql-wasm.wasm` to `ui/public/sql-wasm.wasm` so Vite/static export can serve it from the output root.
- Added `ui/src/api/sqljs/sqlRows.ts` with:
  - `queryAll`
  - `queryOne`
  - `sqlBlobToBytes`
  - `sqlBlobToText`
  - `extractUtf8Range`
- Added `ui/src/api/sqljs/sqlJsDb.ts` with:
  - `getSqlJs()`
  - `getStaticManifest()`
  - `getStaticDb()`
  - `smokeCountCommits()`
  - `resetStaticDbForTests()`
- Marked T00.1 and T01.1–T01.7 complete in the task list.

### Why

- `sql.js` is now the target browser query runtime.
- We need a tiny DB bootstrap layer before implementing higher-level provider methods.
- BLOB helpers are needed because `file_contents.content` stores source files as SQLite BLOBs and body diffs must slice by byte offsets.

### What worked

- `pnpm` installed `sql.js 1.14.1` and `@types/sql.js 1.4.11` successfully.
- The TypeScript default import from `sql.js` typechecked under the current Vite/TypeScript settings.
- `pnpm -C ui run typecheck` passed.

### What didn't work

- No implementation failures in this step.
- `pnpm` reported existing Storybook peer warnings:
  - `@storybook/react 8.6.18` wants `@storybook/test@8.6.18`, found `8.6.15`.
  - `@storybook/react-vite 8.6.18` wants `@storybook/test@8.6.18`, found `8.6.15`.
- These warnings pre-existed the implementation concern and did not block typechecking.

### What I learned

- The installed `@types/sql.js` package exposes types through the `initSqlJs` namespace, so using `import type initSqlJsTypes from 'sql.js'` works for `Database`, `SqlJsStatic`, and `SqlValue`.
- Vite's public directory is the simplest first-pass place to put `sql-wasm.wasm`.

### What was tricky to build

- The main sharp edge is BLOB handling. Browser-side body extraction should slice `Uint8Array` byte ranges before UTF-8 decoding, because Go symbol offsets are byte offsets, not JavaScript UTF-16 string offsets. The helper `extractUtf8Range` is intentionally byte-based.

### What warrants a second pair of eyes

- Confirm that copying `sql-wasm.wasm` into `ui/public/` is the desired long-term asset strategy, or whether export should copy it directly from `node_modules/sql.js/dist`.
- Confirm the fallback behavior in `getStaticManifest()`: it currently defaults to `db/codebase.db` if `manifest.json` is absent, which is convenient for development but should remain compatible with the static-only design.

### What should be done in the future

- Add `SqlJsQueryProvider` on top of these primitives.
- Add a real exported `db/codebase.db` before browser-testing `smokeCountCommits()`.

### Code review instructions

- Start with `ui/src/api/sqljs/sqlJsDb.ts` and `ui/src/api/sqljs/sqlRows.ts`.
- Validate with `pnpm -C ui run typecheck`.
- Confirm `ui/public/sql-wasm.wasm` is present and copied into Vite builds.

### Technical details

Current DB bootstrap flow:

```text
getStaticDb()
  -> getSqlJs()
  -> getStaticManifest()
  -> fetch manifest.db.path or db/codebase.db
  -> new SQL.Database(bytes)
```

## Step 2: Add static-only export packaging and manifest

This step moves the Go export path toward the new static-only model. I added a new `internal/staticapp` package that packages the built SPA, copies the SQLite database to `db/codebase.db`, and writes a small `manifest.json` declaring `sql.js` as the query engine and `hasGoRuntimeServer=false`.

I also refactored `review export` into a thin wrapper around `staticapp.Export`. This intentionally stops treating `precomputed.json` and TinyGo review data as the primary static runtime artifact. The frontend is not fully migrated yet, so the exported bundle is structurally correct for the new architecture but not yet functionally complete until `SqlJsQueryProvider` is wired into the UI.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing GCB-015 one focused slice at a time and document what changes.

**Inferred user intent:** Establish the Go-side packaging foundation for a static-only sql.js browser runtime.

**Commit (code):** pending — static export packaging slice not committed yet

### What I did

- Added `internal/staticapp/manifest.go` with:
  - `Manifest`
  - `DBManifest`
  - `FeatureManifest`
  - `RepoManifest`
  - `CommitManifest`
  - `RuntimeManifest`
- Added `internal/staticapp/export.go` with:
  - `Options`
  - `Export(ctx, opts)`
  - SPA build orchestration
  - SPA asset copy
  - SQLite DB copy to `db/codebase.db`
  - manifest inspection/writing
  - optional source tree copy
- Rewrote `cmd/codebase-browser/cmds/review/export.go` as a thin command wrapper around `staticapp.Export`.
- Added `--repo-root` and `--include-source` flags.
- Made source tree copy optional and defaulted it to false, because the target static runtime should use SQLite/file contents first.
- Marked T02.1–T02.9 complete in `tasks.md`.

### Why

- The target architecture needs Go to package a SQLite-backed static app, not emit a review-specific JSON/WASM runtime.
- `manifest.json` is useful boot metadata for the browser and for humans inspecting an export.
- `db/codebase.db` is the single runtime data source for the browser and the LLM/query artifact.

### What worked

- `gofmt -w internal/staticapp/manifest.go internal/staticapp/export.go cmd/codebase-browser/cmds/review/export.go` succeeded.
- `go test ./internal/staticapp ./cmd/codebase-browser` passed.
- `go build ./cmd/codebase-browser` passed.
- A smoke export succeeded:
  - indexed `/tmp/reviews/static-smoke.md` into `/tmp/gcb015-staticapp.db`;
  - exported to `/tmp/gcb015-staticapp-export`;
  - output contained `manifest.json`, `db/codebase.db`, and `sql-wasm.wasm`.
- Manifest check showed:
  - `kind = codebase-browser-sqljs-static-export`
  - `db.path = db/codebase.db`
  - `runtime.hasGoRuntimeServer = false`
- SQLite check showed the copied DB contained 1 commit and 1 review doc in the smoke fixture.

### What didn't work

- No command failed in this step.
- The exported React app is not expected to be fully usable yet because the frontend still contains old TinyGo/precomputed runtime assumptions. This is an intentional intermediate state and will be fixed by the provider refactor.

### What I learned

- Vite copies `ui/public/sql-wasm.wasm` into the SPA output root, so `staticapp.Export` gets the sql.js WASM asset by copying `ui/dist/public`.
- The export command no longer needs to copy `search.wasm`, `wasm_exec.js`, or `precomputed.json` for the target architecture.

### What was tricky to build

- The main sequencing issue is that Go export packaging can move to the new layout before the frontend is migrated. That means the export command can produce a structurally correct static-only bundle before the current UI can consume it. This is acceptable for the clean-cut ticket, but it must be clearly recorded so reviewers do not expect end-to-end UI behavior from this intermediate commit.

### What warrants a second pair of eyes

- Review whether `--include-source=false` should be the default now or whether source copy should remain default until DB-backed source pages are complete.
- Review whether `staticapp` should mutate the copied DB in later steps for rendered docs/FTS, or whether those tables should be added during indexing.

### What should be done in the future

- Add `static_review_rendered_docs` generation on the copied output DB.
- Add `SqlJsQueryProvider` and refactor the frontend to consume `db/codebase.db`.
- Delete old precomputed/TinyGo static runtime paths once the provider covers the UI.

### Code review instructions

- Start with `cmd/codebase-browser/cmds/review/export.go` to see the command-level simplification.
- Then review `internal/staticapp/export.go` for packaging behavior and manifest generation.
- Validate with:
  - `go test ./internal/staticapp ./cmd/codebase-browser`
  - `go build ./cmd/codebase-browser`
  - a smoke `review export` and inspection of `manifest.json` plus `db/codebase.db`.

### Technical details

Smoke commands used:

```bash
rm -f /tmp/gcb015-staticapp.db
go run ./cmd/codebase-browser review index \
  --commits HEAD~1..HEAD \
  --docs /tmp/reviews/static-smoke.md \
  --db /tmp/gcb015-staticapp.db

rm -rf /tmp/gcb015-staticapp-export
go run ./cmd/codebase-browser review export \
  --db /tmp/gcb015-staticapp.db \
  --out /tmp/gcb015-staticapp-export

sqlite3 /tmp/gcb015-staticapp-export/db/codebase.db \
  'select count(*) from commits; select count(*) from review_docs;'
```

## Step 3: Render review documents into the exported SQLite database

This step adds the first static-only derived table inside the copied browser database. Instead of requiring a Go server to render review markdown on demand, `review export` now renders review documents during export and stores the resulting HTML in `static_review_rendered_docs` inside `db/codebase.db`.

This preserves the existing Go markdown/directive renderer while moving runtime document loading to sql.js. The browser will eventually query `static_review_rendered_docs` directly through `SqlJsQueryProvider`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing the sql.js static architecture in dependency-ordered slices.

**Inferred user intent:** Remove Go runtime server responsibilities by shifting render-time work into the export/index step.

**Commit (code):** pending — rendered-review-docs slice not committed yet

### What I did

- Added `internal/staticapp/reviewdocs.go`.
- Created `static_review_rendered_docs` with columns:
  - `slug`
  - `title`
  - `html`
  - `snippets_json`
  - `errors_json`
  - `rendered_at`
- Implemented `AddRenderedReviewDocs(ctx, dbPath, repoRoot)`.
- Updated `staticapp.Export` to run review doc rendering on the copied output DB.
- Updated the CLI wrapper to set `RenderReviewDocs: true`.
- Marked T03.2, T03.3, T03.4, and T03.6 complete.

### Why

- The target architecture has no Go server, so review docs cannot depend on `internal/review/server.go` at runtime.
- Rendering review docs in Go during export avoids porting the markdown/directive renderer to TypeScript immediately.
- Storing rendered docs in SQLite keeps browser runtime data in the single sql.js database.

### What worked

- `go test ./internal/staticapp ./cmd/codebase-browser` passed.
- A smoke export rendered `/tmp/reviews/static-smoke.md` into the copied DB.
- SQLite verification returned one row:
  - slug `static-smoke`
  - title `Static Export Smoke Review`
  - non-empty HTML
  - non-empty snippets JSON
  - `errors_json = []`

### What didn't work

- Initial verification showed `errors_json` as `null` because `json.Marshal(nil []string)` returns `null`. I fixed this by normalizing nil `page.Errors` to an empty slice before marshaling. I also normalized nil snippets to an empty `[]docs.SnippetRef{}` for consistency.

### What I learned

- The current Go renderer is already suitable for export-time rendering. It returns enough data (`HTML`, `Snippets`, `Errors`) to persist review docs for browser-side loading.
- It is safer to read all `review_docs` rows into memory before upserting rendered docs using the same DB connection, rather than writing while iterating an open query cursor.

### What was tricky to build

- The main subtlety was avoiding accidental mutation of the source DB. `staticapp.Export` copies the DB to `OUT/db/codebase.db` first, then calls `AddRenderedReviewDocs` on the copied DB path. This preserves the source database and makes export a packaging/enrichment step.

### What warrants a second pair of eyes

- Review whether `static_review_rendered_docs` should be created during `review index` instead of export. The current design intentionally creates it during export so the source DB remains a pure index/review DB and the copied DB becomes browser-prepared.
- Review whether renderer placeholder attributes should be changed in Go now or later when the frontend `CodebaseWidget` dispatcher lands.

### What should be done in the future

- Implement `SqlJsQueryProvider.listReviewDocs()` and `getReviewDoc()` against `static_review_rendered_docs`.
- Rename the renderer's widget marker to `data-codebase-widget` during the widget dispatcher refactor.

### Code review instructions

- Start with `internal/staticapp/reviewdocs.go`.
- Check that `AddRenderedReviewDocs` opens the copied DB, creates the table, loads the latest snapshot, renders each doc, and upserts rows.
- Validate with:
  - `go test ./internal/staticapp ./cmd/codebase-browser`
  - `go run ./cmd/codebase-browser review export --db /tmp/gcb015-staticapp.db --out /tmp/gcb015-staticapp-export`
  - `sqlite3 /tmp/gcb015-staticapp-export/db/codebase.db 'select slug,title,errors_json from static_review_rendered_docs;'`

### Technical details

Verification query:

```sql
select slug,title,length(html),length(snippets_json),errors_json
from static_review_rendered_docs;
```

Expected smoke result:

```text
static-smoke|Static Export Smoke Review|4203|3341|[]
```

## Step 4: Add the first SqlJsQueryProvider and move history API to provider query functions

This step adds the first semantic frontend provider over sql.js. The provider can list commits, resolve commit refs, load symbol history, and compute symbol body diffs directly from SQLite BLOB content and symbol byte offsets.

I also rewrote `historyApi.ts` away from server/static endpoint-string routing. Its RTK Query endpoints now call provider methods through `queryFn`, which is the desired target pattern for the static-only runtime.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing the static-only sql.js runtime and record progress/failures.

**Inferred user intent:** Start replacing the old `/api`/TinyGo transport with SQL-backed frontend queries.

**Commit (code):** pending — provider/history slice not committed yet

### What I did

- Added `ui/src/api/queryErrors.ts` with `QueryError` and RTK error normalization.
- Added `ui/src/api/queryProvider.ts` with:
  - `CodebaseQueryProvider` interface;
  - a singleton `getQueryProvider()` returning `SqlJsQueryProvider`;
  - no `ServerQueryProvider` and no runtime-mode branching.
- Added `ui/src/api/sqlJsQueryProvider.ts` with initial implementations for:
  - `listCommits()`;
  - `resolveCommitRef()`;
  - `getCommit()`;
  - `getSymbolHistory()`;
  - `getSymbolBodyDiff()`;
  - placeholder `getCommitDiff()` returning empty diff data;
  - placeholder `getImpact()` returning an empty graph.
- Rewrote `ui/src/api/historyApi.ts` to use RTK `queryFn` calls against the provider instead of `/api/history/*` endpoint strings.
- Marked T04.1–T04.8 and T05.1–T05.6 complete.

### Why

- The target frontend has one runtime data path: `SqlJsQueryProvider -> sql.js -> db/codebase.db`.
- The direct history/body-diff path is the most important first provider capability because it fixes the architectural class of `STATIC_NOT_PRECOMPUTED` failures.
- RTK Query can still be used for caching/hooks, but it should call semantic provider methods rather than encode HTTP URLs.

### What worked

- `pnpm -C ui run typecheck` passed after the provider and `historyApi.ts` rewrite.
- Type-only imports from `historyApi.ts` let provider code reuse existing frontend result shapes without runtime import cycles.
- The byte-offset body extraction path is now represented in TypeScript using `Uint8Array` slicing before UTF-8 decoding.

### What didn't work

- I did not browser-test `/history?symbol=...` yet because other app shell pieces still use the old TinyGo/precomputed index APIs (`indexApi` and `wasmClient`). The provider is typechecked but the whole app still needs the generic browser API migration before end-to-end static runtime validation.
- `getCommitDiff()` and `getImpact()` are placeholders in this step. They exist to satisfy the provider interface while the next phases implement SQL diff and impact semantics.

### What I learned

- It is feasible to keep RTK Query while removing server endpoints: `queryFn` is enough to route hooks to provider methods.
- The current frontend type shapes are PascalCase for history/diff results, so the provider returns those shapes for compatibility with existing widgets.

### What was tricky to build

- The main TypeScript design issue was avoiding a runtime cycle between `historyApi.ts` and provider files. The provider imports history result shapes using `import type`, which is erased at runtime. `historyApi.ts` imports the provider at runtime and calls it from `queryFn`.

### What warrants a second pair of eyes

- Review whether provider result types should continue importing from `historyApi.ts` or move into a neutral `queryTypes.ts` file. A neutral types file would probably be cleaner before the provider expands to packages/symbols/source/review docs.
- Review the placeholder `getCommitDiff()` and `getImpact()` behavior. It is intentionally incomplete and should not be mistaken for final functionality.

### What should be done in the future

- Implement SQL commit diff with SQLite-compatible `UNION ALL` queries.
- Implement refs and impact BFS over `snapshot_refs`.
- Migrate `indexApi`, package/symbol/source pages, and review docs away from TinyGo/precomputed assumptions so the whole app can run from sql.js.

### Code review instructions

- Start with `ui/src/api/historyApi.ts` to see the endpoint-string removal.
- Then review `ui/src/api/queryProvider.ts` for the static-only singleton provider.
- Then review `ui/src/api/sqlJsQueryProvider.ts`, especially `getSymbolBodyDiff()` and its private `getBodyMeta()` / `getContentBytes()` helpers.
- Validate with `pnpm -C ui run typecheck`.

### Technical details

Body diff SQL path:

```text
getSymbolBodyDiff(from, to, symbolId)
  -> resolveCommitRef(from/to)
  -> snapshot_symbols + snapshot_files for byte ranges and content hashes
  -> file_contents for BLOBs
  -> Uint8Array.slice(startOffset, endOffset)
  -> TextDecoder utf-8
  -> simpleUnifiedDiff(oldBody, newBody)
```

## Step 5: Implement SQL commit diffs, refs, and impact BFS in the provider

This step fills in the first non-trivial browser-side graph and diff queries. `SqlJsQueryProvider` now computes commit file/symbol diffs with SQLite-compatible `UNION ALL` queries, computes diff stats in TypeScript, reads refs from `snapshot_refs`, and builds impact graphs with a TypeScript BFS.

This moves impact and commit diff widgets away from precomputed WASM review payloads. The implementation is still blocked from full browser validation by remaining app-shell TinyGo index dependencies, but the provider methods typecheck and are wired through `historyApi`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue converting static browser behavior from precomputed/TinyGo payloads to SQL provider methods.

**Inferred user intent:** Make the SQL-backed provider cover the important interactive widgets, not only history/body diff.

**Commit (code):** pending — SQL diff/impact slice not committed yet

### What I did

- Implemented `getCommitDiff(from, to)` in `SqlJsQueryProvider`.
- Added SQLite-compatible file diff SQL using `UNION ALL` instead of `FULL OUTER JOIN`.
- Added SQLite-compatible symbol diff SQL using `UNION ALL` and the same change classifications as the Go reference:
  - `added`
  - `removed`
  - `modified`
  - `signature-changed`
  - `moved`
- Added TypeScript `diffStats(files, symbols)`.
- Added private `getRefsFrom(symbolId, commit)` and `getRefsTo(symbolId, commit)` methods.
- Implemented `getImpact({ symbolId, direction, depth, commit })` as BFS over refs.
- Added `fallbackName(symbolId)` for external symbols not present in `snapshot_symbols`.
- Marked T06.1–T06.7 complete.

### Why

- SQLite does not support the `FULL OUTER JOIN` shape used by the current Go diff reference, so the browser provider needs SQLite-compatible diff queries.
- Impact should no longer be precomputed for static runtime. It can be computed on demand from `snapshot_refs`.
- Existing impact widgets already call `historyApi.getImpact`, so moving `historyApi` to provider methods makes those widgets use SQL-backed impact once the app shell is fully migrated.

### What worked

- `pnpm -C ui run typecheck` passed.
- The provider now has real SQL implementations for commit diffs and impact instead of empty placeholders.

### What didn't work

- I did not run a browser smoke yet because the app shell still uses `indexApi`/`wasmClient`, which expects the old precomputed/TinyGo runtime files. That will be addressed in later phases.

### What I learned

- The Go reference in `internal/history/diff.go` is useful for semantics, but not directly portable because SQLite lacks `FULL OUTER JOIN`.
- A three-part `UNION ALL` is straightforward and explicit for added/removed/modified file and symbol rows.

### What was tricky to build

- The key correctness risk is preserving the Go diff classification order. In particular, symbol diffs should classify body hash changes as `modified` before checking signature or movement. The SQL `CASE` mirrors that order.
- Impact nodes need to accumulate multiple edges per node while avoiding endless traversal. The provider uses a `visited` set for traversal and a `nodeByID` map for edge accumulation.

### What warrants a second pair of eyes

- Review the SQL parameter order for the `UNION ALL` diff queries. Each query binds old/new commit hashes multiple times.
- Review whether impact should deduplicate duplicate refs or preserve all `snapshot_refs` rows. The current implementation preserves all returned refs as edges.

### What should be done in the future

- Add tests for commit diff SQL against a small fixture DB.
- Add browser validation once package/symbol/source/index APIs are migrated off TinyGo.

### Code review instructions

- Review `ui/src/api/sqlJsQueryProvider.ts`:
  - `fileDiffSQL`
  - `symbolDiffSQL`
  - `diffStats`
  - `getRefsFrom`
  - `getRefsTo`
  - `getImpact`
- Validate with `pnpm -C ui run typecheck`.

### Technical details

Commit diff is intentionally computed in the browser from SQLite:

```text
getCommitDiff(from, to)
  -> resolve old/new refs
  -> fileDiffSQL UNION ALL
  -> symbolDiffSQL UNION ALL
  -> diffStats(files, symbols)
```

Impact is runtime BFS:

```text
getImpact(root, direction, depth, commit)
  -> queue root
  -> getRefsFrom or getRefsTo
  -> add edge to node
  -> enqueue unseen neighbor until max depth
```
