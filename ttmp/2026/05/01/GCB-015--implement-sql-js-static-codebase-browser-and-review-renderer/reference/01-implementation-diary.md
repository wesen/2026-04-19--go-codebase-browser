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
    - Path: cmd/codebase-browser/cmds/review/index.go
      Note: Updated review index help text for static export only in Step 10 (commit 45de723)
    - Path: cmd/codebase-browser/cmds/review/patterns.go
      Note: Moved defaultPatterns helper after deleting serve.go in Step 10 (commit 45de723)
    - Path: cmd/codebase-browser/cmds/review/root.go
      Note: Removed review serve command registration in Step 10 (commit 45de723)
    - Path: cmd/codebase-browser/cmds/review/serve.go
      Note: Deleted deprecated review runtime server command in Step 10 (commit 45de723)
    - Path: docs/help/review-reference.md
      Note: Updated schema help with static export table and no review serve command in Step 10 (commit 45de723)
    - Path: docs/help/review-user-guide.md
      Note: Updated help to document static export workflow only in Step 10 (commit 45de723)
    - Path: internal/review/export.go
      Note: Deleted old PrecomputedReview builder in Step 11 (commit 12d31ec)
    - Path: internal/review/export_test.go
      Note: Deleted tests for removed PrecomputedReview builder in Step 11 (commit 12d31ec)
    - Path: internal/review/server.go
      Note: Deleted deprecated review HTTP wrapper in Step 10 (commit 45de723)
    - Path: internal/staticapp/export.go
      Note: Step 2 static-only export packaging
    - Path: internal/staticapp/export_test.go
      Note: Added static export layout and rendered review-doc tests in Step 12 (commit 35045da)
    - Path: internal/staticapp/manifest.go
      Note: Step 2 manifest schema
    - Path: internal/staticapp/reviewdocs.go
      Note: Step 3 rendered review docs into SQLite
    - Path: internal/wasm/exports.go
      Note: Removed review JS exports and jsonReviewData init arg in Step 11 (commit 12d31ec)
    - Path: internal/wasm/review_types.go
      Note: Deleted WASM ReviewData model in Step 11 (commit 12d31ec)
    - Path: internal/wasm/search.go
      Note: Removed reviewData field and review query methods in Step 11 (commit 12d31ec)
    - Path: ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/design-doc/01-sql-js-static-frontend-architecture-and-implementation-guide.md
      Note: Architecture source for implementation decisions
    - Path: ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md
      Note: Task checklist that drives diary steps
    - Path: ui/index.html
      Note: Removed wasm_exec.js script tag in Step 11 (commit 12d31ec)
    - Path: ui/package.json
      Note: Step 1 added sql.js dependencies
    - Path: ui/pnpm-lock.yaml
      Note: Step 1 dependency lock updates
    - Path: ui/public/sql-wasm-browser.wasm
      Note: Step 9 added after browser requested sql-wasm-browser.wasm
    - Path: ui/public/sql-wasm.wasm
      Note: Step 1 browser sql.js WASM runtime asset
    - Path: ui/public/wasm_exec.js
      Note: Deleted unused Go WASM loader from static sql.js frontend in Step 11 (commit 12d31ec)
    - Path: ui/src/api/docApi.ts
      Note: Step 6 SQL-only review doc API
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

## Step 6: Remove provider wrapper and route review docs through sql.js only

This step responds to the clean-cut instruction directly: I removed the generic provider wrapper file and changed the frontend APIs touched so far to import `SqlJsQueryProvider` directly. I also rewrote `docApi.ts` so review document list/detail queries read from the SQL provider instead of trying server or WASM fallbacks.

This keeps the architecture honest. There is no `ServerQueryProvider`, no runtime-mode switch, and no compatibility fallback for review docs. The only target runtime is sql.js over `db/codebase.db`.

### Prompt Context

**User prompt (verbatim):** "make sure you kill depercated / unnecessary code right away, clean cutover, no backwards compatibililty, no wrapper APIs.

continue"

**Assistant interpretation:** Stop leaving transitional abstraction layers or old fallbacks in place; remove deprecated static/server compatibility paths as soon as the sql.js path exists.

**Inferred user intent:** Keep the implementation simple and avoid accumulating legacy paths that obscure the static-only target architecture.

**Commit (code):** pending — clean-cut frontend API slice not committed yet

### What I did

- Removed `ui/src/api/queryProvider.ts`.
- Removed `ui/src/api/runtimeMode.ts`.
- Added singleton helpers directly in `ui/src/api/sqlJsQueryProvider.ts`:
  - `getSqlJsProvider()`
  - `resetSqlJsProviderForTests()`
- Rewrote `ui/src/api/historyApi.ts` to import `getSqlJsProvider()` directly.
- Rewrote `ui/src/api/docApi.ts` to use provider `queryFn` calls only.
- Removed server fetch fallbacks and WASM fallbacks from `docApi.ts`.
- Implemented review doc list/detail frontend access through:
  - `SqlJsQueryProvider.listReviewDocs()`
  - `SqlJsQueryProvider.getReviewDoc(slug)`
- Marked T08.1, T08.2, T09.1, T09.2, and T09.3 complete.

### Why

- The ticket now explicitly says there is no runtime Go server and no wrapper/provider duality.
- Keeping a generic provider wrapper would be harmless technically, but it violates the user's requested clean cutover and adds a layer without a second implementation.
- Review docs already exist in `static_review_rendered_docs`, so there is no reason for `docApi` to probe `/api/review/docs` or use `wasmBaseQuery`.

### What worked

- `pnpm -C ui run typecheck` passed after deleting the wrapper and runtime-mode file.
- `rg -n "runtimeMode|queryProvider|isStaticExport" ui/src` returned no matches.

### What didn't work

- There are still remaining old TinyGo/WASM users in `indexApi.ts`, `sourceApi.ts`, and `xrefApi.ts`. I did not remove them in this step because their SQL replacements need package/symbol/source/xref provider methods first.

### What I learned

- The review doc path can be cleanly switched to SQL now because Step 3 already writes `static_review_rendered_docs` into the copied DB.
- The frontend no longer needs the runtime-mode helper for the parts already migrated.

### What was tricky to build

- The main risk was removing the provider wrapper while `queryProvider.ts` types were imported by history APIs. The fix was to let `historyApi.ts` and `docApi.ts` call `getSqlJsProvider()` directly and keep the shared return types in their existing API files for now.

### What warrants a second pair of eyes

- Review whether `listDocs` and `getDoc` should remain in `docApi.ts` returning empty/provider review results or be deleted entirely if only review docs are part of the static-only product.
- Review the remaining TinyGo/WASM users and confirm the next cutover order: index/package/symbol/source first, then xrefs/snippets.

### What should be done in the future

- Replace `indexApi.ts`, `sourceApi.ts`, and `xrefApi.ts` with SQL-backed implementations.
- Delete `wasmClient.ts` once no frontend code imports it.

### Code review instructions

- Review `ui/src/api/docApi.ts` for removal of server/WASM fallback behavior.
- Review `ui/src/api/historyApi.ts` for direct `getSqlJsProvider()` usage.
- Review `ui/src/api/sqlJsQueryProvider.ts` for the provider singleton and review doc methods.
- Validate with:
  - `pnpm -C ui run typecheck`
  - `rg -n "runtimeMode|queryProvider|isStaticExport" ui/src`

### Technical details

The target frontend API pattern is now:

```text
RTK Query endpoint
  -> queryFn
  -> getSqlJsProvider().method(...)
  -> sql.js
  -> db/codebase.db
```

There is no `ServerQueryProvider` and no `/api/*` fallback.

## Step 7: Move index/package/symbol/search APIs to sql.js

This step removes another large TinyGo dependency from the app shell. The package list, index summary, symbol detail, and symbol search APIs now query SQLite through `SqlJsQueryProvider` instead of using `wasmBaseQuery`.

The remaining TinyGo users are now narrower: source/snippet and xref APIs. The home/package/symbol/search parts of the app shell have a SQL path, which is necessary before browser-level static validation can work without `precomputed.json`.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue the clean cutover by replacing old WASM/precomputed frontend APIs as soon as equivalent SQL provider methods exist.

**Inferred user intent:** Keep deleting obsolete runtime code and avoid compatibility layers.

**Commit (code):** pending — index API SQL slice not committed yet

### What I did

- Added SQL provider methods:
  - `getIndex()`
  - `getPackageLites()`
  - `getSymbol(id)`
  - `searchSymbols(query, kind)`
- Added private SQL loaders for latest-commit packages, files, and symbols.
- Added `toSymbol(row)` conversion into existing frontend `Symbol` shape.
- Rewrote `ui/src/api/indexApi.ts` to use provider `queryFn` calls only.
- Removed `wasmBaseQuery` usage from `indexApi.ts`.
- Marked T07.1–T07.5 complete.

### Why

- `App.tsx`, `PackagePage`, `SymbolPage`, and `SearchPanel` depend on `indexApi`.
- As long as `indexApi` used TinyGo/precomputed data, the static-only sql.js bundle could not boot cleanly without legacy files.
- SQL already contains packages, files, and symbols for the latest commit, so this is a natural provider responsibility.

### What worked

- `pnpm -C ui run typecheck` passed.
- `indexApi.ts` no longer imports `wasmClient`.

### What didn't work

- Source/snippet rendering still depends on `sourceApi.ts` and `wasmClient`.
- Xrefs still depend on `xrefApi.ts` and `wasmClient`.
- `getIndex()` currently returns empty `module`, `goVersion`, and `generatedAt` fields because those are not yet represented in the SQLite schema/manifest path. The UI can tolerate this, but it should be revisited.

### What I learned

- Existing frontend pages mostly use `getIndex()` as a denormalized latest snapshot. It is straightforward to reconstruct that snapshot from SQL by grouping package file IDs and symbol IDs in TypeScript.
- Keeping existing frontend result shapes reduces the amount of UI refactoring required in this slice.

### What was tricky to build

- `Package` rows need `fileIds` and `symbolIds`, but SQLite package rows do not store those arrays. The provider builds them by loading latest files and symbols and grouping by `packageId`.
- Symbol rows store several JSON fields (`type_params_json`, `tags_json`, `build_tags_json`), so provider conversion must parse those into frontend arrays.

### What warrants a second pair of eyes

- Review `getIndex()` performance. It currently loads all packages/files/symbols for the latest commit and groups in memory, matching the old frontend shape. This is fine for a first cut but may need memoization.
- Review whether `module` should come from manifest metadata, a DB metadata table, or an indexed package/module field.

### What should be done in the future

- Implement source/snippet queries from SQL so `SymbolPage` can render code without TinyGo snippets.
- Implement xref queries from SQL so `XrefPanel` can drop `wasmClient`.

### Code review instructions

- Review `ui/src/api/indexApi.ts` for the removal of `wasmBaseQuery`.
- Review `ui/src/api/sqlJsQueryProvider.ts` around:
  - `getIndex()`
  - `getPackageLites()`
  - `getSymbol()`
  - `searchSymbols()`
  - `toSymbol()`
- Validate with `pnpm -C ui run typecheck`.

### Technical details

Latest snapshot reconstruction:

```text
resolve HEAD
  -> load packages at commit
  -> load files at commit
  -> load symbols at commit
  -> for each package, attach fileIds and symbolIds
  -> return IndexSummary
```

## Step 8: Remove wasmClient and move source/snippet/xref APIs to SQL

This step removes the frontend `wasmClient.ts` file and cuts over the remaining visible source/snippet/xref APIs to `SqlJsQueryProvider`. Source text and snippets now come from `file_contents` in SQLite, and xrefs come from `snapshot_refs`.

This is another clean-cut step: no precomputed JSON fallback, no static source-file fetch fallback, and no TinyGo snippet/xref query path in the frontend APIs.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Continue deleting deprecated frontend runtime paths as soon as SQL equivalents exist.

**Inferred user intent:** Eliminate the old TinyGo/precomputed static runtime rather than carrying it as compatibility code.

**Commit (code):** pending — source/xref SQL cutover not committed yet

### What I did

- Deleted `ui/src/api/wasmClient.ts`.
- Rewrote `ui/src/api/sourceApi.ts` to use SQL provider methods only.
- Rewrote `ui/src/api/xrefApi.ts` to use SQL provider methods only.
- Rewrote `ui/src/api/conceptsApi.ts` to remove `/api` fetching and return static-only unavailable/empty responses for now.
- Added provider methods:
  - `getSource(path)`
  - `getSnippet(symbolId, kind, commitRef)`
  - `getXref(symbolId, commitRef)`
- Added SQL-backed xref grouping for `usedBy` and `uses` responses.
- Updated `DocSnippet` and `AnnotationWidget` to load snippets through `getSqlJsProvider()` instead of `/api/snippet`.
- Marked T07.6, T08.5, and T09.5 complete.

### Why

- `wasmClient.ts` was the center of the old TinyGo/precomputed frontend runtime.
- Source pages, symbol snippets, annotations, and xref panels must all use SQLite for the static-only target.
- Keeping source fallback fetches or `/api/snippet` calls would violate the clean-cut architecture.

### What worked

- `pnpm -C ui run typecheck` passed.
- No frontend code imports `wasmClient` after this step.

### What didn't work

- Snippet refs and source refs currently return empty arrays. This keeps code rendering functional, but clickable token-level source links are not restored yet.
- File xref currently returns an empty structure. Symbol xrefs are SQL-backed, but file-level xref panels need a dedicated SQL implementation later.
- Query concepts are disabled in the static-only runtime for now rather than ported to SQL.

### What I learned

- Most code display can work with only `snapshot_symbols`, `snapshot_files`, and `file_contents`; precomputed snippets are not structurally necessary.
- The richer source-linking experience depends on ref-to-offset indexes that need a SQL implementation rather than JSON maps.

### What was tricky to build

- Xref response shape groups outgoing refs by target symbol and kind. The provider now reconstructs that grouping in TypeScript from flat `snapshot_refs` rows.
- Commit-specific snippets are now possible through the same provider method because it accepts a commit ref and resolves it before reading symbol ranges.

### What warrants a second pair of eyes

- Review whether signature snippets should return `signature` rather than the symbol name. The current quick implementation returns the name for `kind === 'signature'`; this should be corrected when snippet metadata is expanded.
- Review whether `conceptsApi` should be removed from the UI entirely or implemented as SQL-backed saved queries later.

### What should be done in the future

- Implement snippet/source ref queries from `snapshot_refs` so linked code navigation works again.
- Implement file xref SQL response.
- Remove or simplify UI routes for query concepts if they are not part of the static-only product.

### Code review instructions

- Review `ui/src/api/sourceApi.ts` and `ui/src/api/xrefApi.ts` for removal of WASM/fetch fallbacks.
- Review `ui/src/api/sqlJsQueryProvider.ts` for source/snippet/xref methods.
- Review `DocSnippet` and `AnnotationWidget` to confirm `/api/snippet` is gone.
- Validate with `pnpm -C ui run typecheck`.

### Technical details

Source/snippet path:

```text
getSource(path)
  -> snapshot_files by HEAD/path
  -> file_contents by content hash
  -> TextDecoder

getSnippet(sym, kind, commit)
  -> snapshot_symbols + snapshot_files
  -> file_contents
  -> byte range extraction
```

## Step 9: End-of-day handoff — static-only sql.js cutover status

This is the end-of-day handoff entry. Today we moved GCB-015 from design into implementation and made a substantial clean-cut transition toward the sql.js-only runtime. The main pattern is now established: Go packages `db/codebase.db`, React loads it with sql.js, and the frontend APIs increasingly call `SqlJsQueryProvider` directly rather than old Go server endpoints, runtime-mode wrappers, TinyGo WASM, or precomputed JSON.

We did not finish the full migration. The most important thing for tomorrow is to continue from the current partially migrated state, not to reintroduce compatibility paths. The app should keep moving toward one runtime path: `React -> SqlJsQueryProvider -> sql.js -> db/codebase.db`.

### Prompt Context

**User prompt (verbatim):** "Time to write a diary up to now. it's the end of the day and we should write down what we were doing and learned for tomorrow, along with a detailed list of th ethings to read to catch back up on, and what to do next. then stop."

**Assistant interpretation:** Stop implementation work, write a detailed handoff diary entry summarizing progress, lessons, files to read, current state, blockers, and next steps for tomorrow.

**Inferred user intent:** Preserve context at the end of the day so the next session can resume safely without rediscovering decisions or current partial state.

**Commit (code):** pending at time of writing — end-of-day diary and sql-wasm-browser asset should be committed together if no further code changes are made.

### What I did today

#### Ticket/docs setup

- Expanded `GCB-015` tasks into detailed, numbered implementation phases.
- Created this implementation diary:
  - `ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/reference/01-implementation-diary.md`
- Updated changelog after each focused implementation slice.
- Kept `docmgr doctor --ticket GCB-015 --stale-after 30` passing after documented steps.

#### Step 1 — sql.js bootstrap

- Added frontend dependencies:
  - `sql.js`
  - `@types/sql.js`
- Added sql.js WASM asset initially:
  - `ui/public/sql-wasm.wasm`
- Added DB bootstrap helpers:
  - `ui/src/api/sqljs/sqlJsDb.ts`
  - `ui/src/api/sqljs/sqlRows.ts`
- Added helpers for:
  - singleton sql.js initialization;
  - loading `manifest.json`;
  - loading `db/codebase.db`;
  - prepared-statement row helpers;
  - SQLite BLOB to `Uint8Array` / text conversion;
  - byte-offset UTF-8 extraction.

#### Step 2 — static export packaging

- Added new Go package:
  - `internal/staticapp`
- Added:
  - `internal/staticapp/manifest.go`
  - `internal/staticapp/export.go`
- Refactored:
  - `cmd/codebase-browser/cmds/review/export.go`
- New export behavior:
  - builds SPA;
  - copies SPA assets;
  - copies SQLite DB to `OUT/db/codebase.db`;
  - writes `OUT/manifest.json`;
  - declares `runtime.hasGoRuntimeServer=false`;
  - no longer writes `precomputed.json` as the target runtime artifact;
  - no longer copies TinyGo `search.wasm` / `wasm_exec.js` as the target runtime path.

#### Step 3 — rendered review docs in SQLite

- Added:
  - `internal/staticapp/reviewdocs.go`
- Added copied-output-DB table:

```sql
CREATE TABLE IF NOT EXISTS static_review_rendered_docs (
    slug TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    html TEXT NOT NULL,
    snippets_json TEXT NOT NULL DEFAULT '[]',
    errors_json TEXT NOT NULL DEFAULT '[]',
    rendered_at INTEGER NOT NULL DEFAULT 0
);
```

- `review export` now renders review markdown into the copied `db/codebase.db` during export.
- This removes the need for a runtime Go review-doc rendering server.

#### Step 4 — first SqlJsQueryProvider

- Added:
  - `ui/src/api/queryErrors.ts`
  - `ui/src/api/sqlJsQueryProvider.ts`
- Initially added then later removed wrapper file:
  - `ui/src/api/queryProvider.ts`
- Rewrote:
  - `ui/src/api/historyApi.ts`
- Implemented SQL-backed methods for:
  - `listCommits()`;
  - `resolveCommitRef()`;
  - `getCommit()`;
  - `getSymbolHistory()`;
  - `getSymbolBodyDiff()`.
- Body diffs now conceptually come from:
  - `snapshot_symbols` byte ranges;
  - `snapshot_files` content hash;
  - `file_contents` BLOBs;
  - byte slicing before UTF-8 decoding.

#### Step 5 — SQL commit diffs and impact

- Extended `SqlJsQueryProvider` with:
  - `getCommitDiff()`;
  - refs queries;
  - `getImpact()` BFS.
- Replaced SQLite-incompatible Go `FULL OUTER JOIN` diff idea with SQLite-compatible `UNION ALL` queries.
- Impact now computes on demand over `snapshot_refs`, not from precomputed impact maps.

#### Step 6 — clean-cut provider and review docs

- Removed wrapper/runtime-mode files:
  - `ui/src/api/queryProvider.ts`
  - `ui/src/api/runtimeMode.ts`
- Rewrote:
  - `ui/src/api/docApi.ts`
- `docApi` now uses `getSqlJsProvider()` directly.
- Removed server and WASM fallbacks from review doc APIs.
- Added SQL provider methods:
  - `listReviewDocs()`;
  - `getReviewDoc(slug)`.

#### Step 7 — index/package/symbol/search APIs to SQL

- Rewrote:
  - `ui/src/api/indexApi.ts`
- Extended `SqlJsQueryProvider` with:
  - `getIndex()`;
  - `getPackageLites()`;
  - `getSymbol(id)`;
  - `searchSymbols(query, kind)`.
- Reconstructs latest snapshot in TypeScript from SQL:
  - packages;
  - files;
  - symbols;
  - package `fileIds` / `symbolIds`.

#### Step 8 — remove frontend WASM runtime APIs

- Deleted:
  - `ui/src/api/wasmClient.ts`
- Rewrote:
  - `ui/src/api/sourceApi.ts`
  - `ui/src/api/xrefApi.ts`
  - `ui/src/api/conceptsApi.ts`
- Updated:
  - `ui/src/features/doc/DocSnippet.tsx`
  - `ui/src/features/doc/widgets/AnnotationWidget.tsx`
- Source/snippet/xref APIs now use `SqlJsQueryProvider`.
- Remaining limitations from this step:
  - snippet refs return empty arrays;
  - source refs return empty arrays;
  - file xref returns empty structure;
  - query concepts are disabled/unavailable in static runtime for now.

#### End-of-day validation attempt

Ran:

```bash
go test ./internal/staticapp ./cmd/codebase-browser
pnpm -C ui run typecheck
rm -f /tmp/gcb015-sqljs-smoke.db
go run ./cmd/codebase-browser review index \
  --commits HEAD~2..HEAD \
  --docs /tmp/reviews/static-smoke.md \
  --db /tmp/gcb015-sqljs-smoke.db
rm -rf /tmp/gcb015-sqljs-export
go run ./cmd/codebase-browser review export \
  --db /tmp/gcb015-sqljs-smoke.db \
  --out /tmp/gcb015-sqljs-export
```

These passed through export.

Then served the export and opened:

```text
http://localhost:8781/#/review/static-smoke
```

Playwright/browser console showed sql.js failing to load:

```text
Failed to load resource: the server responded with a status of 404 (File not found) @ http://localhost:8781/sql-wasm-browser.wasm:0
wasm streaming compile failed: TypeError: Failed to execute 'compile' on 'WebAssembly': HTTP status code is not ok
falling back to ArrayBuffer instantiation
failed to asynchronously prepare wasm: both async and sync fetching of the wasm failed
RuntimeError: Aborted(both async and sync fetching of the wasm failed). Build with -sASSERTIONS for more info.
```

I then discovered `sql.js/dist` contains both:

```text
sql-wasm.wasm
sql-wasm-browser.wasm
```

The bundled JS requested `sql-wasm-browser.wasm`, not only `sql-wasm.wasm`, so I copied:

```bash
cp ui/node_modules/sql.js/dist/sql-wasm-browser.wasm ui/public/sql-wasm-browser.wasm
```

I rebuilt/exported and confirmed both files are now present in the export:

```text
/tmp/gcb015-sqljs-export/sql-wasm-browser.wasm
/tmp/gcb015-sqljs-export/sql-wasm.wasm
```

I did **not** rerun the browser validation after adding `sql-wasm-browser.wasm`. That is the first thing to do tomorrow.

### What worked today

- The clean-cut architecture is now reflected in actual code, not only docs.
- TypeScript typecheck passed after each frontend migration slice.
- Go package checks/builds passed for the new `internal/staticapp` path.
- `review export` now produces the new structural layout:

```text
export/
  index.html
  assets/
  manifest.json
  db/codebase.db
  sql-wasm.wasm
  sql-wasm-browser.wasm  # added at end of day, needs browser re-test
```

- Review docs are rendered into SQLite table `static_review_rendered_docs`.
- The frontend no longer has `wasmClient.ts`.
- The frontend no longer has `runtimeMode.ts`.
- The frontend no longer has the temporary `queryProvider.ts` wrapper.
- The visible APIs that previously used old server/TinyGo paths were mostly moved to `SqlJsQueryProvider`.

### What failed or remains incomplete

#### Browser validation failed before `sql-wasm-browser.wasm` was added

The browser requested `sql-wasm-browser.wasm`; only `sql-wasm.wasm` had been copied initially. This produced a hard sql.js initialization failure.

Current follow-up:

- `ui/public/sql-wasm-browser.wasm` has been added but not yet committed or browser-validated.
- Tomorrow, rebuild/export/serve/open again and confirm sql.js initializes.

#### Some SQL provider methods are intentionally minimal

Current placeholders/limitations:

- `getSnippetRefs()` returns `[]`.
- `getSourceRefs()` returns `[]`.
- `getFileXref()` returns empty file xref structure.
- `conceptsApi` returns empty/unavailable responses.
- `getSnippet(kind='signature')` currently returns symbol name, not full signature. This is a known buglet from the quick source cutover and should be fixed.

#### App shell may reveal more SQL shape bugs after sql.js loads

Because the browser did not get past sql.js initialization, we have not yet seen whether all queries match actual DB column/value shapes. Expect tomorrow's first real browser run to expose some SQL/runtime bugs.

### What I learned

- The sql.js NPM package can request `sql-wasm-browser.wasm` depending on the bundled module path. Copying only `sql-wasm.wasm` is not enough for the current Vite bundle.
- Removing compatibility layers early is possible but forces decisive migration ordering. This is good for the target architecture but means intermediate commits can temporarily leave routes partially functional.
- The database schema is rich enough for most UI behavior. We do not need precomputed JSON for commits, symbols, history, body diffs, basic snippets, xrefs, or impact.
- The old `precomputed.json` path was mostly a workaround for not having a browser SQL engine. With sql.js, it should stay removed from the target runtime.

### What was tricky

#### Byte offsets vs JavaScript strings

Go symbol ranges are byte offsets. JavaScript strings are UTF-16 indexed. For body snippets and source extraction, the safe path is:

```text
SQLite BLOB -> Uint8Array -> slice byte offsets -> TextDecoder
```

Do not decode the whole file to a string and then slice by Go offsets.

#### SQLite commit diff semantics

The Go diff reference used `FULL OUTER JOIN`, but SQLite does not support it. The browser provider uses three-part `UNION ALL` queries for:

- added;
- removed;
- modified/moved/signature-changed.

Review parameter ordering carefully if debugging diffs tomorrow.

#### Clean cutover means no hiding behind fallback behavior

Because we removed server/WASM fallbacks, any missing SQL method now shows up directly. That is desired, but it means tomorrow's browser test may reveal several honest missing pieces.

### Things to read tomorrow to catch up

Read these in order:

1. **This diary from Step 6 onward**
   - Especially Step 8 and this Step 9.
   - Focus on what has already been removed and what intentionally returns empty data.

2. **Design doc v2**
   - `ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/design-doc/01-sql-js-static-frontend-architecture-and-implementation-guide.md`
   - Re-read the v2 static-only runtime section.

3. **Task list**
   - `ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md`
   - Check which tasks were marked done today and which ones remain open.

4. **Go export package**
   - `internal/staticapp/export.go`
   - `internal/staticapp/manifest.go`
   - `internal/staticapp/reviewdocs.go`
   - Understand the export layout and review-doc table generation.

5. **CLI wrapper**
   - `cmd/codebase-browser/cmds/review/export.go`
   - Confirm flags and defaults.

6. **sql.js bootstrap**
   - `ui/src/api/sqljs/sqlJsDb.ts`
   - `ui/src/api/sqljs/sqlRows.ts`
   - Pay attention to `locateFile` and WASM asset names.

7. **Main provider**
   - `ui/src/api/sqlJsQueryProvider.ts`
   - This is now the central runtime data access file.
   - Review methods in this order:
     - `listCommits`
     - `resolveCommitRef`
     - `getIndex`
     - `getSymbol`
     - `getSymbolHistory`
     - `getSymbolBodyDiff`
     - `getCommitDiff`
     - `getImpact`
     - `getSource`
     - `getSnippet`
     - `getXref`
     - `listReviewDocs`
     - `getReviewDoc`

8. **Frontend APIs now calling provider**
   - `ui/src/api/historyApi.ts`
   - `ui/src/api/docApi.ts`
   - `ui/src/api/indexApi.ts`
   - `ui/src/api/sourceApi.ts`
   - `ui/src/api/xrefApi.ts`
   - `ui/src/api/conceptsApi.ts`

9. **Widgets touched for `/api/snippet` removal**
   - `ui/src/features/doc/DocSnippet.tsx`
   - `ui/src/features/doc/widgets/AnnotationWidget.tsx`

10. **Schema references**
    - `internal/history/schema.go`
    - `internal/review/schema.go`
    - Use these when debugging SQL column names.

### What to do first tomorrow

1. Commit or verify the current uncommitted asset:

```text
ui/public/sql-wasm-browser.wasm
```

2. Re-run baseline validation:

```bash
go test ./internal/staticapp ./cmd/codebase-browser
pnpm -C ui run typecheck
```

3. Rebuild/export the smoke bundle:

```bash
rm -rf /tmp/gcb015-sqljs-export
go run ./cmd/codebase-browser review export \
  --db /tmp/gcb015-sqljs-smoke.db \
  --out /tmp/gcb015-sqljs-export
```

If `/tmp/gcb015-sqljs-smoke.db` no longer exists, recreate it:

```bash
rm -f /tmp/gcb015-sqljs-smoke.db
go run ./cmd/codebase-browser review index \
  --commits HEAD~2..HEAD \
  --docs /tmp/reviews/static-smoke.md \
  --db /tmp/gcb015-sqljs-smoke.db
```

4. Serve and open:

```bash
cd /tmp/gcb015-sqljs-export
python3 -m http.server 8781
```

Open:

```text
http://localhost:8781/#/review/static-smoke
```

5. In browser/Playwright, check:

- Does sql.js initialize now that `sql-wasm-browser.wasm` is present?
- Are there any `/api/*` requests?
- Does the review doc load from `static_review_rendered_docs`?
- Do widgets render?
- What SQL errors appear first?

6. Then open direct history route:

```text
http://localhost:8781/#/history?symbol=sym:github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.Register
```

Check whether body diffs load from SQL. This is the original failure class we are trying to eliminate.

### Concrete next implementation tasks

Recommended order:

1. **Fix sql.js WASM asset resolution if still broken.**
   - Confirm whether `locateFile` needs to explicitly return `sql-wasm-browser.wasm` as well as `sql-wasm.wasm`.
   - Current asset has been copied but not browser-tested.

2. **Fix first SQL runtime errors from browser validation.**
   - Expect possible column alias/type problems in `SqlJsQueryProvider`.
   - Use `sqlite3 /tmp/gcb015-sqljs-export/db/codebase.db` to verify queries.

3. **Fix `getSnippet(kind='signature')`.**
   - It should return signature, not name.
   - Body meta currently does not select signature, so either extend metadata or call `getSymbol`.

4. **Implement snippet/source refs or decide UI degradation.**
   - Current linked code receives empty refs.
   - This is acceptable for first static SQL boot but should be tracked.

5. **Implement file xref SQL.**
   - `FileXrefPanel` currently receives empty data.

6. **Decide what to do with query concepts.**
   - Static-only unavailable state may be fine.
   - If not part of product, consider removing the route/sidebar link rather than keeping a dead page.

7. **Delete/trim remaining TinyGo static export code.**
   - Search for old WASM review exports in Go.
   - Be careful: some generated embedded files may still reference old source until regeneration.

8. **Run browser regression and then commit.**
   - Only commit after the smoke page reaches a meaningful state or after a focused fix with clear diary entry.

### Current git/worktree note

At the time this handoff was written, the only known uncommitted code artifact should be:

```text
ui/public/sql-wasm-browser.wasm
```

This file was added after the browser reported a missing `sql-wasm-browser.wasm`. It should be committed with this diary entry if validation/intent is accepted.

### Review warnings for tomorrow

- Do not reintroduce `wasmClient.ts`.
- Do not reintroduce `runtimeMode.ts`.
- Do not add `ServerQueryProvider`.
- Do not add `/api/*` fallbacks.
- If a feature is not yet SQL-backed, either implement it through `SqlJsQueryProvider` or make it explicitly unavailable in the static-only UI.

## Step 10: Validate sql.js static smoke and delete `review serve`

This step picked up from the end-of-day handoff. I first reran the static export/browser smoke after adding `sql-wasm-browser.wasm`, then removed the old `review serve` runtime command and the review-specific Go HTTP server wrapper.

This is a clean-cut deletion: `review` now indexes and exports, but does not run a Go review server. The exported browser is the runtime surface for review docs and history widgets.

### Prompt Context

**User prompt (verbatim):** "read the diary of GCB-015, catch up, and then continue workign on the tasks, committing at appropriat einterval, and keeping a detailed diary."

**Assistant interpretation:** Re-read the GCB-015 diary/task context, resume from the handoff, validate the sql.js static path, continue focused implementation/deletion tasks, commit, and record the work.

**Inferred user intent:** Continue the static-only migration without losing context or leaving undocumented/deprecated paths behind.

**Commit (code):** 45de723 — "Remove review serve runtime path"

### What I did

- Read the GCB-015 implementation diary, tasks, and design doc context.
- Reran baseline validation:
  - `go test ./internal/staticapp ./cmd/codebase-browser`
  - `pnpm -C ui run typecheck`
- Recreated a smoke DB and static export:
  - `go run ./cmd/codebase-browser review index --commits HEAD~2..HEAD --docs /tmp/reviews/static-smoke.md --db /tmp/gcb015-sqljs-smoke.db`
  - `go run ./cmd/codebase-browser review export --db /tmp/gcb015-sqljs-smoke.db --out /tmp/gcb015-sqljs-export`
- Served the export with `python3 -m http.server 8781`.
- Reopened `http://localhost:8781/#/review/static-smoke` in Playwright.
- Confirmed that after reload, sql.js initialized successfully and browser console had no errors.
- Confirmed no `/api/*` network requests were made.
- Opened the direct history route:
  - `http://localhost:8781/#/history?symbol=sym:github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.Register`
- Confirmed the history page rendered a body diff from SQL without `STATIC_NOT_PRECOMPUTED`.
- Removed `review serve` from the review command tree:
  - deleted `cmd/codebase-browser/cmds/review/serve.go`
  - removed `newServeCmd()` registration from `cmd/codebase-browser/cmds/review/root.go`
- Deleted the review-specific HTTP wrapper:
  - `internal/review/server.go`
- Added `cmd/codebase-browser/cmds/review/patterns.go` to keep `defaultPatterns()` for `review index` and `review db create` after deleting `serve.go`.
- Updated help docs away from server mode:
  - `docs/help/review-user-guide.md`
  - `docs/help/review-reference.md`
- Verified the command list no longer includes `review serve`:
  - `go run ./cmd/codebase-browser review --help`
- Ran validation after deletion:
  - `gofmt -w cmd/codebase-browser/cmds/review/root.go cmd/codebase-browser/cmds/review/index.go cmd/codebase-browser/cmds/review/patterns.go`
  - `go test ./internal/review ./cmd/codebase-browser`
  - `go build ./cmd/codebase-browser`
  - `pnpm -C ui run typecheck`
- Marked completed tasks in `tasks.md`, including:
  - T05.7
  - T07.7
  - T08.6–T08.9
  - T09.7
  - T09.8
  - T10.8
  - T10.10
  - T11.2
  - T11.3

### Why

- The ticket target is static-only: Go indexes/exports, then exits.
- `review serve` kept an explicit runtime Go server path and `/api/review/*` routes alive.
- The static smoke showed sql.js now loads and can answer the original generic history/body-diff route, so deleting the old review runtime path is appropriate.

### What worked

- Adding `sql-wasm-browser.wasm` fixed the previous sql.js initialization failure once the page was reloaded against the newly exported bundle.
- The review doc page loaded from the exported static DB.
- The direct history route rendered the `Register` symbol history and body diff without `STATIC_NOT_PRECOMPUTED`.
- Playwright network inspection found no `/api/*` requests during the checked static pages.
- `review --help` now lists only:
  - `db`
  - `export`
  - `index`

### What didn't work

- The first Playwright navigation still showed old console errors from the previous missing-WASM load. A manual reload of the page after confirming the file existed produced a clean console. This appears to have been browser/session cache state rather than an export problem.
- The repository's embedded source/index/static snapshots still contain old generated references to `review serve` and `NewReviewServer` under paths such as `internal/sourcefs/embed/source/...` and `internal/indexfs/embed/index.json`. I did not regenerate or delete those generated artifacts in this step because the code build and command surface are already clean. This should be decided as a separate generated-artifact cleanup.

### What I learned

- `sql-wasm-browser.wasm` is required by the current Vite/sql.js bundle even though `sql-wasm.wasm` is also present.
- The original `STATIC_NOT_PRECOMPUTED` class of failure is addressed for the tested direct history route: the body diff is now computed from SQLite snapshots and BLOBs in the browser.
- Removing `serve.go` also removed the only definition of `defaultPatterns()`, so a tiny `patterns.go` helper was needed for `review index` and `review db create`.

### What was tricky to build

- The deletion was mostly straightforward, but the stale generated embeds made grep output noisy. Actual source references in `cmd`, `internal/review`, and active help docs are gone, while generated snapshots still reflect older repository state.
- Browser console output needed care: the first console read still included previous errors, but `page.reload({ waitUntil: 'networkidle' })` against the current export produced zero errors.

### What warrants a second pair of eyes

- Decide whether `internal/sourcefs/embed/source/*`, `internal/indexfs/embed/index.json`, and old `internal/sourcefs/embed/source/internal/static/embed/precomputed.json` should be regenerated, deleted, or ignored during this migration. They still mention deleted server paths because they are snapshots.
- Review whether the root `serve` command outside the `review` tree should remain for the older embedded-index browser. This step only removed `review serve`, not the general `serve` command.
- Review the help docs to ensure the static-export workflow is now the only documented review workflow.

### What should be done in the future

- Fix or explicitly defer the remaining placeholder SQL provider methods:
  - snippet refs;
  - source refs;
  - file xref;
  - query concepts.
- Fix `getSnippet(kind='signature')` to return the actual signature.
- Remove old TinyGo/static review export code and `PrecomputedReview` model once no build/runtime path uses it.
- Add real Playwright regression tests for the static review doc route and direct history route.

### Code review instructions

- Start with `cmd/codebase-browser/cmds/review/root.go` to confirm `newServeCmd()` is gone.
- Review `cmd/codebase-browser/cmds/review/patterns.go` to confirm the moved helper is unchanged.
- Review deleted files:
  - `cmd/codebase-browser/cmds/review/serve.go`
  - `internal/review/server.go`
- Review help docs:
  - `docs/help/review-user-guide.md`
  - `docs/help/review-reference.md`
- Validate with:
  - `go test ./internal/review ./cmd/codebase-browser`
  - `go build ./cmd/codebase-browser`
  - `pnpm -C ui run typecheck`
  - `go run ./cmd/codebase-browser review --help`

### Technical details

Static smoke commands used:

```bash
go test ./internal/staticapp ./cmd/codebase-browser
pnpm -C ui run typecheck
rm -f /tmp/gcb015-sqljs-smoke.db
go run ./cmd/codebase-browser review index \
  --commits HEAD~2..HEAD \
  --docs /tmp/reviews/static-smoke.md \
  --db /tmp/gcb015-sqljs-smoke.db
rm -rf /tmp/gcb015-sqljs-export
go run ./cmd/codebase-browser review export \
  --db /tmp/gcb015-sqljs-smoke.db \
  --out /tmp/gcb015-sqljs-export
cd /tmp/gcb015-sqljs-export
python3 -m http.server 8781
```

Validated routes:

```text
http://localhost:8781/#/review/static-smoke
http://localhost:8781/#/history?symbol=sym:github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.Register
```

## Step 11: Remove TinyGo review-data runtime model

This step removes the old review-specific TinyGo/precomputed runtime model now that the visible review/history/static smoke path is SQL-backed. The generic TinyGo package still exists for older generated/embedded paths, but it no longer accepts or exposes `reviewData`.

The important architectural deletion is that `PrecomputedReview` and the WASM `ReviewData` object are gone. The static review browser no longer has a parallel review-data model; review docs, histories, diffs, body diffs, and impact are expected to come from SQLite through `SqlJsQueryProvider`.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Continue focused clean-cut migration after validating the sql.js static runtime smoke.

**Inferred user intent:** Delete old precomputed/WASM review runtime code rather than keeping it as a fallback.

**Commit (code):** 12d31ec — "Remove TinyGo review data runtime model"

### What I did

- Deleted the old review precompute builder:
  - `internal/review/export.go`
  - `internal/review/export_test.go`
- Deleted the WASM review-data type file:
  - `internal/wasm/review_types.go`
- Removed `ReviewData` from `internal/wasm.SearchCtx`.
- Changed `internal/wasm.Init(...)` from seven JSON inputs to six JSON inputs by removing `jsonReviewData`.
- Removed review-data unmarshalling from `internal/wasm/search.go`.
- Removed review-specific WASM query methods from `internal/wasm/search.go`:
  - `GetCommitDiff`
  - `GetSymbolHistory`
  - `GetImpact`
  - `GetSymbolBodyDiff`
  - `GetReviewDocs`
  - `GetReviewDoc`
  - `GetCommits`
- Removed review-specific JS exports from `internal/wasm/exports.go`.
- Removed the unused `strconv` import from `internal/wasm/exports.go`.
- Removed the old Go WASM loader script from the frontend public directory:
  - `ui/public/wasm_exec.js`
- Removed the hard-coded `<script src="/wasm_exec.js"></script>` from `ui/index.html`.
- Marked T09.4 and T09.6 complete in `tasks.md`.

### Why

- The target static runtime is sql.js, not TinyGo reviewData.
- Keeping `PrecomputedReview` after the frontend cutover would invite accidental fallback behavior and confuse future maintenance.
- The static export no longer writes `precomputed.json`, so the review precompute builder was dead runtime code.
- Vite was still copying `wasm_exec.js` from `ui/public`; deleting it ensures the exported sql.js app does not ship an unused Go WASM loader.

### What worked

- `go test ./internal/review ./internal/wasm ./internal/staticapp ./cmd/codebase-browser` passed.
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` passed after removing review exports.
- `pnpm -C ui run typecheck` passed.
- `review export` still succeeded.
- The exported static bundle no longer contains:
  - `wasm_exec.js`
  - `precomputed.json`
- Playwright opened `http://localhost:8781/#/review/static-smoke` successfully after this deletion.
- Browser console had zero errors after the deletion.
- Network inspection still showed no `/api/*` requests.

### What didn't work

I tried to run:

```bash
GOOS=js GOARCH=wasm go test ./internal/wasm
```

It failed with:

```text
fork/exec /tmp/go-build3220019683/b001/wasm.test: exec format error
FAIL	github.com/wesen/codebase-browser/internal/wasm	0.001s
FAIL
```

This is expected for a plain host shell: Go built a JS/WASM test binary and then tried to execute it without a wasm test runner. I switched to:

```bash
GOOS=js GOARCH=wasm go build ./cmd/wasm
```

which validates that the WASM target still compiles without trying to execute it.

### What I learned

- The review-specific TinyGo exports were fully disconnected from the active frontend after the SQL provider cutover; removing them did not affect the static smoke.
- `ui/public` is copied by Vite into the export, so stale public assets are runtime artifacts even if no code imports them.
- A normal `go test` command is not a valid execution test for `GOOS=js GOARCH=wasm` unless a wasm execution environment is configured.

### What was tricky to build

- The generic `internal/wasm` package still contains older index/search/doc-page functionality, so this was not a wholesale TinyGo removal. The safe deletion boundary for this step was review-specific data and exports only.
- `exports.go` and `search.go` had to be edited together because the JS export layer called methods removed from `SearchCtx`.

### What warrants a second pair of eyes

- Decide whether to delete the remaining generic TinyGo path entirely in a later commit. Current active static export does not use it, but old generators and embedded assets still reference it.
- Review whether `internal/bundle/generate_build.go`, `internal/web/generate_build.go`, and `internal/static/generate_build.go` should be removed or rewritten to avoid old `search.wasm` / `precomputed.json` behavior.
- Confirm that deleting `ui/public/wasm_exec.js` is acceptable for every remaining frontend build mode.

### What should be done in the future

- Remove or rewrite old static/bundle/web generators that still copy `search.wasm`, `wasm_exec.js`, or `precomputed.json` if they are no longer used.
- Regenerate or retire stale embedded source/index artifacts that still contain old review server and precomputed runtime references.
- Add explicit tests that `review export` output does not contain `wasm_exec.js`, `search.wasm`, or `precomputed.json`.

### Code review instructions

- Review deleted files first:
  - `internal/review/export.go`
  - `internal/review/export_test.go`
  - `internal/wasm/review_types.go`
  - `ui/public/wasm_exec.js`
- Then review `internal/wasm/search.go` and `internal/wasm/exports.go` to confirm only review-data code was removed.
- Review `ui/index.html` to confirm the Go WASM script tag is gone.
- Validate with:
  - `go test ./internal/review ./internal/wasm ./internal/staticapp ./cmd/codebase-browser`
  - `GOOS=js GOARCH=wasm go build ./cmd/wasm`
  - `pnpm -C ui run typecheck`
  - `go run ./cmd/codebase-browser review export --db /tmp/gcb015-sqljs-smoke.db --out /tmp/gcb015-sqljs-export`
  - `test ! -f /tmp/gcb015-sqljs-export/wasm_exec.js`
  - `test ! -f /tmp/gcb015-sqljs-export/precomputed.json`

### Technical details

Active static export after this step is expected to contain sql.js assets only:

```text
manifest.json
db/codebase.db
sql-wasm.wasm
sql-wasm-browser.wasm
assets/*.js
assets/*.css
```

The old review-data initialization shape is gone:

```text
before: initWasm(index, search, xref, snippets, docManifest, docHTML, reviewData)
after:  initWasm(index, search, xref, snippets, docManifest, docHTML)
```

## Step 12: Add static export layout and rendered review-doc tests

This step adds the first focused Go tests for the new `internal/staticapp` export path. The tests cover the two most important Go-side invariants of the sql.js architecture: static export layout/manifest generation and export-time rendered review docs.

The tests also lock in the clean-cut runtime deletion from the previous steps by asserting that `review export` does not emit legacy TinyGo/precomputed runtime files.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Continue implementation by adding validation around the static-only export path after deleting old runtime code.

**Inferred user intent:** Make the new sql.js/static-only behavior safer to change and easier to review.

**Commit (code):** 35045da — "Test static sql.js export layout"

### What I did

- Added `internal/staticapp/export_test.go`.
- Added `TestExportCopiesDBWritesManifestAndOmitsLegacyRuntimeFiles`:
  - creates a minimal fixture review/history SQLite DB;
  - creates a fake `ui/dist/public` tree in a temp working directory;
  - runs `Export` with `BuildSPA=false`;
  - asserts `index.html` and `db/codebase.db` are copied;
  - asserts legacy runtime files are absent:
    - `precomputed.json`
    - `search.wasm`
    - `wasm_exec.js`
  - asserts `manifest.json` has:
    - `kind = codebase-browser-sqljs-static-export`
    - `db.path = db/codebase.db`
    - `runtime.queryEngine = sql.js`
    - `runtime.hasGoRuntimeServer = false`
- Added `TestAddRenderedReviewDocsCreatesStaticTableOnCopiedDB`:
  - creates a fixture DB with one commit, one minimal snapshot, and one review doc;
  - runs `AddRenderedReviewDocs`;
  - asserts `static_review_rendered_docs` contains the rendered `fixture` row;
  - asserts `snippets_json` and `errors_json` are normalized to `[]`.
- Marked T10.1 and T10.2 complete.

### Why

- The static export layout is now central to the product. It should be protected by tests rather than only smoke scripts.
- The clean-cut removal of TinyGo/precomputed runtime artifacts should be an explicit invariant.
- `static_review_rendered_docs` is the bridge between Go markdown rendering and browser sql.js review loading, so it deserves a unit/integration test.

### What worked

- `go test ./internal/staticapp ./internal/review ./internal/wasm ./cmd/codebase-browser` passed.
- `pnpm -C ui run typecheck` passed.
- The export test successfully runs without invoking the real Vite build by using `BuildSPA=false` and a fake `ui/dist/public` tree.

### What didn't work

The first version of the rendered-doc test inserted markdown content through a raw SQL string with `\n` escapes inside a raw Go string. SQLite stored literal backslash-n characters, so the markdown renderer treated the whole line as the heading and returned:

```text
title = "Fixture Review\\n\\nPlain text."
```

The test failed with:

```text
--- FAIL: TestAddRenderedReviewDocsCreatesStaticTableOnCopiedDB (0.01s)
    export_test.go:85: title = "Fixture Review\\n\\nPlain text."
```

I fixed it by inserting the markdown content as a SQL parameter with real Go newline characters:

```go
db.Exec(`INSERT INTO review_docs(...) VALUES (?, ?, ?, ?, '{}', 100)`,
    "fixture", "Fixture Review", "fixture.md", "# Fixture Review\n\nPlain text.")
```

### What I learned

- Fixture SQL should use parameters for multiline markdown content to avoid confusing raw-string escaping with actual newlines.
- `Export` can be tested without a real frontend build by changing the process working directory to a temp tree containing `ui/dist/public`.
- The absence of old runtime files is easy to assert and should remain part of the regression suite.

### What was tricky to build

- `Export` currently reads `ui/dist/public` relative to the process working directory. The test therefore needs a temporary working directory and cleanup restore. This works, but it highlights that `Export` could eventually accept a SPA build path for easier testing.
- `AddRenderedReviewDocs` calls `review.LoadLatestSnapshot`, so the test DB needs a minimal but valid commit/package/file/content snapshot even though the review markdown itself is simple.

### What warrants a second pair of eyes

- Review whether fixture DB construction should move to a reusable helper for future staticapp/history/sql.js tests.
- Review whether `Export` should accept an explicit `SPAPath` option instead of hard-coding `ui/dist/public`.
- Review whether asserting absence of `search.wasm` is too strict while generic TinyGo artifacts still exist elsewhere in the repository. For `review export`, it should be strict.

### What should be done in the future

- Add TypeScript tests for `resolveCommitRef` and BLOB byte slicing.
- Add Playwright regression tests for zero `/api/*` and the direct history route.
- Add export tests that inspect `static_review_rendered_docs` after a full `Export(RenderReviewDocs=true)` call.

### Code review instructions

- Review `internal/staticapp/export_test.go`.
- Start with the two test functions, then inspect the fixture helpers:
  - `createStaticAppFixtureDB`
  - `writeFakeSPABuild`
  - `withWorkingDir`
- Validate with:
  - `go test ./internal/staticapp ./internal/review ./internal/wasm ./cmd/codebase-browser`
  - `pnpm -C ui run typecheck`

### Technical details

The key runtime-file invariant is now encoded directly in the export test:

```go
for _, legacy := range []string{"precomputed.json", "search.wasm", "wasm_exec.js"} {
    if _, err := os.Stat(filepath.Join(outDir, legacy)); err == nil {
        t.Fatalf("legacy runtime file %s should not be exported", legacy)
    }
}
```
