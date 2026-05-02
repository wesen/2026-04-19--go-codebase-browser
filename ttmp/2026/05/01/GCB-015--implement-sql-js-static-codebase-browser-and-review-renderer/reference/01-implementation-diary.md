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
