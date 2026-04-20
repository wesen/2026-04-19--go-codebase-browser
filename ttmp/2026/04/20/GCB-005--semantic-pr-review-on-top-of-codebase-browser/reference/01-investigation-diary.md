---
Title: Investigation diary
Ticket: GCB-005
Status: active
Topics:
    - codebase-browser
    - pr-review
    - semantic-diff
    - git-integration
    - documentation-tooling
    - react-frontend
    - go-ast
    - ui-design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/20/GCB-005--semantic-pr-review-on-top-of-codebase-browser/design-doc/01-git-level-analysis-mapping-turning-codebase-browser-primitives-into-pr-review-data.md
      Note: Git-mapping design-doc
    - Path: ttmp/2026/04/20/GCB-005--semantic-pr-review-on-top-of-codebase-browser/design-doc/02-ui-affordances-wireframes-and-embeddable-widget-catalog.md
      Note: UI affordances + widget catalog
ExternalSources: []
Summary: Evidence-gathering + design authoring diary for GCB-005 — semantic PR review on top of codebase-browser. Records which internal files were inspected, which invariants are load-bearing, which reuse paths were chosen, and where each design doc lands.
LastUpdated: 2026-04-20T13:40:00Z
WhatFor: Preserve the investigation trail for reviewers + future implementers of the PR-review subsystem.
WhenToUse: Read before modifying the GCB-005 design docs or before implementing any of the new endpoints, directives, or widgets they describe.
---

# Investigation diary

## Goal

Produce a complete design package for a semantic PR-review substrate built on top of the existing codebase-browser. Two long-form design docs, one diary, uploaded to reMarkable. No code changes in this step — the deliverable is design, not implementation.

## Step 1: Ticket scaffold + vocabulary

Created GCB-005 on the existing docmgr ticket tree. GCB-003 and GCB-004 are already in use; GCB-005 is the next slot. Seeded four new vocabulary topics to describe the review subsystem.

### Prompt Context

**User prompt (verbatim):**

```
Create a ticket and and a detailed analysis of especially how the current code can be mapped and used for the git level analysis, and which artifacts can then be created in the UI. Also create a separate document that analyzes which UI affordances we can build, even including a list of ASCII screenshots for showing the wireframes, and a listi of potential reusable widgets to provide different kind of "embeddable" functionality in markdown documents as well.

Upload to remarkable when done.
```

**Assistant interpretation:** Create a docmgr ticket, two separate design-docs (one technical/backend-heavy, one UX/frontend-heavy), a diary, and upload the rendered bundle to reMarkable when complete.

**Inferred user intent:** Produce enough concrete design material that a future implementation session can start without re-deriving the mapping from AST indexes to PR-review data, and enough wireframe-level UX detail that a designer can style the final surfaces without guesswork.

### What I did

1. `docmgr status --summary-only` — confirmed docmgr healthy, 4 tickets, 8 docs.
2. `docmgr vocab list | grep -E "review|pr|git|semantic"` — no existing slugs match. Added four:
   - `pr-review` (Pull-request / code review tooling)
   - `semantic-diff` (Symbol-level diffs derived from AST indexes)
   - `git-integration` (Git history/blame integration in tooling)
   - `ui-design` (UI affordance and wireframe exploration)
3. `docmgr ticket create-ticket --ticket GCB-005 --title "Semantic PR review on top of codebase-browser" --topics codebase-browser,pr-review,semantic-diff,git-integration,documentation-tooling,react-frontend,go-ast,ui-design`.
4. `docmgr doc add` twice for the two design-docs, once for this diary.
5. Seeded 6 tracking tasks on tasks.md via docmgr (subsequently checked off as phases completed).

### Why

Topic slugs matter for future `docmgr doc search` queries — new reviewers of "pr-review" should discover this ticket without knowing its number.

### What worked

`docmgr ticket create-ticket` succeeded cleanly on the first try. `docmgr doc add` produced the expected scaffolds under `design-doc/` and `reference/`. Vocabulary additions were instant.

### What didn't work

Nothing; scaffold step.

### What I learned

The docmgr-seeded design-doc template includes placeholder section headers (Executive Summary, Problem Statement, etc.) that I ended up replacing wholesale because the default is thin. The template is better treated as a structural reminder than a template to fill in.

## Step 2: Evidence gathering

Before writing either design-doc, mapped the existing codebase-browser internals to concrete file+line anchors so every claim in the design has an evidence citation.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

### What I did

1. `wc -l internal/indexer/*.go internal/server/*.go internal/docs/*.go internal/browser/*.go` — confirmed ~2,400 LOC across the backend pieces in scope.
2. `grep -n "mux.HandleFunc" internal/server/server.go` — captured the existing 12 endpoints so the design could state which are reused versus new.
3. `grep -n "^func SymbolID\|^func MethodID\|^func PackageID\|^func FileID" internal/indexer/id.go` — got the ID scheme signature.
4. Read `internal/indexer/types.go:38-47` for the `Range` struct (byte offsets are load-bearing for the git-log-L mapping).
5. Read `internal/indexer/multi.go:30-90` for `Merge` — the set-diff algorithm reuses the same iteration shape.
6. Read `internal/docs/renderer.go:140-165` for `stubHTML` and its attribute contract — this is what the four new directives plug into.
7. Read `ui/src/features/doc/DocSnippet.tsx:22-50` for the hydration dispatch — same extension point on the frontend.
8. Catalogued widget props across 7 React components (`Code`, `SourceView`, `SymbolCard`, `LinkedCode`, `XrefPanel`, `ExpandableSymbol`, `FileXrefPanel`).
9. Probed git readiness with `git log -L 29,30:internal/indexer/id.go --oneline -3` — confirmed git 2.43 + `-L` works as expected.
10. Confirmed no pre-existing `git log`/`os/exec` calls inside `internal/` (only `cmd/build-ts-index/main.go` uses `os/exec` for Dagger fallback).

### Why

Evidence-first is the skill's explicit requirement. It also reduced risk of writing a design that assumed invariants that don't actually hold — e.g. the `SymbolID(importPath, kind, name, signatureForHash)` signature has an optional `#xxxx` suffix (id.go:19-27) that the design doc needed to reason about for signature-hash diff classification.

### What worked

The load-bearing invariants were all documented in existing comments — the `Range` comment at `internal/indexer/types.go:38` says "byte offsets and ... authoritative for slicing", which is exactly the hook for `git log -L`. No new documentation archaeology needed.

### What didn't work

The `SymbolID` helper's optional signature-hash suffix is not currently used anywhere in practice. It's designed for overload handling; the design doc notes this as a latent-but-intended feature.

### What I learned

GCB-004's stub-and-hydrate pipeline is exactly the reuse vector the PR-review directive catalog wants. The decision we made in GCB-004 — "emit raw HTML blocks sandwiched in blank lines so goldmark passes them through" — means the new directives can be added without touching goldmark configuration, just the `resolveDirective` switch and the `DocSnippet.tsx` dispatcher.

### What was tricky to build

Deciding whether to treat GCB-005 as a single large design-doc or two separate ones. The user asked for "a detailed analysis" and "a separate document" for UI affordances — clearly two files. The split landed naturally: `01` covers data flow and server handlers, `02` covers the visual surfaces and the widget catalog. Cross-references between the two are by section number.

### What warrants a second pair of eyes

The §5 algorithms in doc 01 (symbol diff, impact BFS) were written from first principles rather than from an existing implementation — they haven't run against a real two-index pair yet. A reviewer should sanity-check the classification rules (added / removed / signature / body / doc / moved) against edge cases like:

- A symbol renamed in place (new name, same signature) — becomes `removed + added`; we do not attempt rename detection. Comment this as a known limitation.
- A symbol moved across modules (different import path) — becomes `removed + added` because the `importPath` segment of the ID changes. This is probably desirable — crossing a module boundary is a real semantic event.

### What should be done in the future

Steps 3-6 below land the primary deliverables; Step 7 ships them to reMarkable.

### Code review instructions

1. `docmgr doctor --ticket GCB-005 --stale-after 30` — must pass before upload.
2. `ls ttmp/2026/04/20/GCB-005--.../design-doc/` — expect two `.md` files.
3. `ls ttmp/2026/04/20/GCB-005--.../reference/` — expect this diary.

## Step 3: Primary design doc — git-level analysis mapping

Wrote `design-doc/01-git-level-analysis-mapping-turning-codebase-browser-primitives-into-pr-review-data.md` (~550 lines). Structure follows the writing-style recommendation: exec summary → problem → current state → gap → proposed architecture → phased plan → testing → risks → references.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

### What I did

1. Wrote §1-9 of doc 01 in one pass, pulling every architectural claim from the evidence gathered in Step 2.
2. Embedded four algorithmic sketches (symbol diff, impact BFS, snapshot store, history wrapper) with full API signatures and pseudocode per algorithm.
3. Added a phased implementation plan with day estimates (≈12 days for a shipping rough cut).
4. Closed with a testing strategy matrix and an alternatives-considered section (SCIP, browser extension, server-side HTML rendering, git-native comments).

### Why

The user asked for "a detailed analysis of especially how the current code can be mapped and used for the git level analysis, and which artifacts can then be created in the UI." Doc 01 centres the backend + algorithms; doc 02 centres the UI.

### What worked

The mapping from existing primitives to new handlers was essentially mechanical: every new endpoint (`/api/diff`, `/api/impact`, `/api/symbol-history`, `/api/blame`, `/api/comments/*`) has at most one or two pages of logic, reusing an existing data structure. No new infra category.

### What didn't work

Initial outline placed snapshot store under "testing" (accidental cross-reference to a sister skill's test pattern). Corrected in the final pass.

### What I learned

Writing §5.3 (symbol diff classification) forced explicit reasoning about canonical hashing for body comparison. The decision — phase 1 raw-bytes hash, phase 2 Go AST normalisation — is documented in §8.1 risks. Without this step, the "body changed" counter would flood the UI with whitespace noise.

### What was tricky to build

§5.2 (snapshot materialisation modes). Three modes — CI artefact, on-demand worktree, hybrid — are all viable; choosing one required thinking about operator concerns (who runs the server? can they reach CI?) rather than just code structure. The doc presents all three and recommends hybrid with a note on why.

### What warrants a second pair of eyes

§5.3 classification rules might be too aggressive about "body" detection. If a comment-only edit counts as body change, the summary view gets noisy. A cheap tweak: classify "only comments inside the body changed" as a separate `body-comments` status. Left as an open question in §8.3.

### Code review instructions

1. `wc -l design-doc/01-*.md` — expect ~550 lines.
2. Check §5 has an algorithm sketch for each of: snapshots, diff, impact, history, comments.
3. Check §6 phases are actionable (file paths + day estimates).
4. Check §9 references list cites every `internal/` file mentioned in the body.

## Step 4: UI design doc — affordances, wireframes, widget catalog

Wrote `design-doc/02-ui-affordances-wireframes-and-embeddable-widget-catalog.md`. Four main blocks:

- §4 — reviewer surfaces with ASCII wireframes (PR summary, per-symbol diff, impact timeline, hover popover, comment threads, file-diff fallback).
- §5 — new React components (8 components, each with a `props` TypeScript sketch).
- §6 — embeddable widget catalog (7 new directives, each with authoring example + rendering contract + plaintext fallback).
- §7 — interaction spec (keyboard shortcuts + share URLs).

### Prompt Context

**User prompt (verbatim):** (see Step 1)

### What I did

1. Drew five ASCII wireframes (PR summary, per-symbol diff, impact timeline, hover popover, comment thread). Used box-drawing Unicode so each renders legibly in a terminal and on reMarkable PDF.
2. Paired each wireframe with a "composition notes" paragraph that names the exact existing widgets reused and the new components needed.
3. Enumerated 7 new directives: `codebase-diff`, `codebase-impact`, `codebase-history`, `codebase-caller-list`, `codebase-comments`, `codebase-callgraph`, `codebase-file-diff`.
4. Wrote a rendering-contract summary table with the plaintext fallback for each directive — this is the JS-disabled degradation path.
5. Added an interaction spec (§7) covering the minimal keyboard shortcuts a reviewer expects (j/k, o, /, ?).

### Why

The UI is the half that decides whether reviewers actually adopt the tool. Making the wireframes ASCII rather than deferring to "designer will figure it out" forces the spatial argument to be explicit: rough widths, label truncation, column priorities.

### What worked

ASCII wireframes are self-documenting for engineers. The PR-summary wireframe (~30 lines of ASCII) captures density, hierarchy, and hit zones without any figma detour.

### What didn't work

Initial ASCII boxes had inconsistent border widths (some were `╔` + double-line, others `┌` + single-line). Unified on double-line for emphasised regions (before/after panes), single-line for the outer chrome. Cosmetic but improved scanability.

### What I learned

Writing the widget catalog forced a decision about how authors pass extra params: `data-params='{"key":"value"}'` as a JSON-encoded attribute on the stub. This keeps the server-side directive parser simple (free-form `key=value` params) while giving the frontend structured access. Alternative was per-directive `data-*` attributes; JSON is more extensible.

### What was tricky to build

§5.4 `<ImpactPanel>` subsumes the existing `<XrefPanel>` at depth=1. Deciding whether to *merge* the two or keep `XrefPanel` as a backward-compat alias. The doc proposes merging with `<ImpactPanel dir depth withCompatibility>`, with the existing `/symbol/<id>` page using `<ImpactPanel dir="usedby" depth=1 withCompatibility={false}>`. Review needed.

### What warrants a second pair of eyes

§6.6 `codebase-callgraph` — the design handwaves the graph layout ("hierarchical, radial, or force-directed"). A reviewer should decide whether we ship with one default or offer all three; also what library to use (dagre, elk, d3-force). Left as implementation detail.

### What should be done in the future

Storybook stories for each new component (§5.1-5.7). Proposal in §10 Q4 to co-locate them under `ui/src/features/review/stories/`.

### Code review instructions

1. `wc -l design-doc/02-*.md` — expect ~600 lines.
2. Confirm §4 has five wireframes.
3. Confirm §5 lists at least 7 components with TypeScript prop sketches.
4. Confirm §6 catalogues 7 directives with authoring + rendering examples.
5. Confirm §7.1 keyboard shortcuts are complete (movement, action, help, comment).

## Step 5: Bookkeeping

Ran `docmgr doc relate` to attach the key implementation files to both design-docs, then `docmgr changelog update` for each step. Ran `docmgr doctor --ticket GCB-005 --stale-after 30` to validate.

### Code review instructions

1. `docmgr doctor --ticket GCB-005 --stale-after 30` — passes cleanly.
2. `docmgr task list --ticket GCB-005` — all 5 seeded tasks checked off.

## Step 6: reMarkable upload

Used the bundled upload flow: `remarquee status` → dry-run → real upload → verify listing. Bundle includes both design-docs and this diary.

### Code review instructions

1. `remarquee cloud ls /ai/2026/04/20/GCB-005 --long --non-interactive` — expect one `.pdf` entry.
2. Final handoff message includes the remote path + ticket path + doctor result.
