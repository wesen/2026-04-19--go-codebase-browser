# Changelog

## 2026-04-25

- Initial workspace created


## 2026-04-25

Created ticket and wrote full design doc (67KB, 10 sections) covering system architecture, widget catalog with ASCII wireframes, API design, implementation plan, and risk analysis.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/25/GCB-010--embeddable-semantic-diff-widgets-for-literate-pr-code-review-in-markdown/design-doc/01-embeddable-semantic-diff-widgets-design-affordances-and-architecture-for-literate-pr-review.md — Primary design document
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/25/GCB-010--embeddable-semantic-diff-widgets-for-literate-pr-code-review-in-markdown/reference/01-investigation-diary.md — Investigation diary


## 2026-04-25

Uploaded design doc + diary bundle to reMarkable at /ai/2026/04/25/GCB-010


## 2026-04-25

Added Section 11 (incremental vertical slices) to design doc. Rewrote tasks.md to match 6-slice plan (Slice 0: commit= plumbing, Slice 1: diff widget, Slice 2: history, Slice 3: impact+BFS, Slice 4: quick wins, Slice 5: commit walk). Each slice is a complete demonstrable feature with validation steps and decision gates.


## 2026-04-25

Slice 0 implemented: commit= param on existing codebase-snippet/signature directives. 4 commits (server+renderer, frontend, demo page, symbol ID fix). Server running in tmux at :3001 with history.db.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/04-slice0-demo.md — Demo page for slice 0
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go — Added CommitHash to SnippetRef
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_source.go — Extended handleSnippet with commit= param
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocPage.tsx — Extract data-commit from DOM stubs
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocSnippet.tsx — Added useGetSnippetFromCommit hook


## 2026-04-25

Fixed browser validation issues for Slice 0: load wasm_exec.js before app bundle and make docApi prefer live /api/doc endpoints in server-backed mode. Verified with Playwright: 0 console errors and demo page renders at /#/doc/04-slice0-demo.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/index.html — Loads /wasm_exec.js before app bundle
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/public/wasm_exec.js — Go WASM runtime copied into Vite public assets
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/api/docApi.ts — Prefers live server doc API


## 2026-04-25

Added syntax highlighting to commit-resolved snippets and implemented Slice 1 codebase-diff widget. New directive validates sym/from/to, passes JSON params through data-params, hydrates SymbolDiffInlineWidget, and renders highlighted unified diffs from /api/history/symbol-body-diff. Validated with tests, typecheck/build, and Playwright (0 console errors).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/05-slice1-diff-demo.md — Slice 1 demo page
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go — codebase-diff directive and safe data-params emission
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx — New inline diff widget


## 2026-04-25

Implemented Slice 2 codebase-symbol-history widget. New directive renders compact body-hash timeline; clicking a changed row expands SymbolDiffInlineWidget against the predecessor commit. Validated with tests, typecheck/build, and Playwright (0 console errors).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/06-slice2-history-demo.md — Slice 2 demo page
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go — codebase-symbol-history directive
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx — New inline history widget


## 2026-04-25

Implemented Slice 3 codebase-impact widget. Added /api/history/impact BFS endpoint over snapshot_refs, directive support, RTK Query hook, inline grouped impact widget, and Slice 3 demo page. Validated with tests, typecheck/build, and Playwright (0 console errors).

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/07-slice3-impact-demo.md — Slice 3 demo page
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_history.go — /api/history/impact endpoint and BFS helpers
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/ImpactInlineWidget.tsx — New inline impact widget


## 2026-04-25

Polished Slice 3 impact widget links. Impact API now marks local vs external nodes; widget links local rows to history-backed /history?symbol=... so rows like handleConceptDetail resolve even when the static index is stale. External refs render as muted non-links with compact names. Validated with Playwright.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_history.go — Impact nodes now include local flag and external fallback names
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/ImpactInlineWidget.tsx — History-backed links and external-node polish

