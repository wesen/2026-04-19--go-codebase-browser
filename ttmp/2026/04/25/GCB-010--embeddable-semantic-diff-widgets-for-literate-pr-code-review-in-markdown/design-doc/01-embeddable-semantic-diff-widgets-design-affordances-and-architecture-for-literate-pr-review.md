---
Title: Embeddable semantic diff widgets — design, affordances, and architecture for literate PR review
Ticket: GCB-010
Status: active
Topics:
    - codebase-browser
    - pr-review
    - semantic-diff
    - embeddable-widgets
    - markdown-directives
    - history-index
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/history/indexer.go
      Note: Per-commit semantic indexer — the engine that builds per-snapshot symbol tables
    - Path: internal/history/diff.go
      Note: Symbol-level diff — classifies added/removed/modified/moved between two commits
    - Path: internal/history/bodydiff.go
      Note: Per-symbol body diff — extracts old/new bodies and produces unified diff
    - Path: internal/history/schema.go
      Note: SQLite history schema — commits, snapshot_symbols, snapshot_refs, symbol_history view
    - Path: internal/docs/renderer.go
      Note: Markdown directive pipeline — codebase-* fenced blocks → hydrated React stubs
    - Path: internal/server/api_history.go
      Note: History REST API — /api/history/diff, /api/history/symbol-body-diff, symbol-history
    - Path: ui/src/features/history/HistoryPage.tsx
      Note: React history UI — commit timeline, diff view, symbol history, body diff
    - Path: internal/indexer/types.go
      Note: Canonical index schema — Symbol, File, Package, Ref, Range with byte offsets
    - Path: ttmp/2026/04/20/GCB-005--semantic-pr-review-on-top-of-codebase-browser/design-doc/01-git-level-analysis-mapping-turning-codebase-browser-primitives-into-pr-review-data.md
      Note: Predecessor design — PR review data model and git-level analysis
    - Path: ttmp/2026/04/20/GCB-005--semantic-pr-review-on-top-of-codebase-browser/design-doc/02-ui-affordances-wireframes-and-embeddable-widget-catalog.md
      Note: Predecessor design — UI affordances, wireframes, widget catalog
    - Path: ttmp/2026/04/24/GCB-009--git-aware-codebase-index-track-symbol-locations-across-commits-for-per-function-diff-and-change-history/design-doc/01-git-aware-codebase-index.md
      Note: Predecessor design — git-aware indexing, per-commit snapshots, history SQLite store
ExternalSources: []
Summary: A comprehensive design for embedding semantic diff, history, impact, and code-browsing widgets directly into markdown documents, enabling literate PR code review guides where reviewers navigate source, explore changes across commits, and understand blast radius — all inline without opening a new window.
LastUpdated: 2026-04-25T12:00:00Z
WhatFor: Guide the implementation of embeddable semantic diff widgets that turn markdown doc pages into interactive code review workspaces.
WhenToUse: Read before implementing any new codebase-* directive, review widget, or history-backed API endpoint. Also the onboarding reference for new team members joining the project.
---

# Embeddable semantic diff widgets for literate PR code review

## 1. Executive summary

This document describes how to turn markdown documents into interactive code review workspaces by embedding semantic diff, history, impact, and code-browsing widgets directly into the prose.

The codebase-browser already has the three layers needed to make this work:

1. **A semantic index** that extracts every function, type, variable, and cross-reference from Go and TypeScript source code (`internal/indexer/`).
2. **A git history subsystem** that re-runs the indexer at every commit, storing per-commit symbol snapshots in SQLite so you can ask "how did this function change between commit A and commit B?" (`internal/history/`).
3. **A markdown directive pipeline** that replaces fenced code blocks like ` ```codebase-snippet sym=...``` ` with live, React-hydrated widgets inside rendered doc pages (`internal/docs/renderer.go`).

What's new in this ticket is the **combination**: new directives (`codebase-diff`, `codebase-symbol-history`, `codebase-impact`, `codebase-commit-walk`, `codebase-annotation`) that pull data from the history subsystem and render it inline in markdown, plus the new API endpoints and React components to back them.

The target user is a **code reviewer** who reads a literate PR guide — a markdown document that walks them through the change — and can click, expand, and navigate the actual source code and its evolution right there on the page, without switching to a terminal, an IDE, or GitHub.

This document is written to be self-contained. A new intern or a developer joining the project should be able to read it from top to bottom and understand every part of the system — what each component does, why it exists, how the data flows, and what the new widgets will look like.

## 2. Problem statement and motivation

### 2.1 The problem

When you review a pull request, you typically:

1. Open the PR description on GitHub.
2. Open the "Files changed" tab.
3. Read a line-level unified diff.
4. Mentally reconstruct which *functions* changed, which *types* were added, and which *callers* might be affected.
5. Open a terminal or IDE to `git blame`, `git log`, or search for callers.
6. Switch back and forth between GitHub, your IDE, and maybe a design doc.

Steps 4–6 are where time is lost. The reviewer is doing semantic work ("which symbols changed?") with a tool that only shows line-level diffs. The codebase-browser already has the semantic data — symbol IDs, cross-references, byte-accurate ranges, and now per-commit history. The missing piece is making that data available **inline in the review document itself**.

### 2.2 The vision: literate PR review guides

Imagine a markdown document written by the PR author (or auto-generated) that says:

> This PR refactors the indexer to support build tags. Here's what changed:
>
> **1. The `Extract` function gained a new parameter.**
> [inline widget shows the before/after signature and body diff]
>
> **2. Three new callers need updating.**
> [inline widget shows the impact panel with 3 affected call-sites]
>
> **3. The `ExtractOptions` struct was extended.**
> [inline widget shows the full type definition with highlighted new fields]
>
> **4. Here's how `Merge` evolved over the last 5 commits to reach this point.**
> [inline widget shows a compact history timeline]

The reviewer reads the prose, sees the live data inline, clicks to navigate deeper if needed, and never leaves the document. That's the goal.

### 2.3 Why embed in markdown, not a separate page?

The codebase-browser already has a `/history` page (built in GCB-009) with a full commit timeline and symbol diff UI. Why not just link reviewers there?

Three reasons:

1. **Context is king.** A literate guide provides narrative context around each change — "this function was refactored because of X" — that a raw diff page cannot.
2. **Reduced cognitive load.** The reviewer stays in one document. No tab switching, no mental context reconstruction.
3. **Shareability.** A single markdown URL contains the entire review guide. Forward it in Slack, embed it in a wiki, or export it to PDF.

The `/history` page still exists for ad-hoc exploration. The embedded widgets are for **curated** review experiences.

### 2.4 Scope

**In scope for this ticket:**

- New markdown directives for embedding semantic diff, symbol history, impact analysis, commit walks, and annotations.
- New API endpoints (or extensions to existing ones) that serve the data the widgets need.
- New React components that render inline in doc pages.
- A complete widget catalog with ASCII wireframes, API contracts, and authoring ergonomics.

**Out of scope:**

- GitHub API integration (no webhooks, no PR status posting).
- Multi-repository impact analysis (the index is per-module).
- Real-time collaborative review (comments, threading — covered by GCB-005 Phase 4).
- Auto-generation of literate review guides from PR diffs (future work).

## 3. System architecture overview

This section gives a complete picture of the codebase-browser's current architecture, explaining every subsystem that the new widgets build on top of. If you're new to the project, read this carefully — everything that follows assumes this context.

### 3.1 The big picture

The codebase-browser is a single-binary documentation browser for Go + TypeScript codebases. At build time, it indexes the source tree; at runtime, it serves a React SPA plus a JSON API from an embedded HTTP server — no external dependencies.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     BUILD TIME                                        │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐ │
│  │ Go AST       │  │ TS Compiler  │  │ Vite SPA Build               │ │
│  │ Extractor    │  │ API Extractor│  │ (ui/ → internal/web/embed/)  │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┬───────────────┘ │
│         │                 │                          │                │
│         ▼                 ▼                          ▼                │
│  ┌──────────────────────────────┐  ┌──────────────────────────────┐  │
│  │ index.json (merged)          │  │ SPA assets (HTML/JS/CSS)      │  │
│  │ internal/indexfs/embed/      │  │ internal/web/embed/public/    │  │
│  └──────────────────────────────┘  └──────────────────────────────┘  │
│                                                                       │
│  Also embedded: source tree snapshot (internal/sourcefs/embed/)      │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼ go build -tags embed
┌─────────────────────────────────────────────────────────────────────────┐
│                     RUNTIME (single binary)                            │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────────┐ │
│  │  HTTP Server (net/http, Go 1.22+ ServeMux)                     │ │
│  │                                                                  │ │
│  │  /api/index        → full index JSON                            │ │
│  │  /api/symbol/:id   → symbol detail + snippet                    │ │
│  │  /api/source       → file content                               │ │
│  │  /api/snippet      → symbol body at byte offsets                │ │
│  │  /api/search       → symbol search                              │ │
│  │  /api/xref/:id     → cross-references (callers, callees)        │ │
│  │  /api/doc          → rendered markdown pages                    │ │
│  │  /api/history/*    → git history endpoints (GCB-009)            │ │
│  │  /*                → React SPA (fallback)                       │ │
│  └──────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│  ┌──────────────────────┐  ┌──────────────────────────────────────┐  │
│  │ Static Index         │  │ History DB (optional, --history-db)  │  │
│  │ (embedded index.json)│  │ (SQLite, per-commit snapshots)       │  │
│  └──────────────────────┘  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 The semantic index

The index is the heart of the system. It's built by two extractors:

- **Go extractor** (`internal/indexer/extractor.go`, ~394 lines): walks Go packages via `golang.org/x/tools/go/packages` in typed-syntax mode. Emits one `Symbol` per top-level declaration plus one `Ref` per identifier resolved through `types.Info.Uses`.
- **TypeScript extractor** (`tools/ts-indexer/src/extract.ts`, ~450 lines): mirrors the same schema using the TS Compiler API with a two-pass walker (pass 1 registers declarations, pass 2 walks bodies for refs).

Both emit the same canonical shape defined in `internal/indexer/types.go`:

```go
type Index struct {
    Version     string
    GeneratedAt string
    Module      string
    GoVersion   string
    Packages    []Package   // pkg:github.com/foo/bar
    Files       []File       // file:path/to/file.go
    Symbols     []Symbol     // sym:github.com/foo/bar.func.Extract
    Refs        []Ref        // call/uses-type/reads edges
}
```

Every `Symbol` carries **byte offsets** as well as line/column positions:

```go
type Range struct {
    StartLine, StartCol     int   // for display
    EndLine,   EndCol       int
    StartOffset, EndOffset  int   // authoritative for slicing source
}
```

The byte offsets are what make snippet extraction reliable: the renderer reads the source file, slices bytes `[StartOffset:EndOffset]`, and gets the exact function body — no line-counting heuristics.

**Symbol IDs are stable across file moves.** The ID scheme (`internal/indexer/id.go`) is:

```
sym:<importPath>.<kind>.<name>              # top-level
sym:<importPath>.method.<Recv>.<name>       # method
```

Moving `func Extract` from `extractor.go` to `main.go` does not change its ID. This is the property that makes cross-commit diff meaningful.

### 3.3 The cross-reference graph

`Index.Refs` contains edges between symbols:

```go
type Ref struct {
    FromSymbolID string   // who's calling
    ToSymbolID   string   // what's being called
    Kind         string   // "call", "uses-type", "reads", "use"
    FileID       string   // file where the reference occurs
    Range        Range    // byte position of the reference
}
```

On this repo, the merged index has ~960 refs across ~310 symbols. This graph is what powers the "who calls this function?" queries that review widgets need.

### 3.4 The doc rendering pipeline

Doc pages live under `internal/docs/embed/pages/` as regular markdown files. The rendering pipeline (`internal/docs/renderer.go`) is a two-pass system:

1. **Preprocess**: scan for fenced code blocks with info strings starting with `codebase-`. Parse the directive and its parameters, resolve the symbol or file, and emit an HTML `<div>` stub with data attributes.
2. **Render**: feed the preprocessed markdown through goldmark (a Go markdown → HTML converter).

Currently supported directives:

| Directive | Purpose | Example |
|-----------|---------|---------|
| `codebase-snippet` | Full symbol body | `` ```codebase-snippet sym=indexer.Merge``` `` |
| `codebase-signature` | Just the signature line | `` ```codebase-signature sym=indexer.Extract``` `` |
| `codebase-doc` | Godoc / TSDoc comment | `` ```codebase-doc sym=indexer.Extract``` `` |
| `codebase-file` | Whole or partial file contents | `` ```codebase-file path=internal/server/server.go range=28-44``` `` |

Each resolved directive emits a stub div like:

```html
<div class="codebase-snippet" data-codebase-snippet
     data-stub-id="stub-1" data-sym="sym:..." data-directive="codebase-snippet"
     data-kind="func" data-lang="go">
  <pre><code class="language-go">func Merge(base, extra *Index) (*Index, error) { ... }</code></pre>
</div>
```

The React frontend walks the rendered HTML for these stubs after mount (`ui/src/features/doc/DocSnippet.tsx`) and uses `createPortal` to mount rich interactive widgets (with syntax highlighting, xref links, expand/collapse) in place of the plaintext fallback.

**This stub + hydrate pipeline is the extension point for new widgets.** Adding a new directive means: (1) add a `case` in `resolveDirective`, (2) emit a stub with new `data-directive` and `data-params`, (3) add a hydration branch in `DocSnippet.tsx`.

### 3.5 The git history subsystem (GCB-009)

This is the newest and most important subsystem for review widgets. It was built in GCB-009 and adds the ability to track symbol locations across commits.

**The core idea**: run the semantic indexer at every commit, store each result as a "snapshot" in SQLite, then query across snapshots to see how symbols changed over time.

#### 3.5.1 Schema

The history database (`internal/history/schema.go`) has these tables:

```sql
-- Commit metadata
CREATE TABLE commits (
    hash TEXT PRIMARY KEY,
    short_hash TEXT, message TEXT,
    author_name TEXT, author_email TEXT,
    author_time INTEGER, branch TEXT
);

-- Per-commit symbol snapshots
CREATE TABLE snapshot_symbols (
    commit_hash TEXT REFERENCES commits(hash),
    id TEXT,           -- sym:... (same ID scheme as index.json)
    kind TEXT, name TEXT, package_id TEXT, file_id TEXT,
    start_line INT, end_line INT,
    start_offset INT, end_offset INT,
    signature TEXT, doc TEXT,
    body_hash TEXT,     -- SHA-256 of the function body bytes
    exported INT, language TEXT,
    PRIMARY KEY (commit_hash, id)
);

-- Per-commit file snapshots
CREATE TABLE snapshot_files (
    commit_hash TEXT, id TEXT, path TEXT,
    package_id TEXT, sha256 TEXT
);

-- Per-commit cross-reference snapshots
CREATE TABLE snapshot_refs (
    commit_hash TEXT, id INT,
    from_symbol_id TEXT, to_symbol_id TEXT,
    kind TEXT, file_id TEXT
);

-- Cached file contents (deduped by SHA-256)
CREATE TABLE file_contents (
    content_hash TEXT PRIMARY KEY,
    content BLOB
);

-- Convenience view: symbol history across commits
CREATE VIEW symbol_history AS
SELECT s.id, s.name, s.kind, c.hash, c.short_hash, c.message,
       c.author_time, s.body_hash, s.start_line, s.end_line, s.signature
FROM snapshot_symbols s JOIN commits c ON c.hash = s.commit_hash;
```

The `body_hash` column is the key innovation. It's a SHA-256 of the function body bytes at each commit. When the same `sym:...` ID appears in two commits with different `body_hash` values, you know the function body changed — without having to compare the full source text.

#### 3.5.2 Indexing pipeline

The indexer (`internal/history/indexer.go`) works like this:

```
for each commit:
    1. git worktree add /tmp/cb-<sha> <sha>
    2. codebase-browser index build --module-root /tmp/cb-<sha>
    3. Load the resulting Index into snapshot_symbols/snapshot_files/snapshot_refs
    4. Compute body_hash for each symbol from the worktree file
    5. git worktree remove /tmp/cb-<sha>
```

The scanner (`internal/history/scanner.go`) discovers commits to index:

```go
type ScanOptions struct {
    RepoRoot    string
    Range       string   // e.g. "HEAD~20..HEAD"
    Incremental bool     // skip already-indexed commits
    FileFilter  []string // only index commits touching these paths
}
```

#### 3.5.3 Diff engine

The diff engine (`internal/history/diff.go`) compares two commit snapshots:

```go
type SymbolDiff struct {
    SymbolID   string
    Name       string
    Kind       string
    ChangeType ChangeType   // added, removed, modified, signature-changed, moved, unchanged
    OldStartLine, OldEndLine int
    NewStartLine, NewEndLine int
    OldSignature, NewSignature string
    OldBodyHash,  NewBodyHash  string
}

type CommitDiff struct {
    OldHash, NewHash string
    Files   []FileDiff
    Symbols []SymbolDiff
    Stats   DiffStats   // counts of each change type
}
```

The classification logic uses a `FULL OUTER JOIN` on `snapshot_symbols` across two commits, matching by symbol ID. Change types:

- **added**: symbol ID exists in new commit but not old
- **removed**: exists in old but not new
- **modified**: same ID, different `body_hash`
- **signature-changed**: same ID, different `signature`
- **moved**: same ID, different `file_id`
- **unchanged**: same ID, same `body_hash`

#### 3.5.4 Body diff

The body diff (`internal/history/bodydiff.go`) goes one level deeper: for a single symbol that changed between two commits, it extracts the full old and new function bodies and computes a unified diff:

```go
type BodyDiffResult struct {
    SymbolID    string
    OldBody     string      // full old function body
    NewBody     string      // full new function body
    UnifiedDiff string      // line-by-line diff with +/-/  prefixes
    OldRange    string      // "lines 42-78"
    NewRange    string      // "lines 42-85"
}
```

The diff algorithm is a simple LCS-based approach that shows every line (unchanged with `  ` prefix, removed with `- `, added with `+ `) — giving the reviewer full function context, not just the changed region.

### 3.6 The React frontend

The SPA (`ui/`) is a React app built with Vite. Key packages for review widgets:

| Component | Location | What it does |
|-----------|----------|-------------|
| `<Code>` | `ui/src/packages/ui/src/Code.tsx` | Syntax highlighting + per-token xref links |
| `<SourceView>` | `ui/src/packages/ui/src/SourceView.tsx` | Full-file view with linkified identifiers |
| `<SymbolCard>` | `ui/src/packages/ui/src/SymbolCard.tsx` | Kind badge + name + signature + doc |
| `<ExpandableSymbol>` | `ui/src/features/symbol/ExpandableSymbol.tsx` | Collapsible card with lazy snippet loading |
| `<XrefPanel>` | `ui/src/features/symbol/XrefPanel.tsx` | Two-column "used by" / "uses" panel |
| `<HistoryPage>` | `ui/src/features/history/HistoryPage.tsx` | Full history UI (commit timeline, diff, symbol history) |
| `<DocSnippet>` | `ui/src/features/doc/DocSnippet.tsx` | Hydration dispatcher for embedded directives |

Data fetching uses RTK-Query (`ui/src/api/`). The history API slice (`ui/src/api/historyApi.ts`) defines typed hooks for all `/api/history/*` endpoints.

## 4. The widget catalog — new directives and affordances

This section describes every new widget we propose to add. For each widget, we give: its purpose, the markdown authoring syntax, an ASCII wireframe of what it looks like, the API it calls, the React component that renders it, and how it composes with existing primitives.

### 4.1 Widget overview

Seven new directives, each backed by the history subsystem:

| # | Directive | One-line purpose |
|---|-----------|-----------------|
| 1 | `codebase-diff` | Side-by-side symbol body diff between two commits |
| 2 | `codebase-symbol-history` | Compact timeline of commits that touched a symbol |
| 3 | `codebase-impact` | Transitive caller/callee list from a changed symbol |
| 4 | `codebase-commit-walk` | Guided walk through a series of commits with narrative |
| 5 | `codebase-annotation` | Inline code annotation (author highlights lines in a snippet) |
| 6 | `codebase-changed-files` | File-level diff summary between two commits |
| 7 | `codebase-diff-stats` | Compact numeric summary of changes between two commits |

Plus two rendering-mode extensions to existing directives:

| # | Extension | Purpose |
|---|-----------|---------|
| 8 | `codebase-snippet ... commit=<ref>` | Show a symbol as it existed at a specific commit |
| 9 | `codebase-signature ... commit=<ref>` | Show a signature as it existed at a specific commit |

### 4.2 `codebase-diff` — side-by-side symbol body diff

**Purpose.** Show how a single function/type/method changed between two commits, inline in prose. This is the core widget for literate review guides.

**Authoring.**

```markdown
The `Extract` function gained a `strict` parameter in this PR:

```codebase-diff sym=indexer.Extract from=HEAD~3 to=HEAD
```

Note the new error handling path for malformed packages.
```

**ASCII wireframe (rendered inline in the doc page):**

```
┌─── indexer.Extract · body modified ──────────────────────────────────────┐
│  9f8e7d6 (HEAD~3)                          a1b2c3d (HEAD)               │
│ ┌────────────────────────────────┐  ┌────────────────────────────────┐  │
│ │ 22  func Extract(opts ExtractOp│  │ 22  func Extract(opts ExtractOp│  │
│ │ 23      (*Index, error) {      │  │ 23      strict bool) (*Index,  │  │
│ │ 24      pkgs := loadPackages(  │  │ 24      error) {               │  │
│ │ 25          opts.ModuleRoot,   │  │ 25      pkgs := loadPackages(  │  │
│ │ 26      )                      │  │ 26          opts.ModuleRoot,   │  │
│ │ 27      var symbols []Symbol   │  │ 27      )                      │  │
│ │                                │  │ 28      if strict {            │  │
│ │                                │  │ 29          pkgs = filterTags()│  │
│ │                                │  │ 30      }                      │  │
│ │ 28      for _, pkg := range pk │  │ 31      var symbols []Symbol   │  │
│ │ 29          symbols = append(s │  │ 32      for _, pkg := range pk │  │
│ └────────────────────────────────┘  └────────────────────────────────┘  │
│                                                     +3 lines, -0 lines  │
│  [ open full symbol ]  [ view raw diff ]  [ caller impact → ]           │
└──────────────────────────────────────────────────────────────────────────┘
```

**What the reviewer sees:**

- A header showing the symbol name and change classification (body modified / signature changed / moved / added / removed).
- Two scrollable code panes, side-by-side: the "before" version (from the `from` commit) and the "after" version (from the `to` commit).
- Changed lines are highlighted: red background for removed, green for added, grey for unchanged context.
- Every identifier in both panes is a clickable xref link (reuses the existing `<Code>` component with `renderRefLink`).
- A footer with action links: expand to full symbol page, view raw unified diff, see caller impact.

**API contract.**

```
GET /api/history/symbol-body-diff?from=<hash>&to=<hash>&symbol=<sym-id>
→ {
    symbolId, name,
    oldCommit, newCommit,
    oldBody, newBody,
    unifiedDiff,
    oldRange, newRange
  }
```

Also needs the commit-level diff for classification:

```
GET /api/history/diff?from=<hash>&to=<hash>
→ { oldHash, newHash, files[], symbols[], stats }
```

The widget fetches both and matches the requested symbol in the diff's `symbols[]` array to get the `ChangeType`.

**React component.** `<SymbolDiffInlineWidget>` — composed from two `<Code>` panes (from `ui/src/packages/ui/src/Code.tsx`) plus a diff overlay renderer. New file: `ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx`.

**Fallback (JS disabled).** Plaintext: "Diff of indexer.Extract from 9f8e7d6 to a1b2c3d: +3 lines."

### 4.3 `codebase-symbol-history` — compact commit timeline

**Purpose.** Show which commits touched a symbol and when. Useful for answering "has this function been stable?" or "when was this introduced?"

**Authoring.**

```markdown
The merge logic has been iterated several times:

```codebase-symbol-history sym=indexer.Merge limit=8
```
```

**ASCII wireframe:**

```
┌─── History: indexer.Merge ──────────────────────────────────────────────┐
│                                                                         │
│  2026-04-24  Manuel  ● "git-aware indexing: per-function body hash"  HEAD│
│  2026-04-23  Manuel  ● "sqlite index: structured query concepts"        │
│  2026-04-22  Manuel    "refactor: extract gitutil package"   (unchanged) │
│  2026-04-21  Manuel  ● "add cross-reference extraction"                 │
│  2026-04-20  Manuel  ● "Phase 1: Go extractor scaffold"                 │
│                                                                         │
│  ● = body changed   4 modified / 1 unchanged   [ expand all ]          │
│  [ diff first→last ]                                                    │
└─────────────────────────────────────────────────────────────────────────┘
```

**What the reviewer sees:**

- A vertical list of commits, newest first. Each row shows date, author, commit message (truncated), and a dot indicator.
- The dot is filled (●) if the `body_hash` changed at that commit; unfilled if the symbol existed but wasn't modified.
- The "diff first→last" button opens a `codebase-diff` widget comparing the earliest and latest commit.
- Click any commit row to expand a mini body-diff between that commit and the previous one.

**API contract.**

```
GET /api/history/symbols/{symbolID}/history?limit=50
→ [{
    commitHash, shortHash, message, authorTime,
    bodyHash, startLine, endLine, signature, kind
  }]
```

**React component.** `<SymbolHistoryInlineWidget>` — a compact variant of the existing `<SymbolHistoryPanel>` from `HistoryPage.tsx`, but rendered inline without the "from/to" selectors. New file: `ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx`.

**Fallback.** "Symbol indexer.Merge: 5 commits (4 with body changes)."

### 4.4 `codebase-impact` — transitive caller/callee list

**Purpose.** Show the "blast radius" of a change — who calls this function? Who calls those callers? — up to a configurable depth.

**Authoring.**

```markdown
Adding the `strict` parameter to `Extract` affects these callers:

```codebase-impact sym=indexer.Extract dir=usedby depth=2
```
```

**ASCII wireframe:**

```
┌─── Impact: indexer.Extract (used-by, depth 2) ─────────────────────────┐
│                                                                         │
│  Depth 1 — direct callers (5)                                          │
│    func  cmds/index.buildIndex                call         ✓ ok         │
│    func  cmds/index.handleBuild                call         ⚠ review    │
│    func  indexer_test.TestExtract              call         ⚠ review    │
│    func  indexer_test.TestExtractErrors        call         ✓ ok        │
│    func  indexer_test.TestExtractBuildTags     call         ⚠ review    │
│                                                                         │
│  Depth 2 — indirect callers (3)                                        │
│    func  serve.(*Server).handleIndex            via handleBuild         │
│    func  serve.(*Server).handleSnippet          via handleBuild         │
│    func  serve.(*Server).handleSearch           via handleBuild         │
│                                                                         │
│  ⚠ = signature change may affect this caller    [ view as graph → ]    │
└─────────────────────────────────────────────────────────────────────────┘
```

**What the reviewer sees:**

- A grouped list: depth-1 callers first, then depth-2, etc.
- Each row shows: kind badge, symbol name (clickable), ref kind ("call", "uses-type"), and a compatibility indicator.
- The ⚠ indicator is derived from the commit diff: if the target symbol's signature changed, callers that pass positional arguments get flagged.
- "View as graph" renders a mini SVG adjacency diagram for small result sets.

**API contract.** New endpoint:

```
GET /api/history/impact?sym=<id>&dir=usedby&depth=2&from=<hash>&to=<hash>
→ [{
    symbolId, name, kind, depth,
    edges: [{ fromSymbolId, toSymbolId, kind, fileId }],
    compatibility: "ok" | "review" | "unknown"
  }]
```

**Implementation.** BFS over `snapshot_refs` starting from the given symbol. The `from`/`to` params enable the compatibility check: if the symbol's signature differs between `from` and `to`, callers get "review" status.

**React component.** `<ImpactInlineWidget>` — new file: `ui/src/features/doc/widgets/ImpactInlineWidget.tsx`. Reuses the kind-badge rendering from `<SymbolCard>`.

**Fallback.** "5 direct callers of indexer.Extract, 3 indirect (depth 2)."

### 4.5 `codebase-commit-walk` — guided narrative through commits

**Purpose.** The signature widget for literate PR review. Instead of showing one diff, it walks the reviewer through a series of commits with narrative text between each step.

**Authoring.**

```markdown
## PR #42: Build tag support

Let me walk you through the commits in order.

```codebase-commit-walk from=HEAD~4 to=HEAD
step "Phase 1: add ExtractOptions struct"
      Show `indexer.ExtractOptions` signature at this commit.
      Note the new `BuildTags` field.

step "Phase 2: implement tag filtering"
      Show diff of `indexer.Extract` between this and previous commit.
      The function body grew by 12 lines.

step "Phase 3: update callers"
      Show impact of `indexer.Extract` at this commit.
      Three callers needed argument updates.
```

**ASCII wireframe:**

```
┌─── Commit Walk: HEAD~4 → HEAD (4 commits) ─────────────────────────────┐
│                                                                         │
│  Step 1/3: "Phase 1: add ExtractOptions struct"                        │
│  Commit 9f8e7d6 · Manuel · 2026-04-24 18:30                            │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │ type ExtractOptions struct {                                      │  │
│  │     ModuleRoot   string                                           │  │
│  │     Patterns     []string                                         │  │
│  │     IncludeTests bool                                             │  │
│  │ +   BuildTags    []string  // new: filter by build tags           │  │
│  │ }                                                                 │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│  Author's note: "Note the new BuildTags field."                        │
│                                                                         │
│  [ ← prev ]                                    [ next: Phase 2 → ]     │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─  │
│                                                                         │
│  Step 2/3: "Phase 2: implement tag filtering"                          │
│  Commit a1b2c3d · Manuel · 2026-04-24 19:15                            │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │ [ embedded codebase-diff widget showing Extract changes ]         │  │
│  │ +3 lines, body modified                                           │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│  Author's note: "The function body grew by 12 lines."                  │
│                                                                         │
│  [ ← prev ]                                    [ next: Phase 3 → ]     │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─  │
│                                                                         │
│  Step 3/3: "Phase 3: update callers"                                   │
│  Commit c4d5e6f · Manuel · 2026-04-24 20:00                            │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │ [ embedded codebase-impact widget showing 3 affected callers ]    │  │
│  │ ⚠ handleBuild: signature mismatch                                │  │
│  │ ⚠ TestExtract: needs BuildTags in test fixture                    │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  [ ← prev ]                                    [ ✓ review complete ]   │
└─────────────────────────────────────────────────────────────────────────┘
```

**What the reviewer sees:**

- A step-by-step view with one commit (or diff between adjacent commits) per step.
- Each step has a title, commit metadata, the embedded sub-widget (snippet, diff, impact, or annotation), and the author's narrative note.
- Navigation buttons at the bottom: prev/next, with a "review complete" button on the last step.
- The author controls which widget type appears at each step via the `step` sub-directive.

**This is the most complex widget.** It composes the other widgets (diff, impact, history, snippet) into a guided narrative. The markdown authoring syntax uses a simple DSL inside the fenced block:

```
step "<title>"
      Show <widget-type> <params>
      <prose note>
```

Where `<widget-type>` is one of: `signature`, `snippet`, `diff`, `impact`, `history`, or `files`.

**API contract.** No new endpoint. The widget composes calls to existing endpoints:
- `/api/history/commits` for the commit list
- `/api/history/diff` for inter-commit diffs
- `/api/history/symbol-body-diff` for per-symbol diffs
- `/api/history/symbols/{id}/history` for symbol history

**React component.** `<CommitWalkWidget>` — new file: `ui/src/features/doc/widgets/CommitWalkWidget.tsx`. Internal state tracks the current step index. Each step renders one of the sub-widgets.

**Fallback.** "4 commits from 9f8e7d6 to c4d5e6f. Walk through with JS enabled."

### 4.6 `codebase-annotation` — inline code annotation

**Purpose.** Show a code snippet with author-highlighted lines and inline comments. Like a GitHub comment but embedded in the review guide.

**Authoring.**

```markdown
Here's the critical section that changed:

```codebase-annotation sym=indexer.Extract commit=HEAD
highlight 28-30 "New strict-mode gate"
highlight 45-48 "This error path is the main behavioral change"
note 35 "Be careful: this line is performance-sensitive"
```
```

**ASCII wireframe:**

```
┌─── indexer.Extract @ HEAD ─── annotations ──────────────────────────────┐
│                                                                         │
│  22  func Extract(opts ExtractOptions, strict bool) (*Index, error) {   │
│  23      pkgs := loadPackages(opts.ModuleRoot, opts.Patterns)           │
│  24      var symbols []Symbol                                           │
│ ▶25      if strict {                              ← "New strict-mode   │
│  26          pkgs = filterTags(pkgs, opts.BuildTags)      gate"         │
│  27      }                                                              │
│  28      for _, pkg := range pkgs {                                     │
│  29          syms := walkPackage(pkg)                                   │
│  30          symbols = append(symbols, syms...)                         │
│  31      }                                                              │
│ ▶35      result := dedup(symbols)        ← "Be careful: this line is   │
│  36      return &Index{                          performance-sensitive" │
│  37          Symbols: result,                                            │
│  38      }, nil                                                         │
│  39  }                                                                  │
│                                                                         │
│  2 highlighted regions, 1 note                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

**What the reviewer sees:**

- The full symbol body with syntax highlighting and xref links (reuses `<Code>`).
- Highlighted line ranges get a coloured background (e.g. light yellow for "pay attention").
- Notes appear as inline annotations to the right of the relevant line.
- The `▶` marker indicates annotated lines.

**API contract.** Uses existing endpoints:
- `/api/snippet?sym=<id>` for the symbol body
- `/api/source?path=<path>` for the file content (to resolve at a specific commit, uses `/api/history/commits/{hash}/symbols` to get the byte offsets, then reads from the history cache)

**React component.** `<AnnotationWidget>` — wraps `<Code>` with a highlight overlay. New file: `ui/src/features/doc/widgets/AnnotationWidget.tsx`.

**Fallback.** The symbol body is rendered as plain code. Notes are listed as a bulleted list below.

### 4.7 `codebase-changed-files` — file-level diff summary

**Purpose.** Show which files changed between two commits, with line counts.

**Authoring.**

```markdown
This PR touches 4 files:

```codebase-changed-files from=main to=HEAD
```
```

**ASCII wireframe:**

```
┌─── Changed files: main → HEAD ──────────────────────────────────────────┐
│                                                                         │
│  M  internal/indexer/extractor.go           +47  -19                   │
│  M  internal/indexer/types.go               +12   -3                   │
│  M  cmd/codebase-browser/cmds/index/build   +18   -8                   │
│  A  internal/indexer/build_tags.go           +24    -   (new file)      │
│                                                                         │
│  4 files, +101 / -30 lines                                              │
│  [ open file in source view → ]                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

**API contract.** Uses existing `/api/history/diff?from=<hash>&to=<hash>` and extracts the `files[]` array.

**React component.** `<ChangedFilesWidget>` — new file: `ui/src/features/doc/widgets/ChangedFilesWidget.tsx`. Each file name is a `<Link to={/source/<path>}>`.

**Fallback.** "4 files changed: +101/-30 lines."

### 4.8 `codebase-diff-stats` — compact numeric summary

**Purpose.** A one-liner summary of the change, suitable for embedding in a paragraph.

**Authoring.**

```markdown
This is a medium-sized PR:

```codebase-diff-stats from=main to=HEAD
```

with 3 functions modified and 1 type added.
```

**ASCII wireframe (inline):**

```
 ● 3 body-changed  ● 1 signature-changed  ● 1 added  ● 0 removed  ● 4 files
```

**React component.** `<DiffStatsWidget>` — renders as an inline `<span>` with coloured badges. New file: `ui/src/features/doc/widgets/DiffStatsWidget.tsx`.

**Fallback.** "3 modified, 1 signature-changed, 1 added, 0 removed, 4 files."

### 4.9 Extension: `commit=` parameter on existing directives

The existing `codebase-snippet` and `codebase-signature` directives currently resolve against the static HEAD index. We extend them with an optional `commit=<ref>` parameter that resolves the symbol at a specific commit from the history DB.

**Example:**

```markdown
Before this PR, `Extract` looked like this:

```codebase-snippet sym=indexer.Extract commit=HEAD~3
```

And here's the current version:

```codebase-snippet sym=indexer.Extract
```
```

**Implementation.** When `commit=` is present, the renderer queries `snapshot_symbols` for that commit hash + symbol ID to get the byte offsets, then reads the file content from `file_contents` (or falls back to `git show`). The rest of the pipeline is unchanged.

## 5. API design — new and extended endpoints

### 5.1 Existing endpoints (unchanged)

These already exist and are used by the new widgets:

| Endpoint | Purpose | Used by |
|----------|---------|---------|
| `GET /api/history/commits` | List all indexed commits | commit-walk, history |
| `GET /api/history/commits/{hash}` | Single commit detail | commit-walk |
| `GET /api/history/commits/{hash}/symbols` | Symbols at a commit | annotation (commit=), snippet (commit=) |
| `GET /api/history/diff?from=X&to=Y` | Full commit diff | changed-files, diff-stats, diff |
| `GET /api/history/symbol-body-diff?from=X&to=Y&symbol=Z` | Per-symbol body diff | diff widget |
| `GET /api/history/symbols/{id}/history` | Symbol timeline | symbol-history |

### 5.2 New endpoint: impact analysis

```
GET /api/history/impact
    ?sym=<symbol-id>        required
    &dir=usedby|uses        required
    &depth=1..5             default 2
    &from=<commit-hash>     optional — enables compatibility check
    &to=<commit-hash>       optional — enables compatibility check

Response:
{
  "root": "sym:github.com/.../indexer.func.Extract",
  "direction": "usedby",
  "depth": 2,
  "nodes": [
    {
      "symbolId": "sym:...handleBuild",
      "name": "handleBuild",
      "kind": "func",
      "depth": 1,
      "edges": [
        { "from": "sym:...handleBuild", "to": "sym:...Extract", "kind": "call" }
      ],
      "compatibility": "review"   // "ok" | "review" | "unknown"
    },
    {
      "symbolId": "sym:...Server.handleIndex",
      "name": "handleIndex",
      "kind": "method",
      "depth": 2,
      "edges": [
        { "from": "sym:...Server.handleIndex", "to": "sym:...handleBuild", "kind": "call" }
      ],
      "compatibility": "unknown"
    }
  ]
}
```

**Implementation** (`internal/server/api_history.go`):

```go
func (s *Server) handleHistoryImpact(w http.ResponseWriter, r *http.Request) {
    symID := r.URL.Query().Get("sym")
    dir := r.URL.Query().Get("dir")      // "usedby" or "uses"
    depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
    if depth < 1 || depth > 5 { depth = 2 }

    from := r.URL.Query().Get("from")
    to   := r.URL.Query().Get("to")

    // BFS over snapshot_refs at the latest commit (or "to" if given).
    // For each discovered node, check if its signature changed
    // between from and to → set compatibility.
    result := impactBFS(r.Context(), s.History, symID, dir, depth, from, to)
    writeJSON(w, result)
}
```

**BFS pseudocode:**

```
function impactBFS(store, root, dir, maxDepth, from, to):
    visited = {root}
    queue = [(root, 0)]
    nodes = []

    while queue is not empty:
        (symID, depth) = queue.dequeue()
        if depth > 0:
            node = {symbolId: symID, depth: depth, edges: [], compatibility: "unknown"}
            if from and to:
                # Check if this symbol's signature changed
                oldSig = querySignature(store, from, symID)
                newSig = querySignature(store, to, symID)
                if oldSig != "" and newSig != "" and oldSig != newSig:
                    node.compatibility = "review"
                else:
                    node.compatibility = "ok"
            nodes.append(node)

        if depth < maxDepth:
            # Get one-hop neighbours from snapshot_refs
            if dir == "usedby":
                refs = query(store, "SELECT from_symbol_id, kind FROM snapshot_refs
                            WHERE to_symbol_id = ? AND commit_hash = ?", symID, latestCommit)
            else:
                refs = query(store, "SELECT to_symbol_id, kind FROM snapshot_refs
                            WHERE from_symbol_id = ? AND commit_hash = ?", symID, latestCommit)

            for ref in refs:
                if ref.target not in visited:
                    visited.add(ref.target)
                    queue.append((ref.target, depth + 1))

    return {root, direction: dir, depth: maxDepth, nodes}
```

### 5.3 Extended endpoint: snippet at a specific commit

```
GET /api/snippet
    ?sym=<symbol-id>        required
    &commit=<commit-hash>   optional (new parameter)

When commit is present:
  1. Look up the symbol in snapshot_symbols for that commit.
  2. Get the file path from snapshot_files.
  3. Read the file content from file_contents cache (or git show).
  4. Slice bytes at [start_offset:end_offset].
  5. Return the snippet.

When commit is absent (existing behaviour):
  Resolve from the static embedded index.json + source FS.
```

**Server-side change** (`internal/server/api_source.go`):

```go
func (s *Server) handleSnippet(w http.ResponseWriter, r *http.Request) {
    symRef := r.URL.Query().Get("sym")
    commit := r.URL.Query().Get("commit")  // NEW

    if commit != "" && s.History != nil {
        // Resolve from history DB
        snippet, err := s.snippetFromHistory(r.Context(), commit, symRef)
        // ... handle error, write response
        return
    }
    // Existing code path: resolve from static index
    // ...
}
```

### 5.4 Route registration

All new routes are added in `internal/server/server.go`:

```go
// In registerHistoryRoutes():
mux.HandleFunc("GET /api/history/impact", s.handleHistoryImpact)

// handleSnippet gets the commit= parameter (no new route needed)
```

## 6. Data flow for embedded widgets

This section traces the full data path from markdown authoring → server-side rendering → client-side hydration → displayed widget. Understanding this flow is essential for implementing new directives correctly.

### 6.1 The full pipeline

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 1. AUTHOR writes markdown                                              │
│                                                                         │
│    internal/docs/embed/pages/my-review.md:                              │
│                                                                         │
│    Here's what changed in Extract:                                     │
│    ```codebase-diff sym=indexer.Extract from=HEAD~3 to=HEAD            │
│    ```                                                                  │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 2. SERVER preprocesses the directive                                   │
│    (internal/docs/renderer.go:preprocess)                              │
│                                                                         │
│    a. Regex matches the fence opener: ```codebase-diff sym=... from=..│
│    b. resolveDirective("codebase-diff sym=... from=... to=...")        │
│       - Validates params (sym, from, to are all present)               │
│       - Looks up the symbol in the static index for the name/kind      │
│       - Does NOT fetch the diff yet (happens client-side)              │
│       - Returns a SnippetRef with metadata                             │
│    c. stubHTML(ref) emits a <div> with data attributes:                │
│       <div class="codebase-snippet" data-codebase-snippet              │
│            data-stub-id="stub-3"                                       │
│            data-directive="codebase-diff"                              │
│            data-sym="sym:...Extract"                                   │
│            data-params='{"from":"HEAD~3","to":"HEAD"}'>                │
│         Plaintext fallback content                                     │
│       </div>                                                           │
│    d. The fenced block is replaced with this <div>.                    │
│    e. Goldmark renders the rest of the markdown normally.              │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 3. BROWSER receives the HTML page                                      │
│                                                                         │
│    The rendered page has:                                              │
│    - Prose paragraphs (normal HTML)                                    │
│    - Stub <div>s for each codebase-* directive                         │
│                                                                         │
│    React mounts and DocSnippet.tsx walks the DOM:                      │
│    a. querySelectorAll('[data-codebase-snippet]')                      │
│    b. For each stub, read data-directive, data-sym, data-params        │
│    c. Switch on data-directive:                                        │
│       "codebase-diff" → createPortal(<SymbolDiffInlineWidget>, stub)  │
│       "codebase-impact" → createPortal(<ImpactInlineWidget>, stub)    │
│       etc.                                                             │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 4. WIDGET fetches data from the API                                    │
│                                                                         │
│    SymbolDiffInlineWidget does:                                        │
│    a. useGetDiffQuery({ from: "HEAD~3", to: "HEAD" })                 │
│       → fetches /api/history/diff?from=HEAD~3&to=HEAD                 │
│    b. useGetSymbolBodyDiffQuery({ from, to, symbol: "sym:...Extract"}) │
│       → fetches /api/history/symbol-body-diff?from=...&to=...&symbol=│
│    c. Renders two code panes with the diff result                      │
│    d. Every identifier in the code is a clickable xref link            │
└─────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Key design decisions in the pipeline

1. **Server resolves metadata, client fetches data.** The server-side `resolveDirective` validates params and emits metadata, but does not make API calls to the history DB. This keeps the doc-rendering path fast and avoids coupling the renderer to the history subsystem. The client-side widget is responsible for fetching its own data.

2. **Params travel as JSON in `data-params`.** This is extensible: adding new parameters to a directive doesn't require changing the renderer's Go code — just the hydration branch in React.

3. **Plaintext fallback is always meaningful.** Even without JavaScript, the reviewer sees a description of what the widget would show. This is important for PDF export, email clients, and accessibility.

## 7. Implementation plan

### Phase 1: Core directive extensions (3–4 days)

1. **Extend `resolveDirective` in `internal/docs/renderer.go`** with `case "codebase-diff"`, `case "codebase-symbol-history"`, `case "codebase-changed-files"`, `case "codebase-diff-stats"`. Each validates required params and emits a stub with `data-params` JSON.

2. **Add `commit=` parameter to existing directives.** In `resolveDirective`, when `params["commit"]` is set, resolve the symbol from the history DB instead of the static index. This requires passing the `*history.Store` into the renderer.

3. **Write the fallback text generators.** Each new directive produces a useful plaintext fallback.

### Phase 2: API additions (2 days)

4. **Implement `/api/history/impact`** in `internal/server/api_history.go`. BFS over `snapshot_refs` with compatibility checking.

5. **Extend `/api/snippet` with `commit=` parameter.** Resolve from history DB when the param is present.

### Phase 3: React widgets (5–6 days)

6. **`<SymbolDiffInlineWidget>`** — the diff widget. Fetches body-diff API, renders two `<Code>` panes. New file: `ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx`.

7. **`<SymbolHistoryInlineWidget>`** — the timeline widget. Fetches symbol history API, renders compact commit list. New file: `ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx`.

8. **`<ImpactInlineWidget>`** — the impact widget. Fetches impact API, renders grouped caller list. New file: `ui/src/features/doc/widgets/ImpactInlineWidget.tsx`.

9. **`<ChangedFilesWidget>`** — file summary. New file: `ui/src/features/doc/widgets/ChangedFilesWidget.tsx`.

10. **`<DiffStatsWidget>`** — inline stats. New file: `ui/src/features/doc/widgets/DiffStatsWidget.tsx`.

11. **`<AnnotationWidget>`** — annotated code. New file: `ui/src/features/doc/widgets/AnnotationWidget.tsx`.

12. **`<CommitWalkWidget>`** — the guided walk. Composes the other widgets. New file: `ui/src/features/doc/widgets/CommitWalkWidget.tsx`.

13. **Extend `DocSnippet.tsx`** with hydration branches for each new directive.

### Phase 4: Polish and testing (2–3 days)

14. **Storybook stories** for each widget with mocked API data.

15. **E2E test**: render a doc page with all widget types, verify data fetches and display.

16. **Accessibility pass**: keyboard navigation, ARIA labels, screen reader testing.

17. **Documentation**: update `internal/docs/embed/pages/` with a worked example page demonstrating all widgets.

**Total estimate: 12–15 days.**

## 8. Risks, alternatives, and open questions

### 8.1 Risks

1. **History DB not available.** The widgets depend on the history subsystem being populated (`codebase-browser history scan ...`). If a doc page uses a `codebase-diff` directive but the server was started without `--history-db`, the widget should degrade gracefully — show an error message like "History data not available. Start the server with `--history-db history.db`."

2. **Large diffs overwhelm inline rendering.** A symbol that grew by 200 lines creates a very tall widget. Mitigation: add a `maxHeight` parameter (default 400px) with scroll, and a "show full" expander.

3. **Ref parameter resolution.** The `from=HEAD~3` syntax requires resolving git refs to commit hashes. The server needs git ref resolution at render time (or the client needs to resolve them via an API). Mitigation: add a `/api/history/resolve-ref?ref=HEAD~3` endpoint, or require authors to use full commit hashes.

4. **Performance of impact BFS on large repos.** For repos with tens of thousands of refs, depth-3 BFS could be slow. Mitigation: cache impact results per (symbol, depth) pair; set a query timeout.

### 8.2 Alternatives considered

1. **Server-side rendering of widgets to HTML.** Instead of client-side hydration, the server could render the full widget HTML at doc-render time. This would work without JavaScript but couples the renderer to the history DB and makes the response slower. **Rejected** for the primary path; kept as a future export mode (PDF, email).

2. **Using GitHub's PR API as the data source.** Instead of our own history DB, pull diff data from GitHub. **Rejected**: ties us to one forge, doesn't work for local-only repos, and GitHub's API doesn't have semantic symbol data.

3. **Iframe embedding.** Each widget as a separate page loaded in an iframe. **Rejected**: breaks xref navigation (clicking a link inside an iframe doesn't navigate the parent), no shared state, worse accessibility.

### 8.3 Open questions

1. **Should the commit-walk widget support a markdown-first syntax?** The current DSL (`step "..." Show ...`) is custom. An alternative is to allow multiple fenced blocks with interstitial prose, which is more natural in markdown but harder to parse. Recommend the DSL for phase 1, markdown-native syntax as a phase 2 enhancement.

2. **How to handle symbols that don't exist at a given commit?** If the author writes `codebase-diff sym=indexer.Extract from=HEAD~10 to=HEAD` but `Extract` didn't exist 10 commits ago, the "before" pane should show a clear "symbol not present at this commit" message rather than an error.

3. **Should the annotation widget support range-based annotations across files?** Currently scoped to a single symbol. Cross-file annotations (e.g. "this type is used over there") could be handled by hyperlinks to other widgets.

4. **Colour scheme for diff highlights in dark mode.** The standard red/green diff colours have poor contrast in dark mode. Need accessible alternatives.

## 9. File reference index

This section lists every file that needs to be created or modified, with a one-line description of the change.

### New files

| File | Purpose |
|------|---------|
| `ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx` | Side-by-side symbol body diff widget |
| `ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx` | Compact commit timeline widget |
| `ui/src/features/doc/widgets/ImpactInlineWidget.tsx` | Transitive caller/callee impact widget |
| `ui/src/features/doc/widgets/ChangedFilesWidget.tsx` | File-level diff summary widget |
| `ui/src/features/doc/widgets/DiffStatsWidget.tsx` | Inline numeric change summary widget |
| `ui/src/features/doc/widgets/AnnotationWidget.tsx` | Annotated code snippet widget |
| `ui/src/features/doc/widgets/CommitWalkWidget.tsx` | Guided multi-step commit walk widget |
| `ui/src/features/doc/widgets/index.ts` | Barrel export for all widgets |
| `ui/src/features/doc/stories/SymbolDiff.stories.tsx` | Storybook story |
| `ui/src/features/doc/stories/SymbolHistory.stories.tsx` | Storybook story |
| `ui/src/features/doc/stories/Impact.stories.tsx` | Storybook story |
| `ui/src/features/doc/stories/CommitWalk.stories.tsx` | Storybook story |
| `internal/docs/embed/pages/04-review-widgets.md` | Demo page showing all new widgets |

### Modified files

| File | Change |
|------|--------|
| `internal/docs/renderer.go` | Add `resolveDirective` cases for 7 new directives + `commit=` param support |
| `internal/server/api_history.go` | Add `handleHistoryImpact` handler + extend `registerHistoryRoutes` |
| `internal/server/api_source.go` | Extend `handleSnippet` with `commit=` parameter |
| `internal/server/server.go` | Register `/api/history/impact` route |
| `ui/src/features/doc/DocSnippet.tsx` | Add hydration branches for 7 new directives |
| `ui/src/api/historyApi.ts` | Add `useGetImpactQuery` hook |

## 10. Summary

The codebase-browser already has the three pillars needed for literate PR review:

1. **Semantic indexing** — every function, type, and cross-reference is captured with stable IDs.
2. **Git history** — per-commit snapshots in SQLite let us diff any symbol across any two commits.
3. **Embeddable directives** — the `codebase-*` pipeline lets us embed live widgets in markdown.

This ticket connects them. Seven new directives and their backing widgets turn a markdown document into an interactive code review workspace. The reviewer navigates the source, explores diffs, traces impact, and walks through commits — all without leaving the page.

The implementation is incremental: each widget is independent, each builds on existing components, and each degrades gracefully when the history DB is unavailable. The commit-walk widget is the capstone: it composes all the others into a guided narrative that the PR author writes and the reviewer follows step by step.

## 11. Incremental implementation — vertical slices

The phased plan in §7 organises work by layer (directives → APIs → widgets). That's fine for a small team that already knows exactly what to build. But when exploring a new surface — embedding history-backed widgets into markdown — it's better to build **vertical slices**, where each slice delivers one complete, demonstrable feature end-to-end. Each slice touches the renderer, the API, and the React component, so you see a working result immediately and can course-correct before investing in the next widget.

The dependency chain is:

```
Slice 0  ──►  Slice 1  ──►  Slice 2  ──►  Slice 3  ──►  Slice 4  ──►  Slice 5
(commit=     (diff        (symbol       (impact       (annotation   (commit
 on existing  widget)      history       widget)       + stats +     walk:
 directives)               widget)                      files)        capstone)
```

### Slice 0: "Snapshot at a commit" — `commit=` on existing directives

**Time: ~1 day**

**What you can demo:** A markdown doc page that shows the same function at two different commits, side-by-side, as two separate `codebase-snippet` blocks. Not a diff widget yet — just two live code blocks with xref links, each resolved from a different commit.

**Authoring:**

```markdown
Before this PR, `Extract` looked like this:

```codebase-snippet sym=indexer.Extract commit=HEAD~3
```

And now:

```codebase-snippet sym=indexer.Extract
```
```

**Changes needed:**

1. **`internal/docs/renderer.go`** — In `resolveDirective`, when `params["commit"]` is present, don't try to resolve the symbol from the static index. Instead, emit a stub with an extra `data-commit="<hash>"` attribute. The server doesn't need to fetch the content — it just validates the param format and passes it through.

2. **`internal/server/api_source.go`** — Extend `handleSnippet` with a `commit` query parameter. When present, look up the symbol in `snapshot_symbols` for that commit hash, get byte offsets, read file content from `file_contents` (or `git show` fallback), slice, and return.

3. **`ui/src/features/doc/DocSnippet.tsx`** — When the stub has `data-commit`, pass it through to the snippet fetch so the RTK-Query hook includes `&commit=<hash>`.

4. **`internal/docs/embed/pages/04-review-slice0.md`** — A demo page showing a before/after pair.

**Files touched:** `renderer.go`, `api_source.go`, `DocSnippet.tsx`, `historyApi.ts` (minor), new demo page.

**Validation:** Open the demo page. Both snippets render with correct code for their respective commits. Xref links work. No history DB shows a clear error message.

**Why this slice first:** It validates the core plumbing — passing history-aware params through the directive pipeline — with zero new widgets. If `commit=` resolution works here, every subsequent widget inherits the same mechanism.

---

### Slice 1: "The diff widget" — `codebase-diff`

**Time: ~2–3 days**

**What you can demo:** A markdown doc page with a single `codebase-diff` block that renders a live side-by-side diff of one function between two commits. This is the single most important widget — once this works, you know the whole pipeline is viable.

**Authoring:**

```markdown
Here's what changed in `Extract`:

```codebase-diff sym=indexer.Extract from=HEAD~3 to=HEAD
```

The function gained a `strict bool` parameter.
```

**Changes needed:**

1. **`internal/docs/renderer.go`** — Add `case "codebase-diff"` to `resolveDirective`. Validates `sym`, `from`, `to` params. Emits a stub with `data-directive="codebase-diff"` and `data-params='{"from":"HEAD~3","to":"HEAD"}'`. Does NOT fetch the diff server-side — just emits metadata.

2. **`ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx`** — New file. The widget:
   - Reads `from`, `to`, `sym` from the stub's `data-params`.
   - Calls `useGetSymbolBodyDiffQuery({ from, to, symbol: sym })` (existing API hook).
   - Renders two `<pre>` blocks side-by-side with diff highlighting.
   - Reuses the line-colouring logic from `HistoryPage.tsx` (red/green/grey).

3. **`ui/src/features/doc/DocSnippet.tsx`** — Add `if (directive === 'codebase-diff') return <SymbolDiffInlineWidget .../>`.

4. **`ui/src/features/doc/DocPage.tsx`** — When reading stubs, also extract `data-params` and pass it through to `DocSnippet`.

5. **`internal/docs/embed/pages/04-review-slice1.md`** — Demo page with one or two diff blocks.

**Files touched:** `renderer.go`, `DocSnippet.tsx`, `DocPage.tsx`, new widget file, new demo page.

**Key design decision for this slice:** The widget fetches data client-side, not server-side. The renderer just validates and emits metadata. This keeps the doc-rendering path fast and uncoupled from the history DB.

**Validation:** Open the demo page. The diff widget loads and shows the before/after of the function with coloured lines. Expanding to full symbol view works. If the history DB is missing, shows a clear "connect history DB" message.

**Why this is the second slice:** This is the hardest widget to get right. If we can render a clean inline diff, the rest are easier variations. Better to validate the hard case early.

---

### Slice 2: "The history timeline" — `codebase-symbol-history`

**Time: ~1–2 days**

**What you can demo:** A compact commit timeline inline in a doc page. Shows when a function was introduced, how many times it changed, and by whom.

**Authoring:**

```markdown
The merge logic has been iterated several times:

```codebase-symbol-history sym=indexer.Merge limit=5
```
```

**Changes needed:**

1. **`internal/docs/renderer.go`** — Add `case "codebase-symbol-history"`. Validates `sym`. Emits stub with `data-params='{"limit":5}'`.

2. **`ui/src/features/doc/widgets/SymbolHistoryInlineWidget.tsx`** — New file.
   - Calls `useGetSymbolHistoryQuery({ symbolId, limit })` (existing API).
   - Renders a compact vertical list: date · author · commit message · body-hash indicator (● or ○).
   - Click a row to expand a mini body-diff between that commit and the previous one.

3. **`ui/src/features/doc/DocSnippet.tsx`** — Add `if (directive === 'codebase-symbol-history') return ...`.

4. Update demo page with a history block.

**Files touched:** `renderer.go`, `DocSnippet.tsx`, new widget file, demo page update.

**Validation:** The timeline renders inline with correct commit data. Body-hash dots are correct (filled for commits where the body changed). Clicking a row expands a mini-diff.

**Why this slice:** It reuses an existing API (`/api/history/symbols/{id}/history`) so it's fast to build. It's the second-most-useful widget for review (after the diff). And it validates that the existing history API is sufficient for inline rendering — if we need to extend it, we find out now.

---

### Slice 3: "Impact analysis" — `codebase-impact`

**Time: ~2–3 days**

**What you can demo:** An inline caller/callee list showing who's affected by a change, grouped by depth, with compatibility indicators.

**Authoring:**

```markdown
Changing `Extract` affects these callers:

```codebase-impact sym=indexer.Extract dir=usedby depth=2
```
```

**Changes needed:**

1. **`internal/server/api_history.go`** — New handler `handleHistoryImpact`. BFS over `snapshot_refs`. This is the first truly new API endpoint. ~80 lines of Go.

2. **`internal/server/server.go`** — Register `GET /api/history/impact`.

3. **`ui/src/api/historyApi.ts`** — Add `useGetImpactQuery` hook.

4. **`internal/docs/renderer.go`** — Add `case "codebase-impact"`. Validates `sym`, `dir`, `depth`.

5. **`ui/src/features/doc/widgets/ImpactInlineWidget.tsx`** — New file. Grouped list of callers at each depth, with ✓/⚠ indicators.

6. **`ui/src/features/doc/DocSnippet.tsx`** — Add dispatch.

7. Update demo page.

**Files touched:** `api_history.go`, `server.go`, `historyApi.ts`, `renderer.go`, `DocSnippet.tsx`, new widget file, demo page.

**Validation:** The impact widget renders with correct callers at depth 1 and 2. Compatibility indicators show correctly for symbols whose signatures changed. Performance is acceptable on the current repo (~960 refs).

**Why this slice:** This is the first slice that requires a new server endpoint. Building it now means we validate the BFS approach and can tune performance before the more complex widgets that compose on top of it.

---

### Slice 4: "Quick wins" — `codebase-annotation`, `codebase-changed-files`, `codebase-diff-stats`

**Time: ~1–2 days**

**What you can demo:** Three simpler widgets that round out the set:
- An annotated code block with author-highlighted lines
- A file-level change summary
- A one-line numeric diff stat

These are individually simple — each is half a day once the patterns from slices 1–3 are established.

**Changes needed:**

1. **`internal/docs/renderer.go`** — Add three new `case` branches. Each is ~15 lines.

2. **Three new React components** in `ui/src/features/doc/widgets/`:
   - `AnnotationWidget.tsx` — wraps `<Code>` with highlight overlay
   - `ChangedFilesWidget.tsx` — table of changed files with line counts
   - `DiffStatsWidget.tsx` — inline `<span>` with coloured badges

3. **`ui/src/features/doc/DocSnippet.tsx`** — Three more dispatch branches.

4. Update demo page with all three.

**Validation:** Each widget renders correctly in the demo page. The annotation highlights the right lines. The changed-files list matches `git diff --stat`. The diff stats match the `/api/history/diff` response.

**Why this slice:** These are low-risk, high-value polish. After the hard work of slices 1–3, these three round out the catalog and make the demo page feel complete. They also validate that adding new directives is now a routine, ~30-minute operation — proving the architecture is extensible.

---

### Slice 5: "The guided walk" — `codebase-commit-walk`

**Time: ~3–4 days**

**What you can demo:** A multi-step guided walk through a series of commits, with narrative prose and embedded sub-widgets at each step. This is the capstone — the "literate PR review guide" in its fullest form.

**Why last:** This widget composes all the others. It only makes sense to build it once the diff, history, and impact widgets are validated and working. It's also the most complex widget in terms of internal state (step navigation, sub-widget lifecycle).

**Changes needed:**

1. **`internal/docs/renderer.go`** — Add `case "codebase-commit-walk"`. Parse the `step` sub-directives from the block body. Emit a stub with all steps serialised as JSON in `data-params`.

2. **`ui/src/features/doc/widgets/CommitWalkWidget.tsx`** — New file. Internal state tracks current step index. Each step renders the appropriate sub-widget (`<SymbolDiffInlineWidget>`, `<ImpactInlineWidget>`, etc.) plus the author's narrative note.

3. **`ui/src/features/doc/DocSnippet.tsx`** — Add dispatch.

4. **Demo page** — A full literate PR review guide using the commit-walk widget.

**Validation:** Walk through the steps. Each step renders the correct sub-widget. Navigation (prev/next) works. The narrative prose appears between widgets. The whole thing reads as a coherent review guide.

---

### Slice summary and validation checklist

| Slice | Widget | Time | New API? | Demo deliverable |
|-------|--------|------|----------|------------------|
| 0 | `commit=` on existing | 1 day | No | Before/after code blocks |
| 1 | `codebase-diff` | 2–3 days | No | Inline side-by-side diff |
| 2 | `codebase-symbol-history` | 1–2 days | No | Compact commit timeline |
| 3 | `codebase-impact` | 2–3 days | **Yes** (BFS) | Caller list with ⚠ indicators |
| 4 | annotation + files + stats | 1–2 days | No | Three simple widgets |
| 5 | `codebase-commit-walk` | 3–4 days | No | Full guided review walk |
| | **Total** | **10–15 days** | | |

**Validation after each slice:**

1. Write (or update) a demo markdown page at `internal/docs/embed/pages/04-review-widgets.md`.
2. Run `make dev-backend` + `make dev-frontend`.
3. Open the demo page in the browser.
4. Verify the new widget renders with real data from the history DB.
5. Check that existing widgets still work (no regressions).
6. If the result looks wrong or the UX is off, adjust before moving to the next slice.

**Rollback safety:** Each slice adds new files and extends one `switch` statement in the renderer. No slice modifies existing widget code. If a slice goes wrong, revert the new files and the `switch` additions — existing functionality is untouched.

**Decision gates:** After slices 1 and 3, pause and evaluate:
- After slice 1: Is the inline diff useful? Is side-by-side the right layout, or should we try inline (unified) diff first?
- After slice 3: Is the impact BFS fast enough? Is depth=2 the right default? Do the compatibility indicators help or clutter?

These gates prevent over-investing in a direction that doesn't work for reviewers.
