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
