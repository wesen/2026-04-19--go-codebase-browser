---
Title: Implementation Diary
Ticket: GCB-004
Status: active
Topics:
    - codebase-browser
    - documentation-tooling
    - react-frontend
    - embedded-web
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Diary for GCB-004 — replacing inline <pre><code> source snippets on doc pages with React-hydrated <SymbolCard> widgets via server-emitted data-stub divs.
LastUpdated: 2026-04-20T12:20:00Z
WhatFor: Step-by-step record of the stub-and-hydrate implementation including trade-offs, surprises, and review pointers.
WhenToUse: Read alongside the design doc when modifying the server renderer or the DocPage hydration path.
---

# Implementation Diary

## Goal

Ship the Option A design from the preceding conversation: let doc-page `codebase-snippet` directives render as the same `<SymbolCard>` / `<LinkedCode>` widgets used on symbol pages, via server-emitted placeholder stubs that the React frontend hydrates on mount.

## Step 1: Ticket scaffold + design doc

Created the GCB-004 ticket (GCB-003 was already claimed by an unrelated deployment project), filled in the design-doc with the full phased plan, and seeded five implementation tasks on `tasks.md`.

### Prompt Context

**User prompt (verbatim):** "add a ticket for option A and an implementation guide and then implement it."

**Assistant interpretation:** Create a docmgr ticket for the stub-hydration design described in the previous message, write a design/implementation guide detailing server stub emission + frontend hydration flow, then execute the three implementation phases committing at phase boundaries.

**Inferred user intent:** Close the "doc snippets are inert text" gap so that every `codebase-snippet` directive renders with syntax highlighting, xref links, and godoc annotations — matching the richness of `/symbol/{id}` pages — without any changes to doc-page authoring.

### What I did

1. `docmgr ticket create-ticket --ticket GCB-004 ...` (GCB-003 was taken by a pre-existing deployment ticket).
2. `docmgr doc add` for the design-doc + diary.
3. Wrote `design-doc/01-react-hydrated-snippet-widgets-design-and-implementation-guide.md` covering: problem, current state, target design, phased plan, detailed code changes, risks, and testing strategy.
4. Seeded `tasks.md` with five tasks (server renderer change, stub metadata, frontend hydration, meta-page exercise, fallback).

### Why

Having the design written out up front settles two decisions that would otherwise block mid-implementation: (1) the stub carries its plaintext fallback inside it (so JS-disabled readers still see something) and (2) each stub gets its own `createRoot` rather than sharing a single React tree (keeping goldmark-generated prose intact without trying to reconstruct it).

### What worked

Reusing the existing `SymbolCard` + `LinkedCode` components means the whole "hydration" step is maybe 30 lines of glue — `useEffect` + `querySelectorAll('[data-codebase-snippet]')` + `createRoot` per stub. Everything else is already built.

### What didn't work

N/A (scaffold step).

### What I learned

docmgr errors cleanly when a ticket ID collides; it doesn't overwrite the existing one. Found out by creating GCB-003 and getting an ambiguous-ticket error on follow-up commands. Chose GCB-004 instead and the issue disappeared.

### What should be done in the future

Implementation phases 1 (server stubs) and 2 (frontend hydration), then meta-page verification. If either surfaces a surprise, capture it in a follow-up diary step.

### Code review instructions

1. `cat ttmp/2026/04/20/GCB-004--.../design-doc/01-...md` — should contain §1 (exec summary) through §9 (references).
2. `docmgr task list --ticket GCB-004` — should show 5 open tasks.
3. No code changes in this step; the implementation phases land the actual renderer + frontend changes.

### Technical details

Ticket path: `ttmp/2026/04/20/GCB-004--react-hydrated-snippet-widgets-on-doc-pages/`.

## Step 2: Server renders stubs instead of inline source

Swapped the renderer's "replace the fenced block with a go-fenced code block" step for "replace with a raw-HTML `<div data-codebase-snippet …>` carrying the plaintext fallback inside." Goldmark passes the HTML block through unchanged. `SnippetRef` grew a `StubID` + `Language` field so the `/api/doc/{slug}` JSON contract lines up each stub with its metadata entry.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** `5d40b01` — "GCB-004 Phase 1: renderer emits data-codebase-snippet stubs"

### What I did

1. `SnippetRef` gained `StubID string` and `Language string`.
2. `resolveDirective` now reads `sym.Language` (defaults to `"go"`).
3. `preprocess` now:
   - Assigns each resolved directive a monotonic `stub-1`, `stub-2`, ... ID.
   - Emits the stub via a new `stubHTML(ref)` helper that writes `<div class="codebase-snippet" data-codebase-snippet data-stub-id=... data-sym=... data-directive=... data-kind=... data-lang=...><pre><code class="language-X">escaped text</code></pre></div>`.
   - Sandwiches the raw-HTML block with blank lines so goldmark treats it as a standalone HTML block rather than inline HTML (which would escape the `<` characters).
4. `stubHTML` picks the fallback body per directive: `<pre><code>` for snippet, `<pre><code>` for signature (same shape so the visual footprint matches), `<blockquote>` for doc.
5. Added `TestRender_EmitsHydrationStubs` asserting: N snippets produce N stub divs, each stub-ID appears as both a `SnippetRef.StubID` and a `data-stub-id=` attribute, and `data-directive` / `data-lang` reach the HTML correctly.

### Why

The renderer doubles as the hydration contract. If the stub attributes are wrong or duplicated, the frontend can't line them up with the `snippets[]` JSON. Making the IDs deterministic (`stub-N` over a UUID) keeps test output readable and stable.

### What worked

Goldmark's raw-HTML-block handling "just works" once the stub is separated from surrounding prose by blank lines. No custom AST walker or renderer extension needed — the existing pipeline is `markdown → [preprocess swap] → markdown → goldmark → html`, and raw HTML inside markdown is already a markdown feature.

### What didn't work

First attempt emitted the stub directly inside the flow of other paragraphs (no blank lines). Goldmark treated it as inline HTML and escaped the angle brackets, producing `&lt;div data-codebase-snippet&gt;...`. Fixed by inserting blank-line separators before and after the stub.

### What I learned

`html.EscapeString` from the stdlib is enough — I was tempted to pull in a third-party escaper. Also: the `snippets[]` JSON response is still a flat array; each entry's `stubId` is the only way to associate it with an HTML anchor. This deliberately avoids structured JSON for the prose + snippets; the prose stays free-form markdown.

### What was tricky to build

Maintaining backward-compat. The existing renderer tests assert "the signature text appears in HTML." Those still pass because the stub's `<code>` fallback contains the signature. If some future test asserted "the signature is inside a fenced-code-block," it would need updating. Current tests don't, so we're fine.

### What warrants a second pair of eyes

`stubHTML` uses `fmt.Sprintf` with `%q` for attribute values — works for simple strings but if a symbol ID ever contained a double-quote (it can't today) we'd need HTML-entity escaping instead. Flagging for someone to confirm the ID scheme's character set.

### What should be done in the future

Optional: a `codebase-snippet` variant that doesn't render the header (`?headless=true`) for use inline inside prose. Not needed today.

### Code review instructions

1. `go test ./internal/docs` — should pass all 5 tests.
2. `go run ./cmd/codebase-browser serve --addr :3001` then `curl /api/doc/03-meta | jq '.snippets'` — each entry should carry a `stubId`, and `jq '.html' | grep -c data-codebase-snippet` should equal `len(snippets)`.

## Step 3: Frontend hydrates stubs with rich widgets

Added `DocSnippet.tsx` which dispatches on directive (`snippet` → `<LinkedCode>` with xrefs, `signature` → `<Link>`, `doc` → `<blockquote>`). Amended `DocPage.tsx` with a post-mount `useEffect` that scans the article for `[data-codebase-snippet]` stubs, clears each stub's fallback, and uses `createPortal` to mount the right widget in place. Portals inherit the outer `<Provider>` so RTK-Query hooks inside `<DocSnippet>` just work.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** `87af108` — "GCB-004 Phase 2: hydrate doc-page stubs with rich React widgets"

### What I did

1. `docApi.SnippetRef` grew `stubId` + `language` fields to match the server contract.
2. `ui/src/features/doc/DocSnippet.tsx` — new file, 3 sub-components for the 3 directive kinds.
   - Snippet uses `useGetSymbolQuery` + `useGetSnippetQuery` + `useGetSnippetRefsQuery` → `<LinkedCode>`, same stack as `/symbol/{id}` pages.
   - Signature uses `useGetSymbolQuery` → compact `<pre><code>` wrapped in `<Link>`.
   - Godoc uses `useGetSymbolQuery` → `<blockquote>`.
3. `ui/src/features/doc/DocPage.tsx` — added:
   - `articleRef` to point into the SSR'd article element.
   - `useEffect([data?.html])` that walks `articleRef.current.querySelectorAll('[data-codebase-snippet]')`, reads the stub's `data-*` attributes, clears its fallback `innerHTML`, and pushes it onto a `stubs` state array.
   - JSX `{stubs.map(createPortal(<DocSnippet {...s}/>, s.el, key))}` under the article to mount one portal per stub.

### Why

Portals over `createRoot`: a fresh `createRoot` doesn't inherit the outer React context (Redux `<Provider>`, React-Router `<BrowserRouter>`), which means RTK-Query hooks inside the children would fail to find the store. `createPortal` keeps the subtree under the same React context while physically placing it in the stub's DOM node.

### What worked

First try compiled and produced visually correct HTML from the server. A single `useEffect` keyed on `data?.html` handles both the initial mount and re-hydration when a user navigates between two doc pages (React re-runs the effect when the HTML string changes, and the `setStubs([])` path clears old stubs when data is missing).

### What didn't work

`useGetSnippetQuery(sym)` signature — I forgot it takes `{ sym }` as an object (per `sourceApi.ts:21`). Fixed by passing `{ sym }` instead of the bare string. Caught by `pnpm typecheck` on first pass.

### What I learned

`createPortal` is the right primitive when you need to render React children into a DOM node that is *inside* an existing React tree but was inserted imperatively (e.g. by `dangerouslySetInnerHTML`). Think of it as "this child belongs to this component's React context but lives at that DOM location."

### What was tricky to build

Getting the effect dependency right. `[data?.html]` covers the common case (page navigation), but if the same HTML string is served twice (unlikely but possible with RTK-Query caching), the effect doesn't re-run. That's fine — the stubs are still mounted from the first run. The trickier case is if a new doc page happens to emit stubs for the same symbols with the same order; React's key-based portal list (`${slug}-${i}`) ensures each portal gets a stable identity across renders.

### What warrants a second pair of eyes

1. StrictMode double-mount: the effect runs twice in dev mode. Since `setStubs(found)` is idempotent (the list of stubs is the same both times), this should be OK. Worth eyeballing in a live dev session to confirm no duplicate portals.
2. The plaintext fallback gets cleared the moment the effect runs. If the effect runs before the HTML is painted, readers might see a flash. In practice the effect runs after `dangerouslySetInnerHTML` mounts, so the order is paint-then-clear-then-portal. Visual verification useful.

### What should be done in the future

1. Storybook story for `<DocSnippet>` with a mocked RTK-Query provider — makes the three directives developable in isolation.
2. CSS polish: stub-reserved min-height to avoid layout shift when the hydrated widget is taller than the plaintext fallback.

### Code review instructions

1. `pnpm -C ui run typecheck` — clean.
2. `pnpm -C ui run build` — produces `dist/public/` without errors.
3. `make build && ./bin/codebase-browser serve --addr :3001` — open `http://localhost:3001/doc/03-meta` and confirm:
   - All 6 blocks render with syntax highlighting (Go + TS tokens coloured).
   - `Merge` header shows `func` kind badge + clickable name.
   - Identifier names inside `Merge`'s body (e.g. `Package`, `File`) are blue xref links that navigate to their symbol pages.
   - `Server.handleXref` signature block is clickable and jumps to that method's page.

### Technical details

DocPage hydration hook in full (key part):

```tsx
useEffect(() => {
  if (!data || !articleRef.current) { setStubs([]); return; }
  const found: StubHandle[] = [];
  articleRef.current.querySelectorAll<HTMLElement>('[data-codebase-snippet]')
    .forEach((el) => {
      const sym = el.getAttribute('data-sym') ?? '';
      const directive = el.getAttribute('data-directive') ?? '';
      // ...
      el.innerHTML = '';
      found.push({ el, sym, directive, kind, lang });
    });
  setStubs(found);
}, [data?.html]);

// in render:
{stubs.map((s, i) =>
  createPortal(<DocSnippet {...s} />, s.el, `${slug}-${i}`))}
```

The 6 stubs on `03-meta` dispatch to 3 full-snippet widgets (`Merge`, `tokenize`, `isJsxComponentRef`), 2 signature widgets (`handleXref`, `runDagger`), and 1 extra signature (`Merge`). Each full-snippet widget fetches its own `/api/symbol/{id}` + `/api/snippet` + `/api/snippet-refs` lazily via RTK-Query; cached across navigations (`keepUnusedDataFor: 3600`).
