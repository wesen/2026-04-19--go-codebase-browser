---
Title: Investigation diary
Ticket: GCB-011
Status: active
Topics:
  - codebase-browser
  - react-frontend
  - semantic-diff
  - ui-design
  - documentation-tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
  - https://diffs.com/docs
Summary: Chronological diary for the Diffs library adoption ticket.
LastUpdated: 2026-04-25T12:35:00-04:00
WhatFor: Use to resume the Diffs integration work with context about downloaded sources and initial planning.
WhenToUse: Before implementing or reviewing GCB-011.
---

# Investigation diary

## Goal

Track the work to add `@pierre/diffs` support for nicer diff rendering in codebase-browser.

## Context

GCB-010 introduced several hand-rendered diff surfaces:

- `codebase-diff` embedded doc widget
- focused `/history?symbol=...` body diffs
- `codebase-commit-walk` steps that compose the diff widget

Those implementations work, but they are intentionally simple. GCB-011 is the follow-up ticket to integrate a purpose-built diff UI library.

## Step 1: Create ticket and archive upstream docs

Created ticket workspace:

```text
ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/
```

Downloaded upstream docs using Defuddle, as requested:

```bash
defuddle parse https://diffs.com/docs --md \
  -o ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/sources/01-diffs-docs.md

defuddle parse https://diffs.com/docs -p title \
  > ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/sources/00-diffs-docs-title.txt

defuddle parse https://diffs.com/docs -p description \
  > ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/sources/00-diffs-docs-description.txt
```

Captured source files:

- `sources/01-diffs-docs.md` — 475 lines of extracted Markdown docs.
- `sources/00-diffs-docs-title.txt` — `Diffs docs`.
- `sources/00-diffs-docs-description.txt` — short metadata description.

## Step 2: Initial planning notes

Created the design note:

```text
design/01-diffs-library-adoption-plan-for-semantic-diff-widgets.md
```

Key decision in the plan: isolate `@pierre/diffs/react` behind a local wrapper component so the library's early APIs do not spread throughout the codebase. First targets should be `SymbolDiffInlineWidget.tsx` and `SymbolBodyDiffView` in `HistoryPage.tsx`.

## Next steps

1. Install `@pierre/diffs` in the UI package and inspect its TypeScript exports.
2. Prototype a wrapper around `PatchDiff` using existing `unifiedDiff` data.
3. If needed, normalize hunk-only symbol diffs into synthetic patch strings.
4. Replace `SymbolDiffInlineWidget` first.
5. Replace `SymbolBodyDiffView` second.
6. Validate demo pages and focused symbol-history routes with Playwright.

## Step 3: Convert user feedback into implementation phases

The user asked for three concrete improvements after seeing the first Diffs integration direction:

1. Add a unified/split toggle.
2. Make word-level diffs explicit.
3. Fix the annotated snippet widget, which currently looks ugly and does not explain what the reader should see.

I rewrote `tasks.md` into phases:

- Phase 1: first Diffs integration, including installing `@pierre/diffs`, adding the wrapper, and migrating the two existing diff surfaces.
- Phase 2: reviewer controls/readability, including visible unified/split toggle and word-level diff validation.
- Phase 3: annotation widget redesign.
- Phase 4: follow-up theming/performance polish.

### Screenshot diagnosis for annotation widget

The screenshot from `/tmp/pi-clipboard-0e1a6b39-a413-4760-bb34-8fa5db02421a.png` shows that the current annotation renderer displays every line as its own boxed `Code` component. That makes the snippet look like a stack of disconnected cards, and the yellow highlight does not tell the reader why those lines matter. The next implementation should make it read as: “Review note → highlighted lines → compact code frame”.

## Step 4: Implement Diffs wrapper, unified/split toggle, word diffs, and annotation redesign

I continued the implementation after the user asked for a unified toggle, word-level diffs, and a clearer annotation widget.

### Diffs API inspection

Installed the dependency:

```bash
pnpm -C ui add @pierre/diffs
```

Inspected `ui/node_modules/@pierre/diffs/package.json` and the generated `.d.ts` files. The relevant exports are:

- `@pierre/diffs/react`
  - `MultiFileDiff`
  - `PatchDiff`
  - `FileDiff`
- `MultiFileDiff` accepts `oldFile` and `newFile`, which matches our current `/api/history/symbol-body-diff` response (`oldBody`, `newBody`).
- `PatchDiff` is still relevant for future full patch/PR views, but not needed for the current symbol-body endpoints.

Decision: use `MultiFileDiff` first, because it avoids synthesizing patch headers around our custom hunk-only diffs.

### Diffs wrapper

Added:

```text
ui/src/features/diff/DiffsUnifiedDiff.tsx
```

The wrapper:

- imports `MultiFileDiff` from `@pierre/diffs/react` in one place only;
- accepts `oldText`, `newText`, name/labels/language;
- passes `diffStyle` through to Diffs;
- enables `lineDiffType: 'word'` for word-level changes;
- uses GitHub light/dark Shiki themes;
- disables the worker pool for now to keep first integration simple;
- includes an error boundary and fallback to a small hand-rendered diff if the third-party renderer fails.

### Unified/split toggle

Added a visible toggle inside the shared wrapper:

- `Unified` button
- `Split` button
- `aria-pressed` reflects the current state
- the active button is visually emphasized

Because the toggle is in the shared wrapper, it appears automatically in:

- embedded `codebase-diff` docs;
- commit-walk diff steps;
- focused symbol-history body diffs.

### Word-level diffs

Set:

```ts
lineDiffType: 'word'
```

in the Diffs options. The wrapper also shows a small label:

```text
Rendered with Diffs · word-level changes enabled
```

This makes it clear to reviewers why the rendering is more detailed than the old line-only highlighting.

### Migrated diff surfaces

Updated:

```text
ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx
ui/src/features/history/HistoryPage.tsx
```

Both now render body diffs through `DiffsUnifiedDiff` instead of hand-rendered `<pre><span>` lines. `CommitWalkWidget` benefits indirectly because its `diff` step composes `SymbolDiffInlineWidget`.

### Annotation redesign

Updated:

```text
ui/src/features/doc/widgets/AnnotationWidget.tsx
```

The old widget rendered each source line as a separate `<Code>` block. In the screenshot the result looked like a stack of disconnected boxes, and the annotation did not explain what the reader should notice.

The new widget is framed as a review note:

- header: `Review note: <symbol>`
- metadata: commit + highlighted line range
- explanation/note appears above the code
- code renders in one compact frame
- highlighted lines use a subtle yellow row background and left accent bar
- line numbers remain visible, but the code no longer appears as separate cards

This keeps `codebase-annotation` useful for “look here, this is the part of the code the guide is discussing” without pretending it is a diff.

### Validation

Commands run:

```bash
pnpm -C ui run typecheck
pnpm -C ui build
go test ./internal/docs ./internal/server
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Copied `ui/dist/public/*` into `internal/web/embed/public/` and restarted the tmux server on `:3001`.

Playwright validation:

- `/#/doc/05-slice1-diff-demo`
  - two Diffs wrappers rendered;
  - no fallback renderer;
  - Shadow DOM diff header present;
  - Split toggle changes `aria-pressed` state.
- `/#/history?symbol=sym:...Server.handleSnippet`
  - one Diffs wrapper rendered;
  - no fallback renderer;
  - toggle visible;
  - Shadow DOM diff header present.
- `/#/doc/08-slice4-quick-wins-demo`
  - one annotation widget rendered;
  - five highlighted line rows rendered;
  - text now starts with `Review note: handleSnippet` and explains the highlighted lines.
- Browser console errors: 0.

### Build/performance note

The Vite build now emits many Shiki language/theme chunks and large bundle warnings. This is expected from the Diffs/Shiki integration but should be handled in Phase 4, likely by lazy-loading the Diffs wrapper or otherwise reducing initial bundle impact.
