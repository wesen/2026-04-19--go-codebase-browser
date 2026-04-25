---
Title: Future assessment guide for Diffs-powered review UX
Ticket: GCB-012
Status: active
Topics:
  - codebase-browser
  - react-frontend
  - semantic-diff
  - ui-design
  - performance
  - documentation-tooling
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/diff/DiffsUnifiedDiff.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/diff/DiffsUnifiedDiffRenderer.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/AnnotationWidget.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/CommitWalkWidget.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/history/HistoryPage.tsx
  - /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_history.go
ExternalSources:
  - https://diffs.com/docs
Summary: Assessment guide for future improvements after adopting @pierre/diffs for semantic code review widgets.
LastUpdated: 2026-04-25T13:05:00-04:00
WhatFor: Use to plan and prioritize future Diffs-powered review UX improvements.
WhenToUse: Use when deciding the next roadmap after GCB-010 and GCB-011.
---

# Future assessment guide for Diffs-powered review UX

## Purpose

GCB-010 and GCB-011 established the first end-to-end version of embeddable semantic review widgets:

- history-backed snippets and diffs;
- symbol history and impact widgets;
- guided commit walks;
- `@pierre/diffs` rendering with word-level changes and unified/split toggles;
- lazy-loaded Diffs/Shiki rendering;
- cleaned up annotation widgets.

This ticket is not an implementation ticket. It is a future assessment workspace: collect everything we should, can, or could improve; evaluate impact and cost; then split concrete work into later tickets.

## Current baseline

### Implemented baseline

- `DiffsUnifiedDiff` is a light shell that lazy-loads `DiffsUnifiedDiffRenderer`.
- `DiffsUnifiedDiffRenderer` uses `MultiFileDiff` because the backend already returns `oldBody` and `newBody`.
- Runtime language requests are constrained to:
  - `go`
  - `typescript`
  - `tsx`
  - `text`
- Word-level diffs are enabled with `lineDiffType: 'word'`.
- A local unified/split toggle exists per diff widget.
- The hand-rendered fallback remains available if Diffs fails.

### Known tradeoffs

- Vite still emits many Shiki language/theme chunks because upstream Shiki exposes a bundled-language dynamic import map.
- The Diffs renderer is lazy-loaded, so the main app chunk is much smaller, but the lazy Diffs route still has significant code.
- `codebase-annotation` is a custom highlighted snippet, not yet a native Diffs line annotation.
- We use `MultiFileDiff`, not `PatchDiff`; this is ideal for symbol body diffs but not necessarily for full file/PR patches.
- Diff widgets currently maintain local layout state; unified/split preference is not persisted.

## Assessment areas

## 1. Bundle size and Shiki language/theme control

### Question

Can we reduce emitted Shiki language/theme chunks to only what codebase-browser needs?

### Current evidence

Runtime language loading is constrained, but build output still includes chunks for many Shiki languages. This comes from Shiki's dynamic import map, not from codebase-browser directly requesting those languages.

### Options

1. **Keep current lazy-loaded setup**
   - Lowest risk.
   - Main app load is already improved.
   - Accept lazy chunk footprint for now.
2. **Vite/Rollup manual chunk tuning**
   - Could group Shiki chunks more predictably.
   - May improve cache behavior but not total emitted bytes.
3. **Alias `shiki` to a limited compatibility module**
   - Potentially biggest reduction.
   - High risk because `@pierre/diffs` imports several Shiki exports.
4. **Patch or upstream-request a Diffs option for custom highlighter/language registry**
   - Best long-term solution if supported.
   - Requires upstream compatibility.
5. **Use SSR/pre-rendering utilities**
   - Could move highlighting work out of browser for static docs, but history-backed diffs are dynamic.

### Assessment criteria

- Main route JS size.
- Lazy diff route JS size.
- First diff render latency.
- Risk of breaking Shiki themes/languages.
- Maintainability during Diffs upgrades.

## 2. Diffs line annotations for `codebase-annotation`

### Question

Should `codebase-annotation` become a native Diffs line annotation instead of a separate highlighted snippet widget?

### Why it matters

The current annotation widget is now clearer, but it is still separate from the diff renderer. Diffs supports line annotations and render hooks, which could let us attach review notes directly to code or diff rows.

### Possible design

A future annotation step could render either:

- a single-file Diffs `File` component with line annotations; or
- a `MultiFileDiff`/`FileDiff` component where annotations attach to addition/deletion/context rows.

### Use cases

- Explain why a highlighted branch matters.
- Point out API compatibility risk.
- Attach caller/callee impact notes to specific changed lines.
- Show “review checklist” notes inside commit-walk steps.

### Open questions

- Does Diffs line annotation API work well inside Shadow DOM for our markdown layout?
- Can we target relative snippet lines robustly?
- How should annotations map across old/new sides in split view?
- Should annotations be authored in directive params, fence body DSL, or a YAML-like block?

## 3. Full file and PR-level patch support

### Question

Should codebase-browser grow from symbol-body diffs to full file/PR patch review?

### Diffs features to evaluate

- `PatchDiff`
- `parsePatchFiles`
- `trimPatchContext`
- `MultiFileDiff`

### Candidate widgets

- `codebase-file-diff from=... to=... path=...`
- `codebase-patch from=... to=... paths=...`
- `codebase-pr-review base=... head=...`
- `codebase-changed-files` with expandable inline file diffs.

### Backend needs

- Endpoint for file contents at commit.
- Endpoint for full file diff/patch by path.
- Endpoint for multi-file patch by commit pair.
- Optional path filters and max-size guardrails.

### Risks

- Large diffs may need virtualization/worker pool.
- Full patches may duplicate GitHub/GitLab capabilities unless semantic overlays are strong.
- Review docs should not become too heavy to read.

## 4. Commit-walk authoring improvements

### Current state

`codebase-commit-walk` uses line-oriented `step key=value` directives. It supports quoted titles and bodies, but long prose is awkward.

### Improvements to assess

- YAML-ish step DSL.
- Markdown body per step.
- Nested directives inside steps.
- Step templates: `stats`, `files`, `symbol`, `impact`, `annotation`.
- Persistent step progress.
- Deep-link to a specific step.

### Example future shape

```markdown
```codebase-commit-walk
- title: Start with the risk profile
  kind: stats
  from: ...
  to: ...
  body: |
    This commit is small but affects history-backed rendering.
- title: Inspect the routing branch
  kind: annotation
  sym: sym:...
  lines: 9-13
  body: |
    This branch is the semantic hinge of the change.
```
```

## 5. Review ergonomics and state persistence

### Potential improvements

- Persist unified/split preference in local storage.
- Add global display preferences for diff style and word/char/no line diff mode.
- Add keyboard shortcuts in commit walks.
- Add deep links to a commit-walk step.
- Add “copy review link” buttons for diff widgets and annotations.
- Add “open in history page” from embedded diff widgets.

### Assessment criteria

- Does the control reduce repeated reviewer work?
- Does it clutter embeddable docs?
- Can it be implemented in the shared wrappers rather than per widget?

## 6. Accessibility

### Required checks

- Keyboard navigation for diff toggles.
- `aria-pressed` correctness.
- Commit walk step buttons and `aria-current`.
- Color contrast for additions/deletions and annotation highlights.
- Screen reader behavior with Shadow DOM diff content.
- Reduced-motion / high-contrast compatibility.

### Possible outputs

- Accessibility audit doc.
- Playwright accessibility smoke tests.
- Storybook stories for keyboard/ARIA review.

## 7. Theming

### Current state

The wrapper uses GitHub light/dark Shiki themes and Diffs Shadow DOM styling.

### Future questions

- Should Diffs use Pierre Light/Dark, GitHub Light/Dark, or codebase-browser custom themes?
- Should app theme be passed explicitly instead of `themeType: 'system'`?
- Which CSS variables from codebase-browser should map into Diffs `unsafeCSS`?
- Can screenshots remain visually coherent across README and docs?

## 8. Backend/data model improvements

### Current state

Symbol body diffs use `oldBody` and `newBody` from history DB + repo checkout. Impact uses `snapshot_refs` BFS.

### Potential backend tasks

- Return file path/language in `BodyDiffResult` so the frontend does not assume `.go`.
- Return structured diff metadata instead of only raw old/new bodies and simple unified diff.
- Add file-content-at-commit endpoint.
- Add multi-file patch endpoint.
- Add max-size and truncation metadata for huge symbols/files.
- Add concept/symbol compatibility classification for diff rows.

## Prioritization matrix

| Area | User value | Risk | Suggested priority |
| --- | --- | --- | --- |
| Persist unified/split preference | Medium | Low | Soon |
| Backend language/path metadata | Medium | Low | Soon |
| Diffs line annotations | High | Medium | Next assessment spike |
| Commit-walk richer DSL | High | Medium | Next assessment spike |
| Full file diff widget | Medium/High | Medium | After backend endpoints |
| Custom Shiki limited bundle | Medium | High | Only if bundle size is painful |
| Worker pool/virtualization | Medium | Medium | Only for large diffs |
| Accessibility audit | High | Low/Medium | Soon |
| Theming integration | Medium | Medium | Alongside design polish |

## Recommended next assessment procedure

1. Re-open GCB-010 and GCB-011 demos.
2. Record current UX friction with screenshots.
3. Measure current bundle sizes and first diff render time.
4. Pick one high-value/low-risk improvement and one high-value/spike item.
5. Create implementation tickets from this assessment ticket.
6. Keep this ticket as the roadmap index, not a dumping ground for implementation commits.

## Suggested next tickets

- **GCB-013**: Persist diff view preferences and add deep links for commit-walk steps.
- **GCB-014**: Diffs-native line annotations for `codebase-annotation`.
- **GCB-015**: Backend file-diff/patch endpoints and `codebase-file-diff` widget.
- **GCB-016**: Accessibility audit for semantic review widgets.
- **GCB-017**: Shiki/Diffs bundle-size spike with custom language/theme registry.
