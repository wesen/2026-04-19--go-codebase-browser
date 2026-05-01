---
Title: Static Export Review — Markdown, WASM, and API Wiring Assessment
Ticket: GCB-013
Status: active
Topics:
    - codebase-browser
    - pr-review
    - code-review
    - sqlite-index
    - markdown-docs
    - literate-programming
    - glazed-help
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/review/export.go
      Note: |-
        Static export command; recently fixed stale dist copy but still carries bundler and source-copy issues.
        Documents export bundler behavior and stale dist fix.
    - Path: internal/docs/renderer.go
      Note: |-
        Directive resolver; explains the observed symbol-not-found error for malformed sym refs.
        Documents symbol reference resolution and authoring error root cause.
    - Path: internal/review/export.go
      Note: |-
        Precomputes reviewData docs/diffs/histories/impacts; important foundation with schema mismatches and commit-key limitations.
        Documents reviewData precomputation
    - Path: ui/src/api/docApi.ts
      Note: |-
        Review-doc and regular-doc API layer currently tries HTTP first, then falls back to WASM; source of expected 404s in static export.
        Documents why static export currently probes /api/doc and /api/review/docs before WASM fallback.
    - Path: ui/src/api/historyApi.ts
      Note: |-
        History widgets still use fetchBaseQuery('/api/history'); source of failing diff/history/impact widgets in static export.
        Documents why review widgets still call /api/history in static export.
    - Path: ui/src/api/wasmClient.ts
      Note: |-
        WASM transport has review doc and review query functions, but only some UI code uses them.
        Documents available WASM review functions and missing widget integration.
    - Path: ui/src/features/doc/DocSnippet.tsx
      Note: |-
        Markdown snippet hydration dispatches review widgets to components that currently depend on historyApi.
        Documents markdown directive dispatch path from rendered stubs to widgets.
    - Path: ui/src/features/doc/widgets/DiffStatsWidget.tsx
      Note: Documents observed Failed to load diff stats failure path.
    - Path: ui/src/features/doc/widgets/ImpactInlineWidget.tsx
      Note: Documents payload shape mismatch between ImpactResponse and exported ImpactLite.
    - Path: ui/src/features/review/ReviewDocPage.tsx
      Note: |-
        Newly added static review doc renderer; good direction but incomplete without static widget data source.
        Documents newly added static review doc route and hydration approach.
ExternalSources: []
Summary: Review of the GCB-013 markdown/static-WASM export work after browser errors showed HTTP fallback leaks, unresolved symbols, and incomplete widget wiring.
LastUpdated: 2026-05-01T10:39:15.604667701-04:00
WhatFor: Use this to reassess the current GCB-013 implementation before continuing static export work.
WhenToUse: Read before fixing review doc widgets, adding sql.js, changing the export bundler, or declaring static export complete.
---


# Static Export Review — Markdown, WASM, and API Wiring Assessment

## Goal

This document reassesses the GCB-013 implementation after the static export showed these browser-visible failures:

```text
New export command

    doc error: symbol "github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.newExportCmd" not found (codebase-snippet sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.newExportCmd)

Pre-computed diffs
Failed to load diff stats

XHR GET http://localhost:8771/api/doc                                  404
XHR GET http://localhost:8771/api/review/docs                          404
XHR GET http://localhost:8771/api/review/docs/pr-42                    404
XHR GET http://localhost:8771/api/history/diff?from=HEAD~1&to=HEAD     404
```

The purpose is not to dunk on the previous work. The purpose is to separate the useful foundations from the incomplete assumptions, explain how the failures could have been predicted from the code, and define what to keep, what to fix, what to redo, and what to leave aside.

## Executive summary

The implementation made real progress in the backend and export-data layers, but it declared the static export “working” before the frontend data-source contract was complete.

What is genuinely good:

1. The review database and indexer are useful artifacts.
2. The export precomputation (`reviewData`) is the right general direction for offline review guides.
3. The WASM module now exposes review-oriented functions (`getReviewDoc`, `getCommitDiff`, `getSymbolHistory`, `getImpact`).
4. The new `ReviewDocPage` and sidebar route prove that pre-rendered review markdown can be displayed in the static SPA.
5. The export command bug where it copied stale `dist/` instead of the fresh Vite output was correctly identified and fixed.

What is still wrong:

1. The static export still performs server-first HTTP requests for docs. Those 404s are expected from the current code, not surprising browser noise.
2. The history widgets embedded in markdown still call `/api/history/*` directly through `historyApi`. They do not use the new WASM review functions.
3. `codebase-diff-stats from=HEAD~1 to=HEAD` cannot work offline as implemented because export diffs are keyed by full commit hashes only.
4. The impact data shape precomputed in `internal/review/export.go` does not match the `ImpactInlineWidget`/`historyApi` frontend type contract.
5. Symbol references in markdown are easy to write incorrectly, and the test markdown used the wrong short-reference form.
6. The implementation lacks tests that assert “no `/api/*` requests happen in a static export.” Without that test, this regression was almost guaranteed.

Bottom line: keep the database/indexer/export-data/WASM foundation, but introduce a real static-vs-server data-source abstraction before adding more UI. The next work should be a small, precise fix pass, not more feature work.

## Current-state evidence

### 1. Review doc fetching intentionally tries HTTP first

`ui/src/api/docApi.ts` currently fetches `/api/doc` before falling back to WASM:

```ts
// ui/src/api/docApi.ts:40-50
listDocs: b.query<PageMeta[], void>({
  queryFn: async (_arg, api, extraOptions) => {
    try {
      const resp = await fetch('/api/doc');
      if (resp.ok) return { data: await resp.json() };
    } catch {}
    return wasmBaseQuery('docPages', api as any, extraOptions as any) as any;
  },
}),
```

The same pattern exists for review docs:

```ts
// ui/src/api/docApi.ts:61-87
listReviewDocs: b.query<ReviewDocMeta[], void>({
  queryFn: async (_arg, api, extraOptions) => {
    try {
      const resp = await fetch('/api/review/docs');
      if (resp.ok) { ... }
    } catch {}
    const result = await wasmBaseQuery('reviewDocs', api as any, extraOptions as any) as any;
    ...
  },
}),
getReviewDoc: b.query<DocPage, string>({
  queryFn: async (slug, api, extraOptions) => {
    try {
      const resp = await fetch(`/api/review/docs/${encodeURIComponent(slug)}`);
      if (resp.ok) return { data: await resp.json() };
    } catch {}
    return wasmBaseQuery(`reviewDoc:${slug}`, api as any, extraOptions as any) as any;
  },
}),
```

Therefore these network entries are expected with the current code:

```text
GET /api/doc                 404
GET /api/review/docs         404
GET /api/review/docs/pr-42   404
```

They are not proof that the fallback is broken. They are proof that the static export has no “static mode” switch. The code probes the server and only then uses WASM. That is tolerable in development, but bad for a polished static artifact because it creates scary red console entries and masks real failures.

### 2. Markdown history widgets still use the server-only history API

The real functional failure is the history widgets.

`ui/src/api/historyApi.ts` is hard-wired to `fetchBaseQuery({ baseUrl: '/api/history' })`:

```ts
// ui/src/api/historyApi.ts:121-153
export const historyApi = createApi({
  reducerPath: 'historyApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/history' }),
  endpoints: (builder) => ({
    getDiff: builder.query<CommitDiff, { from: string; to: string }>({
      query: ({ from, to }) => `/diff?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    }),
    getSymbolHistory: builder.query<SymbolHistoryEntry[], { symbolId: string; limit?: number }>({ ... }),
    getSymbolBodyDiff: builder.query<BodyDiffResult, { from: string; to: string; symbolId: string }>(...),
    getImpact: builder.query<ImpactResponse, { sym: string; dir?: 'usedby' | 'uses'; depth?: number; commit?: string }>({ ... }),
  }),
});
```

The markdown hydration layer dispatches `codebase-diff-stats` directly to widgets that consume `historyApi`:

```ts
// ui/src/features/doc/DocSnippet.tsx:82-106
if (directive === 'codebase-diff') {
  return <SymbolDiffInlineWidget sym={sym} from={params?.from ?? ''} to={params?.to ?? ''} />;
}
...
if (directive === 'codebase-diff-stats') {
  return <DiffStatsWidget from={params?.from ?? ''} to={params?.to ?? ''} />;
}
if (directive === 'codebase-changed-files') {
  return <ChangedFilesWidget from={params?.from ?? ''} to={params?.to ?? ''} />;
}
```

`DiffStatsWidget` calls `useGetDiffQuery`:

```ts
// ui/src/features/doc/widgets/DiffStatsWidget.tsx:9-17
export function DiffStatsWidget({ from, to }: DiffStatsWidgetProps) {
  const { data, isLoading, error } = useGetDiffQuery({ from, to }, { skip: !from || !to });
  ...
  if (error) {
    ...
    return <span data-part="error">Failed to load diff stats</span>;
  }
```

Therefore this network entry is not accidental:

```text
GET /api/history/diff?from=HEAD~1&to=HEAD 404
```

It is exactly what the current code says to do.

### 3. The WASM review functions exist, but the widgets do not use them

The WASM client added review helpers:

```ts
// ui/src/api/wasmClient.ts:156-171
export async function getCommitDiff(oldHash: string, newHash: string): Promise<unknown> { ... }
export async function getSymbolHistory(symbolID: string): Promise<unknown> { ... }
export async function getImpact(symbolID: string, direction: string, depth: number): Promise<unknown> { ... }
```

The base query routes review docs through WASM:

```ts
// ui/src/api/wasmClient.ts:137-140
} else if (endpoint.startsWith('reviewDoc:')) {
  result = window.codebaseBrowser.getReviewDoc(endpoint.slice(10));
} else if (endpoint === 'reviewDocs') {
  result = window.codebaseBrowser.getReviewDocs();
}
```

But there is no equivalent `historyApi` fallback to `getCommitDiff`, `getSymbolHistory`, or `getImpact`. The function exists; the UI path does not call it. This is a classic integration gap.

### 4. Diffs are precomputed only for adjacent full-hash pairs

`internal/review/export.go` precomputes diffs for adjacent commits only, and keys them as `oldHash + ".." + newHash`:

```go
// internal/review/export.go:117-134
for i := 1; i < len(out.Commits); i++ {
    oldHash := out.Commits[i-1].Hash
    newHash := out.Commits[i].Hash
    key := oldHash + ".." + newHash

    diff, err := store.History.DiffCommits(ctx, oldHash, newHash)
    ...
    out.Diffs[key] = &DiffLite{...}
}
```

The WASM lookup does the same exact full-string key construction:

```go
// internal/wasm/search.go:245-252
func (s *SearchCtx) GetCommitDiff(oldHash, newHash string) []byte {
    if s.ReviewData == nil { return []byte("null") }
    key := oldHash + ".." + newHash
    data, _ := json.Marshal(s.ReviewData.Diffs[key])
    return data
}
```

So a doc directive like this cannot work offline unless the frontend first resolves `HEAD~1` and `HEAD` to full commit hashes:

```markdown
```codebase-diff-stats from=HEAD~1 to=HEAD
```
```

In server mode, the server could resolve revspecs. In static mode, there is no Git. The static artifact only has the precomputed commit list.

### 5. The impact data shape does not match the widget contract

The frontend history API expects impact nodes with `edges` and `local`, and the response has `root` and `commit`:

```ts
// ui/src/api/historyApi.ts:103-119
export interface ImpactNode {
  symbolId: string;
  name: string;
  kind: string;
  depth: number;
  edges: ImpactEdge[];
  compatibility: string;
  local: boolean;
}

export interface ImpactResponse {
  root: string;
  direction: string;
  depth: number;
  commit: string;
  nodes: ImpactNode[];
}
```

The export data has a different shape:

```go
// internal/review/export.go:54-67
type ImpactLite struct {
    RootSymbol string       `json:"rootSymbol"`
    Direction  string       `json:"direction"`
    Depth      int          `json:"depth"`
    Nodes      []ImpactNode `json:"nodes"`
}

type ImpactNode struct {
    SymbolID      string `json:"symbolId"`
    Name          string `json:"name"`
    Kind          string `json:"kind"`
    Depth         int    `json:"depth"`
    Compatibility string `json:"compatibility"`
}
```

The widget reads `data.commit`, `node.local`, and `node.edges.length`:

```ts
// ui/src/features/doc/widgets/ImpactInlineWidget.tsx:40-50, 95-99
const localCount = data.nodes.filter((node) => node.local).length;
...
<code>{data.commit.slice(0, 7)}</code>
...
{node.edges.length} edge{node.edges.length === 1 ? '' : 's'}
```

Even after switching `ImpactInlineWidget` from HTTP to WASM, the current `ImpactLite` payload would not satisfy the existing component contract. The previous implementation missed this because it tested that the function returned “something,” not that the consumer could render it.

### 6. The symbol-not-found error is a documentation and authoring-contract issue

The user saw:

```text
doc error: symbol "github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.newExportCmd" not found
```

The renderer accepts two forms:

```go
// internal/docs/renderer.go:528-545
// resolveSymbol accepts either a full "sym:..." ID or a short form
// "pkg/import/path.Name" / "pkg/import/path.Recv.Method".
func resolveSymbol(ref string, l *browser.Loaded) (*indexer.Symbol, error) {
    if strings.HasPrefix(ref, "sym:") { ... }
    dot := strings.LastIndex(ref, ".")
    importPath := ref[:dot]
    name := ref[dot+1:]
```

The bad reference omitted `sym:` but included the full-ID internal `.func.` segment. Because it did not start with `sym:`, the short-form resolver split it into:

```text
importPath = github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func
name       = newExportCmd
```

No package has import path `.../review.func`, so resolution failed. The author should have used either:

```markdown
```codebase-snippet sym=sym:github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.newExportCmd
```
```

or the short form without `.func.`:

```markdown
```codebase-snippet sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd
```
```

This is not primarily a WASM problem. It is an authoring-contract problem. The implementation should make this easier to discover by documenting accepted forms, showing examples in `review-user-guide`, and possibly adding a symbol search/autocomplete helper.

## What the previous work did well

### Keep: the review DB schema and indexing direction

The review DB as a unified SQLite artifact is a strong foundation. It supports both LLM querying and UI export. The `review_docs` / `review_doc_snippets` tables are the right seam between markdown authoring and code-index data.

### Keep: pre-rendered review docs in `reviewData`

`internal/review/export.go` reads review docs, renders them through `docs.Render`, and stores HTML/snippets in `ReviewDocLite`:

```go
// internal/review/export.go:334-364
page, err := docs.Render(slug, []byte(content), loaded, sourceFS)
...
out.Docs = append(out.Docs, ReviewDocLite{
    Slug:     page.Slug,
    Title:    page.Title,
    HTML:     page.HTML,
    Snippets: page.Snippets,
})
```

That is the correct model for static export. Markdown rendering should happen at export time, not at browser runtime.

### Keep: WASM review lookups

The added WASM functions are conceptually right:

1. `GetReviewDocs`
2. `GetReviewDoc`
3. `GetCommitDiff`
4. `GetSymbolHistory`
5. `GetImpact`
6. `GetCommits`

They are not enough on their own, but they are the right browser-side query surface.

### Keep: `ReviewDocPage` as a first UI bridge

`ui/src/features/review/ReviewDocPage.tsx` correctly uses the same stub-hydration pattern as regular docs:

```ts
// ui/src/features/review/ReviewDocPage.tsx:31-52
articleRef.current
  .querySelectorAll<HTMLElement>('[data-codebase-snippet]')
  .forEach((el) => { ... found.push({ el, sym, directive, kind, lang, commit, params }); });
```

And it hydrates those stubs with `DocSnippet`:

```tsx
// ui/src/features/review/ReviewDocPage.tsx:68-82
<div dangerouslySetInnerHTML={{ __html: data.html }} />
{stubs.map((s, i) =>
  createPortal(<DocSnippet ... />, s.el, `${slug}-${i}`),
)}
```

That is the right shape. The problem is below it: `DocSnippet` dispatches to widgets that still use server APIs.

### Keep: the stale-bundler fix

The export command previously built the SPA but copied stale root `dist/` output. The current code copies from `ui/dist/public` after running Vite:

```go
// cmd/codebase-browser/cmds/review/export.go:104-120
if err := buildSPA(); err != nil { ... }
...
spaDir := "ui/dist/public"
if err := copyTree(spaDir, outDir); err != nil { ... }
```

This is an important fix. Without it, UI changes do not show up in the static export.

## What the previous work did poorly

### 1. It conflated “WASM function exists” with “UI uses WASM”

The previous validation called `window.codebaseBrowser.getCommits()` and `getSymbolHistory()` directly. That proves the functions exist. It does not prove that the rendered markdown widgets use them.

A better validation would have opened a rendered review document containing each directive and asserted no `/api/history/*` requests occurred.

### 2. It did not trace the data path from markdown to widget to transport

The failing path is:

```text
markdown fence
  → internal/docs.Render
  → HTML stub with data-directive="codebase-diff-stats"
  → ReviewDocPage hydrates stub
  → DocSnippet dispatches to DiffStatsWidget
  → DiffStatsWidget calls useGetDiffQuery
  → historyApi fetches /api/history/diff
  → static server returns 404
```

The earlier work validated only these two disconnected facts:

```text
reviewData.docs exists
WASM getCommitDiff exists
```

It did not validate the end-to-end path between them.

### 3. It left server-first probing in static mode

The doc endpoints intentionally try HTTP first. That may be acceptable for a hybrid dev/server app, but a static export should know it is static and skip HTTP completely.

The browser errors for `/api/doc` and `/api/review/docs` are a symptom of missing mode detection.

### 4. It ignored shape compatibility between server API types and export JSON

The impact payload mismatch is the clearest example. A function returning JSON is not enough. The JSON has to match what the consumer expects.

### 5. It did not resolve Git revspecs for static mode

Static browser code cannot resolve `HEAD~1`. The export should either:

1. rewrite doc directive params from revspecs to hashes at export time;
2. include a rev alias map (`HEAD`, `HEAD~1`, short hashes, branch tips) in `reviewData`; or
3. require full hashes in docs and fail loudly during export if a revspec appears.

The current implementation does none of these.

### 6. It over-marked tasks as complete

Some task checkboxes were marked complete after partial manual validation. In particular, “static export workflow” should not have been checked off while rendered widgets still fetched `/api/history/*` and failed offline.

## How they could have known better

The missing information was already in the code. The right review procedure would have been:

1. Search for hard-coded `/api/` calls:
   ```bash
   rg -n "fetch\(|fetchBaseQuery|/api/history|/api/review|/api/doc" ui/src -S
   ```
2. Trace every markdown directive in `DocSnippet.tsx` to its widget.
3. For each widget, identify its data source.
4. Compare each data source against the static artifact contract.
5. Run browser tests with request capture and assert zero unexpected `/api/*` calls.

That process would have immediately revealed:

1. doc APIs have server-first fallback;
2. history widgets are server-only;
3. `HEAD~1` cannot resolve in static mode;
4. impact JSON shape differs from frontend contract.

## What information was missing

The previous implementer seemed to lack or not apply four key pieces of information:

1. **Static export is a transport problem, not just a data problem.** Precomputing JSON is necessary but insufficient. Every UI consumer must route to the static transport.
2. **Browser console/network tab is part of the contract.** A static export should not rely on failing HTTP requests as a normal path.
3. **Directive params are user-facing API.** If docs say `from=HEAD~1`, the static exporter must either support it or reject it clearly.
4. **Type shapes matter across server/WASM boundaries.** Existing React components were written against `historyApi` shapes, not the new `reviewData` shapes.

## What to keep

Keep these components and build on them:

1. `internal/review/schema.go`, `store.go`, `indexer.go`: good foundation.
2. `internal/review/export.go`: keep the `PrecomputedReview` concept, but revise key normalization and payload shapes.
3. `internal/wasm/review_types.go` and review methods in `internal/wasm/search.go`: keep but align schemas with frontend expectations.
4. `ReviewDocPage`: keep as the route-level renderer for review guides.
5. `review export` command: keep the fresh-SPA copy fix; later refactor shared bundling logic to avoid duplicate code.
6. `review db create` / `review index` / `review serve`: keep; server mode remains useful.

## What to fix next

### Fix 1: Add explicit runtime/export mode

Do not probe `/api/*` in static mode. Add a tiny mode detector:

```ts
export function isStaticExport(): boolean {
  return import.meta.env.VITE_STATIC_EXPORT === '1' || Boolean((window as any).codebaseBrowser);
}
```

Then in `docApi`:

```ts
if (!isStaticExport()) {
  const resp = await fetch('/api/review/docs');
  if (resp.ok) return { data: await resp.json() };
}
return wasmBaseQuery('reviewDocs', api, extraOptions);
```

The export command should run Vite with `VITE_STATIC_EXPORT=1`.

### Fix 2: Replace or wrap `historyApi` for static mode

Create a transport abstraction with identical hook-level semantics:

```ts
// reviewHistoryApi.ts, or a static-aware historyApi baseQuery
getDiff({ from, to })
  server mode: GET /api/history/diff?from=...&to=...
  static mode: resolveCommitRef(from/to), call getCommitDiff(oldHash, newHash)

getSymbolHistory({ symbolId, limit })
  server mode: GET /api/history/symbols/:id/history
  static mode: call getSymbolHistory(symbolId), apply limit client-side

getImpact({ sym, dir, depth })
  server mode: GET /api/history/impact
  static mode: call getImpact(sym, dir, depth)
```

Do not leave the widgets themselves responsible for knowing whether they are in server or static mode.

### Fix 3: Normalize commit refs during export

Add to `PrecomputedReview`:

```go
type PrecomputedReview struct {
    ...
    CommitAliases map[string]string `json:"commitAliases"`
}
```

Populate aliases:

```text
HEAD      -> newest commit hash
HEAD~1    -> previous commit hash
HEAD~2    -> older commit hash
<short>   -> full hash
<full>    -> full hash
```

Then static `getDiff` can resolve `HEAD~1` and `HEAD` to full hashes before calling `getCommitDiff`.

### Fix 4: Align export JSON with existing frontend types

Either change the frontend widget types or change export JSON. Prefer changing export JSON to match current `historyApi` interfaces:

```ts
ImpactResponse {
  root: string;
  direction: string;
  depth: number;
  commit: string;
  nodes: ImpactNode[];
}
ImpactNode {
  symbolId: string;
  name: string;
  kind: string;
  depth: number;
  edges: ImpactEdge[];
  compatibility: string;
  local: boolean;
}
```

This makes server and static transports interchangeable.

### Fix 5: Improve directive authoring errors

The symbol reference error is technically correct but not helpful enough. Improve it to say:

```text
symbol "...review.func.newExportCmd" not found.
This looks like a full symbol ID without the "sym:" prefix.
Use either:
  sym=sym:github.com/.../review.func.newExportCmd
or short form:
  sym=github.com/.../review.newExportCmd
```

This could be implemented in `resolveSymbol` when a non-`sym:` ref contains `.func.` or `.method.`.

### Fix 6: Add browser network tests

Add a Playwright test or script that:

1. creates a tiny review DB with two commits and one markdown doc;
2. exports static site;
3. serves it with `python3 -m http.server`;
4. opens `/review/<slug>`;
5. fails if any request URL contains `/api/` except maybe explicitly allowed favicon probes;
6. asserts review doc text and widgets render.

Pseudocode:

```ts
const apiRequests: string[] = [];
page.on('request', req => {
  if (new URL(req.url()).pathname.startsWith('/api/')) apiRequests.push(req.url());
});
await page.goto(`${base}/#/review/pr-42`);
await expect(page.getByText('PR #42')).toBeVisible();
expect(apiRequests).toEqual([]);
```

## What to redo

### Redo the static history-widget wiring

The current WASM helpers are not useless, but the frontend wiring should be redone around a shared `historyApi` contract. Do not patch each widget ad hoc. The widgets should continue to call one query abstraction, and that abstraction should select server vs static transport.

### Redo impact export shape

The current `ImpactLite` was a rough BFS result, not a frontend-compatible API payload. It needs to be rebuilt to match `ImpactResponse` or the widgets need a separate static-only renderer. Prefer matching `ImpactResponse`.

### Redo static-export validation claims

Do not call static export complete until a real review doc with all supported directives works without server requests.

## What to leave aside for now

### Leave sql.js aside until the base static transport is correct

sql.js is still valuable for ad-hoc LLM SQL queries, but it should not be used to paper over the broken review widgets. First make the intended precomputed-WASM path work. Then add sql.js as an optional console.

### Leave source-tree size optimization aside until correctness is fixed

The export still copies `internal/sourcefs/embed/source` into every review export. That is large and should be optimized. But the immediate correctness issues are the server-bound widgets and commit-ref resolution. Fix correctness first.

### Leave UI polish aside

The current visible doc errors are acceptable during validation. Better empty states and prettier error blocks can wait until transport behavior is correct.

## Recommended repair plan

### Phase A — static mode gate, one day

1. Add `VITE_STATIC_EXPORT=1` to `review export` SPA build.
2. Add `isStaticExport()` helper.
3. Stop doc APIs from probing HTTP in static mode.
4. Add a Playwright network assertion that review docs do not call `/api/doc` or `/api/review/docs` in static mode.

### Phase B — history transport, one to two days

1. Add commit alias map to `reviewData`.
2. Replace static `historyApi` behavior with WASM calls.
3. Support `getDiff`, `getSymbolHistory`, and `getImpact` first.
4. Either hide `codebase-diff` body diff in static mode or precompute symbol body diffs; current `GetCommitDiff` is not enough to render body-level diffs.

### Phase C — payload compatibility, one day

1. Align `DiffLite`, `HistoryEntryLite`, and `ImpactLite` with TypeScript expectations.
2. Add fixture-based JSON contract tests.
3. Add frontend tests for `DiffStatsWidget`, `SymbolHistoryInlineWidget`, and `ImpactInlineWidget` in static mode.

### Phase D — authoring ergonomics, half day

1. Improve symbol-not-found hints.
2. Add review guide examples using valid symbol forms.
3. Add a command or SQL snippet to list candidate symbol IDs for a name.

### Phase E — optional sql.js, later

1. Add sql.js only after widgets are correct.
2. Use it for a SQL console, not for core widget rendering.
3. Keep it lazy-loaded.

## Quick reference: observed failures mapped to root causes

| Symptom | Root cause | Evidence | Fix |
|---|---|---|---|
| `GET /api/doc 404` | `docApi.listDocs` probes server first | `ui/src/api/docApi.ts:40-49` | Static mode gate |
| `GET /api/review/docs 404` | `docApi.listReviewDocs` probes server first | `ui/src/api/docApi.ts:61-72` | Static mode gate |
| `GET /api/review/docs/pr-42 404` | `docApi.getReviewDoc` probes server first | `ui/src/api/docApi.ts:80-86` | Static mode gate |
| `GET /api/history/diff?... 404` | `historyApi` always uses `/api/history` | `ui/src/api/historyApi.ts:121-153` | Static history transport using WASM |
| `Failed to load diff stats` | `DiffStatsWidget` consumes server-bound `useGetDiffQuery` | `ui/src/features/doc/widgets/DiffStatsWidget.tsx:9-17` | Route `getDiff` to WASM in static mode |
| `symbol ...review.func.newExportCmd not found` | Non-`sym:` ref incorrectly includes `.func.` | `internal/docs/renderer.go:528-545` | Use `sym:` full ID or short form without `.func.`; improve error hints |
| Impact widget would fail after WASM switch | Export impact shape lacks `commit`, `edges`, `local` | `internal/review/export.go:54-67`; `ui/src/api/historyApi.ts:103-119` | Align payload shape |
| UI changes did not appear in export initially | `review export` copied stale root `dist/` | fixed in `cmd/.../review/export.go:104-120` | Keep fresh-SPA copy or refactor bundler |

## Final verdict

The previous work is not throwaway. It built useful backend, export, and WASM foundations. But it stopped at “data exists” and did not finish “the rendered markdown uses the data without a server.” The errors reported by the user are exactly what the current code would produce.

The correct next step is not sql.js and not more broad feature work. The correct next step is to finish the static transport boundary:

1. no server probing in static mode;
2. history widgets backed by WASM in static mode;
3. full-hash or alias-resolved commit refs;
4. payload shapes compatible with existing widgets;
5. browser tests that fail on unexpected `/api/*` requests.

Once that is in place, sql.js becomes a clean optional add-on for exploratory SQL, rather than a workaround for incomplete static widget wiring.

## Related

- `design-doc/02-standalone-wasm-export-browser-side-sqlite-queries-for-code-review.md`
- `design-doc/03-tinygo-vs-sql-js-feasibility-assessment-for-browser-side-sqlite.md`
- `reference/01-investigation-diary.md`

## Addendum — first repair pass completed on 2026-05-01

After this review was written, a first repair pass implemented the highest-priority transport fixes:

1. `review export` now builds the SPA with `VITE_STATIC_EXPORT=1`.
2. `docApi` skips `/api/doc` and `/api/review/docs` probes in static mode.
3. `historyApi` now has a static-aware base query. In static mode it routes commit lists, commit diffs, symbol histories, and impact lookups through the WASM review helpers instead of `/api/history`.
4. Static history lookup resolves `HEAD`, `HEAD~N`, full hashes, and short hashes from `reviewData.commits`.
5. Static diff/history/impact results are normalized into the TypeScript shapes expected by existing widgets.
6. Symbol body diffs remain explicitly not precomputed; static mode now returns a local `STATIC_NOT_PRECOMPUTED` error instead of making an HTTP request.

Validation command sequence:

```bash
pnpm -C ui run typecheck
go build ./cmd/codebase-browser
go run ./cmd/codebase-browser review index --commits HEAD~2..HEAD --docs /tmp/reviews/pr-static.md --db /tmp/review-static-smoke.db
go run ./cmd/codebase-browser review export --db /tmp/review-static-smoke.db --out /tmp/review-static-export
python3 -m http.server 8772 --directory /tmp/review-static-export
```

Browser validation opened `/#/review/pr-static`; Playwright's `/api/` network filter returned no requests, and `codebase-diff-stats from=HEAD~1 to=HEAD` rendered from WASM-backed static data.

The remaining recommendations still apply for body-level diffs, richer impact payloads, source-tree export size, and optional sql.js.

## Addendum — body-diff static support completed on 2026-05-01

A second repair pass added static support for `codebase-diff` body diffs.

Changes:

1. `PrecomputedReview` now includes `bodyDiffs`.
2. Export precomputes body diffs for changed symbols in adjacent commit diffs and for explicit `codebase-diff` snippets found in review docs.
3. WASM exposes `getSymbolBodyDiff(oldHash, newHash, symbolID)`.
4. Static `historyApi` now resolves `/symbol-body-diff?...` through WASM instead of reporting `STATIC_NOT_PRECOMPUTED`.
5. `internal/history/bodydiff.go` no longer panics when tests use short synthetic commit hashes.

Validation opened a static review doc containing both `codebase-diff` and `codebase-diff-stats`; Playwright observed no `/api/` requests, the body diff rendered with the Diffs widget, diff stats rendered, and no error elements were present.

Remaining caution: body diffs are still precomputed selectively, not for all symbols/all commit pairs. That is intentional for artifact size, but it should be documented as an export limitation.
