# Changelog

## 2026-04-30

- Initial workspace created


## 2026-04-30

Created ticket and wrote comprehensive 63KB design doc covering architecture, schema, CLI commands, server routes, and Glazed help entries

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/30/GCB-013--code-review-command-index-commit-ranges-serve-markdown-review-guides-and-produce-llm-queryable-sqlite-dbs/design-doc/01-code-review-tool-analysis-design-and-implementation-guide.md — Primary design document


## 2026-04-30

Wrote standalone WASM export design doc — three approaches compared, hybrid A+B recommended, with implementation phases

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/30/GCB-013--code-review-command-index-commit-ranges-serve-markdown-review-guides-and-produce-llm-queryable-sqlite-dbs/design-doc/02-standalone-wasm-export-browser-side-sqlite-queries-for-code-review.md — WASM export design document


## 2026-04-30

Wrote feasibility assessment: TinyGo WASM proven for widgets, pure-Go SQLite in TinyGo not feasible, sql.js remains necessary only for ad-hoc LLM SQL

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/30/GCB-013--code-review-command-index-commit-ranges-serve-markdown-review-guides-and-produce-llm-queryable-sqlite-dbs/design-doc/03-tinygo-vs-sql-js-feasibility-assessment-for-browser-side-sqlite.md — Feasibility assessment document


## 2026-04-30

Created comprehensive task list with 35 tasks across 10 phases — from schema/store foundation through static WASM export to end-to-end testing

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/30/GCB-013--code-review-command-index-commit-ranges-serve-markdown-review-guides-and-produce-llm-queryable-sqlite-dbs/tasks.md — Task list document


## 2026-05-01

Added a static export review document reassessing markdown rendering, WASM query wiring, server-bound API leaks, symbol-reference errors, and required repair plan after browser validation showed /api/* 404s in static export.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/30/GCB-013--code-review-command-index-commit-ranges-serve-markdown-review-guides-and-produce-llm-queryable-sqlite-dbs/reference/02-static-export-review-markdown-wasm-and-api-wiring-assessment.md — New detailed review deliverable
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/docApi.ts — Primary source of static export /api/doc and /api/review/docs probes
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/historyApi.ts — Primary source of static export /api/history calls


## 2026-05-01

Implemented the first static transport repair pass: static exports now set VITE_STATIC_EXPORT, docs skip HTTP probes in static mode, history widgets route diff/history/impact queries through WASM instead of /api/history, and browser validation showed no /api requests for a review doc smoke test.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/review/export.go — Builds SPA with VITE_STATIC_EXPORT=1
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/docApi.ts — Skips doc/review-doc HTTP probes in static mode
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/historyApi.ts — Static-aware history transport using WASM review helpers
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/runtimeMode.ts — Static export runtime mode helper


## 2026-05-01

Added static body-diff support for codebase-diff: review export now precomputes body diffs for changed symbols and explicit codebase-diff snippets, WASM exposes getSymbolBodyDiff, and static historyApi serves symbol-body-diff without /api calls.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/history/bodydiff.go — Safe short-hash errors
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/review/export.go — Precomputes reviewData.bodyDiffs
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/exports.go — JS export getSymbolBodyDiff
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/search.go — WASM body-diff lookup
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/historyApi.ts — Static /symbol-body-diff transport


## 2026-05-01

Enriched static impact payloads: review export now mirrors server BFS with commit, edges, and local/external node metadata, honors explicit codebase-impact dir/depth/commit parameters, and static WASM getImpact supports commit-specific lookups.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/review/export.go — Server-compatible impact BFS and keyed precomputation
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/review_types.go — Impact response includes commit
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/wasm/search.go — Commit-aware getImpact lookup
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/historyApi.ts — Static impact transport passes resolved commit

