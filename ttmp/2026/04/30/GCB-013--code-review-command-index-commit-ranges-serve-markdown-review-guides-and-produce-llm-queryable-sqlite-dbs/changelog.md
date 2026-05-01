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

