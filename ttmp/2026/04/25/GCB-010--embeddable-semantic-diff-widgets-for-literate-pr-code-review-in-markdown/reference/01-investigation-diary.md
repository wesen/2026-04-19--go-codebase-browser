---
Title: Investigation diary
Ticket: GCB-010
Status: active
Topics:
    - codebase-browser
    - pr-review
    - semantic-diff
    - embeddable-widgets
    - markdown-directives
    - history-index
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological diary of the design investigation for embeddable semantic diff widgets.
LastUpdated: 2026-04-25T12:00:00Z
WhatFor: Record what was analysed, what decisions were made, and where to pick up next.
WhenToUse: Read before resuming work on GCB-010.
---

# Diary

## Goal

Record the investigation and design process for GCB-010 — embeddable semantic diff widgets that allow literate PR code review guides to be written as markdown with live, navigable source browsing inline.

## Step 1: Project analysis and evidence gathering

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create embeddable semantic diff widgets into markdown files. The idea is to write up literate pull request code review guides in markdown to guide a reviewer, and allow them to navigate the source right there in the markdown files without having to open a new window. This can be plain code browsing, browsing semantic changes that happened over a series of commits, etc... Analyze the current project to know what this is all about, about the functionality we built wrt git commit semantic indexing, and then write a detailed design / brainstorm document about affordances to support code review with rich primitives, including with ASCII widget screenshots and such."

**Assistant interpretation:** Analyse the full codebase-browser project, understand the semantic indexing pipeline (Go/TS AST extraction → index.json → embedded server → React SPA), the git history subsystem (per-commit snapshots in SQLite, symbol diff, body diff, history tracking), and the markdown doc system (codebase-* directives → hydrated React stubs). Then produce a comprehensive design document for embeddable widgets that make literate PR review guides possible directly in markdown.

**Inferred user intent:** Manuel wants a thorough design doc that captures the full system architecture and proposes concrete widget affordances for embedding semantic diff, history, impact, and code-browsing primitives into markdown — targeting code reviewers who can read everything in one document without switching windows.

### What I did

- Read the full README.md for the project overview
- Read `internal/history/indexer.go` — the per-commit semantic indexer that creates worktrees and runs AST extraction at each commit
- Read `internal/history/schema.go` — the SQLite history schema (commits, snapshot_symbols, snapshot_refs, file_contents, symbol_history view)
- Read `internal/history/diff.go` — the symbol-level diff engine comparing two commit snapshots
- Read `internal/history/bodydiff.go` — per-symbol body diff with unified diff output
- Read `internal/history/scanner.go` — git commit discovery and filtering
- Read `internal/history/loader.go` — bulk-loading commit snapshots into the history DB
- Read `internal/history/cache.go` — file content caching and retrieval
- Read `internal/gitutil/log.go` — git CLI wrapper for commit listing, changed files, ancestry checks
- Read `internal/docs/renderer.go` — the full markdown directive pipeline (preprocess → stub emission → goldmark rendering)
- Read `internal/server/api_history.go` — all /api/history/* REST endpoints
- Read `internal/server/server.go` — the full route table
- Read `internal/indexer/types.go` — canonical index schema (Index, Symbol, File, Package, Ref, Range)
- Read `internal/concepts/types.go` — structured query concepts
- Read `ui/src/features/history/HistoryPage.tsx` — the full React history UI (commit timeline, diff view, symbol history, body diff)
- Read `internal/docs/embed/pages/03-meta.md` — self-documenting example page
- Read GCB-005 design doc #1 (git-level analysis mapping)
- Read GCB-005 design doc #2 (UI affordances and widget catalog)
- Created the docmgr ticket GCB-010 with design doc and diary doc
- Related 11 key files to the design doc

### Why

Understanding the full pipeline from git commits → AST extraction → SQLite snapshots → REST API → React widgets → markdown directives is essential before designing new widgets that bridge these layers for literate review guides.

### What worked

- The project has excellent traceability: every design decision is in prior tickets (GCB-001 through GCB-009)
- The history subsystem (GCB-009) already provides the core primitives: per-commit snapshots, symbol diff, body diff, history tracking
- The doc directive pipeline (GCB-004) already provides the embedding mechanism: codebase-* directives → stubs → hydrated React components
- GCB-005 already sketched the PR review architecture and widget catalog, which is a strong foundation

### What didn't work

- N/A — evidence gathering was straightforward

### What I learned

- The system has three layers that compose cleanly:
  1. **Data layer**: `internal/history/` — SQLite with per-commit symbol snapshots, body hashes, file contents
  2. **API layer**: `internal/server/api_history.go` — REST endpoints for commits, diffs, symbol history, body diffs
  3. **Rendering layer**: `ui/src/features/history/` + `internal/docs/renderer.go` — React components + markdown directive pipeline
- The key insight for this ticket: the existing infrastructure was designed for a full-page history view, but the *same primitives* can be embedded inline in markdown for literate review guides
- The `codebase-*` directive system already supports 4 directives (`codebase-snippet`, `codebase-signature`, `codebase-doc`, `codebase-file`). Extending it with diff/history/impact directives is a natural evolution
- The `symbol_history` view in the history DB already joins symbols across commits, which is exactly what inline diff widgets need

### What was tricky to build

- Understanding the interaction between the static index (index.json for HEAD) and the dynamic history DB (SQLite for multiple commits) — the review widgets need to draw from both
- The body diff currently requires the repo root on disk (for `git show` fallback) — embedded widgets in a served doc page may not have this available, which constrains the design

### What warrants a second pair of eyes

- The assumption that all widget rendering can happen client-side (React) with server-side data fetching, rather than server-side HTML rendering of the diff widgets
- The interaction between the history DB and the static index — should the new directives query both, or only the history DB?

### What should be done in the future

- Implement the new directives and widgets per the design doc
- Add E2E tests for the embedded widget rendering pipeline
- Consider static HTML rendering of diff widgets for non-JS environments (e.g., email, PDF export)

### Code review instructions

- Start with the design doc at `design-doc/01-*.md`
- Cross-reference with `internal/history/diff.go` and `internal/docs/renderer.go` for the extension points
- Verify the API contracts match the existing `api_history.go` patterns

### Technical details

Files analysed (with line counts):

| File | Lines | Role |
|------|-------|------|
| `internal/history/indexer.go` | ~130 | Per-commit indexing orchestrator |
| `internal/history/schema.go` | ~85 | SQLite history schema |
| `internal/history/diff.go` | ~120 | Symbol-level diff engine |
| `internal/history/bodydiff.go` | ~130 | Per-symbol body diff |
| `internal/history/scanner.go` | ~60 | Git commit discovery |
| `internal/history/loader.go` | ~160 | Snapshot bulk loader |
| `internal/history/cache.go` | ~80 | File content caching |
| `internal/docs/renderer.go` | ~250 | Markdown directive pipeline |
| `internal/server/api_history.go` | ~170 | History REST API |
| `ui/src/features/history/HistoryPage.tsx` | ~430 | React history UI |

## Step 2: Write the design document

### Prompt Context

**User prompt (verbatim):** "write the design document in smaller parts to avoid having a massive write."

**Assistant interpretation:** Write the design document in sequential chunks using bash append, to avoid hitting any write limits.

**Inferred user intent:** Complete the full design document efficiently by breaking it into manageable sections.

### What I did

- Wrote the design doc in 4 chunks:
  1. Frontmatter + Section 1 (executive summary) + Section 2 (problem statement)
  2. Section 3 (system architecture overview — 6 subsections covering index, xrefs, doc pipeline, history subsystem, React frontend)
  3. Section 4 (widget catalog — 9 widgets with ASCII wireframes, API contracts, authoring syntax)
  4. Sections 5–10 (API design, data flow, implementation plan, risks, file reference index, summary)
- Created docmgr tasks (8 tasks for the phased implementation plan)
- Updated the ticket index
- Added 3 vocabulary entries (embeddable-widgets, history-index, markdown-directives)
- Ran docmgr doctor — all checks pass

### Why

The design doc needs to be self-contained for onboarding new developers. Breaking it into sections made it manageable to write while maintaining coherence.

### What worked

- The bash `cat >> file << 'EOF'` pattern worked well for appending sections
- The widget catalog with ASCII wireframes gives a concrete picture of each affordance
- The data flow diagram (Section 6) traces the full pipeline end-to-end

### What didn't work

- N/A — smooth execution

### What I learned

- The design naturally fell into 7 new directives plus 2 extensions to existing ones
- The commit-walk widget is the most complex but also the most valuable — it's the "killer feature" for literate review guides
- The existing API endpoints cover most widget needs; only `/api/history/impact` is truly new

### What was tricky to build

- Keeping the ASCII wireframes legible at different terminal widths
- Balancing detail vs. brevity — the doc is 67KB, which is long but necessary for onboarding

### What warrants a second pair of eyes

- The commit-walk widget's DSL syntax (`step "..." Show ...`) — is this the right authoring ergonomics?
- The impact BFS implementation — is the compatibility check logic sound?

### What should be done in the future

- Upload to reMarkable (next step)
- Implement Phase 1 after review

### Code review instructions

- Read the full design doc: `design-doc/01-embeddable-semantic-diff-widgets-design-affordances-and-architecture-for-literate-pr-review.md`
- Start with Sections 1–2 for motivation, Section 3 for architecture, Section 4 for the widget catalog
- Cross-reference the ASCII wireframes with the existing HistoryPage.tsx to verify composition

### Technical details

Design doc final size: 67,512 bytes (10 sections)

## Step 3: Rewrite implementation plan as incremental vertical slices

### Prompt Context

**User prompt (verbatim):** "we want to build this in incremental phases to see results as we build to make sure we are going in the right direction, what would that look like?"

**Assistant interpretation:** Rewrite the implementation plan as vertical slices where each slice delivers a complete, demonstrable feature end-to-end, so the team can validate each widget before investing in the next.

**Inferred user intent:** Avoid the waterfall approach (all directives → all APIs → all widgets). Instead, build one working widget at a time, demo it, validate the direction, then move on.

### What I did

- Analysed the dependency chain between widgets
- Designed 6 vertical slices (Slice 0–5), each with: what you can demo, files touched, validation steps, and decision gates
- Wrote Section 11 of the design doc (~10KB) with detailed per-slice breakdowns
- Rewrote the tasks.md to match the slice structure (5 slices + polish phase, 33 tasks total)
- Updated the diary

### Why

The original phased plan organised work by layer (directives, APIs, widgets). This is efficient if you already know exactly what to build, but risky when exploring a new surface. Vertical slices let you course-correct early: if the inline diff doesn't feel right after Slice 1, you can adjust the approach before building 6 more widgets.

### What worked

- Slice 0 (`commit=` on existing directives) is a great starting point — it's 1 day of work and validates the entire plumbing without building any new widget
- The two decision gates (after Slice 1: diff layout; after Slice 3: BFS performance) give natural pause points
- The rollback story is clean: each slice only adds new files and extends a switch statement, never modifies existing widget code

### What didn't work

- N/A

### What I learned

- The dependency chain is flatter than expected: most widgets only depend on the plumbing from Slice 0, not on each other. Only the commit-walk widget (Slice 5) composes the others.
- Three of the seven widgets (changed-files, diff-stats, annotation) are trivially simple once the patterns from Slices 1–3 are established — they can be batched into Slice 4.

### What was tricky to build

- Balancing slice size: Slice 1 (the diff widget) is 2–3 days, which feels large for a "vertical slice". But it's the hardest widget and the most important one to validate early, so it earns its size.

### What warrants a second pair of eyes

- The slice ordering: is the diff widget (Slice 1) really the right first widget, or should we start with something even simpler like diff-stats?
- The decision gates: are they at the right points?

### What should be done in the future

- Execute Slice 0 first, validate the plumbing, then decide on Slice 1 vs. a simpler alternative

### Code review instructions

- Read Section 11 of the design doc (the new incremental plan)
- Compare with the tasks.md to verify alignment
- Consider whether the slice ordering matches your intuition for "fastest path to a demo"

## Step 4: Implement Slice 0 — commit= on existing directives

### Prompt Context

**User prompt (verbatim):** "Implement slice 0, commit at appropriate intervals, keep a diary. Run server in tmux, ask me to validate when done."

**Assistant interpretation:** Implement the full Slice 0 vertical slice: extend existing codebase-snippet/signature/doc directives with a `commit=<hash>` parameter so they resolve from the history DB instead of the static index. Commit at logical checkpoints. Start the server for validation.

**Inferred user intent:** Get a working, demonstrable feature end-to-end that shows the same symbol at two different commits, then have the user validate it.

### What I did

(Recording as I go)

### What I did (implementation)

**Commit 1** (`3eed622`): Server + renderer changes
- `internal/server/api_source.go`: Extended `handleSnippet` with `commit=` query param. When present and history DB is available, resolves from `snapshot_symbols` + `snapshot_files` + `file_contents` via new `handleSnippetFromHistory` method. Extracted `applyKind` helper to DRY the kind logic.
- `internal/docs/renderer.go`: `SnippetRef` gains `CommitHash` field. `resolveDirective` captures the `commit=` param. `stubHTML` emits `data-commit` attribute on the stub div.

**Commit 2** (`e457069`): Frontend changes
- `ui/src/features/doc/DocPage.tsx`: `StubHandle` interface gains optional `commit` field. The stub walker extracts `data-commit` from the DOM. Passes it through to `DocSnippet`.
- `ui/src/features/doc/DocSnippet.tsx`: Added `commit` prop. Created `useGetSnippetFromCommit` hook that fetches from `/api/snippet?commit=<hash>` when set. `DocFullSnippet` renders a simple code block with commit label for non-HEAD snippets, preserving the existing `ExpandableSymbol` treatment for HEAD.

**Commit 3** (`0db8698`): Demo page
- `internal/docs/embed/pages/04-slice0-demo.md`: Shows `stubHTML` and `handleSnippet` at two different commits.

**Commit 4** (`2fbddb1`): Fix demo page symbol IDs
- `handleSnippet` is a method (`method.Server.handleSnippet`), not a func. Fixed the demo page to use the correct IDs.

### What worked

- The `handleSnippetFromHistory` SQL query (JOIN between `snapshot_symbols` and `snapshot_files`) works perfectly on the first try
- The `applyKind` extraction DRYed up the code nicely
- The `data-commit` attribute propagation (renderer → HTML → DOM → React → fetch) is clean

### What didn't work

- First attempt at the demo page used short-form symbol references (`github.com/.../docs.func.stubHTML`) which don't resolve correctly — the `LastIndex(".")` split treats `.func.` as part of the import path. Fixed by using full `sym:...` IDs.
- Also used `func.handleSnippet` when it's actually `method.Server.handleSnippet`. Fixed.
- The initial server restart didn't pick up the rebuilt binary (user error — needed to kill and restart tmux session)

### What I learned

- Short-form symbol resolution is fragile for nested packages. Full `sym:...` form is safer for demo/test pages.
- The `useGetSnippetFromCommit` hook uses direct `fetch()` instead of RTK-Query. This is fine for Slice 0 but should migrate to a proper RTK-Query endpoint in later slices for caching and loading state.

### What was tricky to build

- The `useGetSnippetFromCommit` hook needs to avoid re-fetching on re-render. Used a `useRef<Map>` cache pattern with `useReducer` for forcing re-renders. This is a common pattern but has edge cases with concurrent mode that should be revisited.

### What warrants a second pair of eyes

- The `useGetSnippetFromCommit` hook — is the cache+forceUpdate pattern correct, or should we use RTK-Query's `fetchBaseQuery` directly?
- Should `handleSnippetFromHistory` resolve short-form symbol IDs too, or only full `sym:...` IDs?

### What should be done in the future

- Migrate `useGetSnippetFromCommit` to a proper RTK-Query endpoint
- Add short-form resolution to `handleSnippetFromHistory`
- Consider adding `commit=` to the static `/api/snippet` endpoint for cases where history DB is available but the symbol isn't in the static index

### Code review instructions

- Read `internal/server/api_source.go` — the `handleSnippetFromHistory` method and the `applyKind` helper
- Read `ui/src/features/doc/DocSnippet.tsx` — the `useGetSnippetFromCommit` hook and the `DocFullSnippet` commit branch
- Read `ui/src/features/doc/DocPage.tsx` — the `data-commit` extraction from DOM stubs
- Open http://localhost:3001/doc/04-slice0-demo to validate

### Technical details

4 commits total. Server running in tmux session `codebase-browser` at :3001.

## Step 5: Fix WASM runtime loading and validate Slice 0 with Playwright

The first browser validation surfaced a frontend startup error: `Go WASM runtime not loaded. Include wasm_exec.js before loading this module.` This happened because the rebuilt Vite `index.html` loaded the app bundle directly without first loading Go's `wasm_exec.js`. The binary already had `wasm_exec.js` in embedded public assets from older builds, but the generated `index.html` no longer referenced it.

I fixed this as a real frontend issue, not as a manual local patch: copied `internal/wasm/embed/wasm_exec.js` into `ui/public/wasm_exec.js`, added `<script src="/wasm_exec.js"></script>` before the module script in `ui/index.html`, and rebuilt the frontend. While testing with Playwright, I also found that the UI doc list and doc route were still backed by WASM precomputed docs, so the new demo page did not appear even though `/api/doc/04-slice0-demo` worked. I changed `ui/src/api/docApi.ts` to prefer live `/api/doc` and `/api/doc/:slug` in server-backed mode, falling back to WASM for static deployments.

### What I did

- Added `ui/public/wasm_exec.js`
- Added the `wasm_exec.js` script tag before the app module in `ui/index.html`
- Updated `ui/src/api/docApi.ts` to prefer live server `/api/doc` endpoints, falling back to WASM
- Ran `pnpm -C ui run typecheck`
- Ran `pnpm -C ui build`
- Rebuilt the binary with `go build -tags embed -o codebase-browser ./cmd/codebase-browser/`
- Restarted tmux server with `./codebase-browser serve --addr :3001 --history-db history.db --repo-root .`
- Used Playwright to load `http://localhost:3001/#/doc/04-slice0-demo`
- Verified console has 0 errors
- Verified page text contains the Slice 0 demo and the old/new commit snippets

### What worked

- After a hard reload, Playwright showed scripts loaded in the right order: `/wasm_exec.js` before `/assets/index-*.js`
- The Slice 0 demo page now appears in the left Docs nav
- The demo page renders the old `stubHTML` implementation at `c913257` and the new implementation at `e457069`
- Playwright console errors: 0

### What didn't work

- Initial Playwright navigation to `/doc/04-slice0-demo` was wrong because this SPA uses hash routing; correct URL is `/#/doc/04-slice0-demo`
- The browser initially had the older asset URL cached; a hard reload picked up the new bundle

### What I learned

- For server-backed validation, doc pages should prefer `/api/doc` over WASM precomputed docs. Otherwise new markdown pages require regenerating the static WASM bundle before they appear in the UI.
- The `wasm_exec.js` dependency is easy to regress because Vite's generated `index.html` only includes scripts declared in `ui/index.html`; keeping it in `ui/public` plus an explicit script tag makes this robust.

### What warrants a second pair of eyes

- Whether `docApi` should always prefer `/api/doc` in production serve mode, or whether we should detect static mode explicitly. Current implementation tries `/api/doc` and falls back to WASM, which is simple and robust.

### Code review instructions

- Inspect `ui/index.html`, `ui/public/wasm_exec.js`, and `ui/src/api/docApi.ts`
- Use Playwright or browser console to confirm there are no WASM runtime errors
- Validate `http://localhost:3001/#/doc/04-slice0-demo`

## Step 6: Add syntax highlighting and implement Slice 1 (`codebase-diff`)

The user asked for syntax highlighting on commit-resolved snippets and then to proceed to the next phase. I first replaced the plain `<pre><code>` rendering for `commit=` snippets with the shared `<Code>` component, so history-backed snippets now get the same token spans and colours as normal snippets. Playwright confirmed 1,900+ `data-tok` spans on the Slice 0 demo and 0 console errors.

Then I implemented Slice 1 as a complete vertical slice: `codebase-diff` in the markdown renderer, `data-params` plumbing through doc hydration, a new `SymbolDiffInlineWidget`, and a demo page.

### What I did

- `ui/src/features/doc/DocSnippet.tsx`: imported `<Code>` and used it for commit-resolved snippets and signatures
- `internal/docs/renderer.go`: added `Params` to `SnippetRef`, emitted safe `data-params` HTML attributes, added `case "codebase-diff"`
- `ui/src/features/doc/DocPage.tsx`: parsed `data-params` JSON from stubs
- `ui/src/features/doc/DocSnippet.tsx`: dispatched `codebase-diff` to the new widget
- `ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx`: new inline unified diff widget backed by `/api/history/symbol-body-diff`
- `internal/docs/embed/pages/05-slice1-diff-demo.md`: demo page with `stubHTML` and `handleSnippet` diffs

### Validation

Commands run:

```bash
go test ./internal/docs ./internal/server
pnpm -C ui run typecheck
pnpm -C ui build
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Server restarted in tmux:

```bash
./codebase-browser serve --addr :3001 --history-db history.db --repo-root .
```

Playwright validation:

- `http://localhost:3001/#/doc/04-slice0-demo`: syntax highlighted commit snippets (`data-tok` spans present), 0 console errors
- `http://localhost:3001/#/doc/05-slice1-diff-demo`: `codebase-diff` widgets render, 0 console errors

### What didn't work

- The first `data-params` implementation used `%q`/backslash-escaped JSON in an HTML attribute. Browsers do not treat backslashes as quote escapes in HTML, so `getAttribute('data-params')` returned only `{\`. I fixed it by HTML-escaping JSON quotes (`&#34;`) with `html.EscapeString`, so the browser decodes the attribute back to valid JSON.
- The first Playwright check reused an old tab with cached RTK Query state. Opening a fresh tab showed the corrected `data-params` and widget rendering.

### What I learned

- For arbitrary JSON in data attributes, use HTML entity escaping, not JS-style backslash escaping.
- The minimal Slice 1 widget can reuse the existing `/api/history/symbol-body-diff` endpoint with no new server API.

### What warrants a second pair of eyes

- `SymbolDiffInlineWidget` currently renders a unified diff, not side-by-side. This is intentional for the first vertical slice, but we should evaluate whether reviewers prefer unified or split before investing in a richer layout.

### Code review instructions

- Start with `internal/docs/renderer.go` (`codebase-diff` case and `data-params` escaping)
- Then review `SymbolDiffInlineWidget.tsx`
- Validate both demo pages in the browser

## Step 7: Implement Slice 2 (`codebase-symbol-history`)

I continued with Slice 2 after the syntax-highlighting and Slice 1 work. The goal was a compact inline history timeline that shows which indexed commits contain a symbol, marks body changes with a filled dot, and lets the reviewer click a changed row to expand the existing inline diff widget against its predecessor.

### What I did

- `internal/docs/renderer.go`: added `case "codebase-symbol-history"`, requiring `sym=` and accepting optional `limit=`. The directive resolves the symbol to its full `sym:...` ID and emits `data-params` for the limit.
- `ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx`: new component using `useGetSymbolHistoryQuery`. It renders commit rows with body-change dots, date, short hash, message, and line range.
- `ui/src/features/doc/DocSnippet.tsx`: dispatches `codebase-symbol-history` to the new widget and parses `limit` as a number.
- `internal/docs/embed/pages/06-slice2-history-demo.md`: demo page for `stubHTML` and `handleSnippet` histories.

### Validation

Commands run:

```bash
go test ./internal/docs ./internal/server
pnpm -C ui run typecheck
pnpm -C ui build
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Server restarted in tmux:

```bash
./codebase-browser serve --addr :3001 --history-db history.db --repo-root .
```

Playwright validation:

- Loaded `http://localhost:3001/#/doc/06-slice2-history-demo`
- Verified two history widgets render (`stubHTML`, `handleSnippet`)
- Clicked a changed row (`3eed622` for `stubHTML`) and verified an inline diff expands below the timeline
- Console errors: 0

### What worked

- Reusing `SymbolDiffInlineWidget` for row expansion made the implementation small and kept the UI consistent with Slice 1.
- The existing `/api/history/symbols/{id}/history` endpoint had enough data for the timeline.

### What didn't work

- My first Playwright click targeted the newest row (`e457069`) which was unchanged relative to its predecessor, so no diff appeared. Clicking a filled-dot row validated the expansion path.

### What I learned

- The history API orders newest-first, so the predecessor of row `i` is row `i+1`.
- Filled-dot semantics should be explained in the widget footer because rows can exist without a body change.

### What warrants a second pair of eyes

- The current grid layout is dense and works for the demo, but it may need responsive tuning for smaller screens.
- The widget marks the root commit as changed because it has no predecessor in the window. That is technically true for the visible window but may be confusing; we might want a distinct "introduced/out-of-window" marker later.

### Code review instructions

- Review `SymbolHistoryInlineWidget.tsx` first; it owns the interaction and row/diff expansion logic.
- Then review the `codebase-symbol-history` case in `renderer.go`.
- Validate `/#/doc/06-slice2-history-demo` and click the filled-dot rows.

## Step 8: Implement Slice 3 (`codebase-impact`)

I continued with Slice 3, the first slice that needed a genuinely new server endpoint. The goal was an inline caller/callee impact widget backed by a bounded BFS over the per-commit `snapshot_refs` table.

### What I did

- `internal/server/api_history.go`: added `GET /api/history/impact` and helper functions:
  - `latestHistoryCommit`
  - `impactBFS`
  - `impactOneHop`
  - `impactSymbolMeta`
- `internal/docs/renderer.go`: added `case "codebase-impact"`, accepting `sym=`, `dir=usedby|uses`, `depth=`, and optional `commit=`.
- `ui/src/api/historyApi.ts`: added `ImpactResponse`, `ImpactNode`, `ImpactEdge` types and `useGetImpactQuery`.
- `ui/src/features/doc/widgets/ImpactInlineWidget.tsx`: new widget grouping impact nodes by depth.
- `ui/src/features/doc/DocSnippet.tsx`: dispatches `codebase-impact` to the new widget.
- `internal/docs/embed/pages/07-slice3-impact-demo.md`: demo page showing both `usedby` and `uses` directions.

### Validation

Commands run:

```bash
go test ./internal/docs ./internal/server
pnpm -C ui run typecheck
pnpm -C ui build
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Server restarted in tmux:

```bash
./codebase-browser serve --addr :3001 --history-db history.db --repo-root .
```

Playwright validation:

- Loaded `http://localhost:3001/#/doc/07-slice3-impact-demo`
- Verified two impact widgets render
- `writeJSON usedby depth=2` showed 21 symbols: 18 at depth 1, 3 at depth 2
- `handleSnippet uses depth=2` showed 26 symbols: 16 at depth 1, 10 at depth 2
- Console errors: 0

### What worked

- The `snapshot_refs` table already had enough information for useful impact lists.
- The endpoint is fast on the current history DB and returns compact JSON.
- The widget works for both graph directions (`usedby` and `uses`).

### What didn't work / caveats

- Some external symbols (e.g. `sym:fmt.func.Sprintf`) don't have `snapshot_symbols` metadata in our DB, so the widget displays the full symbol ID and `kind=symbol`. This is acceptable for Slice 3 but should be polished later.
- Compatibility indicators are currently placeholders (`unknown`). The first useful version focuses on impact reachability; true signature compatibility can come later when we compare from/to diffs.

### What I learned

- Depth-2 impact is already useful; for high fan-in helpers like `writeJSON`, depth 2 exposes route registration paths (`Handler`, `registerHistoryRoutes`).
- External refs need separate display treatment because they are present as ref targets but not necessarily indexed as local snapshot symbols.

### What warrants a second pair of eyes

- The BFS response currently updates a pointer map and a response slice in sync. It works, but could be simplified by collecting pointers first and flattening at the end.
- The default `commit` selection is latest indexed commit by `author_time`; this is practical for demos but the UI should expose/accept explicit commit selection later.

### Code review instructions

- Start with `handleHistoryImpact` and `impactBFS` in `internal/server/api_history.go`.
- Then review `ImpactInlineWidget.tsx` for rendering and grouping semantics.
- Validate `/#/doc/07-slice3-impact-demo`.

## Step 9: Polish Slice 3 impact links and external symbols

The user noticed that clicking `handleConceptDetail` in the `writeJSON` "used by" list did not resolve. The underlying issue is that impact nodes come from the history DB snapshot, while `/symbol/:id` resolves against the static embedded HEAD index. When the static index is older or does not contain that symbol, the click lands on "Symbol not found" even though the history DB has the symbol.

I polished Slice 3 by making impact rows history-backed: the server now marks each impact node as `local` if it exists in `snapshot_symbols`, and the frontend links local rows to `/history?symbol=...` instead of `/symbol/...`. This makes `handleConceptDetail` resolve to the standalone symbol history view. External nodes are displayed as non-links with a compact fallback name and an `external` badge.

### What I did

- `internal/server/api_history.go`
  - Added `local` boolean to impact nodes
  - Changed `impactSymbolMeta` to report whether a symbol is local to the snapshot
  - Added `impactFallbackName` for external refs such as `sym:fmt.func.Sprintf`
- `ui/src/api/historyApi.ts`
  - Added `local: boolean` to `ImpactNode`
- `ui/src/features/doc/widgets/ImpactInlineWidget.tsx`
  - Link local rows to `/history?symbol=<id>`
  - Render external rows as non-links with muted style and `external` badge
  - Improved header counts (local/external) and included the commit hash
  - Added a footer explaining why links use the history-backed route

### Validation

Commands run:

```bash
gofmt -w internal/server/api_history.go
go test ./internal/server ./internal/docs
pnpm -C ui run typecheck
pnpm -C ui build
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Playwright validation:

- Loaded `http://localhost:3001/#/doc/07-slice3-impact-demo`
- Verified the `writeJSON` impact widget renders with local counts
- Clicked `handleConceptDetail`
- Browser navigated to `/history?symbol=sym:...handleConceptDetail`
- Verified the symbol history view renders and shows the body diff
- Console errors: 0

### What worked

- `/history?symbol=...` is the right target for history-backed impact rows because it does not depend on the static embedded index.
- External refs are much clearer now: `Sprintf` instead of the full `sym:fmt.func.Sprintf` as the primary label, with the full ID retained as a tooltip.

### What warrants a second pair of eyes

- Whether local impact rows should eventually offer both links: "open history" and "open current symbol" when the current static index contains the symbol.
- Whether external refs should be hidden by default for `usedby`, or grouped separately.

### Code review instructions

- Review `ImpactInlineWidget.tsx` link behaviour first.
- Confirm `handleConceptDetail` in `/#/doc/07-slice3-impact-demo` opens a usable history view.

## Step 10: Hide redundant commit sidebar in symbol-history deep links

The user pointed out that `/#/history?symbol=...` still showed the left commit picker, which felt redundant. That route is used by impact-widget rows to open a focused symbol history view. The standalone symbol history panel already has its own from/to selectors for body diff, so the page-level commit-pair sidebar duplicates controls and distracts from the focused symbol.

### What I did

- Updated `ui/src/features/history/HistoryPage.tsx` so `CommitTimeline` immediately returns `<StandaloneSymbolHistory>` when `initialSymbol` is present.
- Removed the now-unreachable `initialSymbol ? ...` branch inside the normal two-column commit diff layout.

### Validation

Commands run:

```bash
pnpm -C ui run typecheck
pnpm -C ui build
go build -tags embed -o codebase-browser ./cmd/codebase-browser/
```

Playwright validation:

- Loaded `/#/history?symbol=sym:...handleConceptDetail`
- Verified `hasCommitsBox: false`
- Verified the focused symbol history table and body diff still render
- Console errors: 0

### What worked

- This is a small UI-only change with clear behaviour: ad-hoc `/history` keeps the commit picker; focused `/history?symbol=...` hides it.

### What warrants a second pair of eyes

- The copy at the top still says "5 indexed commit(s). Select two commits to diff." In symbol mode the instruction is slightly wrong; we should probably adjust the header copy in a later polish pass.

## Step 11: Fix symbol-mode history header copy

After hiding the redundant commit picker in `/#/history?symbol=...`, the top copy still said "Codebase history" and "Select two commits to diff", which no longer matched the focused symbol-history route. I updated the copy to be mode-aware.

### What I did

- `ui/src/features/history/HistoryPage.tsx`: added `symbolMode` derived from `initialSymbol`.
- In symbol mode, the page now says:
  - Heading: `Symbol history`
  - Description: `<N> indexed commit(s). Review this symbol across commits.`
- In normal history mode, existing copy remains unchanged.

### Validation

Ran frontend typecheck/build and rebuilt the embedded binary. Playwright loaded `/#/history?symbol=sym:...handleConceptDetail` and confirmed:

- Heading: `Symbol history`
- Copy: `5 indexed commit(s). Review this symbol across commits.`
- Console errors: 0
