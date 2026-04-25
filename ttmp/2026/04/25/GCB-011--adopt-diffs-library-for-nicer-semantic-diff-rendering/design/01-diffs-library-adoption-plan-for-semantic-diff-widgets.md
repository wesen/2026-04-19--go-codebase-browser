---
Title: Diffs library adoption plan for semantic diff widgets
Ticket: GCB-011
Status: active
Topics:
  - codebase-browser
  - react-frontend
  - semantic-diff
  - ui-design
  - documentation-tooling
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/history/HistoryPage.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/CommitWalkWidget.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/AnnotationWidget.tsx
ExternalSources:
  - https://diffs.com/docs
Summary: Plan for replacing hand-rolled diff rendering with @pierre/diffs while preserving semantic review widgets.
LastUpdated: 2026-04-25T12:35:00-04:00
WhatFor: Use when implementing nicer diff rendering across codebase-browser history pages and embeddable docs.
WhenToUse: Before changing diff widgets, history body diffs, or commit-walk rendering.
---

# Diffs library adoption plan for semantic diff widgets

## Goal

Adopt [`@pierre/diffs`](https://diffs.com/docs) to make codebase-browser diffs more readable, themeable, expandable, and future-proof while preserving the semantic/history-backed behavior introduced in GCB-010.

## Source snapshot

The upstream docs were downloaded with Defuddle and stored in this ticket:

- [`../sources/01-diffs-docs.md`](../sources/01-diffs-docs.md)
- [`../sources/00-diffs-docs-title.txt`](../sources/00-diffs-docs-title.txt)
- [`../sources/00-diffs-docs-description.txt`](../sources/00-diffs-docs-description.txt)

Important upstream notes from the docs:

- Package: `@pierre/diffs`
- React entrypoint: `@pierre/diffs/react`
- Patch rendering component: `PatchDiff`
- File comparison components/utilities: `FileDiff`, `MultiFileDiff`, `parseDiffFromFile`, `parsePatchFiles`
- Syntax highlighting is Shiki-based.
- Components render into Shadow DOM with CSS variables and optional `unsafeCSS` hooks.
- Virtualization and worker pools exist for large diffs but are optional for an initial integration.
- APIs are marked early/active-development, so the first implementation should isolate the library behind local wrapper components.

## Current diff rendering surfaces

Initial targets:

1. `ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx`
   - Embeddable `codebase-diff` widget.
   - Currently renders unified diff lines by hand.
2. `ui/src/features/history/HistoryPage.tsx`
   - `SymbolBodyDiffView` for focused history pages and selected symbol diffs.
   - Recently fixed to color hand-rendered `+` / `-` lines.
3. `ui/src/features/doc/widgets/CommitWalkWidget.tsx`
   - Composes diff widgets; should benefit automatically if the child diff widget is upgraded.
4. Future file-level diff surfaces
   - `codebase-changed-files` remains a list, but future file-level diff widgets can use `PatchDiff` or `MultiFileDiff`.

## Recommended integration strategy

### Phase 1: wrapper component

Create a small local wrapper, for example:

```text
ui/src/features/diff/DiffsUnifiedDiff.tsx
```

Responsibilities:

- Import from `@pierre/diffs/react` in exactly one place.
- Accept the current backend shape:
  - `unifiedDiff: string`
  - optional `filename`, `language`, `oldHash`, `newHash`
- Render via `PatchDiff` if it accepts raw patch text cleanly.
- If `PatchDiff` expects file patch headers, normalize symbol-body unified diffs into a synthetic patch:
  - `diff --git a/<name> b/<name>`
  - `--- a/<name>`
  - `+++ b/<name>`
  - append existing hunk lines.
- Expose a fallback path to the current hand-rendered diff for failures or missing library support.

### Phase 2: replace symbol diff rendering

Update `SymbolDiffInlineWidget.tsx` to delegate to the wrapper. This gives the embedded docs and commit walk immediate improvement while minimizing blast radius.

### Phase 3: replace history body diff rendering

Update `SymbolBodyDiffView` in `HistoryPage.tsx` to use the same wrapper so the focused `/history?symbol=...` route matches doc-widget diffs.

### Phase 4: polish theming and performance

- Align Diffs theme variables with codebase-browser CSS variables.
- Decide whether to use Pierre Light/Dark or an existing Shiki theme.
- Add `unsafeCSS` only through the wrapper if needed.
- Defer worker pool/virtualization until there are measured large-diff performance problems.

## Dependency concerns

- Add dependency to `ui/package.json` with pnpm lockfile update.
- Confirm Vite handles the package and Shiki assets without extra config.
- Check bundle size impact.
- Because docs say APIs are early, avoid spreading Diffs imports across many feature widgets.

## Validation plan

Run:

```bash
pnpm -C ui install
pnpm -C ui run typecheck
pnpm -C ui build
go test ./internal/docs ./internal/server
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Browser validation:

- `/#/doc/05-slice1-diff-demo`
- `/#/doc/09-slice5-commit-walk-demo` step 3
- `/#/history?symbol=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet`

Playwright checks:

- No console errors.
- Diff widget renders.
- Added and removed lines are visually distinct.
- Long hunks remain scrollable and readable.
- Commit-walk navigation still works.

## Open questions

- Does `PatchDiff` accept hunk-only unified diff strings, or do we need synthetic file headers?
- Which theme should be default for light/dark codebase-browser themes?
- Does Shadow DOM styling conflict with current screenshot/readme aesthetics?
- Should annotation widgets also use Diffs token hooks later, or remain separate?
