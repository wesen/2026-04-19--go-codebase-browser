# Tasks

## TODO

### Phase 1 — Dependencies and export package shape

- [ ] Add `sql.js` to `ui/package.json` and ensure `sql-wasm.wasm` is available in static builds.
- [ ] Add a static export manifest writer that emits `manifest.json`.
- [ ] Change static export to copy the SQLite database to `db/codebase.db`.
- [ ] Decide whether static export mutates a copied DB or the source DB when adding static-only tables.

### Phase 2 — Go-side static export preparation

- [ ] Create `internal/staticapp` package for sql.js static export packaging.
- [ ] Add manifest Go types and tests.
- [ ] Add optional `static_export_metadata` table population.
- [ ] Add `static_review_rendered_docs` table population using the existing Go markdown/directive renderer.
- [ ] Optionally add `symbol_search_fts` creation/population for faster browser search.
- [ ] Refactor `cmd/codebase-browser/cmds/review/export.go` into a thin call to `staticapp.Export`.

### Phase 3 — Frontend sql.js DB layer

- [ ] Add `ui/src/api/sqljs/sqlJsDb.ts` to initialize sql.js and open `db/codebase.db`.
- [ ] Add `ui/src/api/sqljs/sqlRows.ts` for prepared statements, row conversion, and statement cleanup.
- [ ] Add BLOB/text decoding helpers for `file_contents.content`.
- [ ] Add a basic smoke query that can count commits from the static DB.

### Phase 4 — Query provider abstraction

- [ ] Add `CodebaseQueryProvider` interface.
- [ ] Add `SqlJsQueryProvider` implementation.
- [ ] Do not add a `ServerQueryProvider`; the target runtime has only `SqlJsQueryProvider`.
- [ ] Add a simple provider singleton for the SQL provider; remove runtime-mode branching.
- [ ] Add structured `QueryError` handling and RTK Query result normalization.

### Phase 5 — Core history and body diff support

- [ ] Implement `listCommits` and `resolveCommitRef` in `SqlJsQueryProvider`.
- [ ] Implement `getSymbolHistory` from the `symbol_history` view.
- [ ] Implement `getSymbolBodyDiff` using `snapshot_symbols`, `snapshot_files`, `file_contents`, byte offsets, and JS diff generation.
- [ ] Refactor `ui/src/api/historyApi.ts` to use provider `queryFn` methods instead of static endpoint-string parsing.
- [ ] Verify `/history?symbol=sym:...Register` works in static export without `STATIC_NOT_PRECOMPUTED`.

### Phase 6 — Generic browser SQL coverage

- [ ] Implement package list/package detail queries.
- [ ] Implement symbol lookup and symbols-at-commit queries.
- [ ] Implement source file content queries from `file_contents`.
- [ ] Implement symbol search, initially with LIKE and later with FTS.
- [ ] Implement refs/xrefs queries.
- [ ] Implement commit diff in SQL using `UNION ALL` instead of `FULL OUTER JOIN`.
- [ ] Implement impact BFS in TypeScript over SQL refs.

### Phase 7 — Review renderer SQL coverage

- [ ] Implement `listReviewDocs` and `getReviewDoc` from `static_review_rendered_docs`.
- [ ] Change review placeholder marker to generic `data-codebase-widget`.
- [ ] Add a generic `CodebaseWidget` dispatcher for review widgets.
- [ ] Ensure `codebase-snippet`, `codebase-diff`, `codebase-diff-stats`, `codebase-symbol-history`, and `codebase-impact` call the provider.

### Phase 8 — Remove old TinyGo review transport

- [ ] Remove `reviewData` from WASM initialization.
- [ ] Remove or stop using WASM review exports for commits, histories, diffs, impacts, body diffs, and review docs.
- [ ] Remove `PrecomputedReview` as the static runtime data model.
- [ ] Remove all `/api/history` endpoint parsing from frontend API files; no application API exists in the target runtime.

### Phase 9 — Tests and validation

- [ ] Add Go tests for static manifest/export packaging.
- [ ] Add Go tests for rendered review docs table generation.
- [ ] Add TypeScript tests for sql.js provider commit ref resolution.
- [ ] Add TypeScript tests for body extraction from BLOB content using byte offsets.
- [ ] Add Playwright regression for static review doc rendering with zero `/api/*` requests.
- [ ] Add Playwright regression for direct `/history?symbol=sym:...Register` body diff rendering.

### Phase 10 — Documentation

- [ ] Update Glazed help entries for sql.js static export behavior.
- [ ] Update implementation diary and changelog as work progresses.
- [ ] Upload revised implementation notes to reMarkable after major changes.
