---
Title: Git-level analysis mapping — turning codebase-browser primitives into PR-review data
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/docs/renderer.go
      Note: |-
        Directive parser + stub emitter to extend with codebase-diff, -history, -impact
        Directive parser + stub emission extended with codebase-diff etc.
    - Path: internal/indexer/id.go
      Note: |-
        Stable ID scheme that is the join key across commits
        Stable ID scheme (load-bearing cross-commit join key)
    - Path: internal/indexer/multi.go
      Note: |-
        Merge + duplicate-ID detection — reused for diff set ops
        Merge pattern reused for set-diff
    - Path: internal/indexer/types.go
      Note: |-
        Canonical schema (Symbol, File, Package, Ref, Range with byte offsets)
        Schema reused across snapshots
    - Path: internal/indexer/xref.go
      Note: Existing ref extraction (basis for impact BFS)
    - Path: internal/server/api_xref.go
      Note: |-
        Existing xref + file-xref handlers; new /api/impact reuses patterns
        Pattern for new /api/impact handler
    - Path: internal/server/server.go
      Note: |-
        Existing /api/* route table to extend with review endpoints
        Route table extended with /api/diff|impact|symbol-history|comments
    - Path: ui/src/features/doc/DocSnippet.tsx
      Note: Hydration dispatcher that grows new directive branches
ExternalSources: []
Summary: How the codebase-browser's existing extractor, xref graph, byte-accurate ranges, and stub/hydration pipeline can be composed into a semantic PR-review substrate. Covers per-symbol history via `git log -L`, symbol-by-symbol diff between two commit indexes, recursive impact analysis over the ref graph, symbol-anchored review comments, and the narrow set of new Go and TypeScript surfaces each piece needs.
LastUpdated: 2026-04-20T13:30:00Z
WhatFor: Decide how much of the codebase-browser can be reused versus rebuilt to deliver a PR-review tool that operates on symbols, not diff hunks.
WhenToUse: Read before starting implementation of any PR-review feature; also read before adding a new review-shaped endpoint or directive, because the reuse strategy is opinionated and the invariants it relies on are easy to violate.
---


# Git-level analysis mapping

## 1. Executive summary

GitHub-style unified diffs show line-level changes inside files. Every reviewer mentally re-derives the semantic change — *which function moved*, *whose signature flipped*, *who calls this thing* — by running grep in a second terminal. The codebase-browser has the semantic layer that the diff view lacks: stable IDs that survive file moves (`internal/indexer/id.go:19`), byte-accurate ranges on every symbol (`internal/indexer/types.go:38`), a full cross-reference graph (`internal/indexer/types.go:74`), and an HTTP+React rendering stack that already knows how to splice interactive widgets into markdown (`internal/docs/renderer.go:125`).

This document proposes mapping those primitives onto four PR-review capabilities:

1. **Per-symbol history** — "which commits touched this function?" Answered by running `git log -L <startLine>,<endLine>:<path>` against a symbol's `Range`. No new extraction needed.
2. **Semantic diff between commits** — "what symbols changed between base and head?" Answered by running the existing extractor against two commits and set-diffing the resulting `Index.Symbols` by ID.
3. **Impact analysis** — "who might break if I change this?" Answered by BFS over `Index.Refs` from the changed symbols outward.
4. **Symbol-anchored review comments** — "comments that follow the symbol when it moves." Answered by using `sym:...` as the comment anchor instead of `file:line`.

Each maps to a small number of new Go handlers (`internal/server/api_review_*.go`) plus one or two new renderer directives and React widgets. The per-commit extractor invocation is the only non-trivial moving part, and it is a single `codebase-browser diff --base X --head Y` CLI addition on top of the existing `index build` command.

The load-bearing invariant is the symbol-ID scheme. As long as `sym:<importPath>.<kind>.<name>` remains stable across module-internal moves (which is exactly what GCB-001/002 guaranteed), review data stitches together cleanly. Violating that invariant — for example by folding file paths into symbol IDs — would break every downstream PR-review surface silently. The rest of this document treats that invariant as a contract.

## 2. Problem statement and scope

### 2.1 Problem

GitHub's PR review is effective for visual diff comprehension but lacks four capabilities that semantic indexes make cheap:

1. **Identity across moves.** When a function is relocated to a new file, GitHub either renders a full delete + full add (useless for review) or, with `--find-renames`, a single renamed-file header with no intra-file change context. We already have a stable symbol ID; we can correlate the before and after bodies directly.
2. **Impact blast radius.** A reviewer staring at a changed `func Extract(...)` has no native way to see "23 callers, 12 of which pass arguments matching the old signature." We have that data in `Index.Refs`.
3. **Cross-commit timeline per symbol.** `git blame` tells you the last commit per line; it does not summarise "this function has been touched by these 5 commits over its lifetime." `git log -L <range>:<file>` does exactly that but nobody runs it from a browser tab.
4. **Comment rot.** A review comment at `path/to/file.go:42` on GitHub survives line-number changes only if the author force-pushes; comments at `sym:...Extract` would survive any module-internal move.

### 2.2 In scope

1. A `codebase-browser diff --base <ref> --head <ref>` command that loads (or generates) two indexes and emits a JSON diff.
2. A `/api/diff?base=X&head=Y` handler that returns the symbol-level diff.
3. A `/api/symbol-history/{id}` handler that shells to `git log -L`.
4. A `/api/impact/{id}?depth=N` handler that walks the ref graph.
5. A minimal symbol-anchored review-comment store (file-backed JSON is enough for phase 1; SQLite is the obvious phase-2 upgrade).
6. Three new doc-page directives (`codebase-diff`, `codebase-history`, `codebase-impact`) plus their React hydration components.
7. A `/pr/<id>` route in the SPA that composes the PR-summary view.

### 2.3 Out of scope

1. GitHub API integration. Reviewers paste a base/head pair; the tool does not poll webhooks or post statuses.
2. Cross-repository impact (tracing a Go change into a downstream TS repo via HTTP contracts). The index is per-module.
3. LSP-grade "find all references" from arbitrary positions. We only surface top-level-declaration-to-top-level-declaration edges, which is what the existing extractor emits.
4. Auto-rebase, auto-merge, or any write operations against the git repo.
5. Rich WYSIWYG review comment authoring. Markdown is assumed.

## 3. Current-state analysis

### 3.1 The extraction pipeline

The extractor (`internal/indexer/extractor.go`, 394 lines) walks Go packages via `golang.org/x/tools/go/packages` in typed-syntax mode and emits one `Symbol` per top-level declaration, plus one `Ref` per identifier use resolved via `types.Info.Uses`. The TypeScript extractor (`tools/ts-indexer/src/extract.ts`, 450 lines) mirrors the same schema using the TS Compiler API with a two-pass walker. Both emit the canonical `Index` shape declared at `internal/indexer/types.go:3-13`:

```go
type Index struct {
    Version     string
    GeneratedAt string
    Module      string
    GoVersion   string
    Packages    []Package
    Files       []File
    Symbols     []Symbol
    Refs        []Ref
}
```

Every `Symbol` carries byte offsets as well as line/col positions (`internal/indexer/types.go:38-47`):

```go
type Range struct {
    StartLine, StartCol int
    EndLine, EndCol     int
    StartOffset, EndOffset int
}
```

The byte offsets are authoritative for slicing source; the line/col pair is used for display and, critically for this proposal, as the input to `git log -L <startLine>,<endLine>:<path>`.

### 3.2 The ID scheme

`internal/indexer/id.go:19-40` defines the IDs that make cross-commit correlation possible:

```go
func SymbolID(importPath, kind, name, signatureForHash string) string {
    base := fmt.Sprintf("sym:%s.%s.%s", importPath, kind, name)
    if signatureForHash == "" {
        return base
    }
    h := sha256.Sum256([]byte(signatureForHash))
    return base + "#" + hex.EncodeToString(h[:4])
}

func MethodID(importPath, recvType, name string) string {
    recv := strings.TrimPrefix(recvType, "*")
    return fmt.Sprintf("sym:%s.method.%s.%s", importPath, recv, name)
}

func FileID(relPath string) string  { return "file:" + relPath }
func PackageID(importPath string) string { return "pkg:" + importPath }
```

Four properties are load-bearing for review:

1. **Move-stability.** Moving `func Extract` from `internal/indexer/extractor.go` to `internal/indexer/main.go` does not change its ID. The diff between two commits sees the same ID in both indexes.
2. **Signature-overload discriminator.** The optional `#xxxx` suffix (currently unused in practice) lets two overloads coexist without ID collision.
3. **Method-kind segment.** `method.<Recv>.<Name>` disambiguates method vs top-level function with the same name (hit during GCB-002 when `GoExtractor.Extract` collided with top-level `Extract`).
4. **Language-neutral form.** The scheme works for Go and TS today; a Python or Rust extractor would mint IDs in the same family.

### 3.3 The xref graph

`internal/indexer/types.go:74-82` defines the ref edge:

```go
type Ref struct {
    FromSymbolID string
    ToSymbolID   string
    Kind         string // call | uses-type | reads | use
    FileID       string
    Range        Range
}
```

Two separate passes populate `Index.Refs`: the Go side (`internal/indexer/xref.go`) walks each function body's `ast.Inspect` tree and asks `p.TypesInfo.Uses[ident]`; the TS side does the same via `checker.getSymbolAtLocation` with alias-following. Both only emit edges whose target is indexed (external-package refs are dropped), which is exactly the rule a PR-review UI wants: "show me callers inside this repo."

For reference: the current merged index on this repo carries ~960 refs across 310 symbols. An "impact radius" BFS two levels out from a changed symbol typically returns a double-digit caller set, which is a reasonable thing to render inline.

### 3.4 The server and its endpoints

`internal/server/server.go:28-44` registers the current routes:

```go
mux.HandleFunc("/api/index", s.handleIndex)
mux.HandleFunc("/api/packages", s.handlePackages)
mux.HandleFunc("/api/symbol/", s.handleSymbol)
mux.HandleFunc("/api/source", s.handleSource)
mux.HandleFunc("/api/snippet", s.handleSnippet)
mux.HandleFunc("/api/search", s.handleSearch)
mux.HandleFunc("/api/doc", s.handleDocList)
mux.HandleFunc("/api/doc/", s.handleDocPage)
mux.HandleFunc("/api/xref/", s.handleXref)
mux.HandleFunc("/api/snippet-refs", s.handleSnippetRefs)
mux.HandleFunc("/api/source-refs", s.handleSourceRefs)
mux.HandleFunc("/api/file-xref", s.handleFileXref)
```

All of these are loaded-index operations: the server holds a `browser.Loaded` (see `internal/browser/index.go`) that wraps a single `Index` JSON and a source `fs.FS`. The review endpoints need to either:

- Extend `browser.Loaded` to hold a pair of `Index` values (base, head), or
- Keep one loaded index as "current" and open others on demand via a small git-worktree indirection.

We recommend the second pattern (§5.4) because it keeps the existing server unchanged for non-review users and avoids doubling memory consumption for everyone.

### 3.5 The doc renderer and stub pipeline (GCB-004 legacy)

`internal/docs/renderer.go` contains two non-trivial mechanisms the review system reuses verbatim:

1. **Directive dispatch** (`internal/docs/renderer.go:176-243`): a fence with info string `codebase-X` resolves to a `SnippetRef` with `Directive`, `SymbolID`, `Kind`, `Language`, and optionally `Text`. Extending this with new directive names is a ten-line `switch` addition.
2. **Stub emission** (`internal/docs/renderer.go:140-165`): each resolved directive renders as a `<div class="codebase-snippet" data-codebase-snippet data-sym="..." data-directive="..." ...>` carrying a plaintext fallback. The React frontend walks the article for stubs after mount (`ui/src/features/doc/DocPage.tsx:23-50`) and uses `createPortal` to mount the right widget in each stub's place.

Review widgets slot into this pipeline with zero new plumbing: add a new directive name, add a stub data attribute, add a new branch in `DocSnippet.tsx` that dispatches to the review widget. This is exactly what GCB-004 made possible; we cash that in here.

### 3.6 Existing React widgets worth reusing

| Widget | Location | What it gives us |
|---|---|---|
| `<Code refs renderRefLink>` | `ui/src/packages/ui/src/Code.tsx:36` | Syntax highlighting + per-token link promotion when a ref matches |
| `<SourceView refs renderRefLink>` | `ui/src/packages/ui/src/SourceView.tsx:27` | Full-file view with linkified identifiers (byte-offset matched) |
| `<SymbolCard symbol snippet>` | `ui/src/packages/ui/src/SymbolCard.tsx:17` | Kind badge + name + signature + doc + snippet slot |
| `<LinkedCode refs language>` | `ui/src/features/symbol/LinkedCode.tsx:18` | Router-aware Code wrapper (Link to /symbol/{id}) |
| `<ExpandableSymbol symbol defaultOpen>` | `ui/src/features/symbol/ExpandableSymbol.tsx:27` | Collapsible card with lazy /api/snippet + /api/snippet-refs |
| `<XrefPanel symbolId>` | `ui/src/features/symbol/XrefPanel.tsx:13` | Two-column used-by / uses with target links |
| `<FileXrefPanel path>` | `ui/src/features/source/FileXrefPanel.tsx:14` | File-level used-by / uses (intra-file edges dropped) |
| `<DocSnippet sym directive>` | `ui/src/features/doc/DocSnippet.tsx:22` | Dispatch for embedded markdown directive |

Every review UI surface in §5 is composed from these.

## 4. Gap analysis

### 4.1 What's missing for review

1. **Multi-index loading.** `browser.Loaded` holds one `Index`. Review needs two (base and head), or more for timeline views. Introduce a `browser.Snapshot` struct keyed by `(repo-root, ref)` so the server can cache a few of them.
2. **Per-commit index materialisation.** Today `go generate ./internal/indexfs` produces one index for `HEAD`. Reviews need an index per commit-of-interest (base + head, minimum). Either pre-generate (via CI) or shell to `git worktree add` + `codebase-browser index build` on demand.
3. **Git invocation layer.** No code in `internal/` shells to git today (confirmed by `grep -rn 'git log\|exec.*git' internal/`). Add one small package `internal/gitops` with `History(path, startLine, endLine)`, `BlameLine(path, line)`, `ResolveRef(ref)`.
4. **Symbol diff algorithm.** No existing function computes the set-diff between two indexes. Straightforward but needs careful handling of the `#xxxx` signature-hash suffix so "same ID, different signature hash" is *not* reported as "removed+added" — it's "signature changed."
5. **Impact BFS.** `/api/xref` returns direct edges only. Review needs recursive descent (up to depth N) with cycle detection.
6. **Symbol-anchored comment store.** No persistence today. The simplest form is a sidecar JSON file keyed by `sym:...` plus `line` and an author/timestamp.

### 4.2 What we do *not* need to build

1. A second extractor. The existing Go + TS extractors run per-commit unchanged.
2. A second rendering stack. Markdown pages with hydrated stubs are exactly the shape the review system wants.
3. A second ID scheme. The `sym:...` IDs are the join key; any new data structure just references them.
4. A second frontend package. All review widgets go into `ui/src/features/review/`; widget-package primitives (`<Code>`, `<SourceView>`, `<SymbolCard>`) stay untouched.

## 5. Proposed architecture

### 5.1 Data flow

```
┌───────────────┐   ┌─────────────┐    ┌────────────────┐
│  git repo     │──▶│  worktree   │──▶ │ codebase-brwsr │
│               │   │ at <ref>    │    │ index build    │
└───────────────┘   └─────────────┘    └────────┬───────┘
                                                │
                                                ▼ per-commit Index JSON (~300KB)
                                       ┌──────────────────┐
                                       │ index cache      │
                                       │ keyed by SHA     │
                                       └────────┬─────────┘
                                                │
                       ┌────────────────────────┴──────────────────────────┐
                       ▼                                                    ▼
                ┌─────────────┐                                      ┌─────────────┐
                │ /api/diff   │  base, head → {added, removed,        │ /api/impact │
                │             │   signatureChanged, bodyChanged}      │ BFS over    │
                └──────┬──────┘                                       │ Refs        │
                       │                                              └──────┬──────┘
                       │                                                     │
                       ▼                                                     ▼
                ┌────────────────────────────────────────────────────────────────┐
                │  React SPA /pr/:id                                             │
                │   ─ SymbolDiffView  ─ ImpactGraph  ─ HistoryTimeline  ─ ...    │
                └────────────────────────────────────────────────────────────────┘
```

### 5.2 Index materialisation per commit

Three sourcing modes, all compatible:

1. **CI artefact mode.** Every PR build produces an `index.json` artefact for the head SHA and for the merge-base SHA. The review server resolves `(repo, sha)` to a path under `/var/codebase-browser/indexes/<repo>/<sha>.json`. No runtime extraction cost.
2. **On-demand worktree mode.** Reviewer opens `/pr/42`; server runs `git worktree add /tmp/cb-<sha> <sha>` and `codebase-browser index build --out /tmp/cb-<sha>/index.json`. First request is slow (~30 s on a warm pnpm cache), subsequent requests within TTL are cached.
3. **Hybrid.** Check the cache first; fall back to worktree. Recommended default.

The index materialiser lives behind a new interface:

```go
// internal/review/snapshots.go (new)
type Snapshot struct {
    Ref      string          // "a1b2c3d" or "main"
    Resolved string          // full SHA
    Index    *indexer.Index
    Source   fs.FS           // typically os.DirFS(worktreePath)
}

type SnapshotStore interface {
    Get(ctx context.Context, ref string) (*Snapshot, error)
    // Release drops the snapshot's refcount; when zero, the worktree may be
    // GC'd. No-op for the CI-artefact backing.
    Release(ref string)
}
```

Two implementations:

- `CachedSnapshotStore` wraps a directory of pre-built `index.json` files (CI mode).
- `WorktreeSnapshotStore` runs git + the extractor on demand, with an LRU cap of ~4 worktrees.

### 5.3 Symbol diff algorithm

Input: two `*indexer.Index`. Output: a deterministic `Diff` struct.

```go
// internal/review/diff.go (new)
type SymbolChange struct {
    ID          string           // sym:... (stable)
    Status      ChangeStatus     // added | removed | signature | body | doc | moved
    Before      *indexer.Symbol  // nil on added
    After       *indexer.Symbol  // nil on removed
    // For body-only changes: a canonical-normalised hash pair.
    BeforeBodyHash, AfterBodyHash string
}

type FileDiff struct {
    Path     string
    Added    []string // symbol IDs
    Removed  []string
    Changed  []string
}

type Diff struct {
    BaseRef, HeadRef string
    Symbols []SymbolChange
    Files   []FileDiff
    // Aggregate counters for the summary view.
    Counts struct{ Added, Removed, SignatureChanged, BodyChanged int }
}

func Compute(base, head *indexer.Index) *Diff
```

Classification rules:

1. **added.** ID in `head` but not `base`.
2. **removed.** ID in `base` but not `head`.
3. **signature changed.** Same ID in both, `Before.Signature != After.Signature`.
4. **body changed.** Same ID, same signature, but the source bytes in the declaration's byte range differ. Use a canonical-AST hash (or, phase-1, a normalised-whitespace hash) so formatting-only edits don't flood the UI.
5. **doc changed.** Same ID, same signature/body, but `Before.Doc != After.Doc`.
6. **moved.** Same ID, `Before.FileID != After.FileID`. Orthogonal to the above; a symbol can be moved *and* body-changed.

Implementation sketch (~80 lines):

```go
func Compute(base, head *indexer.Index) *Diff {
    bMap := map[string]*indexer.Symbol{}
    for i := range base.Symbols { bMap[base.Symbols[i].ID] = &base.Symbols[i] }
    hMap := map[string]*indexer.Symbol{}
    for i := range head.Symbols { hMap[head.Symbols[i].ID] = &head.Symbols[i] }

    d := &Diff{BaseRef: base.GeneratedAt, HeadRef: head.GeneratedAt}
    for id, b := range bMap {
        h, ok := hMap[id]
        if !ok {
            d.Symbols = append(d.Symbols, SymbolChange{ID: id, Status: Removed, Before: b})
            continue
        }
        if c := classify(b, h); c != nil { d.Symbols = append(d.Symbols, *c) }
    }
    for id, h := range hMap {
        if _, ok := bMap[id]; !ok {
            d.Symbols = append(d.Symbols, SymbolChange{ID: id, Status: Added, After: h})
        }
    }
    sort.Slice(d.Symbols, func(i, j int) bool { return d.Symbols[i].ID < d.Symbols[j].ID })
    // ... counts + FileDiff rollup
    return d
}
```

`classify` compares signature, body bytes (via each Snapshot's source FS), and doc strings.

### 5.4 Impact analysis

The existing `/api/xref/{id}` handler (`internal/server/api_xref.go:26`) emits one-hop edges. For PR review we want the transitive closure bounded by depth and dedup'd per target:

```go
// internal/review/impact.go (new)
type ImpactNode struct {
    SymbolID string
    Depth    int
    Edges    []indexer.Ref // the refs that put this node at this depth
}

func Impact(idx *indexer.Index, roots []string, depth int) []ImpactNode {
    // BFS over idx.Refs. Predecessor graph for "used by" (incoming refs);
    // successor graph for "uses" (outgoing refs). Review usually wants
    // predecessors ("who could break if I change this") but both are cheap.
}
```

Handler:

```go
GET /api/impact?head=<ref>&sym=<id>&dir=usedby|uses&depth=2
→ [{symbolId, depth, edges:[{fromSymbolId, toSymbolId, kind, fileId, range}]}]
```

Cost: O(depth × |refs|) with a visited set. On this repo (≈960 refs) depth-3 from any symbol completes in low-single-digit milliseconds. Ship with depth cap of 5.

### 5.5 Per-symbol history via `git log -L`

Given a symbol's `Range.StartLine` and `Range.EndLine`, git's `-L` flag gives us exactly the commits that modified the byte range — with automatic rename following via `--follow`-equivalent behaviour inside `-L`:

```bash
git log -L <startLine>,<endLine>:<filePath> --format='%H%x09%aI%x09%an%x09%s'
```

Wrapper:

```go
// internal/gitops/history.go (new)
type Commit struct {
    SHA          string    `json:"sha"`
    AuthoredAt   time.Time `json:"authoredAt"`
    Author       string    `json:"author"`
    Subject      string    `json:"subject"`
    // Optional: diff hunks for the range, so the frontend can render mini-diffs.
    HunkOld, HunkNew string
}

func History(ctx context.Context, repoRoot, relPath string, startLine, endLine int) ([]Commit, error)
```

Handler:

```go
GET /api/symbol-history/{sym} → [{sha, authoredAt, author, subject}...]
```

Cost: one git process per request, ~50 ms for a symbol with a few commits. Cache-friendly (the commit list for a range is stable once the range is in the past).

### 5.6 Symbol-anchored review comments

Phase 1 storage: a JSON sidecar at `.codebase-browser/comments.json` inside the repo. Phase 2: SQLite. The shape is language-agnostic:

```go
type Comment struct {
    ID         string    `json:"id"`
    AnchorSym  string    `json:"anchorSym"`   // sym:...  load-bearing
    AnchorLine int       `json:"anchorLine,omitempty"`  // optional, best-effort on move
    Author     string    `json:"author"`
    CreatedAt  time.Time `json:"createdAt"`
    Body       string    `json:"body"`        // markdown
    Resolved   bool      `json:"resolved"`
}
```

Handlers:

```
GET  /api/comments?sym=<id>      → list comments on a symbol
POST /api/comments               → create a comment
POST /api/comments/{id}/resolve
```

Move handling: when a symbol's `FileID` changes between base and head, comments follow the ID unchanged. When the ID itself disappears (symbol removed), the UI surfaces outstanding comments as "orphaned — delete or re-anchor?" prompts.

### 5.7 Endpoint additions

Summary of new handlers, all slotted into `internal/server/server.go` alongside existing routes:

```
GET  /api/diff?base=<ref>&head=<ref>
GET  /api/impact?sym=<id>&dir=usedby|uses&depth=N&head=<ref>
GET  /api/symbol-history/{sym}?head=<ref>
GET  /api/blame?path=<p>&line=<n>&head=<ref>
GET  /api/comments?sym=<id>
POST /api/comments
POST /api/comments/{id}/resolve
```

`?head=<ref>` is a new query parameter accepted by existing endpoints too: passing it switches the loaded snapshot. Omitted → the default HEAD snapshot. This keeps the blast radius small — no breaking changes, and single-snapshot users see the same behaviour.

### 5.8 Directive additions on the markdown renderer

Four new directive names, handled by `internal/docs/renderer.go:178-210`:

```markdown
```codebase-diff sym=<id> base=<ref> head=<ref>
```
```codebase-impact sym=<id> dir=usedby depth=2
```
```codebase-history sym=<id>
```
```codebase-caller-list sym=<id>
```
```

Each resolves to a stub div with the same attribute contract as `codebase-snippet` (see `internal/docs/renderer.go:140-165`). Hydration dispatches in `ui/src/features/doc/DocSnippet.tsx:22` grow matching branches that fetch from the new endpoints.

## 6. Phased implementation plan

### Phase 1 — snapshots + diff (3 days)

1. `internal/gitops/` package with `ResolveRef`, `History`, `BlameLine`.
2. `internal/review/snapshots.go` with `SnapshotStore` + two backends.
3. `internal/review/diff.go` with `Compute()`.
4. `codebase-browser diff --base X --head Y --out diff.json` CLI.
5. Go tests: set-diff edge cases, signature-hash collision handling.

### Phase 2 — server endpoints (2 days)

6. `/api/diff`, `/api/impact`, `/api/symbol-history`, `/api/blame`.
7. Snapshot load-on-demand in the server (`browser.Server` grows a `SnapshotStore` field).
8. `?head=<ref>` plumbing across existing endpoints.

### Phase 3 — React review surfaces (4 days)

9. `/pr/:id` route + loader (reads `?base=...&head=...` query string).
10. `<SymbolDiffView>` composing two `<ExpandableSymbol>`s side-by-side.
11. `<ImpactGraph>` (flat list first; graph-vis is a nice-to-have).
12. `<HistoryTimeline>`.
13. Extend `<DocSnippet>` dispatch with the four new directives.

### Phase 4 — symbol-anchored comments (2 days)

14. File-backed comment store.
15. `/api/comments/*` endpoints.
16. `<CommentThread sym>` widget + `codebase-comments` directive.
17. Orphan detection when an anchor symbol disappears from head.

### Phase 5 — CI artefact wiring (1 day)

18. GitHub Action that runs `codebase-browser index build` on PR + posts a link to the review UI with `?base=<sha>&head=<sha>` baked in.
19. Documentation page showing the full workflow.

Total: ≈12 days for a shipping rough cut. A further week of polish covers the impact-graph visualisation, orphan-comment UX, and BFS performance tuning on very large indexes.

## 7. Testing and validation strategy

| Layer | Test | Tooling |
|---|---|---|
| gitops | `History` against a fixture repo with N known commits on a known line range | `testing/fstest` + `git init` in a temp dir |
| diff | Added/removed/signature/body/moved classifications | Two fabricated `*indexer.Index` values |
| impact | BFS depth 1/2/3 correctness + cycle termination | Synthetic refs ring |
| snapshots | Worktree creation, cleanup on Release, LRU eviction | Temp dirs + a pinned `git` version |
| endpoints | httptest against loaded snapshots; JSON shape assertions | Existing `internal/server/server_test.go` pattern |
| widgets | Storybook stories for each new widget with mocked RTK-Query data | Existing Storybook setup |
| E2E | Render `/pr/<id>` with a two-commit fixture; assert counts, changed symbol list, impact panel | Playwright or Cypress (phase 5) |

Two invariant tests worth locking down in CI:

1. **Determinism across re-extraction.** Given the same commit, two independent `codebase-browser index build` runs produce byte-identical JSON (modulo `generatedAt`). Already true today; worth asserting.
2. **ID stability across a fixture move.** Move a function file in a fixture repo; assert the diff reports a `moved` (not `removed+added`).

## 8. Risks, alternatives, open questions

### 8.1 Risks

1. **Body-diff noise.** Whitespace-only reformatting registers as a body change. Mitigation: normalise via AST pretty-printer before hashing. For Go, `go/printer` into a canonical config is ~5 lines. For TS, a `prettier --check` style normalisation is more invasive; use raw-byte hash in phase 1 and iterate.
2. **Large indexes strain server memory.** Two snapshots of a large monorepo could be hundreds of MB. Mitigation: load diffs as streams where possible, keep a hard cap on concurrent snapshots, add `Release()` support to `SnapshotStore`.
3. **`git log -L` performance on hot functions.** A function touched by hundreds of commits over years produces a long history. Mitigation: paginate (`--max-count=50` by default) and surface "view all" as an explicit action.
4. **Force-push changes the SHA.** Symbol-anchored comments survive; URL-anchored snapshots (`?head=<sha>`) become stale. Mitigation: store comments keyed by `(sym, sha-family)` where "sha-family" is resolved via `git merge-base` at comment creation time; stale URLs redirect to the newest SHA with a banner.
5. **Snapshot-store concurrency.** If two reviewers hit `/pr/42` simultaneously, on-demand worktree creation races. Mitigation: per-ref `singleflight.Group` around the build, which is a 10-line Go idiom.

### 8.2 Alternatives considered

1. **Replace the extractor with an LSP index (SCIP, LSIF).** SCIP is richer (every identifier position, including parameters and locals), but the codebase-browser's emitter is ours to change and already encodes exactly the scope we want (top-level symbols, call/uses-type/reads edges). Switching to SCIP means losing the single-binary property and adopting a schema that is not designed for byte-range rendering. Rejected.
2. **Use GitHub's own PR API + a browser extension.** Overlays GitHub's diff with xref data fetched from our server. Lower-friction adoption, but tied to one forge; no support for internal/self-hosted diffs. Worth prototyping as a phase-2 delivery channel rather than a replacement for the standalone UI.
3. **Server-side diff rendering (no snapshots in memory).** `codebase-browser diff` produces an HTML blob; the server just serves it. Simpler, but gives up interactive xref navigation because the ref graph is gone from the HTML. Rejected for the flagship path; usable as an email-friendly "shareable review" export.
4. **Git-native per-hunk comments, pointed at symbols post-hoc.** Keep GitHub's line comments as primary; display them overlaid on our symbol view by resolving hunk → symbol via byte range. Pragmatic integration path; does not conflict with native symbol-anchored comments.

### 8.3 Open questions

1. Where do we store the per-commit index artefacts when there is no CI? Options: a designated remote (S3, a configured git remote tracking `refs/codebase-index/<sha>`), or lazy on-demand only. Recommend CI-first with on-demand as a fallback.
2. How deep should impact BFS go by default? Phase 1 proposal: `depth=2` default, `depth=5` cap. Revisit once we see real usage.
3. Should the diff classify "doc-only" as its own status, or fold it into body? Opinion: separate. Doc changes need less scrutiny than body changes but are still notable.
4. Should review comments exist on `file:...` anchors too, not just `sym:...`? Yes, for prose-level feedback about the whole file, but that's a phase-3 extension.
5. How do we surface non-indexed changes (e.g. a new markdown file)? Probably as a plain file-level diff rendered via the existing `<SourceView>`, annotated as "no symbols — plain diff."

## 9. References

1. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/indexer/types.go` — canonical schema
2. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/indexer/id.go` — ID scheme
3. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/indexer/multi.go` — Merge + dedup
4. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/indexer/xref.go` — ref extraction
5. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/server.go` — route table
6. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_xref.go` — xref + file-xref handlers
7. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go` — directive parsing + stub emission
8. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocSnippet.tsx` — hydration dispatch
9. `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/packages/ui/src/SymbolCard.tsx` — reusable symbol surface
10. `git log -L` documentation — https://git-scm.com/docs/git-log#Documentation/git-log.txt--Lltstartgtltendgtltfilegt
11. GCB-002 design-doc — `ttmp/2026/04/20/GCB-002--.../design-doc/01-typescript-extractor-design-and-implementation-guide.md`
12. GCB-004 design-doc — `ttmp/2026/04/20/GCB-004--.../design-doc/01-react-hydrated-snippet-widgets-design-and-implementation-guide.md`
