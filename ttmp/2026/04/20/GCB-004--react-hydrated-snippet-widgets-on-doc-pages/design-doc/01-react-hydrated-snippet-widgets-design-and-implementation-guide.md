---
Title: React-hydrated snippet widgets — Design and Implementation Guide
Ticket: GCB-004
Status: active
Topics:
    - codebase-browser
    - documentation-tooling
    - react-frontend
    - embedded-web
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/docs/renderer.go
      Note: Markdown renderer that currently inlines raw source snippets
    - Path: ui/src/features/doc/DocPage.tsx
      Note: Frontend consumer that dangerouslySetInnerHTML's the server HTML
    - Path: ui/src/packages/ui/src/SymbolCard.tsx
      Note: Reusable rich snippet widget to hydrate with
    - Path: ui/src/features/symbol/LinkedCode.tsx
      Note: Router-aware code view with click-through xrefs
ExternalSources: []
Summary: Replace the raw <pre><code> snippets that the doc-page renderer inlines into HTML with placeholder stubs that the React frontend hydrates into full <SymbolCard> widgets. Result is that every `codebase-snippet`/`codebase-signature`/`codebase-doc` directive in a doc page renders the same interactive component used on symbol pages — syntax highlighting, click-through xrefs, godoc annotations, no duplication.
LastUpdated: 2026-04-20T12:18:00Z
WhatFor: Make doc pages first-class consumers of the same rich UI widgets symbol pages use, so live source snippets get syntax highlighting + xref links instead of inert text.
WhenToUse: Read before touching internal/docs/renderer.go or adding a second frontend doc-rendering path.
---

# React-hydrated snippet widgets — Design and Implementation Guide

## 1. Executive summary

Today's doc pages serve markdown → HTML with `codebase-snippet` directives resolved server-side into raw `<pre><code>` blocks, then the frontend just `dangerouslySetInnerHTML`s the string. The snippets render, but they're text-only: no syntax highlighting, no click-through xrefs, no godoc annotation spans, no inline expansion. Symbol pages have all of those via `<SymbolCard>` + `<LinkedCode>`.

This ticket makes doc pages reuse those widgets. The renderer emits empty `<div data-codebase-snippet data-sym="..." data-directive="...">` stubs at each directive position. The frontend scans the rendered HTML for those stubs after mount, mounts React-rendered `<SymbolCard>`s (or a thin `<DocSnippet>` wrapper) in their place, and lets the existing RTK-Query hooks handle data fetching.

The design preserves two important properties:

- **Backward compatibility** — the server still resolves directives against the index (so `symbol not found` errors still fire at the API boundary), and the response JSON still includes a `snippets` array listing what was resolved. Hydration is additive.
- **Degrade gracefully** — each stub contains a small plaintext fallback inside it (either the signature or "Snippet: <sym id>"), so JS-disabled or pre-hydration readers see a sensible placeholder instead of blank space.

## 2. Problem statement and scope

### 2.1 Why

Three pain points:

1. **Duplicated rendering logic.** `internal/docs/renderer.go` has its own fence-to-`<pre><code>` rendering separate from `<Code>`. When we added JSX-component linkification in the widget package, doc pages didn't benefit.
2. **No xrefs on doc pages.** The `/doc/01-overview` page shows the `SymbolID` function body, but can't click through to `PackageID` (which it references). On the symbol page for `SymbolID` you *can* click through. Asymmetric.
3. **No godoc annotations.** The Phase 7b annotator (`highlight/annotations.ts`) only runs inside `<Code>`.

### 2.2 In scope

1. Change `internal/docs/renderer.go` to emit placeholder stubs for all three directives.
2. New `DocSnippet` component on the frontend that resolves a symbol ID into a rendered widget.
3. Hydration logic in `DocPage.tsx` that walks the server-rendered HTML and replaces stubs with React components.
4. Backward-compatible JSON response — `snippets[]` stays the same shape; HTML payload changes but old consumers still see readable fallback text.
5. Update the meta doc page to exercise all three directive kinds across both languages to confirm hydration works.

### 2.3 Out of scope

1. Rewriting the entire `/api/doc/*` contract to return structured blocks instead of HTML strings. We keep the HTML string; stubs live *inside* it.
2. Authoring UI for creating doc pages.
3. Per-page CSS themes.
4. Server-side React rendering (SSR). The current plain-text fallback is good enough.
5. Long-form changes to how `codebase-signature` / `codebase-doc` resolve — just their rendering shape.

## 3. Current state

### 3.1 How a doc page flows today

```
markdown source        goldmark preprocess            React render
─────────────────     ──────────────────────         ──────────────
01-overview.md    →   scan fenced blocks        →    DocPage.tsx
(codebase-snippet     resolve sym → slice bytes       dangerouslySetInnerHTML
 sym=...)             substitute body                 → <pre><code>raw text</code></pre>
                      goldmark md → html
```

The relevant server code: `internal/docs/renderer.go` lines ~135-210. Two passes:

1. `fenceRe` scans for fences with info strings `codebase-snippet` / `codebase-signature` / `codebase-doc`.
2. Each matched block is replaced with the resolved text wrapped in a new fenced block with a language hint (`go` or `ts`).

Then goldmark renders the resulting markdown → HTML.

### 3.2 What the frontend does today

`ui/src/features/doc/DocPage.tsx`:

```tsx
return (
  <article data-part="doc-page">
    <div dangerouslySetInnerHTML={{ __html: data.html }} />
    <footer>Resolved {data.snippets.length} snippet(s) from the live index.</footer>
  </article>
);
```

Raw dump. The `data.snippets` field (server returns a list of resolved symbol IDs + directive kinds) is today only used to compose the footer count.

### 3.3 What we already have on the React side

- `<SymbolCard symbol={} snippet={} />` — renders the symbol header + kind badge + code block. Reused on search results, package pages, etc.
- `<LinkedCode text={} refs={} language={} />` — the code viewer with click-through xrefs. Router-aware via the app layer.
- `<ExpandableSymbol symbol={} />` — collapsed-by-default card with a "show more" toggle.
- RTK-Query hooks: `useGetSymbolQuery(id)`, `useGetSnippetQuery(id)`, `useGetSnippetRefsQuery(id)`.

The only missing piece is a small "given a symbol ID and directive, render the right thing" wrapper.

## 4. Target design

### 4.1 Server side

Replace the current "substitute the fenced block with resolved text" step with "substitute with a stub div". The stub carries the information the frontend needs to hydrate, plus a plaintext fallback:

```html
<div class="codebase-snippet"
     data-codebase-snippet
     data-sym="sym:github.com/.../indexer.func.SymbolID"
     data-directive="codebase-snippet"
     data-kind="func"
     data-lang="go">
  <pre><code class="language-go">func SymbolID(importPath, kind, name, typeParams string) string</code></pre>
</div>
```

The `<pre><code>` inside the stub is the existing resolved text. Browsers without JS (or during the pre-hydration flash) see exactly what they see today. When React hydrates, it clears the inner HTML and mounts the rich widget.

The `snippets` array in the JSON response keeps its current shape — one entry per directive — with a new `stubId` field that matches a `data-stub-id` attribute on the div. This lets the frontend do a stable DOM walk rather than parsing string attributes.

### 4.2 Frontend side

New component `<DocSnippet stubId sym directive />` that picks the right widget:

- `codebase-snippet` → `<SymbolCard symbol={sym} snippet={snippet} refs={refs} />` (reuses the symbol page's card).
- `codebase-signature` → a compact `<code>` with the signature only + a click-through link to `/symbol/{id}`.
- `codebase-doc` → a `<blockquote>` with the godoc paragraph.

`DocPage.tsx` uses `useRef` + `useEffect` to scan the article element after mount, find each `[data-codebase-snippet]` node, collect its attributes, and React-render a `DocSnippet` into it via `createPortal` (or `createRoot` targeting the stub). We use portals because the stubs are nested inside goldmark-generated HTML — we can't easily rebuild that tree from structured JSON without losing the prose.

### 4.3 Data flow

```
DocPage mount
    ↓
useGetDocQuery(slug)
    ↓
<article ref={articleRef}>
  <div dangerouslySetInnerHTML={html} />
</article>
    ↓
useEffect scans articleRef.current for [data-codebase-snippet]
    ↓
For each stub node:
  createPortal(<DocSnippet ...stubData/>, stubNode)
    ↓
<DocSnippet> dispatches per-directive:
  snippet    → useGetSnippetQuery + useGetSnippetRefsQuery → SymbolCard
  signature  → useGetSymbolQuery → compact link
  doc        → useGetSymbolQuery → blockquote
```

The `createPortal` approach keeps the goldmark-generated prose intact while giving each stub its own React subtree with its own hooks.

## 5. Implementation plan (phased)

### Phase 1 — Server renders stubs

1. Refactor `resolveFence` to return a `Directive` struct carrying `{ directive, sym, kind, lang, resolvedText, stubId }` rather than a single `Text` string.
2. `Render` assembles stub HTML from the directive struct and splices it into the markdown before goldmark runs. Pre-rendering stubs as HTML strings bypasses goldmark's processing of their content — important so the `<div>` isn't escaped into `&lt;div&gt;`. Use goldmark's `ast.RawHTML` node or just emit raw HTML passthrough.

The simplest form: emit the stub as a fenced HTML block (```html), which goldmark passes through untouched.

3. `PageResult.Snippets[]` gets a `StubID` field; the HTML's `data-stub-id` uses the same value so the frontend can line them up.
4. Renderer tests updated: each resolved directive produces exactly one stub; the snippet list length matches the stub count.

### Phase 2 — Frontend hydration wrapper

1. New file `ui/src/features/doc/DocSnippet.tsx` — the per-stub component.
2. Amend `ui/src/features/doc/DocPage.tsx` with a `useEffect` that scans, portals, and cleans up on unmount.
3. Use React 18 `createRoot` on each stub rather than `createPortal` if the stub tree isn't already under React control. (Both work; `createRoot` is cleaner because the goldmark HTML isn't a React subtree.) Each stub gets its own root that we `unmount()` on cleanup.

### Phase 3 — Meta page refresh + Storybook

1. Update `03-meta.md` to include one block per directive kind (snippet / signature / doc) so the hydration path is exercised across all three.
2. Add a Storybook story `DocSnippet` with mocked RTK-Query data (plain snippet, signature, doc) so the component is developable in isolation.

## 6. Detailed changes

### 6.1 `internal/docs/types.go`

```go
type SnippetRef struct {
    StubID     string  // NEW — UUID or hash, matches data-stub-id
    Directive  string
    SymbolID   string
    Kind       string  // copied from Symbol.Kind for the frontend's benefit
    Language   string  // "go" | "ts"
    StartLine  int
    EndLine    int
    Text       string  // plaintext fallback; frontend uses this pre-hydration
}
```

### 6.2 `internal/docs/renderer.go`

New helper:

```go
func stubHTML(ref *SnippetRef) string {
    var body string
    switch ref.Directive {
    case "codebase-signature":
        body = "<code>" + html.EscapeString(ref.Text) + "</code>"
    case "codebase-doc":
        body = "<blockquote>" + html.EscapeString(ref.Text) + "</blockquote>"
    default: // codebase-snippet
        body = "<pre><code class=\"language-" + ref.Language +
               "\">" + html.EscapeString(ref.Text) + "</code></pre>"
    }
    return fmt.Sprintf(
        `<div class="codebase-snippet" data-codebase-snippet data-stub-id=%q `+
            `data-sym=%q data-directive=%q data-kind=%q data-lang=%q>%s</div>`,
        ref.StubID, ref.SymbolID, ref.Directive, ref.Kind, ref.Language, body,
    )
}
```

The existing fence-substitution step writes `stubHTML(ref)` directly into the markdown buffer instead of emitting a new fenced block. Goldmark's default behaviour passes raw HTML through when it sees a standalone HTML block — no custom AST walking needed.

### 6.3 `ui/src/features/doc/DocSnippet.tsx`

```tsx
interface DocSnippetProps {
  stubId: string;
  sym: string;
  directive: string;
  kind: string;
  lang: string;
  fallback: string;  // original innerHTML before hydration
}

export function DocSnippet({ sym, directive, lang }: DocSnippetProps) {
  if (directive === 'codebase-signature') {
    const { data } = useGetSymbolQuery(sym);
    return (
      <Link to={`/symbol/${encodeURIComponent(sym)}`} data-part="doc-sig">
        <code data-tok="kw">{data?.signature ?? sym}</code>
      </Link>
    );
  }
  if (directive === 'codebase-doc') {
    const { data } = useGetSymbolQuery(sym);
    return <blockquote data-part="doc-godoc">{data?.doc ?? ''}</blockquote>;
  }
  // codebase-snippet
  const { data: sym2 } = useGetSymbolQuery(sym);
  const { data: snippet } = useGetSnippetQuery(sym);
  const { data: refs } = useGetSnippetRefsQuery(sym);
  if (!sym2 || !snippet) return null;
  return <SymbolCard symbol={sym2} snippet={snippet} refs={refs} showHeader />;
}
```

### 6.4 `ui/src/features/doc/DocPage.tsx`

```tsx
export function DocPage() {
  // ... existing slug + query code ...
  const articleRef = useRef<HTMLElement>(null);
  const rootsRef = useRef<ReactDOMRoot[]>([]);

  useEffect(() => {
    if (!data || !articleRef.current) return;
    const stubs = articleRef.current.querySelectorAll('[data-codebase-snippet]');
    stubs.forEach((el) => {
      const sym = el.getAttribute('data-sym')!;
      const directive = el.getAttribute('data-directive')!;
      const kind = el.getAttribute('data-kind') ?? '';
      const lang = el.getAttribute('data-lang') ?? 'go';
      const fallback = el.innerHTML;
      el.innerHTML = '';
      const root = createRoot(el);
      root.render(<DocSnippet sym={sym} directive={directive} kind={kind}
                              lang={lang} fallback={fallback}
                              stubId={el.getAttribute('data-stub-id') ?? ''} />);
      rootsRef.current.push(root);
    });
    return () => {
      rootsRef.current.forEach((r) => r.unmount());
      rootsRef.current = [];
    };
  }, [data?.html]);

  return (
    <article data-part="doc-page" ref={articleRef}>
      <div dangerouslySetInnerHTML={{ __html: data.html }} />
      <footer data-part="symbol-doc" style={{ marginTop: 32, fontSize: 12 }}>
        Resolved {data.snippets.length} snippet(s) from the live index.
      </footer>
    </article>
  );
}
```

## 7. Risks, alternatives, open questions

### 7.1 Risks

1. **React warning about clearing DOM managed by dangerouslySetInnerHTML.** Mitigation: wrap the inner stub in its own div that we own (the outer `<div data-codebase-snippet>` is not the same React-rendered div). React doesn't warn about mutating descendants of `dangerouslySetInnerHTML`.
2. **StrictMode double-mount of roots.** `createRoot` is idempotent per-element but double-rendering could create visual flicker. Mitigation: track which elements already have a root via a `WeakMap<Element, Root>`.
3. **Scroll jank when hydration expands a stub to a much taller SymbolCard.** Mitigation: render the card inside a wrapper that reserves a min-height based on the fallback's line count.

### 7.2 Alternatives considered

1. **Return structured JSON blocks instead of HTML string.** Would work but throws away prose authoring via plain markdown. Rejected — markdown-first authoring is the point.
2. **Server-side React render.** Adds a Node runtime dependency to the Go server. Rejected — contradicts the single-binary invariant.
3. **Custom goldmark AST nodes rendered to React placeholders via a custom HTML renderer.** Cleaner semantically but much more code. The raw-HTML passthrough achieves the same result in 20 lines.

### 7.3 Open questions

1. Should `codebase-doc` stubs link to the owning symbol page, or stay inline? Recommended: small link icon.
2. Should the snippet card on doc pages be collapsed by default (`ExpandableSymbol`) for long bodies? Probably yes for snippets over 40 lines.
3. Backward-compat window: do we need a query parameter `?hydrate=0` to fall back to the old raw HTML? Probably no — the fallback is already inside the stub.

## 8. Testing strategy

| Layer | Test | Tooling |
|---|---|---|
| Renderer | Each directive produces one stub div; stub carries sym/directive/kind/lang; fallback body contains resolved text | Go table-test for `Render()` |
| Renderer | `snippets[]` `StubID` values match the `data-stub-id` attributes in the HTML | Go |
| DocSnippet | Each of the three directives renders the right sub-component in isolation | Storybook / vitest later |
| E2E | Fire up the server, fetch `/api/doc/03-meta`, assert the HTML contains the expected number of stubs, each with the expected attributes | Go httptest |
| Visual | `pnpm storybook` → `Widgets/Docs` → CodeBlockDoc story renders | Storybook |

## 9. References

1. GCB-002 TS extractor design — `ttmp/2026/04/20/GCB-002--.../design-doc/01-...md`
2. Existing renderer — `internal/docs/renderer.go`
3. SymbolCard — `ui/src/packages/ui/src/SymbolCard.tsx`
4. LinkedCode — `ui/src/features/symbol/LinkedCode.tsx`
5. React 18 `createRoot` + portals: https://react.dev/reference/react-dom/client/createRoot
