# Tasks

## Phase 0 — Ticket setup and upstream docs

- [x] Create GCB-011 ticket workspace
- [x] Download `https://diffs.com/docs` with Defuddle into `sources/`
- [x] Capture Defuddle title/description metadata in `sources/`
- [x] Write initial Diffs adoption design plan

## Phase 1 — First Diffs integration

- [x] Add `@pierre/diffs` to `ui/package.json` and update pnpm lockfile
- [x] Inspect React exports and choose `MultiFileDiff` for current old/new body data
- [x] Prototype a local wrapper around `@pierre/diffs/react`
- [x] Replace `SymbolDiffInlineWidget` rendering with the wrapper
- [x] Replace `SymbolBodyDiffView` in `HistoryPage.tsx` with the wrapper
- [x] Validate `/#/doc/05-slice1-diff-demo`
- [x] Validate `/#/doc/09-slice5-commit-walk-demo` step 3
- [x] Validate `/#/history?symbol=...Server.handleSnippet`

## Phase 2 — Reviewer controls and readability

- [x] Add a visible unified/split diff toggle to the shared Diffs wrapper
- [x] Enable and verify word-level diffs in the shared Diffs wrapper
- [x] Ensure the toggle works in embedded docs, commit-walk steps, and focused symbol history
- [x] Validate no console errors and no fallback renderer on supported diffs

## Phase 3 — Annotation widget redesign

- [x] Redesign `codebase-annotation` as a clear “review note on code” widget
- [x] Move explanatory note above the code so readers know what to look for
- [x] Replace the ugly per-line `<Code>` boxes with one compact code frame and highlighted line rows
- [x] Update demo copy if needed so the annotation’s purpose is obvious
- [x] Validate `/#/doc/08-slice4-quick-wins-demo` and commit-walk annotation step

## Phase 4 — Follow-up polish / performance

- [x] Document theming/performance findings
- [x] Lazy-load the Diffs renderer to keep Diffs/Shiki out of the initial app chunk
- [x] Refresh README screenshots after Diffs UI changes
- [ ] Consider Diffs line annotations for a future richer `codebase-annotation`
- [ ] Consider `PatchDiff` / `parsePatchFiles` for future full file or PR patch views
