---
Title: Tasks
Ticket: GCB-010
Status: active
Topics: []
DocType: tasks
Intent: working
Owners: []
LastUpdated: 2026-04-25T12:30:00Z
---

# Tasks

## Slice 0: "Snapshot at a commit" — commit= on existing directives (~1 day)

- [x] **T0a** Extend `handleSnippet` in `internal/server/api_source.go` with `commit=` query param — resolve from history DB snapshot_symbols + file_contents
- [x] **T0b** Extend `resolveDirective` in `internal/docs/renderer.go` — when `params["commit"]` present, emit `data-commit` attribute on stub (don't resolve server-side)
- [x] **T0c** Extend `DocSnippet.tsx` + `DocPage.tsx` — pass `data-commit` through to snippet fetch
- [x] **T0d** Create demo page `internal/docs/embed/pages/04-review-slice0.md` with before/after code blocks
- [x] **T0e** Validate: open demo page, both snippets render at correct commits, xrefs work

**Decision gate: none** — this is pure plumbing, no UX decisions.

## Slice 1: "The diff widget" — codebase-diff (~2–3 days)

- [x] **T1a** Add `case "codebase-diff"` in `internal/docs/renderer.go` — validate sym/from/to, emit stub with data-params JSON
- [x] **T1b** Extend `DocPage.tsx` — extract `data-params` JSON from stubs, pass to DocSnippet
- [x] **T1c** Create `ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx` — fetch body-diff API, render side-by-side with diff colours
- [x] **T1d** Add `codebase-diff` dispatch in `DocSnippet.tsx`
- [x] **T1e** Update demo page with diff blocks
- [x] **T1f** Validate: diff renders, lines are coloured, expand to full symbol works

**Decision gate: Is side-by-side the right layout? Should we also try inline unified diff?**

## Slice 2: "The history timeline" — codebase-symbol-history (~1–2 days)

- [x] **T2a** Add `case "codebase-symbol-history"` in `internal/docs/renderer.go`
- [x] **T2b** Create `ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx` — compact commit list with body-hash dots
- [x] **T2c** Add dispatch in `DocSnippet.tsx`
- [x] **T2d** Update demo page with history block
- [x] **T2e** Validate: timeline renders with correct commit data, click-to-expand mini-diff works

## Slice 3: "Impact analysis" — codebase-impact (~2–3 days)

- [ ] **T3a** Implement `handleHistoryImpact` in `internal/server/api_history.go` — BFS over snapshot_refs with compatibility checking
- [ ] **T3b** Register `GET /api/history/impact` in `internal/server/server.go`
- [ ] **T3c** Add `useGetImpactQuery` hook in `ui/src/api/historyApi.ts`
- [ ] **T3d** Add `case "codebase-impact"` in `internal/docs/renderer.go`
- [ ] **T3e** Create `ui/src/features/doc/widgets/ImpactInlineWidget.tsx` — grouped caller list with ✓/⚠
- [ ] **T3f** Add dispatch in `DocSnippet.tsx`
- [ ] **T3g** Update demo page with impact block
- [ ] **T3h** Validate: callers render at correct depths, compatibility indicators work, performance acceptable

**Decision gate: Is BFS fast enough? Is depth=2 the right default?**

## Slice 4: "Quick wins" — annotation + changed-files + diff-stats (~1–2 days)

- [ ] **T4a** Add three `case` branches in `internal/docs/renderer.go` (codebase-annotation, codebase-changed-files, codebase-diff-stats)
- [ ] **T4b** Create `ui/src/features/doc/widgets/AnnotationWidget.tsx`
- [ ] **T4c** Create `ui/src/features/doc/widgets/ChangedFilesWidget.tsx`
- [ ] **T4d** Create `ui/src/features/doc/widgets/DiffStatsWidget.tsx`
- [ ] **T4e** Add three dispatch branches in `DocSnippet.tsx`
- [ ] **T4f** Update demo page with all three
- [ ] **T4g** Validate: all three render correctly, adding a new directive takes <30 min

## Slice 5: "The guided walk" — codebase-commit-walk (~3–4 days)

- [ ] **T5a** Add `case "codebase-commit-walk"` in `internal/docs/renderer.go` — parse `step` sub-directives, serialise as JSON in data-params
- [ ] **T5b** Create `ui/src/features/doc/widgets/CommitWalkWidget.tsx` — step navigation, sub-widget composition
- [ ] **T5c** Add dispatch in `DocSnippet.tsx`
- [ ] **T5d** Write full literate PR review guide demo page
- [ ] **T5e** Validate: walk through all steps, sub-widgets render, navigation works, reads as coherent guide

## Polish (after all slices)

- [ ] **TP1** Storybook stories for all widgets with mocked API data
- [ ] **TP2** E2E test: render doc page with all widget types
- [ ] **TP3** Accessibility pass: keyboard nav, ARIA labels
- [ ] **TP4** Upload final design doc + demo page to reMarkable
