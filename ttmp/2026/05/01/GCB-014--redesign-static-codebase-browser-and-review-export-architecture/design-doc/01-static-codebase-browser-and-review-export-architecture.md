---
Title: Static codebase browser and review export architecture
Ticket: GCB-014
Status: active
Topics:
    - codebase-browser
    - static-export
    - wasm
    - react-frontend
    - review-docs
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/review/export.go
      Note: Current review export command that should become a thin wrapper around a static bundle builder
    - Path: internal/history/schema.go
      Note: Authoritative history SQLite schema for static bundle loaders
    - Path: internal/review/export.go
      Note: Current review-centric precompute bundle to replace with generic static bundle data
    - Path: internal/review/schema.go
      Note: Review document tables that remain the source for review renderer data
    - Path: internal/wasm/search.go
      Note: Current WASM query exports that should be refactored around static bundle data
    - Path: ui/src/api/historyApi.ts
      Note: Current static/server transport split to replace with query provider
    - Path: ui/src/features/history/HistoryPage.tsx
      Note: Generic browser history route that must be supported by the static browser contract
    - Path: ui/src/features/review/ReviewDocPage.tsx
      Note: Review markdown renderer that should hydrate generic widget stubs
ExternalSources: []
Summary: Clean-cut architecture for separating a generic static codebase browser from a static rich review markdown renderer while sharing one explicit static data contract.
LastUpdated: 2026-05-01T19:18:00.439895666-04:00
WhatFor: Use this as the implementation guide for replacing the current review-centric static export with a generic static browser foundation plus review docs layered on top.
WhenToUse: Use before changing review export, static WASM data, history widgets, review markdown widgets, or any route expected to work in a standalone static bundle.
---


# Static codebase browser and review export architecture

## Executive Summary

The current static export grew from a narrow goal: render review markdown documents with enough precomputed data to satisfy the widgets that appeared inside those documents. That approach worked for the first smoke tests, but it has now exposed a fundamental design problem: the application also ships a generic codebase browser with routes such as `/history`, `/symbol`, `/source`, and search. Those routes imply open-ended exploration, while the current export contains only a review-document-shaped subset of data.

This ticket intentionally makes a clean cutoff. The application is not yet used anywhere externally, so we do **not** need compatibility wrappers, migration shims, legacy endpoint shapes, or old export formats. We can replace the current static export with a clearer architecture.

The new design has two first-class static features:

1. **Generic static codebase browser**
   - This is the foundation.
   - It allows a user to explore indexed commits, packages, symbols, source, xrefs, diffs, symbol history, body diffs, and selected impact graphs without a server.
   - It owns the static data contract.

2. **Static rich review markdown renderer**
   - This is a layer on top of the static codebase browser.
   - It renders authored markdown review documents.
   - Markdown directives hydrate into React widgets.
   - Review documents can cross-link into the generic browser routes.
   - Review widgets use the same static query API as the generic browser.

There is also a third artifact that remains important but is conceptually separate:

3. **SQLite LLM/query artifact**
   - This is the `review.db`/history database.
   - It is the build-time source for the static bundle.
   - It can also be shipped for LLMs, scripts, or humans who want SQL queries.
   - It is not the browser runtime data model.

The main architectural shift is this:

```text
Current prototype:

  review export
      -> reviewData blob
      -> enough WASM helpers for observed review widgets
      -> generic browser routes accidentally partially work

New design:

  static codebase bundle
      -> explicit manifest and data shards
      -> generic browser works by contract
      -> review documents are optional authored content on top
```

The immediate blocker that revealed the flaw was a static history page error:

```text
Failed to load body diff: STATIC_NOT_PRECOMPUTED
symbol body diff not precomputed: sym:...review.func.Register
```

That error is not just a missing body diff. It shows that `/history?symbol=...` is an open-ended browser route asking a review-centric export for data the export never promised to include. The fix is not another one-off precompute rule. The fix is a static browser data contract that explicitly covers symbol histories and the body diffs needed by those histories.

## Problem Statement

### What exists today

The current implementation includes several useful pieces:

- `internal/history/schema.go`
  - Defines `commits`, `snapshot_files`, `snapshot_symbols`, `snapshot_refs`, `file_contents`, and the `symbol_history` view.
- `internal/review/schema.go`
  - Adds `review_docs` and `review_doc_snippets` on top of the history database.
- `internal/review/export.go`
  - Builds `PrecomputedReview` with commits, adjacent diffs, histories, body diffs, impacts, and rendered review docs.
- `cmd/codebase-browser/cmds/review/export.go`
  - Builds the SPA, copies `search.wasm`, `wasm_exec.js`, source files, `review.db`, and writes `precomputed.json`.
- `internal/wasm/search.go`
  - Exposes WASM functions such as `GetCommitDiff`, `GetSymbolHistory`, `GetImpact`, `GetSymbolBodyDiff`, `GetReviewDocs`, and `GetReviewDoc`.
- `ui/src/api/historyApi.ts`
  - In static mode, rewrites some `/api/history/*` requests to WASM-backed lookups.
- `ui/src/features/review/ReviewDocPage.tsx`
  - Renders pre-rendered review document HTML and hydrates stubs with React widgets.
- `ui/src/features/history/HistoryPage.tsx`
  - Provides generic history exploration and symbol history pages.

These pieces prove the concept. Static rendering, WASM lookup, and hydrated review widgets can work.

### What is wrong with the current design

The problem is that the current data model is centered on review documents:

```go
// internal/review/export.go today

type PrecomputedReview struct {
    Version     string
    GeneratedAt string
    CommitRange string
    Commits     []CommitLite
    Diffs       map[string]*DiffLite
    Histories   map[string][]HistoryEntryLite
    Impacts     map[string]*ImpactLite
    BodyDiffs   map[string]*history.BodyDiffResult
    Docs        []ReviewDocLite
}
```

This type is named and shaped as review data. It contains browser data because the review widgets needed browser-like lookups. Over time it became a mixed bag:

- commits for history widgets;
- diffs for diff stats;
- histories for symbol history;
- body diffs for explicit `codebase-diff` snippets;
- impact graphs for explicit `codebase-impact` snippets;
- rendered review docs.

That shape is not wrong as a prototype, but it does not define a reliable static browser.

The frontend has the same ambiguity. `ui/src/api/historyApi.ts` still exposes server-shaped RTK Query endpoints:

```ts
getDiff({ from, to }) -> /diff?from=...&to=...
getSymbolHistory({ symbolId }) -> /symbols/:id/history
getSymbolBodyDiff({ from, to, symbolId }) -> /symbol-body-diff?...
getImpact({ sym, dir, depth, commit }) -> /impact?...
```

In server mode those are real HTTP routes. In static mode they are intercepted and translated to WASM lookups. This is useful but incomplete because there is no manifest that says which static queries are guaranteed.

### Symptoms

The symptoms are predictable:

- A review doc widget works because it was explicitly precomputed.
- A cross-link to `/history?symbol=...` may partially work.
- The History page may ask for body diffs that are not in `reviewData.bodyDiffs`.
- Impact queries may work only for review-requested depth/direction combinations.
- Some routes imply complete browser functionality even when the static bundle only contains curated review data.
- Missing data appears as technical errors such as `STATIC_NOT_PRECOMPUTED` instead of a product-level capability message.

### The design question

The central question is:

> What is the static export product?

For this ticket, the answer is explicit:

> The static export product is a generic static codebase browser. The review markdown renderer is a separate feature layered on top of that browser. Both must work without a server and share one static query contract.

## Goals

### Product goals

- Export a standalone directory that can be served by any static file server.
- Support generic codebase browsing and history exploration as the foundation.
- Support rich review markdown documents as a layer on top.
- Allow review documents to cross-link into generic browser pages.
- Include an optional SQLite database artifact for LLMs and scripts.
- Avoid server API calls in static mode.
- Make static capabilities explicit through a manifest.

### Engineering goals

- Cleanly separate build-time indexing from runtime browser querying.
- Replace the review-centric `PrecomputedReview` with a static bundle schema.
- Replace scattered static-mode patches with a dedicated static query provider.
- Keep the frontend widgets transport-agnostic.
- Use a clean cutoff; no backwards compatibility with the current `precomputed.json` shape is required.
- Make failure modes human-readable and tied to manifest coverage.

### Non-goals for the first cleanup pass

- Do not implement in-browser SQLite as the primary runtime path.
- Do not preserve old `reviewData` API shapes.
- Do not support arbitrary commit-pair diffs for every possible pair unless configured.
- Do not precompute deep impact graphs for every symbol by default.
- Do not optimize every bundle size issue before the data contract is correct.

## Vocabulary

### Static bundle

The exported directory containing the SPA, WASM, manifest, data files, optional SQLite DB, and optional source tree.

### Static browser

The browser application running without a Go server. It uses local JSON/WASM data from the static bundle.

### Review renderer

The feature that renders markdown review documents and hydrates inline widgets.

### LLM/query artifact

The SQLite database used by LLMs, scripts, or humans for SQL inspection. It is also the build-time input for the static bundle.

### Coverage

A manifest-declared statement of what data is present. Examples:

- adjacent commit diffs are included;
- history body diffs are included;
- impact graphs are included only for review-requested queries;
- source files are included or omitted.

### Static query provider

The TypeScript runtime layer that answers browser/widget queries from static data. It replaces ad-hoc `isStaticExport()` branching scattered across API files.

## Desired System Shape

The target architecture has four layers.

```text
+-------------------------------------------------------------+
| Feature UIs                                                  |
|                                                             |
|  A. Generic Static Codebase Browser                          |
|     - packages, symbols, source, search, xrefs               |
|     - commits, diffs, history, body diffs, impact            |
|                                                             |
|  B. Static Review Markdown Renderer                          |
|     - rendered markdown                                      |
|     - hydrated React widgets                                 |
|     - cross-links into the browser                           |
+------------------------------+------------------------------+
                               |
                               v
+-------------------------------------------------------------+
| Static Query Provider                                        |
|                                                             |
|  TypeScript API used by all widgets and pages.               |
|  It reads manifest coverage and lazy-loads data shards.      |
|  It has no dependency on HTTP server endpoints.              |
+------------------------------+------------------------------+
                               |
                               v
+-------------------------------------------------------------+
| Static Data Package                                          |
|                                                             |
|  manifest.json                                               |
|  data/commits.json                                           |
|  data/symbols.json                                           |
|  data/files.json                                             |
|  data/refs.json                                              |
|  data/diffs.json                                             |
|  data/body-diffs.json                                        |
|  data/impacts.json                                           |
|  data/review-docs.json                                       |
|  search.wasm                                                 |
|  assets/                                                     |
+------------------------------+------------------------------+
                               |
                               v
+-------------------------------------------------------------+
| SQLite DB                                                    |
|                                                             |
|  Authoritative build-time source.                            |
|  Optional shipped artifact for LLMs and SQL tools.           |
+-------------------------------------------------------------+
```

The important rule is:

> Feature UIs do not know whether they are in server mode or static mode. They call a query provider. The provider decides how to answer.

## Data Sources and Current File References

### History database schema

File:

```text
internal/history/schema.go
```

Key tables:

- `commits`
  - one row per indexed commit;
  - includes hash, short hash, author, timestamp, parents, and indexing errors.
- `snapshot_packages`
  - package metadata per commit.
- `snapshot_files`
  - file metadata per commit;
  - includes path, language, line count, SHA/content hash.
- `snapshot_symbols`
  - symbol metadata per commit;
  - includes symbol ID, kind, name, file ID, ranges, signature, body hash.
- `snapshot_refs`
  - symbol reference edges per commit;
  - used for xrefs and impact analysis.
- `file_contents`
  - content blob cache keyed by content hash.
- `symbol_history` view
  - joins symbols with commits for history queries.

This schema should remain the authoritative build-time model.

### Review database schema

File:

```text
internal/review/schema.go
```

Key tables:

- `review_docs`
  - slug, title, path, raw markdown content, frontmatter.
- `review_doc_snippets`
  - resolved markdown directives;
  - stores directive name, symbol ID, file path, kind, language, params, lines, commit.

This schema should remain the source for the review renderer.

### Current export command

File:

```text
cmd/codebase-browser/cmds/review/export.go
```

Current responsibilities:

1. open review DB;
2. load latest snapshot;
3. build search and xref indexes;
4. extract snippets/source refs;
5. load review-specific export data;
6. build SPA;
7. copy static assets;
8. copy source tree;
9. write `precomputed.json`;
10. copy `review.db`.

This command currently mixes static browser export, review export, source export, and LLM DB export in one place. The cleanup should split this into a package-level builder with explicit options.

### Current review export builder

File:

```text
internal/review/export.go
```

Current responsibilities:

- load commits;
- compute adjacent diffs;
- compute histories;
- compute some body diffs;
- compute some impact graphs;
- render review docs.

This should be replaced or substantially renamed. The browser data builder should not live in `internal/review` because the browser is the foundation and review docs are optional.

### Current WASM review types

File:

```text
internal/wasm/review_types.go
```

Current `ReviewData` mirrors `PrecomputedReview`. In the new design, WASM should receive a `StaticBundleData` or load specific data shards, not a review-specific blob.

### Current static transport

Files:

```text
ui/src/api/wasmClient.ts
ui/src/api/historyApi.ts
ui/src/api/docApi.ts
ui/src/api/runtimeMode.ts
```

The static transport is currently split across API files and WASM helper functions. The cleanup should create one provider layer, then have API slices or hooks call into that provider.

### Current feature UIs

Files:

```text
ui/src/app/App.tsx
ui/src/features/history/HistoryPage.tsx
ui/src/features/review/ReviewDocPage.tsx
ui/src/features/doc/DocSnippet.tsx
ui/src/features/symbol/SymbolPage.tsx
ui/src/features/source/SourcePage.tsx
ui/src/features/tree/HomePage.tsx
ui/src/features/tree/SearchPanel.tsx
```

These should become consumers of the query provider. They should not contain static/server branching.

## Proposed Static Bundle Layout

The output directory should look like this:

```text
export/
  index.html
  assets/
    index-....js
    index-....css
    ...

  wasm_exec.js
  search.wasm

  manifest.json

  data/
    commits.json
    packages.json
    files.json
    symbols.json
    refs.json
    search-index.json
    xref-index.json
    snippets.json
    source-refs.json
    file-xref-index.json
    histories.json
    diffs.json
    body-diffs.json
    impacts.json
    review-docs.json

  review.db           # optional, for LLMs/scripts
  source/             # optional, for source browsing fallback or downloads
```

The first implementation may still load a single `precomputed.json` if that is faster, but the target design should be sharded. The reason is size and lazy loading:

- a home page does not need all body diffs;
- a symbol page does not need all review docs;
- a review doc does not need all refs unless widgets request them;
- impact graphs can become large.

A clean cutoff means the old `precomputed.json` shape can be removed when the new layout lands.

## Manifest Design

Create a manifest at:

```text
manifest.json
```

Suggested shape:

```json
{
  "schemaVersion": 1,
  "kind": "codebase-browser-static-export",
  "generatedAt": "2026-05-01T23:00:00Z",
  "generator": {
    "name": "codebase-browser",
    "version": "dev"
  },
  "repo": {
    "module": "github.com/wesen/codebase-browser",
    "rootLabel": "codebase-browser"
  },
  "commitRange": "HEAD~20..HEAD",
  "commits": {
    "count": 21,
    "oldest": "abc...",
    "newest": "def..."
  },
  "features": {
    "codebaseBrowser": true,
    "reviewDocs": true,
    "llmDatabase": true,
    "sourceTree": true
  },
  "coverage": {
    "commitDiffs": "adjacent-plus-review-requested",
    "bodyDiffs": "history-transitions-plus-review-requested",
    "impacts": "review-requested",
    "histories": "all-symbols-with-multiple-entries",
    "refs": "all-indexed-commits",
    "symbols": "all-indexed-commits",
    "files": "all-indexed-commits",
    "source": "copied-source-tree"
  },
  "dataFiles": {
    "commits": "data/commits.json",
    "packages": "data/packages.json",
    "files": "data/files.json",
    "symbols": "data/symbols.json",
    "refs": "data/refs.json",
    "histories": "data/histories.json",
    "diffs": "data/diffs.json",
    "bodyDiffs": "data/body-diffs.json",
    "impacts": "data/impacts.json",
    "reviewDocs": "data/review-docs.json"
  },
  "routes": {
    "supported": [
      "/",
      "/packages/:id",
      "/symbol/:id",
      "/source/*",
      "/history",
      "/review",
      "/review/:slug"
    ],
    "unsupported": []
  }
}
```

### Why a manifest matters

Without a manifest, the UI discovers missing data by failing. With a manifest, the UI can explain limitations:

```text
This static export includes impact graphs only for review-requested queries.
The requested graph is not included. Re-export with --impact=depth1 or use server mode.
```

That is much better than:

```json
{"status":"STATIC_NOT_PRECOMPUTED"}
```

## Static Data Model

### Commits

File:

```text
data/commits.json
```

TypeScript shape:

```ts
interface StaticCommit {
  hash: string;
  shortHash: string;
  message: string;
  authorName: string;
  authorEmail: string;
  authorTime: number;
  parentHashes: string[];
  branch: string;
  error: string;
}
```

Required operations:

```ts
listCommits(): Promise<StaticCommit[]>;
resolveCommitRef(ref: string): Promise<string>;
getCommit(ref: string): Promise<StaticCommit>;
```

Commit resolution must support:

- `HEAD`
- `HEAD~N`
- full hash
- short hash
- unique hash prefix

Pseudocode:

```ts
function resolveCommitRef(ref, commits) {
  const ordered = sortByAuthorTimeAscending(commits);
  const newestIndex = ordered.length - 1;

  if (ref === "HEAD") return ordered[newestIndex].hash;

  if (ref matches /^HEAD~(\d+)$/) {
    const offset = parseInt(match[1]);
    return ordered[newestIndex - offset]?.hash;
  }

  const exact = ordered.find(c => c.hash === ref || c.shortHash === ref);
  if (exact) return exact.hash;

  const prefixMatches = ordered.filter(c => c.hash.startsWith(ref));
  if (prefixMatches.length === 1) return prefixMatches[0].hash;

  throw new QueryError("commit ref is unknown or ambiguous");
}
```

### Packages

File:

```text
data/packages.json
```

Shape:

```ts
interface StaticPackage {
  commitHash: string;
  id: string;
  importPath: string;
  name: string;
  doc: string;
  language: string;
}
```

Required operations:

```ts
listPackages(commit?: string): Promise<StaticPackage[]>;
getPackage(id: string, commit?: string): Promise<StaticPackage>;
```

### Files

File:

```text
data/files.json
```

Shape:

```ts
interface StaticFile {
  commitHash: string;
  id: string;
  path: string;
  packageId: string;
  size: number;
  lineCount: number;
  sha256: string;
  contentHash: string;
  language: string;
}
```

Required operations:

```ts
listFiles(commit?: string): Promise<StaticFile[]>;
getFileById(fileId: string, commit?: string): Promise<StaticFile>;
getFileByPath(path: string, commit?: string): Promise<StaticFile>;
getFileContent(path: string, commit?: string): Promise<string>;
```

Source content can be handled in two ways:

1. copy `source/` and fetch by path;
2. store content chunks in data files keyed by content hash.

For a clean first pass, keep copied `source/` if it already works, but make it explicit in the manifest.

### Symbols

File:

```text
data/symbols.json
```

Shape:

```ts
interface StaticSymbol {
  commitHash: string;
  id: string;
  kind: string;
  name: string;
  packageId: string;
  fileId: string;
  startLine: number;
  endLine: number;
  startOffset: number;
  endOffset: number;
  signature: string;
  bodyHash: string;
  exported: boolean;
  language: string;
}
```

Required operations:

```ts
searchSymbols(query: string, options?: { kind?: string; commit?: string }): Promise<StaticSymbol[]>;
getSymbol(id: string, commit?: string): Promise<StaticSymbol>;
getSymbolsAtCommit(commit?: string): Promise<StaticSymbol[]>;
getSymbolsByFile(fileId: string, commit?: string): Promise<StaticSymbol[]>;
```

The current WASM search path may still be useful. The difference is that it should operate on static bundle data, not a review-specific blob.

### Refs and xrefs

File:

```text
data/refs.json
```

Shape:

```ts
interface StaticRef {
  commitHash: string;
  id: number;
  fromSymbolId: string;
  toSymbolId: string;
  kind: string;
  fileId: string;
  startLine: number;
  endLine: number;
}
```

Required operations:

```ts
getRefsFrom(symbolId: string, commit?: string): Promise<StaticRef[]>;
getRefsTo(symbolId: string, commit?: string): Promise<StaticRef[]>;
```

Indexes to build at export time:

```text
refsByCommitThenFrom[commitHash][fromSymbolId] -> StaticRef[]
refsByCommitThenTo[commitHash][toSymbolId] -> StaticRef[]
```

### Histories

File:

```text
data/histories.json
```

Shape:

```ts
interface StaticSymbolHistoryEntry {
  symbolId: string;
  commitHash: string;
  shortHash: string;
  message: string;
  authorTime: number;
  bodyHash: string;
  signature: string;
  fileId: string;
  startLine: number;
  endLine: number;
  kind: string;
}
```

Required operation:

```ts
getSymbolHistory(symbolId: string): Promise<StaticSymbolHistoryEntry[]>;
```

Coverage:

```text
all symbols that appear in more than one indexed commit
```

History entries must be sorted in a single documented order. The recommendation is newest first for UI display, but body-diff precomputation can internally use oldest-to-newest. The provider should normalize this.

### Commit diffs

File:

```text
data/diffs.json
```

Key:

```text
<oldHash>..<newHash>
```

Shape:

```ts
interface StaticCommitDiff {
  oldHash: string;
  newHash: string;
  stats: DiffStats;
  files: FileDiff[];
  symbols: SymbolDiff[];
}
```

Default coverage:

```text
adjacent commit pairs in the exported range
plus pairs explicitly referenced by review docs
```

### Body diffs

File:

```text
data/body-diffs.json
```

Key:

```text
<oldHash>..<newHash>|<symbolId>
```

Shape:

```ts
interface BodyDiffResult {
  symbolId: string;
  name: string;
  oldCommit: string;
  newCommit: string;
  oldBody: string;
  newBody: string;
  unifiedDiff: string;
  oldRange: string;
  newRange: string;
}
```

Default coverage for the generic browser:

```text
body diffs for each adjacent transition in each exported symbol history
plus explicit review-document body diffs
```

This is the key correction for the `Register` error.

Pseudocode:

```go
func ComputeHistoryBodyDiffs(ctx, store, histories, repoRoot) map[string]BodyDiffResult {
    result := map[string]BodyDiffResult{}

    for symbolID, entries := range histories {
        ordered := sortByAuthorTimeAscending(entries)

        for i := 1; i < len(ordered); i++ {
            oldCommit := ordered[i-1].CommitHash
            newCommit := ordered[i].CommitHash

            if ordered[i-1].BodyHash == ordered[i].BodyHash {
                continue // no body change; UI can show unchanged
            }

            key := oldCommit + ".." + newCommit + "|" + symbolID
            if _, exists := result[key]; exists {
                continue
            }

            diff, err := history.DiffSymbolBodyWithContent(ctx, store.History, repoRoot, oldCommit, newCommit, symbolID)
            if err != nil {
                recordCoverageWarning(key, err)
                continue
            }

            result[key] = diff
        }
    }

    return result
}
```

The UI rule should be:

- If the body hash changed and a body diff exists, render it.
- If the body hash did not change, render “body unchanged”.
- If the body hash changed but a body diff is missing, render a capability-aware message, not a raw RTK error.

### Impact graphs

File:

```text
data/impacts.json
```

Key:

```text
<commitHash>|<symbolId>|<direction>|<depth>
```

Shape:

```ts
interface ImpactResponse {
  root: string;
  direction: 'usedby' | 'uses';
  depth: number;
  commit: string;
  nodes: ImpactNode[];
}

interface ImpactNode {
  symbolId: string;
  name: string;
  kind: string;
  depth: number;
  edges: ImpactEdge[];
  compatibility: string;
  local: boolean;
}

interface ImpactEdge {
  fromSymbolId: string;
  toSymbolId: string;
  kind: string;
  fileId: string;
}
```

Default coverage recommendation:

```text
review-requested impact graphs only
```

Optional future coverage:

```text
all changed symbols at depth 1, both directions
all local symbols at depth 1, both directions
all local symbols at configurable depth
```

Impact is the highest-risk data class for size growth, so the manifest must clearly state coverage.

### Review docs

File:

```text
data/review-docs.json
```

Shape:

```ts
interface StaticReviewDoc {
  slug: string;
  title: string;
  html: string;
  snippets: SnippetRef[];
  errors: string[];
}
```

The review docs are pre-rendered markdown HTML with placeholder elements for widgets. The placeholders should include all directive parameters as data attributes.

Example placeholder:

```html
<div
  data-codebase-widget="codebase-impact"
  data-directive="codebase-impact"
  data-sym="sym:..."
  data-params='{"dir":"uses","depth":"2"}'
></div>
```

The current `ReviewDocPage.tsx` already hydrates `[data-codebase-snippet]`. Rename this marker to something widget-neutral such as `[data-codebase-widget]` during cleanup.

## Static Query Provider API

Create a new frontend abstraction:

```text
ui/src/api/queryProvider.ts
ui/src/api/staticQueryProvider.ts
ui/src/api/serverQueryProvider.ts
```

The provider interface should describe product-level operations, not HTTP endpoints.

```ts
export interface CodebaseQueryProvider {
  manifest(): Promise<StaticManifest>;

  listCommits(): Promise<CommitRow[]>;
  getCommit(ref: string): Promise<CommitRow>;
  resolveCommitRef(ref: string): Promise<string>;

  listPackages(options?: { commit?: string }): Promise<PackageRow[]>;
  getPackage(id: string, options?: { commit?: string }): Promise<PackageRow>;

  searchSymbols(query: string, options?: { kind?: string; commit?: string }): Promise<SymbolRow[]>;
  getSymbol(id: string, options?: { commit?: string }): Promise<SymbolRow>;
  getSymbolsAtCommit(commit?: string): Promise<SymbolRow[]>;

  getRefsFrom(symbolId: string, options?: { commit?: string }): Promise<RefEdge[]>;
  getRefsTo(symbolId: string, options?: { commit?: string }): Promise<RefEdge[]>;

  getSymbolHistory(symbolId: string): Promise<SymbolHistoryEntry[]>;

  getCommitDiff(from: string, to: string): Promise<CommitDiff>;
  getSymbolBodyDiff(from: string, to: string, symbolId: string): Promise<BodyDiffResult>;

  getImpact(options: {
    symbolId: string;
    direction: 'usedby' | 'uses';
    depth: number;
    commit?: string;
  }): Promise<ImpactResponse>;

  listReviewDocs(): Promise<ReviewDocSummary[]>;
  getReviewDoc(slug: string): Promise<ReviewDoc>;
}
```

Then create two implementations:

```ts
class StaticQueryProvider implements CodebaseQueryProvider {
  // Reads manifest.json and data/*.json.
}

class ServerQueryProvider implements CodebaseQueryProvider {
  // Calls /api/* endpoints.
}
```

A runtime factory chooses the provider:

```ts
export function createQueryProvider(): CodebaseQueryProvider {
  if (import.meta.env.VITE_STATIC_EXPORT === '1') {
    return new StaticQueryProvider('/');
  }
  return new ServerQueryProvider('/api');
}
```

### Why this is better than endpoint interception

Current code asks endpoint-shaped strings in `historyApi.ts` and then decodes them in static mode:

```ts
if (arg.startsWith('/symbol-body-diff?')) { ... }
```

That keeps HTTP routing concepts in the static runtime. The new provider asks semantic questions:

```ts
provider.getSymbolBodyDiff(from, to, symbolId)
```

This makes widgets and pages simpler and avoids reimplementing URL parsing.

## React API Design

The app can still use RTK Query, but RTK Query should wrap provider methods, not raw endpoint URLs.

Example:

```ts
const providerBaseQuery: BaseQueryFn<ProviderRequest, unknown, QueryError> = async (request) => {
  const provider = getProvider();

  switch (request.type) {
    case 'listCommits':
      return { data: await provider.listCommits() };
    case 'getSymbolHistory':
      return { data: await provider.getSymbolHistory(request.symbolId) };
    case 'getSymbolBodyDiff':
      return { data: await provider.getSymbolBodyDiff(request.from, request.to, request.symbolId) };
  }
}
```

Or skip RTK Query for some static pages and use direct hooks:

```ts
export function useSymbolHistory(symbolId: string) {
  return useQuery(['symbol-history', symbolId], () => provider.getSymbolHistory(symbolId));
}
```

The important rule is that React components do not call `fetch('/api/...')` and do not parse static endpoint strings.

## Export Command Design

The command should express the two features and optional artifacts.

Initial clean command:

```bash
codebase-browser export static \
  --db review.db \
  --out ./export \
  --features browser,review \
  --body-diffs history \
  --impact review \
  --include-db \
  --include-source
```

If keeping the existing command tree is preferable, use:

```bash
codebase-browser review export \
  --db review.db \
  --out ./export \
  --features browser,review \
  --body-diffs history \
  --impact review
```

Because there is no backwards compatibility requirement, either command shape is acceptable. The implementation guide below assumes we keep `review export` initially to reduce CLI churn, but internally it should call a new static bundle builder.

### Suggested Go options

Create a new package:

```text
internal/staticbundle
```

Core types:

```go
type Feature string

const (
    FeatureBrowser Feature = "browser"
    FeatureReview  Feature = "review"
    FeatureLLMDB   Feature = "llm-db"
    FeatureSource  Feature = "source"
)

type BodyDiffCoverage string

const (
    BodyDiffNone    BodyDiffCoverage = "none"
    BodyDiffReview  BodyDiffCoverage = "review"
    BodyDiffHistory BodyDiffCoverage = "history"
    BodyDiffAll     BodyDiffCoverage = "all"
)

type ImpactCoverage string

const (
    ImpactNone       ImpactCoverage = "none"
    ImpactReview     ImpactCoverage = "review"
    ImpactChangedD1  ImpactCoverage = "changed-depth-1"
    ImpactAllLocalD1 ImpactCoverage = "all-local-depth-1"
)

type Options struct {
    DBPath string
    OutDir string
    RepoRoot string

    Features []Feature

    BodyDiffs BodyDiffCoverage
    Impact    ImpactCoverage

    IncludeDB bool
    IncludeSource bool

    BuildSPA bool
}
```

Main entrypoint:

```go
func Export(ctx context.Context, opts Options) error {
    db, err := review.Open(opts.DBPath)
    if err != nil { return err }
    defer db.Close()

    bundle, err := Build(ctx, db, opts)
    if err != nil { return err }

    if opts.BuildSPA {
        if err := BuildSPA(ctx); err != nil { return err }
    }

    if err := Write(ctx, bundle, opts.OutDir); err != nil { return err }

    if opts.IncludeDB {
        copyDB(opts.DBPath, filepath.Join(opts.OutDir, "review.db"))
    }

    if opts.IncludeSource {
        copySource(opts.RepoRoot, filepath.Join(opts.OutDir, "source"))
    }

    return nil
}
```

### Builder responsibilities

```go
func Build(ctx context.Context, store *review.Store, opts Options) (*Bundle, error) {
    commits := LoadCommits(ctx, store)
    packages := LoadPackages(ctx, store)
    files := LoadFiles(ctx, store)
    symbols := LoadSymbols(ctx, store)
    refs := LoadRefs(ctx, store)

    histories := ComputeHistories(commits, symbols)
    diffs := ComputeCommitDiffs(ctx, store, commits, opts)
    bodyDiffs := ComputeBodyDiffs(ctx, store, histories, opts)
    impacts := ComputeImpacts(ctx, store, refs, opts)

    reviewDocs := nil
    if opts.HasFeature(FeatureReview) {
        reviewDocs = RenderReviewDocs(ctx, store, latestSnapshot)
        AddReviewRequestedCoverage(reviewDocs, diffs, bodyDiffs, impacts)
    }

    manifest := BuildManifest(opts, counts, coverage)

    return &Bundle{Manifest: manifest, Data: ...}, nil
}
```

## Clean Cut Migration Plan

### Phase 1: Write the static bundle contract

Deliverables:

- `internal/staticbundle` package skeleton.
- Manifest Go types.
- Data file Go types.
- TypeScript mirror types.

Files to add:

```text
internal/staticbundle/manifest.go
internal/staticbundle/types.go
internal/staticbundle/options.go
ui/src/api/staticBundleTypes.ts
```

Validation:

- Go tests can marshal/unmarshal manifest and example data.
- TypeScript typecheck passes.

### Phase 2: Move export build logic out of `internal/review/export.go`

Current code in `internal/review/export.go` mixes browser and review concepts. Move browser logic to `internal/staticbundle`.

New package structure:

```text
internal/staticbundle/
  export.go          # high-level Export/Build/Write
  manifest.go        # manifest structs
  load.go            # load commits/packages/files/symbols/refs from SQLite
  diffs.go           # commit diff coverage
  histories.go       # symbol histories
  bodydiffs.go       # history/review body diff coverage
  impacts.go         # impact graph precompute
  reviewdocs.go      # review doc rendering, optional
  write.go           # write data shards
  spa.go             # optional SPA build/copy helpers
```

The old `internal/review/export.go` can be deleted or reduced to review-doc-specific helpers. Since this is a clean cutoff, deleting and replacing is preferred.

### Phase 3: Add history body diffs as browser core coverage

This phase fixes the known `/history?symbol=...` class of failures.

Algorithm:

1. Build histories for all symbols that appear in multiple commits.
2. For each symbol history, sort by commit time ascending.
3. For each adjacent pair:
   - if body hash is unchanged, no diff needed;
   - if body hash changed, compute `DiffSymbolBodyWithContent`;
   - store by `<old>..<new>|<symbol>`.
4. Record warnings in manifest if a diff could not be computed.

Manifest addition:

```json
{
  "warnings": [
    {
      "kind": "body-diff-missing",
      "key": "old..new|sym:...",
      "message": "symbol not found in old commit"
    }
  ]
}
```

UI behavior:

- body unchanged: show a neutral message;
- diff present: show diff;
- diff missing with manifest warning: show a human-readable explanation.

### Phase 4: Introduce `StaticQueryProvider`

Add:

```text
ui/src/api/queryProvider.ts
ui/src/api/staticQueryProvider.ts
ui/src/api/serverQueryProvider.ts
ui/src/api/queryErrors.ts
```

`StaticQueryProvider` should load `manifest.json` first, then lazy-load data files.

Pseudocode:

```ts
class StaticQueryProvider implements CodebaseQueryProvider {
  private manifestPromise?: Promise<StaticManifest>;
  private cache = new Map<string, Promise<unknown>>();

  manifest() {
    return this.loadJson('manifest.json');
  }

  private async loadJson<T>(path: string): Promise<T> {
    if (!this.cache.has(path)) {
      this.cache.set(path, fetch(path).then(r => {
        if (!r.ok) throw new QueryError('DATA_FILE_MISSING', path);
        return r.json();
      }));
    }
    return this.cache.get(path) as Promise<T>;
  }

  async getSymbolBodyDiff(from, to, symbolId) {
    const oldHash = await this.resolveCommitRef(from);
    const newHash = await this.resolveCommitRef(to);
    const bodyDiffs = await this.loadJson<Record<string, BodyDiffResult>>(
      (await this.manifest()).dataFiles.bodyDiffs
    );
    const key = `${oldHash}..${newHash}|${symbolId}`;
    const result = bodyDiffs[key];
    if (!result) throw this.notIncluded('bodyDiffs', key);
    return result;
  }
}
```

### Phase 5: Refactor RTK Query slices to use provider operations

Instead of parsing endpoint strings in `historyApi.ts`, define semantic requests.

Current pattern to remove:

```ts
if (arg.startsWith('/symbol-body-diff?')) {
  const params = paramsFor(arg);
  ...
}
```

Target pattern:

```ts
getSymbolBodyDiff: builder.query<BodyDiffResult, { from: string; to: string; symbolId: string }>({
  queryFn: async (args) => providerResult(() => provider.getSymbolBodyDiff(args.from, args.to, args.symbolId)),
})
```

This removes HTTP endpoint strings from static mode entirely.

### Phase 6: Make ReviewDocPage widget-neutral

Current code in `ReviewDocPage.tsx` looks for `[data-codebase-snippet]`, but it now hydrates more than snippets. Replace with:

```text
[data-codebase-widget]
```

Suggested handle:

```ts
interface WidgetStub {
  el: HTMLElement;
  directive: string;
  symbolId?: string;
  kind?: string;
  language?: string;
  commit?: string;
  params: Record<string, string>;
}
```

Then render a generic widget dispatcher:

```tsx
<CodebaseWidget directive={stub.directive} symbolId={stub.symbolId} params={stub.params} />
```

The dispatcher chooses:

- snippet widget;
- diff stats widget;
- body diff widget;
- symbol history widget;
- impact widget;
- changed files widget.

### Phase 7: Route guarantees and unsupported states

`ui/src/app/App.tsx` currently exposes routes regardless of static capability. In the new design, the manifest should define supported routes. The router can still register routes, but pages should check capability.

Example:

```tsx
function StaticCapabilityGate({ feature, children }) {
  const manifest = useManifest();
  if (!manifest.features[feature]) {
    return <UnsupportedFeature feature={feature} manifest={manifest} />;
  }
  return children;
}
```

For this ticket's desired product, the generic browser and review docs should be supported when exported with `--features browser,review`.

### Phase 8: Browser regression tests

Add a static export regression test that runs the real app in a browser.

Test fixture:

1. create small review DB over two commits;
2. create review doc containing:
   - snippet;
   - diff stats;
   - body diff;
   - symbol history link;
   - impact widget;
3. export static bundle;
4. serve with static HTTP server;
5. open review doc;
6. open `/history?symbol=...` for a symbol not explicitly in a body-diff widget;
7. assert no `/api/*` requests;
8. assert no widget error parts;
9. assert body diff appears or a capability-aware message appears.

Because the desired product includes generic history, the test should assert body diff appears for history transitions.

## Specific Fix for the Known Register Error

The observed error:

```text
Failed to load body diff: {"status":"STATIC_NOT_PRECOMPUTED","data":"symbol body diff not precomputed: sym:github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.func.Register"}
```

Cause:

- `/history?symbol=...Register` loads symbol history.
- The UI asks for a body diff between adjacent history entries.
- The export did not precompute body diffs for all symbol history transitions.
- It only precomputed changed adjacent diff symbols and explicit `codebase-diff` review snippets.

Correct fix under this design:

- Include `history-transitions` body diff coverage in the generic browser export.
- The History page should be part of the browser guarantee.
- Therefore every changed transition shown on the History page should have either:
  - a body diff; or
  - a manifest warning explaining why the diff could not be produced.

Implementation location:

```text
internal/staticbundle/bodydiffs.go
```

Pseudocode:

```go
func AddHistoryBodyDiffs(ctx context.Context, b *Bundle, store *review.Store, repoRoot string) {
    for symbolID, history := range b.Histories.BySymbol {
        entries := SortOldestToNewest(history)
        for i := 1; i < len(entries); i++ {
            old := entries[i-1]
            newer := entries[i]

            if old.BodyHash == newer.BodyHash {
                continue
            }

            key := BodyDiffKey(old.CommitHash, newer.CommitHash, symbolID)
            if b.BodyDiffs[key] != nil {
                continue
            }

            diff, err := historypkg.DiffSymbolBodyWithContent(ctx, store.History, repoRoot, old.CommitHash, newer.CommitHash, symbolID)
            if err != nil {
                b.Manifest.Warnings = append(b.Manifest.Warnings, Warning{
                    Kind: "body-diff-missing",
                    Key: key,
                    Message: err.Error(),
                })
                continue
            }

            b.BodyDiffs[key] = diff
        }
    }
}
```

## CLI Design

Because no backwards compatibility is required, the CLI can be cleaned up.

Recommended long-term command tree:

```text
codebase-browser index history
codebase-browser export static
codebase-browser serve
```

But a lower-risk intermediate shape is:

```text
codebase-browser review db create
codebase-browser review index
codebase-browser review serve
codebase-browser review export
```

For this cleanup ticket, keep the existing command names but change the internals and flags.

Suggested `review export` flags:

```text
--db string
    Path to SQLite DB.

--out string
    Output directory.

--features browser,review
    Comma-separated features. Valid: browser,review,llm-db,source.

--body-diffs history
    Body diff coverage. Valid: none,review,history,all.

--impact review
    Impact coverage. Valid: none,review,changed-depth-1,all-local-depth-1.

--include-db
    Copy SQLite DB into output.

--include-source
    Copy source tree into output.

--repo-root .
    Repository root used for source/body extraction fallback.
```

Default recommendation:

```text
--features browser,review
--body-diffs history
--impact review
--include-db true
--include-source true initially, optional later
```

## Intern Implementation Guide

This section is written as a step-by-step guide for someone new to the project.

### Step 0: Understand the build-time flow

Read these files first:

```text
internal/history/schema.go
internal/review/schema.go
internal/review/indexer.go
internal/history/diff.go
internal/history/bodydiff.go
internal/review/export.go
cmd/codebase-browser/cmds/review/export.go
```

The build-time flow is:

```text
git commit range
  -> history indexer
  -> SQLite history tables
  -> optional review docs indexed into review tables
  -> static export builder reads SQLite
  -> static bundle written to disk
```

### Step 1: Add `internal/staticbundle`

Create:

```text
internal/staticbundle/options.go
internal/staticbundle/manifest.go
internal/staticbundle/types.go
internal/staticbundle/export.go
internal/staticbundle/load.go
```

Start with types and tests only. Do not move all logic at once.

Test example:

```go
func TestManifestRoundTrip(t *testing.T) {
    m := StaticManifest{SchemaVersion: 1, Kind: "codebase-browser-static-export"}
    data, err := json.Marshal(m)
    require.NoError(t, err)
    var out StaticManifest
    require.NoError(t, json.Unmarshal(data, &out))
    require.Equal(t, m.Kind, out.Kind)
}
```

### Step 2: Implement loaders

Write SQL loaders from the existing DB tables:

```go
func LoadCommits(ctx context.Context, db *sql.DB) ([]Commit, error)
func LoadPackages(ctx context.Context, db *sql.DB) ([]Package, error)
func LoadFiles(ctx context.Context, db *sql.DB) ([]File, error)
func LoadSymbols(ctx context.Context, db *sql.DB) ([]Symbol, error)
func LoadRefs(ctx context.Context, db *sql.DB) ([]Ref, error)
```

Keep these boring and direct. Each loader should have a test using a small temp SQLite DB or an existing review store helper.

### Step 3: Implement histories

Use loaded symbols and commits or query the `symbol_history` view.

Important: include enough fields for the History page:

- symbol ID;
- commit hash;
- short hash;
- message;
- author time;
- body hash;
- signature;
- file ID;
- start/end lines;
- kind.

### Step 4: Implement body diff coverage

Move current body diff logic from `internal/review/export.go` into `internal/staticbundle/bodydiffs.go` and expand it.

Required sources:

1. history transitions if `--body-diffs=history` or `all`;
2. explicit review `codebase-diff` snippets if review feature is enabled;
3. changed symbols from adjacent diffs if useful, but history transitions are the browser guarantee.

### Step 5: Implement impact coverage

Move the current enriched BFS logic from `internal/review/export.go` into `internal/staticbundle/impacts.go`.

Keep default impact coverage as review-requested.

Future modes can be added after the base design is stable.

### Step 6: Write data shards

Create:

```go
func WriteBundle(ctx context.Context, bundle *Bundle, outDir string) error
```

It should write:

```text
manifest.json
data/commits.json
data/packages.json
data/files.json
data/symbols.json
data/refs.json
data/histories.json
data/diffs.json
data/body-diffs.json
data/impacts.json
data/review-docs.json
```

Use stable indentation for debuggability during development.

### Step 7: Refactor `review export` command

Replace most of `cmd/codebase-browser/cmds/review/export.go` with a call to `staticbundle.Export`.

The command should be thin:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    opts := staticbundle.Options{...from flags...}
    return staticbundle.Export(cmd.Context(), opts)
}
```

### Step 8: Implement frontend static provider

Create:

```text
ui/src/api/staticBundleTypes.ts
ui/src/api/queryProvider.ts
ui/src/api/staticQueryProvider.ts
```

The first version can load all JSON files eagerly for simplicity, then optimize later.

Provider pseudocode:

```ts
export class StaticQueryProvider implements CodebaseQueryProvider {
  async listCommits() {
    return this.load('commits');
  }

  async getSymbolHistory(symbolId) {
    const histories = await this.load('histories');
    return histories.bySymbol[symbolId] ?? [];
  }

  async getSymbolBodyDiff(from, to, symbolId) {
    const oldHash = await this.resolveCommitRef(from);
    const newHash = await this.resolveCommitRef(to);
    const key = `${oldHash}..${newHash}|${symbolId}`;
    const diffs = await this.load('bodyDiffs');
    if (!diffs[key]) throw new NotIncludedError('bodyDiffs', key, await this.manifest());
    return diffs[key];
  }
}
```

### Step 9: Replace static branch logic in `historyApi.ts`

Remove URL-string static parsing. Use provider methods.

The RTK endpoints should be semantic:

```ts
getSymbolHistory: builder.query({
  queryFn: ({ symbolId }) => providerResult(() => provider.getSymbolHistory(symbolId)),
})
```

This should work in both server and static mode because the provider factory chooses the implementation.

### Step 10: Update ReviewDocPage widget hydration

Change marker attributes:

```text
data-codebase-snippet -> data-codebase-widget
```

Create a widget dispatcher so review docs can embed any codebase widget without the review page knowing each widget's internals.

### Step 11: Browser validation

Create a smoke review with:

```markdown
# Static Export Smoke Review

```codebase-snippet sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd
```

```codebase-diff-stats from=HEAD~1 to=HEAD
```

```codebase-diff sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd from=HEAD~1 to=HEAD
```

```codebase-symbol-history sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.Register
```

```codebase-impact sym=github.com/wesen/codebase-browser/cmd/codebase-browser/cmds/review.newExportCmd dir=uses depth=2
```
```

Then validate:

- `/review/static-smoke` renders;
- `/history?symbol=sym:...Register` renders;
- body diffs on the history page render;
- no `/api/*` requests appear;
- no `[data-part="error"]` widgets appear.

## Testing Strategy

### Go unit tests

Add tests for:

- manifest marshal/unmarshal;
- commit ref resolution;
- body diff key creation;
- history transition enumeration;
- impact BFS on a tiny graph;
- review-requested coverage extraction.

### TypeScript unit tests

Add tests for:

- static provider commit ref resolution;
- static provider data file loading;
- missing data errors with capability messages;
- body diff lookup keys;
- impact lookup keys.

### Browser integration tests

Use Playwright to verify the final product.

Required assertions:

```ts
expect(apiRequests).toHaveLength(0);
expect(page.locator('[data-part="error"]')).toHaveCount(0);
await expect(page.getByText('Static Export Smoke Review')).toBeVisible();
await expect(page.getByText('Diff: newExportCmd')).toBeVisible();
await expect(page.getByText('Impact: newExportCmd')).toBeVisible();
```

Also test direct browser route:

```text
/#/history?symbol=sym:...Register
```

This route should be part of the generic browser contract.

## Failure and UX Design

Do not surface raw implementation errors to users when data is not included. Use structured errors.

```ts
class QueryError extends Error {
  code: 'NOT_FOUND' | 'NOT_INCLUDED' | 'AMBIGUOUS_REF' | 'DATA_FILE_MISSING';
  feature?: string;
  key?: string;
  hint?: string;
}
```

Example user message:

```text
This static bundle does not include impact graphs for depth 3.
Included coverage: review-requested impact graphs.
Re-export with --impact=all-local-depth-1 or use server mode for ad-hoc impact queries.
```

For body diffs in the default browser export, missing diffs should be rare and should have manifest warnings.

## Design Decisions

### Decision 1: Generic static browser is the foundation

Rationale:

- Review docs should be able to cross-link into browser pages.
- `/history?symbol=...` should not be an accidental partial feature.
- The browser data contract is broader and more fundamental than review docs.

### Decision 2: Review docs are a separate layer

Rationale:

- Review docs are authored narrative content.
- They embed widgets, but they should not own the data model.
- Keeping them separate avoids review-centric data blobs creeping into browser features.

### Decision 3: Keep SQLite as build-time source and optional LLM artifact

Rationale:

- SQLite is excellent for indexing and LLM/script queries.
- Browser runtime should use static data shards for predictable static hosting.
- In-browser SQLite can be revisited later, but it is not required for the core product.

### Decision 4: Use a manifest and coverage model

Rationale:

- Static bundles are finite.
- Some data, especially impact graphs, cannot be assumed complete.
- The UI needs to explain what is included.

### Decision 5: Clean cutoff instead of compatibility wrappers

Rationale:

- The app is not used externally.
- Maintaining old `reviewData` and new static bundle formats would slow down the cleanup.
- A clean replacement reduces complexity and confusion for future contributors.

## Alternatives Considered

### Alternative A: Keep patching `reviewData`

Rejected.

This is the current path. It leads to repeated fixes whenever another route asks for missing data. It keeps the review docs at the center even though the desired product has a generic browser foundation.

### Alternative B: Ship SQLite and run all queries in browser

Rejected for the first cleanup pass.

This might be useful later through sql.js, but it introduces binary size, loading, and query-planning concerns. It also does not remove the need for a frontend query provider and manifest.

### Alternative C: Disable generic browser routes in static export

Rejected for this ticket.

This would be acceptable for a review-only artifact, but the desired product explicitly requires a generic static browser and history foundation.

### Alternative D: Precompute everything eagerly

Partially rejected.

Some things should be broadly precomputed, such as histories and history body diffs. Other things, such as deep impact graphs for every symbol, can explode in size. Use explicit coverage modes instead.

## Risks

### Bundle size growth

History body diffs and refs can be large. Mitigations:

- shard data files;
- lazy-load by feature;
- skip unchanged body transitions;
- add export flags for coverage.

### Slow export time

Computing body diffs for all changed history transitions may take time. Mitigations:

- cache by content/body hash pair;
- parallelize safely later;
- show progress output.

### UI refactor scope

Replacing static endpoint interception with a provider layer touches many files. Mitigation:

- add provider behind existing RTK endpoints first;
- then refactor components gradually;
- no backward compatibility with old static data format is required.

### Source content ambiguity

Body diff computation needs file content. The DB caches `file_contents`, but fallback reads need `repoRoot`. Mitigation:

- make `--repo-root` explicit;
- prefer cached `file_contents`;
- warn in manifest if content is unavailable.

## Acceptance Criteria

The cleanup is done when:

1. Static export writes `manifest.json` and `data/*.json` shards.
2. Generic static browser routes work without server APIs:
   - home/package/symbol/source/search;
   - commits/diffs;
   - `/history?symbol=...` including body diffs for changed transitions.
3. Review docs render from static data.
4. Review widgets call the same query provider as browser pages.
5. Playwright confirms no `/api/*` requests in static mode.
6. The `Register` history body-diff scenario no longer reports `STATIC_NOT_PRECOMPUTED`.
7. Missing impact coverage produces a human-readable capability message.
8. `review.db` remains optionally included for LLM/script queries.
9. Old `reviewData`-centric static transport is removed.

## References

Current implementation files:

- `cmd/codebase-browser/cmds/review/export.go`
- `internal/review/export.go`
- `internal/review/schema.go`
- `internal/history/schema.go`
- `internal/history/bodydiff.go`
- `internal/history/diff.go`
- `internal/wasm/search.go`
- `internal/wasm/review_types.go`
- `ui/src/api/historyApi.ts`
- `ui/src/api/wasmClient.ts`
- `ui/src/api/docApi.ts`
- `ui/src/app/App.tsx`
- `ui/src/features/history/HistoryPage.tsx`
- `ui/src/features/review/ReviewDocPage.tsx`
- `ui/src/features/doc/DocSnippet.tsx`

Related prior ticket:

- `GCB-013`: code review command, static review markdown/WASM export, and earlier repair passes.
