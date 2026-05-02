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
