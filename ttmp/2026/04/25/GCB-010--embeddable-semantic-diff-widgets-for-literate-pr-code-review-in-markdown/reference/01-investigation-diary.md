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
