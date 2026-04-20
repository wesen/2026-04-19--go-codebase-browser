---
Title: UI affordances, wireframes, and embeddable widget catalog
Ticket: GCB-005
Status: active
Topics:
    - codebase-browser
    - pr-review
    - react-frontend
    - documentation-tooling
    - ui-design
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/docs/renderer.go
      Note: Server-side stub emission that every new widget plugs into
    - Path: ui/src/features/doc/DocSnippet.tsx
      Note: |-
        Hydration dispatcher that grows review-widget branches
        Hydration dispatcher extended with review directives
    - Path: ui/src/features/source/FileXrefPanel.tsx
      Note: |-
        File-level xref panel reused by CommitXrefPanel
        Precedent for commit-scope xref views
    - Path: ui/src/features/symbol/ExpandableSymbol.tsx
      Note: |-
        Collapsible card reused by diff view and impact panel
        Reused inside SymbolDiffView panes
    - Path: ui/src/features/symbol/XrefPanel.tsx
      Note: |-
        Two-column used-by/uses reused by ImpactPanel (depth-1 case)
        Evolves into ImpactPanel depth=1 case
    - Path: ui/src/packages/ui/src/SymbolCard.tsx
      Note: |-
        Reused by SymbolDiffView's before/after panes
        Kind badge + signature header reused throughout
ExternalSources: []
Summary: ASCII-wireframed walkthrough of the user-facing surfaces needed for semantic PR review (PR summary, symbol-diff view, impact timeline, symbol-anchored comments, hover affordances), plus a catalog of reusable markdown-embeddable widgets (codebase-diff, codebase-impact, codebase-history, codebase-caller-list, codebase-comments, codebase-callgraph) with their authoring ergonomics and rendering contracts.
LastUpdated: 2026-04-20T13:35:00Z
WhatFor: Give designers, frontend engineers, and doc authors a concrete picture of the review surface shape before implementation — including which primitives are reused and which are new.
WhenToUse: Read alongside the git-mapping design-doc before implementing any /pr/ route, PR-review component, or new codebase-* directive.
---


# UI affordances, wireframes, and embeddable widget catalog

## 1. Executive summary

This document covers the user-facing half of GCB-005. The companion design-doc (`01-git-level-analysis-mapping...md`) describes how the existing codebase-browser primitives get mapped to git-level review data. This doc describes what reviewers and doc authors *see and do* with that data.

Two audiences and two surfaces:

1. **Reviewers** use a dedicated `/pr/:id` page and hover affordances wired into existing `/symbol/` and `/source/` views. This is the primary delivery target.
2. **Doc authors** use new markdown directives that embed the same widgets inline — e.g. a `codebase-impact` block inside an architecture doc that shows "here are the current callers of this interface, as of HEAD."

Both surfaces share the same React components; the doc-page path is the existing GCB-004 stub+portal pipeline (`internal/docs/renderer.go:140`, `ui/src/features/doc/DocSnippet.tsx:22`) with four new directive branches. The `/pr/` path is a new route composing those same components into a richer shell.

This document proceeds from the largest surface (the PR summary page) down to the smallest (hover popovers), then enumerates the widget catalog usable from markdown. Every wireframe is ASCII for durability — the final UI will render richer, but the spatial logic and information density should survive the translation.

## 2. Problem statement

GitHub-style PR review optimises for line-level code comprehension. Our index-backed data unlocks four orthogonal capabilities that want surfacing:

1. **Semantic overview.** "What changed in terms of symbols, not lines?"
2. **Side-by-side diff with live xrefs.** "Show me the before and after of `Extract`, and let me click through to callers on either side."
3. **Impact awareness.** "Who's downstream of this change?"
4. **History at a glance.** "When did this function last change, and by whom?"

A fifth, lower-frequency capability — *symbol-anchored review comments* — is layered over the above.

The tension throughout is information density versus readability. We lean toward density because reviewers are power users by default; collapse toggles buy us a soft floor for casual readers.

## 3. Design principles

1. **Symbol is the atomic unit.** Every navigation, every link, every anchor points at a `sym:...` ID. File and line anchors are fallback only.
2. **Reuse before inventing.** Every wireframe below is a composition of components that already exist. Where a new component is strictly necessary, it gets named explicitly in §5.
3. **Progressive disclosure.** Top of page = counts and classification. Click into a row = the full `<ExpandableSymbol>` + xrefs. Hover = compact popovers. No single view should render more than ~8 screen-heights of content without an explicit expand.
4. **Keyboardable navigation.** `j`/`k` between changed symbols, `o` to open a symbol, `/` to search within the diff, `?` for help. Matches what heavy-review users expect from GitHub's file-tree pane.
5. **Stable anchors.** Every collapsible has a URL fragment (`#sym:...`) so a reviewer can link "look at this one" in Slack.
6. **Doc widgets render identically inline.** The same `<SymbolDiffView>` a reviewer sees at `/pr/42` renders inside a markdown `codebase-diff` block. No second implementation.

## 4. Reviewer surfaces

### 4.1 PR summary page (`/pr/:id` or `/diff?base=X&head=Y`)

The landing page of a review. Its job is to let the reviewer answer three questions in five seconds: *how big is this change?*, *does anything look risky at a glance?*, *where do I start?*

```
┌──────────────────────────────────────────────────────────────────────────┐
│  PR #42 · rework Extract to support build tags                           │
│  base 9f8e7d6 (main)      head a1b2c3d       Manuel · 2026-04-20 11:42   │
│  [ review ]  [ open in IDE ]  [ share link ]                             │
├──────────────────────────────────────────────────────────────────────────┤
│  SEMANTIC DIFF                                                           │
│   ● 3 functions body-changed        ● 1 type added                       │
│   ● 2 signatures altered  ⚠         ● 0 functions removed                │
│   ● 4 doc comments updated          ● 1 function moved (file → file)     │
├──────────────────────────────────────────────────────────────────────────┤
│  CHANGED SYMBOLS                                        IMPACT  HISTORY  │
│  ─────────────────────────────────────────────────────────────────────── │
│   ⚠ func   indexer.Extract                signature    23 (12⚠)     4    │
│      "Extract(opts) (*Index, error)"                                     │
│      → "Extract(opts, strict bool) (*Index, error)"                      │
│   ● func   indexer.addRefsForFile         body          6           3    │
│   ● func   indexer.NewGoExtractor         body          5           2    │
│   + type   indexer.ExtractOptions         added         —           0    │
│   M func   cmds/index.Register            moved→index/  1           8    │
│  ─────────────────────────────────────────────────────────────────────── │
│  FILES (4 touched, 0 pure additions)                                     │
│   internal/indexer/extractor.go              +47  -19                    │
│   internal/indexer/types.go                  +12   -3                    │
│   cmd/codebase-browser/cmds/index/build.go   +18   -8                    │
│   internal/indexer/extractor_test.go         +24   -0                    │
├──────────────────────────────────────────────────────────────────────────┤
│  COMMENTS (2 open, 1 resolved)                                           │
│   ┌──────────────────────────────────────────────────────────────┐       │
│   │ @alice on indexer.Extract                                    │       │
│   │ "Signature change needs a changelog note + bump"   [ reply ] │       │
│   └──────────────────────────────────────────────────────────────┘       │
└──────────────────────────────────────────────────────────────────────────┘
```

Key spatial facts:

1. The semantic-diff counter row is the new thing. GitHub's "Files changed" tab has a file-count and line-count badge; this one classifies by symbol behaviour.
2. The changed-symbols table is the primary list. Each row has a kind badge, a symbol name, a one-line change summary, the impact count (direct callers; the ⚠ flag marks callers potentially affected by a signature change), and the history count. Click a row → expand inline or navigate to the per-symbol diff.
3. The files section is kept, but demoted — reviewers who want the line-level view still have it.
4. Comment threads are surfaced inline so the reviewer knows about open feedback before diving in.

Composition: `<DiffSummaryBanner>` (new) + `<SymbolChangeRow>` (new) + `<FileChangeRow>` (thin wrapper over existing file metadata) + `<CommentThread sym>` (new) at the bottom.

### 4.2 Per-symbol diff (`/pr/:id/sym/:sym` or inline expand)

The core reading surface for a single changed symbol. Two `<ExpandableSymbol>` side-by-side plus an `<ImpactPanel>` beneath.

```
┌──────────────────────────────────────────────────────────────────────────┐
│  indexer.Extract  · signature change                                     │
│  [ ← back to PR ]  [ history ]  [ raw diff ]                             │
├──────────────────────────────────────────────────────────────────────────┤
│  ╔═══ BEFORE (9f8e7d6) ═══════╗  ╔═══ AFTER (a1b2c3d) ═══════╗           │
│  ║ func Extract(opts ExtractOp║  ║ func Extract(opts ExtractOp║          │
│  ║   (*Index, error)          ║  ║   strict bool) (*Index, err║          │
│  ║                            ║  ║                            ║          │
│  ║ // body with xref links... ║  ║ // body with xref links... ║          │
│  ║ [ExpandableSymbol with     ║  ║ [ExpandableSymbol with     ║          │
│  ║  LinkedCode + xref panel ] ║  ║  LinkedCode + xref panel ] ║          │
│  ╚════════════════════════════╝  ╚════════════════════════════╝          │
├──────────────────────────────────────────────────────────────────────────┤
│  CALLERS (23)                                                            │
│  ✓ handleBuild                 opts-only call       ok                   │
│  ⚠ packageLoader.Load          positional call      review signature     │
│  ⚠ testSetup (3x)              _test.go files       update tests         │
│  ✓ Register (cmds/index)       constructs options   ok                   │
│  ... 18 more  [ show all ]                                               │
├──────────────────────────────────────────────────────────────────────────┤
│  HISTORY                                                                 │
│   2026-04-19  Manuel   "Phase 1: Go extractor scaffold"         #1       │
│   2026-04-19  Manuel   "Phase 6: cross-references"              #42      │
│   2026-04-20  Manuel   "Phase 1 refactor: Extractor interface"  #101     │
│   2026-04-20  Manuel   "rework Extract to support build tags"(this PR)   │
├──────────────────────────────────────────────────────────────────────────┤
│  COMMENTS (1 open)                                                       │
│   ┌─────────────────────────────────────────────────┐                    │
│   │ @alice "Signature change needs a changelog note │                    │
│   │         + bump"              [ reply ]          │                    │
│   └─────────────────────────────────────────────────┘                    │
└──────────────────────────────────────────────────────────────────────────┘
```

Interaction notes:

1. The two snippet panes scroll independently but share a synchronised scroll mode (toggle in the header) that aligns matching line numbers.
2. Every identifier in either pane is an xref link — clicking `ExtractOptions` in the "before" pane navigates to the base-version symbol page; in the "after" pane, to the head-version. This reuses `<LinkedCode refs renderRefLink>` unchanged; the only new thing is that `renderRefLink` gets a `?head=<sha>` query-string suffix.
3. The caller list is the `<ImpactPanel>` with a new "callsite compatibility" column derived from cheap AST heuristics: a call whose positional arity changed gets ⚠; a call through an interface is flagged as "review manually."
4. History is the `<HistoryTimeline>` widget (§5.5). "This PR" is always bolded and anchored at the bottom.

### 4.3 Impact timeline — "blast radius over time"

For a chosen symbol, a horizontal timeline showing commits that *or anything downstream of it* changed. Useful for bisecting "when did this stop working?"

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Impact timeline: indexer.Merge                                         │
│  [ ← back ]   depth: [1][2][●3][4][5]   direction: [ usedby / uses ]    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  2026-04-19 ─●────────────────●───●─────────────●─●─── 2026-04-20       │
│              │                │   │             │ │                     │
│              ↓                ↓   ↓             ↓ ↓                     │
│             self           caller[Extract]    self [this PR]            │
│             (Phase 1)     (Phase 1 refactor)                            │
│                                                                         │
│  Expanded (click any dot):                                              │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │ 2026-04-20 20:30 · Manuel · "rework Extract to support build tags"│  │
│  │   Direct: Extract (signature changed)                             │  │
│  │   Depth-2: Merge (called from Extract)                            │  │
│  │   [ open PR #42 ]                                                 │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

Composition: `<HistoryTimeline>` (new) overlaid on an SVG axis + `<ImpactCommitCard>` (new) for the expanded popover.

### 4.4 Hover popovers — attaching to existing /symbol/ and /source/ views

Non-review users still benefit from semantic history *on every page*. A small hover popover over any symbol name renders:

```
 ┌─────────────────────────────────────┐
 │  indexer.Extract                    │
 │  Signature: func Extract(opts Extra…│
 │                                     │
 │  Last touched: 2026-04-20 by Manuel │
 │  History: 4 commits   Impact: 23    │
 │  [ open full diff history ]         │
 └─────────────────────────────────────┘
```

Implementation: a new `<SymbolPopover sym>` component, triggered by a `title`-attr-equivalent rehydrated hover handler on every `<Link data-role="xref">` — the same selector the existing xref infrastructure already produces. Fetches `/api/symbol-history/{sym}?limit=1` + `/api/impact/{sym}?depth=1` on hover with a 300 ms debounce to avoid a request storm during code reading.

### 4.5 Symbol-anchored comment threads

Comments live on `sym:...` anchors (see git-mapping §5.6). The UI renders them in three places:

1. Inline under `<ExpandableSymbol>` on symbol pages and in the per-symbol diff.
2. In the PR summary's comments section (flat, newest-first).
3. As a badge on `<SymbolChangeRow>` rows and `<SymbolCard>` headers (count of open threads).

Thread UI:

```
┌────────────────────────────────────────────────────────────┐
│  Thread on indexer.Extract (2 messages, open)              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ @alice  2026-04-20 14:12                             │  │
│  │ Signature change needs a changelog note + a version  │  │
│  │ bump in semver minor at least.                       │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ @manuel  2026-04-20 14:30                            │  │
│  │ Good catch, added to the PR body.                    │  │
│  └──────────────────────────────────────────────────────┘  │
│  [ reply ]   [ resolve ]                                   │
└────────────────────────────────────────────────────────────┘
```

When a thread's anchor symbol disappears from head (symbol removed), the thread is not deleted — the UI renders it with an "orphaned anchor" badge and a "re-anchor or archive?" prompt when the reviewer opens the PR.

### 4.6 PR-level file view (soft fallback)

For files without indexed symbols (plain .json, .md, proto), the review falls back to a traditional unified diff using the existing `<SourceView>` as the renderer for each side. No new component; the `base-ref` / `head-ref` plumbing is enough.

## 5. New React components

Everything in §4 decomposes into the following:

### 5.1 `<DiffSummaryBanner>`

```tsx
interface DiffSummaryBannerProps {
  baseRef: string;
  headRef: string;
  counts: { added, removed, signatureChanged, bodyChanged, docChanged, moved, typeAdded, typeRemoved };
}
```

Three-line card at the top of `/pr/:id`. Pure presentation over `/api/diff` response's `counts` block.

### 5.2 `<SymbolChangeRow>`

```tsx
interface SymbolChangeRowProps {
  change: SymbolChange;
  impactCount: number;
  impactWarnings: number;
  historyCount: number;
  threadCount: number;
  expanded: boolean;
  onToggle: () => void;
}
```

One row in the changed-symbols table. Click to expand inline; expand state renders `<SymbolDiffView>` in-place. Body renders a kind badge + name + status pill + the signature delta on one line (truncated, full on hover).

### 5.3 `<SymbolDiffView>` and `<SideBySideSymbol>`

```tsx
interface SymbolDiffViewProps { sym: string; baseRef: string; headRef: string; }
interface SideBySideSymbolProps { sym: string; ref: string; }  // one pane
```

Composes two `<ExpandableSymbol>` (with the new `ref` prop threading `?head=<sha>` through RTK-Query keys) in a responsive two-column layout. Collapses to stacked panes below 900 px width.

### 5.4 `<ImpactPanel>`

```tsx
interface ImpactPanelProps {
  sym: string;
  dir: 'usedby' | 'uses';
  depth?: number;       // default 2
  withCompatibility?: boolean;  // show ✓ / ⚠ column
}
```

Supersedes `<XrefPanel>` (depth=1 special case). The `withCompatibility` prop toggles the "callsite ok / review needed" column — only meaningful on the review surface; off by default on standalone symbol pages.

### 5.5 `<HistoryTimeline>`

```tsx
interface HistoryTimelineProps {
  sym: string;
  headRef?: string;   // for "as-of" views
  limit?: number;      // default 50
  compact?: boolean;   // one-line rendering for embeds
}
```

Vertical list or horizontal axis view (a `mode` prop picks). Each commit row: date · author · subject · SHA badge · "this PR" badge when applicable.

### 5.6 `<CommentThread>` / `<CommentList>`

```tsx
interface CommentThreadProps { sym: string; }
interface CommentListProps { sym?: string; file?: string; }  // one xor the other
```

Compose-mode adds a markdown editor at the bottom. `<CommentList>` is the unfiltered stream; `<CommentThread>` is the filtered, per-symbol list with reply/resolve controls.

### 5.7 `<SymbolPopover>`

```tsx
interface SymbolPopoverProps { sym: string; anchor: HTMLElement | null; }
```

Renders the hover card from §4.4. Lives outside the component tree via `createPortal` to `document.body` so it escapes overflow clipping.

### 5.8 Composition notes

The app-level `<PRSummaryPage>` is assembled from the above in ~80 lines of JSX. No new styling system; reuses the existing `data-part` / `data-role` convention (`ui/src/packages/ui/src/parts.ts`).

## 6. Embeddable widget catalog

Every surface in §4 is also reachable from a markdown doc page via the `codebase-*` directive family introduced by GCB-004 (`internal/docs/renderer.go:140`). This section catalogs the new directives and pairs each with its authoring scenario, runtime contract, and rendered shape.

### 6.1 `codebase-diff`

**Purpose.** Show a single symbol's before/after across two commits, inline in prose.

**Authoring.**

````markdown
Here's what changed in the extractor between release v0.5 and main:

```codebase-diff sym=sym:.../indexer.func.Extract base=v0.5 head=main
```

The argument list grew by one required parameter, `strict`.
````

**Rendering.** The entire `<SymbolDiffView>` (§5.3) mounted inline. Defaults to open; the author can pass `mode=signature-only` for a compact one-line rendering.

**Attributes.** `sym`, `base`, `head`, optional `mode=full|signature-only|body-only`, optional `height=N` (px cap before scroll).

**Server contract.** Resolves in `internal/docs/renderer.go:resolveDirective` to a `SnippetRef` with `Directive="codebase-diff"` and extra params stashed on `Text` as JSON. The hydration picks them up from the stub's `data-params`.

### 6.2 `codebase-impact`

**Purpose.** Show the transitive callers (or callees) of a symbol as a flat list or mini-graph.

**Authoring.**

````markdown
Adding a new extractor language touches these symbols:

```codebase-impact sym=sym:.../indexer.iface.Extractor dir=usedby depth=2
```
````

**Rendering.** An `<ImpactPanel dir depth>` (§5.4). Optional `mode=list|graph` — graph mode renders an SVG of the adjacency if the count is below ~30; list above.

**Attributes.** `sym`, `dir=usedby|uses`, `depth=1..5`, `mode=list|graph`, optional `filter=internal|external|all`.

### 6.3 `codebase-history`

**Purpose.** Show the commit history that touched a symbol.

**Authoring.**

````markdown
The merge logic has been iterated several times:

```codebase-history sym=sym:.../indexer.func.Merge limit=10
```
````

**Rendering.** The `<HistoryTimeline compact=true>` variant. Suitable for a sidebar.

**Attributes.** `sym`, optional `limit=N`, optional `mode=vertical|horizontal`, optional `since=<ref>` (show only commits since X).

### 6.4 `codebase-caller-list`

**Purpose.** Compact "who uses this" list — a narrow subset of `codebase-impact` for signature-only views.

**Authoring.**

````markdown
`FileXrefPanel` is used by:

```codebase-caller-list sym=sym:ui/src/features/source/FileXrefPanel.func.FileXrefPanel
```
````

**Rendering.** Flat list of `<Link to=/symbol/{id}>` rows with the kind badge prefix. Functionally equivalent to `codebase-impact depth=1 dir=usedby mode=list` but with a terser default rendering tuned for in-prose use.

### 6.5 `codebase-comments`

**Purpose.** Pin a running discussion thread inside a doc page. Useful for design docs that want inline reader Q&A.

**Authoring.**

````markdown
## Why we chose JSON over protobuf

```codebase-comments sym=sym:.../indexer.type.Index
```

Discussion about the schema choice lives here, threaded under the Index type symbol.
````

**Rendering.** `<CommentThread sym>` (§5.6). Compose-mode enabled only for authenticated reviewers.

### 6.6 `codebase-callgraph`

**Purpose.** Render a mini-graph of the n-hop neighbourhood around a symbol.

**Authoring.**

````markdown
The request lifecycle touches:

```codebase-callgraph sym=sym:.../server.method.Server.handleXref depth=2 layout=hierarchical
```
````

**Rendering.** SVG adjacency graph (nodes = symbols, edges = refs, node colour = kind). Reuses the data from `<ImpactPanel>` but renders it differently. Layout options: `hierarchical`, `radial`, `force`.

Cost and sanity: capped at ~50 nodes; above that it renders as a list with a "too dense for graph view" banner.

### 6.7 `codebase-file-diff`

**Purpose.** Side-by-side for a whole file across two refs — useful when discussing a file that has no indexed symbols (e.g. a new migration or markdown page).

**Authoring.**

````markdown
```codebase-file-diff path=internal/indexer/extractor.go base=main head=HEAD
```
````

**Rendering.** Two `<SourceView>` panes side-by-side with synchronised scroll. A thin `<FileDiffHeader>` shows +/- line counts.

### 6.8 Widget rendering contract summary

All widgets follow the GCB-004 stub-and-hydrate convention:

1. Server `internal/docs/renderer.go:resolveDirective` resolves the directive, validates required params, and writes a `SnippetRef`.
2. `stubHTML()` emits `<div class="codebase-snippet" data-codebase-snippet data-directive="codebase-..." data-sym="..." data-params='{"key":"value"}'>` with a plaintext fallback inside.
3. `ui/src/features/doc/DocSnippet.tsx` grows a branch per directive and mounts the corresponding component via the existing `createPortal` path.
4. Each component is responsible for its own data fetching via RTK-Query.

Plaintext fallbacks for each widget:

| Directive | Fallback |
|---|---|
| `codebase-diff` | "See diff at /pr?base=X&head=Y&sym=Z" |
| `codebase-impact` | "X callers (see /symbol/Z)" |
| `codebase-history` | "Touched in N commits (latest: SUBJECT)" |
| `codebase-caller-list` | Unordered list of caller IDs |
| `codebase-comments` | "N messages" |
| `codebase-callgraph` | "N nodes, see /symbol/Z/callgraph" |
| `codebase-file-diff` | "Unified diff for path at base..head" |

JS-disabled readers always see *some* useful content, even if the rich widget never hydrates.

## 7. Interaction spec

### 7.1 Keyboard shortcuts (review surface only)

```
  j / k           next / previous changed symbol
  o               open currently-focused symbol (expand inline)
  Enter           open symbol in its own /symbol/ page in a new tab
  /               focus in-page search
  ?               open keyboard help modal
  [ / ]           previous / next commit in timeline (when focused)
  c               start a new comment on the currently-focused symbol
  r               resolve current comment thread
  shift + r       reply in current comment thread
  g s             go to summary (top of /pr/)
  g c             go to first open comment
```

All shortcuts target the symbol or comment currently under `aria-selected`; focus ring is always visible (no implicit cursors).

### 7.2 Linking and URLs

- `/pr/42` — default summary view; `base` and `head` derived from the configured PR source.
- `/pr/42#sym:...Extract` — scrolls to a specific changed symbol and expands it.
- `/diff?base=X&head=Y` — ad-hoc comparison, no PR id needed.
- `/symbol/:id?head=<sha>` — the existing symbol page rendered at a given SHA (reuses all existing infrastructure).
- `/source/:path?head=<sha>` — same, but file view.

### 7.3 Share links

Every non-trivial state has a shareable URL. Examples:

```
/pr/42                                       the whole PR summary
/pr/42/sym/sym:...Extract                    one symbol, expanded diff
/symbol/sym:...Merge/history                 history timeline for Merge
/symbol/sym:...Merge/impact?depth=3          impact at depth 3
```

These URLs are also what `codebase-*` directives generate as their fallback links, which keeps markdown and UI navigation consistent.

## 8. Visual-design outline (non-binding)

Deliberately deferred to implementation. Notes for the designer:

1. Density: ~12 rows of changed-symbols visible on a 1080p laptop without scrolling; ~24 on a 4K.
2. Colour: reuse `data-role` tokens — `kind` roles colour-code the badge; `xref` role colours the link; new `status` role for `added/removed/signature/body/doc/moved`.
3. Typography: monospace for all code identifiers, proportional for prose, 16 px body minimum.
4. Motion: avoid animation inside the diff panes (causes re-reading overhead). Animate only the sidebar highlights on `j`/`k`.
5. Dark mode: both themes must be legible; kind colours need a dark-mode pairing.

## 9. Accessibility

1. Every collapsible row is a `<button aria-expanded>` — the existing `<ExpandableSymbol>` already follows this pattern.
2. Keyboard focus order matches reading order; no tab traps.
3. The side-by-side panes on narrow viewports collapse to a single column with a "before / after" toggle (no fixed-width assumption).
4. The impact graph has a list-mode sibling; screen readers always see the list.
5. Comment bodies render through the same markdown renderer as doc pages — no inline HTML allowed from untrusted input. Already handled by goldmark's default safety settings (`html.WithUnsafe()` is currently enabled for doc pages; for comments we would disable it).

## 10. Open UI questions

1. **Three-pane vs two-pane diff.** Some tools show base / merge-base / head. For now we assume two panes; reviewers can open a second diff window if they need the merge-base.
2. **Inline vs split diff for bodies.** Industry convention is split (side-by-side) for bodies longer than ~30 lines, inline otherwise. Open question: let the user pick per-symbol, or enforce the rule?
3. **How to visualise moved + body-changed symbols.** A symbol can be both moved and body-changed in one PR. The current design shows the "moved" badge in the row, and the body-diff normally inside. Might want a compact "before at path X, after at path Y" header.
4. **Where do Storybook stories live?** Proposal: `ui/src/features/review/stories/` with mocked MSW handlers for `/api/diff` etc., so widgets develop in isolation without a real repo pair.
5. **Widget ergonomics in markdown.** Authors will want helpers like `{sym:short/form}` that expand to full IDs. We kept short-form resolution in the directive parser; consider extending it so authors can write `codebase-impact sym=indexer.Extract` (no `sym:` prefix) and get a sensible lookup.

## 11. References

1. Existing dispatch: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocSnippet.tsx`
2. Existing stub emitter: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go:140-165`
3. Reusable card: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/symbol/ExpandableSymbol.tsx:27`
4. Xref panel: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/symbol/XrefPanel.tsx:13`
5. File xref panel: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/source/FileXrefPanel.tsx:14`
6. Code + refs: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/packages/ui/src/Code.tsx:36`
7. Companion design doc: `01-git-level-analysis-mapping-turning-codebase-browser-primitives-into-pr-review-data.md`
