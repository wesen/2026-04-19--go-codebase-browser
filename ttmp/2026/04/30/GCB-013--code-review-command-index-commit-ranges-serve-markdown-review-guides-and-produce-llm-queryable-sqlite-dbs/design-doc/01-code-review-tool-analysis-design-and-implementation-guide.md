---
Title: Code Review Tool — Analysis, Design and Implementation Guide
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
    - intern-onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/history/scan.go
      Note: Commit range scanner — reused by review index
    - Path: cmd/codebase-browser/cmds/index/build.go
      Note: Index builder — the model for review db creation
    - Path: cmd/codebase-browser/cmds/serve/run.go
      Note: |-
        Existing serve command — the model for review serve
        Existing serve command — model for review serve implementation
    - Path: cmd/codebase-browser/main.go
      Note: |-
        Root command registration — where the new review command tree attaches
        Root command registration point for new review command tree
    - Path: internal/browser/index.go
      Note: Loaded index — symbol/file/package lookup maps
    - Path: internal/docs/pages.go
      Note: Doc page listing and metadata
    - Path: internal/docs/renderer.go
      Note: |-
        Markdown directive pipeline — codebase-* fenced blocks → HTML stubs
        Markdown directive pipeline — core rendering engine for review guides
    - Path: internal/gitutil/log.go
      Note: Git log parsing — commit range resolution
    - Path: internal/history/indexer.go
      Note: Per-commit indexing pipeline — git worktrees + AST extraction
    - Path: internal/history/schema.go
      Note: |-
        History SQLite schema — commits, snapshot_symbols, snapshot_refs tables
        History SQLite schema — basis for review DB schema extensions
    - Path: internal/history/store.go
      Note: |-
        History store — Open, Create, ResetSchema, DB methods
        History store — needs NewFromDB constructor for review store reuse
    - Path: internal/indexer/id.go
      Note: Stable symbol ID generation
    - Path: internal/indexer/types.go
      Note: Canonical Index, Symbol, Ref, Range types
    - Path: internal/server/api_doc.go
      Note: Doc API handlers — /api/doc list and page rendering
    - Path: internal/server/server.go
      Note: HTTP server — route registration pattern for new review endpoints
    - Path: internal/sqlite/schema.go
      Note: Codebase SQLite schema — packages, files, symbols, refs tables
    - Path: ui/src/features/doc/DocPage.tsx
      Note: React doc page — renders markdown HTML + hydrates widgets
    - Path: ui/src/features/doc/DocSnippet.tsx
      Note: Widget hydration dispatcher — mounts React components into stubs
ExternalSources: []
Summary: A comprehensive design guide for turning codebase-browser into a standalone code review tool. Covers the new `review` CLI command (index, serve, db-create), the SQLite schema extensions for review documents, the markdown rendering pipeline for non-embedded docs, and the Glazed help entry documentation system. Written for a new intern — every subsystem is explained from first principles with prose, diagrams, pseudocode, API contracts, and file references.
LastUpdated: 2026-04-30T12:00:00Z
WhatFor: Guide the implementation of the code-review command and onboard new team members to the codebase-browser architecture
WhenToUse: Read before implementing any part of GCB-013, or when joining the team and needing to understand how the review tool works
---






# Code Review Tool — Analysis, Design and Implementation Guide

## 1. Executive summary

This document describes how to extend the `codebase-browser` into a **standalone code review tool** that can be pointed at a git commit range and a set of markdown files, index both into a SQLite database, and serve interactive review guides — or simply produce the database for consumption by an LLM.

The codebase-browser already has three powerful subsystems:
1. A **semantic indexer** that extracts every symbol, file, package, and cross-reference from Go and TypeScript source code (`internal/indexer/`).
2. A **git history subsystem** that stores per-commit symbol snapshots in SQLite, enabling queries like "how did this function change between commit A and B?" (`internal/history/`).
3. A **markdown directive pipeline** that turns fenced code blocks like ` ```codebase-diff sym=...``` ` into interactive React widgets (`internal/docs/`).

What is missing is a **unified CLI workflow** that ties these together for the code review use case. Today, a reviewer must:
- Run `codebase-browser history scan` to discover commits
- Run `codebase-browser history index` to build a history.db
- Write markdown files manually and hope the paths line up
- Run `codebase-browser serve` with the right `--history-db` flag
- Manually manage which markdown files correspond to which review

This ticket introduces a new `review` command tree:
- `codebase-browser review index --commits RANGE --docs DIR --db FILE` — index commits and markdown docs into a single review database
- `codebase-browser review serve --db FILE --addr :PORT` — serve the review docs with full widget support
- `codebase-browser review db create --commits RANGE --db FILE` — create just the SQLite DB, no docs, for LLM querying

Alongside the implementation, we need **Glazed help entries** — structured markdown documentation with frontmatter — that serve as both a reference manual and a user guide, discoverable via `codebase-browser help <topic>`.

This document is written for a **new intern**. Every concept is explained from first principles. Read it top to bottom before touching any code.

## 2. Problem statement and scope

### 2.1 The problem

Code review today is fragmented. The PR author writes a description on GitHub. The reviewer opens "Files changed," reads line-level diffs, and mentally reconstructs which functions changed, which types were affected, and whether callers need updates. If the reviewer wants deeper context — "who calls this function?" or "how did this evolve?" — they switch to a terminal or IDE.

The codebase-browser has the data to answer all of these questions semantically. What it lacks is:
1. **A single artifact per review** — one database file that contains both the indexed commits and the review narrative.
2. **A CLI workflow for review authors** — a simple command to index a commit range and a set of markdown files into that artifact.
3. **A serve mode for review consumers** — a lightweight server that renders the markdown files with all widgets live.
4. **A DB-only mode for LLMs** — a way to produce just the structured database so an LLM can query it.
5. **Documentation** — reference and user guide help entries in the Glazed format so users can discover commands and concepts without reading the source.

### 2.2 The vision

A PR author writes a review guide as a markdown file:

```markdown
# PR #42: Add strict mode to Extract

## Motivation
The `Extract` function needs to support build tag filtering.

## Changes

### 1. New parameter
```codebase-diff sym=indexer.Extract from=HEAD~1 to=HEAD
```

### 2. Updated callers
```codebase-impact sym=indexer.Extract dir=usedby depth=2
```
```

They run:

```bash
codebase-browser review index \
  --commits HEAD~5..HEAD \
  --docs ./reviews/pr-42.md \
  --db ./reviews/pr-42.db
```

This produces `pr-42.db`, a single SQLite file containing:
- The 5 commits, their metadata, and per-commit symbol snapshots
- The markdown document content and parsed metadata
- Resolved snippet references (which symbols are embedded where)

The author then runs:

```bash
codebase-browser review serve --db ./reviews/pr-42.db --addr :3002
```

And shares `http://localhost:3002` with reviewers. The reviewers see the markdown rendered with live widgets — diff panes, impact analysis, symbol history — all backed by the data in `pr-42.db`.

Alternatively, the author can produce just the DB for an LLM:

```bash
codebase-browser review db create --commits HEAD~5..HEAD --db pr-42.db
```

And then prompt the LLM: "Query the `snapshot_symbols` and `commits` tables in `pr-42.db` to tell me which functions changed their signatures in this PR."

### 2.3 Scope

**In scope for this ticket:**
- The `review` command tree with three subcommands: `index`, `serve`, and `db create`
- SQLite schema extensions for storing review documents and their metadata
- Integration with the existing markdown directive pipeline for non-embedded (on-disk) docs
- A serve mode that reads docs from the review DB or from disk, backed by the history DB
- Glazed help entries: one reference guide (concepts and data model) and one user guide (commands and workflows)
- The Go wiring to embed help entries into the CLI

**Out of scope:**
- Auto-generating review guides from PR diffs (future work)
- Multi-repository reviews
- Real-time collaborative annotations or comments
- GitHub API integration
- PDF export of review guides

## 3. Current-state architecture

Before we design the new feature, you must understand the existing system. This section is a complete tour of every subsystem the review tool builds on. If you are new to the project, read this carefully.

### 3.1 The codebase-browser at a glance

The codebase-browser is a single-binary documentation server for Go and TypeScript codebases. It is built in Go 1.22+ with a React SPA frontend. The binary embeds its own index, source tree, and web assets — no runtime dependencies.

The repository lives at `/home/manuel/code/wesen/2026-04-19--go-codebase-browser`. Key directories:

| Directory | Purpose |
|-----------|---------|
| `cmd/codebase-browser/` | Main CLI entry point and command definitions |
| `internal/indexer/` | Go AST extractor and canonical index types |
| `internal/browser/` | Index loader with lookup maps |
| `internal/server/` | HTTP API handlers and route registration |
| `internal/docs/` | Markdown rendering pipeline with codebase-* directives |
| `internal/history/` | Git-aware per-commit indexing and SQLite store |
| `internal/sqlite/` | Structured query SQLite store for concepts |
| `internal/sourcefs/` | Source tree embedding (for snippet slicing) |
| `internal/web/` | SPA asset embedding |
| `ui/` | React SPA (Vite + RTK-Query) |
| `tools/ts-indexer/` | TypeScript extractor (Node + TS Compiler API) |

### 3.2 The CLI root and command registration

Open `cmd/codebase-browser/main.go`:

```go
var rootCmd = &cobra.Command{
    Use:     "codebase-browser",
    Short:   "Index, query, and serve the Go source of this very binary",
    Version: version,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return logging.InitLoggerFromCobra(cmd)
    },
}

func main() {
    cobra.CheckErr(logging.AddLoggingSectionToRootCommand(rootCmd, "codebase-browser"))

    helpSystem := help.NewHelpSystem()
    help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

    cobra.CheckErr(index.Register(rootCmd))
    cobra.CheckErr(symbol.Register(rootCmd))
    cobra.CheckErr(query.Register(rootCmd, conceptRepositoryFlags))
    cobra.CheckErr(doc.Register(rootCmd))
    cobra.CheckErr(history.Register(rootCmd))
    registerServe(rootCmd)
    // ...
}
```

This is the **root command**. Every subcommand (`index`, `symbol`, `query`, `doc`, `history`, `serve`) registers itself by calling a `Register(*cobra.Command)` function. The `helpSystem` is a Glazed construct — it loads markdown help entries with frontmatter and makes them queryable via `codebase-browser help <slug>`.

**Your task:** add `review.Register(rootCmd)` after the existing registrations.

### 3.3 The semantic index

The index is the heart of the system. It is a JSON file (`index.json`) that contains every symbol, file, package, and cross-reference extracted from the source code.

The canonical types live in `internal/indexer/types.go`:

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

type Symbol struct {
    ID          string
    Kind        string   // "func", "type", "var", "const", "method", ...
    Name        string
    PackageID   string
    FileID      string
    Range       Range    // byte offsets + line/column positions
    Doc         string   // godoc / tsdoc comment
    Signature   string   // e.g. "func Merge(base, extra *Index) (*Index, error)"
    Receiver    *Receiver
    Exported    bool
    Language    string   // "go" or "ts"
    // ...
}

type Ref struct {
    FromSymbolID string
    ToSymbolID   string
    Kind         string   // "call", "uses-type", "reads", "use"
    FileID       string
    Range        Range
}
```

**Symbol IDs are stable across file moves.** The scheme in `internal/indexer/id.go` is:

```
sym:<importPath>.<kind>.<name>              # top-level declaration
sym:<importPath>.method.<Recv>.<name>       # method
```

This stability is what makes cross-commit diff meaningful. If `func Extract` moves from `extractor.go` to `builder.go`, its ID does not change.

**Byte offsets are authoritative.** Every `Symbol.Range` contains `StartOffset` and `EndOffset`. When the renderer needs to show a snippet, it reads the source file and slices `data[StartOffset:EndOffset]`. This is exact — no line-counting heuristics.

The index is built by `cmd/codebase-browser/cmds/index/build.go`. The `BuildCommand` uses Glazed's command description system:

```go
desc := cmds.NewCommandDescription(
    "build",
    cmds.WithShort("Build index.json from Go (and optionally TypeScript) source"),
    cmds.WithFlags(
        fields.New("module-root", fields.TypeString, ...),
        fields.New("patterns", fields.TypeStringList, ...),
        // ...
    ),
)
```

### 3.4 The markdown directive pipeline

The doc rendering pipeline lives in `internal/docs/renderer.go`. It is a **two-pass system**:

**Pass 1: preprocess.** Scan the markdown for fenced code blocks whose info string starts with `codebase-`. Parse the directive and parameters. Resolve the symbol or file. Emit an HTML `<div>` stub with data attributes.

**Pass 2: render.** Feed the preprocessed markdown through `goldmark`, a Go markdown-to-HTML converter.

Example directive:

````markdown
```codebase-snippet sym=indexer.Merge
```
````

This resolves to:

```html
<div class="codebase-snippet" data-codebase-snippet
     data-stub-id="stub-1" data-sym="sym:github.com/.../indexer.func.Merge"
     data-directive="codebase-snippet" data-kind="func" data-lang="go">
  <pre><code class="language-go">func Merge(base, extra *Index) (*Index, error) { ... }</code></pre>
</div>
```

The React frontend (`ui/src/features/doc/DocSnippet.tsx`) walks the rendered HTML after mount, finds these stubs, and uses `createPortal` to mount rich interactive widgets in their place.

**This stub + hydrate pipeline is the extension point for new widgets.** Adding a new directive means:
1. Add a `case` in `resolveDirective()` in `internal/docs/renderer.go`
2. Emit a stub with new `data-directive` and `data-params`
3. Add a hydration branch in `DocSnippet.tsx`

Currently supported directives (as of GCB-010):
- `codebase-snippet` — full symbol body
- `codebase-signature` — just the signature line
- `codebase-doc` — godoc/tsdoc comment
- `codebase-file` — whole or partial file contents
- `codebase-diff` — side-by-side symbol body diff between two commits
- `codebase-symbol-history` — compact timeline of commits that touched a symbol
- `codebase-impact` — transitive caller/callee list
- `codebase-commit-walk` — guided narrative through commits
- `codebase-annotation` — inline code annotation with highlights and notes
- `codebase-changed-files` — file-level diff summary
- `codebase-diff-stats` — compact numeric summary

### 3.5 The git history subsystem (GCB-009)

The history subsystem tracks symbol locations across commits. It stores per-commit snapshots in SQLite.

**Schema** (`internal/history/schema.go`):

```sql
CREATE TABLE commits (
    hash TEXT PRIMARY KEY,
    short_hash TEXT NOT NULL,
    message TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_time INTEGER NOT NULL,
    parent_hashes TEXT NOT NULL DEFAULT '[]',
    tree_hash TEXT NOT NULL DEFAULT '',
    indexed_at INTEGER NOT NULL DEFAULT 0,
    branch TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE snapshot_symbols (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id TEXT NOT NULL,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    package_id TEXT NOT NULL,
    file_id TEXT NOT NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_col INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_col INTEGER NOT NULL DEFAULT 0,
    start_offset INTEGER NOT NULL DEFAULT 0,
    end_offset INTEGER NOT NULL DEFAULT 0,
    doc TEXT NOT NULL DEFAULT '',
    signature TEXT NOT NULL DEFAULT '',
    receiver_type TEXT NOT NULL DEFAULT '',
    receiver_pointer INTEGER NOT NULL DEFAULT 0,
    exported INTEGER NOT NULL DEFAULT 0,
    language TEXT NOT NULL DEFAULT 'go',
    type_params_json TEXT NOT NULL DEFAULT '[]',
    tags_json TEXT NOT NULL DEFAULT '[]',
    body_hash TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE snapshot_files (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id TEXT NOT NULL,
    path TEXT NOT NULL,
    package_id TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    line_count INTEGER NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'go',
    build_tags_json TEXT NOT NULL DEFAULT '[]',
    content_hash TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE snapshot_refs (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id INTEGER NOT NULL,
    from_symbol_id TEXT NOT NULL,
    to_symbol_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    file_id TEXT NOT NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_col INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_col INTEGER NOT NULL DEFAULT 0,
    start_offset INTEGER NOT NULL DEFAULT 0,
    end_offset INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE file_contents (
    content_hash TEXT PRIMARY KEY,
    content BLOB NOT NULL
);
```

**Key concept: `body_hash`.** This is a SHA-256 of the function body bytes at each commit. When the same `sym:...` ID appears in two commits with different `body_hash` values, you know the function changed — without comparing the full source text.

**Indexing pipeline** (`internal/history/indexer.go`):

```
for each commit in range:
    1. git worktree add /tmp/cb-<sha> <sha>
    2. Run Go/TS extractor on the worktree
    3. Insert commit metadata into commits table
    4. Insert symbols into snapshot_symbols
    5. Insert files into snapshot_files
    6. Insert refs into snapshot_refs
    7. Cache file contents into file_contents (deduped by SHA-256)
    8. git worktree remove /tmp/cb-<sha>
```

The `Store` type (`internal/history/store.go`) provides `Open`, `Create`, `ResetSchema`, `HasCommit`, `GetCommit`, `ListCommits`, and `DB()` for direct queries.

### 3.6 The HTTP server

The server lives in `internal/server/server.go`. It uses Go 1.22+'s `http.ServeMux` with path patterns:

```go
func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    s.mux = mux

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
    mux.HandleFunc("/api/query-concepts", s.handleConceptList)
    mux.HandleFunc("/api/query-concepts/", s.handleConceptSubtree)

    s.registerHistoryRoutes()

    mux.Handle("/", s.spaHandler())
    return withCommonHeaders(mux)
}
```

The `Server` struct holds:
- `Loaded *browser.Loaded` — the static index
- `SourceFS fs.FS` — source tree for snippet slicing
- `SPAFS fs.FS` — React SPA assets
- `SQLite *cbsqlite.Store` — structured query concepts
- `ConceptCatalog *concepts.Catalog` — query concept definitions
- `History *history.Store` — optional git history DB
- `RepoRoot string` — path to git repo for reading file contents at commits

### 3.7 The existing serve command

`cmd/codebase-browser/cmds/serve/run.go` defines the `serve` command. It:
1. Loads the embedded `index.json`
2. Opens the SQLite concept store (`--db`)
3. Optionally opens the history DB (`--history-db`)
4. Constructs a `server.Server` and starts an HTTP server

This is the model for `review serve`, but with two differences:
- `review serve` reads its docs from the review DB or from disk, not from `internal/docs/embed/pages/`
- `review serve` always has a history DB (it's part of the review DB)

### 3.8 Glazed help system

The codebase-browser already uses Glazed's help system. In `main.go`:

```go
helpSystem := help.NewHelpSystem()
help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
```

This wires `codebase-browser help <slug>` to look up markdown files with Glazed frontmatter. The frontmatter format is:

```yaml
---
Title: "..."
Slug: "unique-slug"
Short: "One-sentence summary."
Topics:
- topic-a
Commands:
- command-a
Flags:
- flag-a
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic   # or Example, Application, Tutorial
---
```

Help pages are embedded into the binary using `//go:embed` and loaded at startup. We will add two new help entries:
1. A **reference guide** explaining the review data model, SQLite schema, and query patterns (SectionType: GeneralTopic)
2. A **user guide** explaining the commands and workflows (SectionType: Tutorial)

## 4. Gap analysis

### 4.1 What exists vs. what we need

| Capability | Exists? | Location | Gap |
|-----------|---------|----------|-----|
| Semantic index builder | Yes | `internal/indexer/`, `cmd/codebase-browser/cmds/index/build.go` | Reuse as-is |
| Per-commit history indexer | Yes | `internal/history/indexer.go` | Reuse as-is |
| History SQLite store | Yes | `internal/history/store.go`, `schema.go` | Extend with review doc tables |
| Markdown directive renderer | Yes | `internal/docs/renderer.go` | Needs to work with non-embedded docs |
| Doc page serving | Yes | `internal/server/api_doc.go` | Serves only embedded docs; needs disk/DB mode |
| HTTP server | Yes | `internal/server/server.go` | Needs review-specific routes |
| React widget hydration | Yes | `ui/src/features/doc/DocSnippet.tsx` | Reuse as-is |
| CLI command framework | Yes | Glazed + Cobra | New `review` command tree needed |
| Help entry system | Yes | Glazed `help.NewHelpSystem()` | New entries needed |
| Unified review DB | **No** | — | **New** |
| Review doc storage schema | **No** | — | **New** |
| Review index command | **No** | — | **New** |
| Review serve command | **No** | — | **New** |
| Review DB-create command | **No** | — | **New** |

### 4.2 Key gaps explained

**Gap 1: No unified review artifact.** Today, history.db (commits + snapshots) and markdown files (on disk) are separate. The reviewer must know which markdown file goes with which database. We need a single SQLite file that contains both.

**Gap 2: No schema for review documents.** The history schema has no tables for markdown content, doc metadata, or resolved snippet references. We need new tables.

**Gap 3: Doc renderer is embedded-only.** `internal/docs/renderer.go` reads source files from an `fs.FS` (either embedded or on-disk). But `docs.ListPages()` and the server handlers in `api_doc.go` only know about `internal/docs/embed/pages/`. We need a mode where docs are read from disk or from a database.

**Gap 4: No CLI workflow for review authors.** There is no single command that says "index these commits and these docs into a review database." The user must run `history scan`, `history index`, and then manually manage markdown files.

**Gap 5: No help documentation for the review feature.** Users need a reference guide ("what tables are in the review DB?") and a user guide ("how do I write a review markdown file?").

## 5. Proposed architecture

### 5.1 The big picture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         BUILD TIME (review index)                           │
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐  │
│  │ Git commit range │    │ Markdown files  │    │ Review SQLite DB        │  │
│  │ (e.g. HEAD~5..HEAD)│   │ (./reviews/*.md)│    │ (./reviews/pr-42.db)    │  │
│  └────────┬────────┘    └────────┬────────┘    └─────────────────────────┘  │
│           │                      │                           ▲              │
│           ▼                      ▼                           │              │
│  ┌───────────────────────────────────────────────────────────┐             │
│  │  codebase-browser review index                            │             │
│  │  ───────────────────────────────────────────────────────  │             │
│  │  1. Parse commit range (git log)                          │             │
│  │  2. Index each commit → snapshot_symbols, snapshot_files  │             │
│  │  3. Read markdown files → review_docs, review_doc_snippets│             │
│  │  4. Write everything to --db                              │             │
│  └───────────────────────────────────────────────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         RUNTIME (review serve)                              │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  codebase-browser review serve --db ./reviews/pr-42.db --addr :3002  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  HTTP Server (net/http.ServeMux)                                     │   │
│  │                                                                      │   │
│  │  /api/review/docs              → list review docs                    │   │
│  │  /api/review/docs/:slug        → rendered markdown page              │   │
│  │  /api/review/commits           → list commits in this review         │   │
│  │  /api/history/*                → existing history endpoints          │   │
│  │  /*                            → React SPA fallback                  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌──────────────────────────┐  ┌──────────────────────────────────────┐     │
│  │  Review DB (SQLite)      │  │  React SPA (from codebase-browser)   │     │
│  │  ───────────────────────  │  │  ──────────────────────────────────  │     │
│  │  commits                 │  │  DocPage.tsx renders markdown HTML   │     │
│  │  snapshot_symbols        │  │  DocSnippet.tsx hydrates widgets     │     │
│  │  snapshot_files          │  │  HistoryPage.tsx for deep dives      │     │
│  │  snapshot_refs           │  │                                      │     │
│  │  file_contents           │  │                                      │     │
│  │  review_docs             │  │                                      │     │
│  │  review_doc_snippets     │  │                                      │     │
│  └──────────────────────────┘  └──────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 New packages and files

We will create the following new files:

```
cmd/codebase-browser/cmds/review/
├── root.go        # Register review command tree
├── index.go       # review index subcommand
├── serve.go       # review serve subcommand
├── db.go          # review db create subcommand

internal/review/
├── schema.go      # SQLite schema extensions for review docs
├── store.go       # ReviewStore — Open, Create, DB, Close
├── indexer.go     # Review indexer — commits + docs → DB
├── loader.go      # DocLoader — reads markdown from DB or disk
├── server.go      # ReviewServer — HTTP handlers for review routes

docs/help/
├── review-reference.md    # Glazed help entry: reference guide
├── review-user-guide.md   # Glazed help entry: tutorial / user guide
```

And modify:

```
cmd/codebase-browser/main.go           # Add review.Register(rootCmd)
internal/server/server.go              # Add review route registration
internal/docs/renderer.go              # Minor: support commit= on all directives
```

### 5.3 The review SQLite schema

The review DB reuses the history schema **in full** (commits, snapshot_symbols, snapshot_files, snapshot_refs, file_contents) and adds two new tables for review documents.

**New schema** (`internal/review/schema.go`):

```sql
-- Review document metadata and content
CREATE TABLE review_docs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    frontmatter_json TEXT NOT NULL DEFAULT '{}',
    indexed_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_review_docs_slug ON review_docs(slug);

-- Resolved snippet references within review docs
CREATE TABLE review_doc_snippets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_id INTEGER NOT NULL REFERENCES review_docs(id) ON DELETE CASCADE,
    stub_id TEXT NOT NULL,
    directive TEXT NOT NULL,
    symbol_id TEXT,
    file_path TEXT,
    kind TEXT,
    language TEXT,
    text TEXT NOT NULL DEFAULT '',
    params_json TEXT NOT NULL DEFAULT '{}',
    start_line INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    commit_hash TEXT,
    UNIQUE(doc_id, stub_id)
);

CREATE INDEX idx_review_doc_snippets_doc ON review_doc_snippets(doc_id);
CREATE INDEX idx_review_doc_snippets_sym ON review_doc_snippets(symbol_id);
```

**Why these tables?**

- `review_docs` stores the raw markdown content. The `slug` is derived from the filename (e.g., `pr-42.md` → `pr-42`). The `frontmatter_json` stores any YAML frontmatter the author includes (title, author, PR number, etc.).
- `review_doc_snippets` stores the resolved snippet metadata. When `review index` processes a markdown file, it calls `docs.Render()` and captures the resulting `[]SnippetRef`. These are stored so that:
  1. The serve mode can return them to the frontend without re-rendering
  2. An LLM can query "which symbols are referenced in this review?"
  3. We can validate that all referenced symbols exist in the indexed commits

### 5.4 The review store

The `ReviewStore` wraps both the history store and the review doc tables.

```go
// internal/review/store.go

package review

import (
    "database/sql"
    "fmt"

    "github.com/wesen/codebase-browser/internal/history"
)

// Store owns a SQLite connection for the unified review database.
type Store struct {
    db *sql.DB
    // History provides the history schema and operations.
    History *history.Store
}

// Open opens an existing review database.
func Open(path string) (*Store, error) {
    db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
    if err != nil {
        return nil, fmt.Errorf("open review db: %w", err)
    }
    if err := configure(db); err != nil {
        _ = db.Close()
        return nil, err
    }

    // The history store uses the same DB connection.
    hist := &history.Store{}
    // We can't use history.Open directly because it opens a new connection.
    // Instead, we expose a way to construct a history.Store from an existing *sql.DB.
    // (This requires a small change to internal/history/store.go)

    return &Store{db: db, History: hist}, nil
}

// Create opens path, drops any existing tables, and recreates the full schema.
func Create(path string) (*Store, error) {
    store, err := Open(path)
    if err != nil {
        return nil, err
    }
    if err := store.ResetSchema(context.Background()); err != nil {
        _ = store.Close()
        return nil, err
    }
    return store, nil
}

// DB exposes the underlying database for direct queries.
func (s *Store) DB() *sql.DB { return s.db }

// Close checkpoints WAL and closes the connection.
func (s *Store) Close() error {
    _, _ = s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`)
    return s.db.Close()
}

// ResetSchema drops and recreates all tables (history + review).
func (s *Store) ResetSchema(ctx context.Context) error {
    if _, err := s.db.ExecContext(ctx, history.DropSchemaSQL); err != nil {
        return fmt.Errorf("drop history schema: %w", err)
    }
    if _, err := s.db.ExecContext(ctx, history.CreateSchemaSQL); err != nil {
        return fmt.Errorf("create history schema: %w", err)
    }
    if _, err := s.db.ExecContext(ctx, dropReviewSchemaSQL); err != nil {
        return fmt.Errorf("drop review schema: %w", err)
    }
    if _, err := s.db.ExecContext(ctx, createReviewSchemaSQL); err != nil {
        return fmt.Errorf("create review schema: %w", err)
    }
    return nil
}
```

**Note on history.Store reuse:** `internal/history/store.go` currently opens its own `*sql.DB`. We need to add a constructor that accepts an existing `*sql.DB`:

```go
// In internal/history/store.go, add:

// NewFromDB creates a Store from an existing *sql.DB.
func NewFromDB(db *sql.DB) (*Store, error) {
    if err := configure(db); err != nil {
        return nil, err
    }
    return &Store{db: db}, nil
}
```

### 5.5 The review indexer

The review indexer is the heart of `codebase-browser review index`. It:
1. Resolves the commit range to a list of commits
2. Indexes each commit into the history tables (reusing `history.IndexCommits`)
3. Reads the markdown files
4. Renders each markdown file to resolve snippets
5. Stores the docs and snippets in the review tables

```go
// internal/review/indexer.go

package review

import (
    "context"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/wesen/codebase-browser/internal/browser"
    "github.com/wesen/codebase-browser/internal/docs"
    "github.com/wesen/codebase-browser/internal/gitutil"
    "github.com/wesen/codebase-browser/internal/history"
    "github.com/wesen/codebase-browser/internal/indexer"
)

// IndexOptions controls the review indexing process.
type IndexOptions struct {
    RepoRoot     string   // path to git repository
    CommitRange  string   // e.g. "HEAD~10..HEAD"
    DocsPaths    []string // paths to markdown files or directories
    Patterns     []string // Go package patterns for extraction
    IncludeTests bool
    Parallelism  int      // max concurrent worktrees
    OnProgress   func(phase string, done, total int, detail string)
}

// IndexResult describes what the indexer did.
type IndexResult struct {
    CommitsIndexed int
    DocsIndexed    int
    SnippetsIndexed int
    Duration       time.Duration
    Errors         []IndexError
}

type IndexError struct {
    Phase   string
    Detail  string
    Err     error
}

// IndexReview builds a review database from commits and markdown docs.
func IndexReview(ctx context.Context, store *Store, opts IndexOptions) (*IndexResult, error) {
    start := time.Now()
    result := &IndexResult{}

    // ── Phase 1: resolve commit range ──
    commits, err := gitutil.ParseCommitRange(ctx, opts.RepoRoot, opts.CommitRange)
    if err != nil {
        return nil, fmt.Errorf("parse commit range %q: %w", opts.CommitRange, err)
    }

    // ── Phase 2: index commits (reuse history indexer) ──
    histOpts := history.IndexOptions{
        RepoRoot:     opts.RepoRoot,
        Commits:      commits,
        Patterns:     opts.Patterns,
        IncludeTests: opts.IncludeTests,
        Parallelism:  opts.Parallelism,
        OnProgress: func(done, total int, shortHash, message string) {
            result.CommitsIndexed = done
            if opts.OnProgress != nil {
                opts.OnProgress("commits", done, total, shortHash)
            }
        },
    }

    histResult, err := history.IndexCommits(ctx, store.History, histOpts)
    if err != nil {
        return nil, fmt.Errorf("index commits: %w", err)
    }
    result.CommitsIndexed = histResult.Indexed

    // ── Phase 3: discover markdown files ──
    docPaths, err := discoverDocs(opts.DocsPaths)
    if err != nil {
        return nil, fmt.Errorf("discover docs: %w", err)
    }

    // We need a loaded index to resolve snippets. Use the latest commit's snapshot.
    loaded, err := loadLatestSnapshot(ctx, store)
    if err != nil {
        return nil, fmt.Errorf("load latest snapshot: %w", err)
    }

    // ── Phase 4: index each markdown file ──
    for i, path := range docPaths {
        if err := indexDoc(ctx, store, path, loaded); err != nil {
            result.Errors = append(result.Errors, IndexError{
                Phase:  "doc",
                Detail: path,
                Err:    err,
            })
            continue
        }
        result.DocsIndexed++
        if opts.OnProgress != nil {
            opts.OnProgress("docs", i+1, len(docPaths), filepath.Base(path))
        }
    }

    result.Duration = time.Since(start)
    return result, nil
}
```

**Discovering docs:**

```go
func discoverDocs(paths []string) ([]string, error) {
    var result []string
    for _, p := range paths {
        info, err := os.Stat(p)
        if err != nil {
            return nil, err
        }
        if info.IsDir() {
            entries, err := os.ReadDir(p)
            if err != nil {
                return nil, err
            }
            for _, e := range entries {
                if strings.HasSuffix(e.Name(), ".md") {
                    result = append(result, filepath.Join(p, e.Name()))
                }
            }
        } else {
            result = append(result, p)
        }
    }
    return result, nil
}
```

**Loading the latest snapshot:**

To resolve `codebase-*` directives, we need a `*browser.Loaded`. The review DB stores per-commit snapshots, not a single `index.json`. We reconstruct a `Loaded` from the latest commit's data:

```go
func loadLatestSnapshot(ctx context.Context, store *Store) (*browser.Loaded, error) {
    // Find the latest commit by author_time.
    row := store.DB().QueryRowContext(ctx, `
        SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1
    `)
    var hash string
    if err := row.Scan(&hash); err != nil {
        return nil, fmt.Errorf("find latest commit: %w", err)
    }

    // Build an indexer.Index from snapshot_symbols, snapshot_files, snapshot_refs, snapshot_packages.
    idx := &indexer.Index{Version: "review-snapshot"}

    // Query snapshot_packages for this commit.
    rows, err := store.DB().QueryContext(ctx, `
        SELECT id, import_path, name, doc, language
        FROM snapshot_packages WHERE commit_hash = ?
    `, hash)
    // ... populate idx.Packages

    // Similar for snapshot_files, snapshot_symbols, snapshot_refs.

    return browser.LoadFromBytes(marshalIndex(idx))
}
```

**Note:** This is a simplified sketch. In practice, we may want to add a helper in `internal/history/store.go` that reconstructs an `*indexer.Index` from a commit hash, so the review indexer can reuse it.

**Indexing a single doc:**

```go
func indexDoc(ctx context.Context, store *Store, path string, loaded *browser.Loaded) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    slug := strings.TrimSuffix(filepath.Base(path), ".md")

    // We need a source FS to resolve snippets. For review docs, we use
    // the repo root as the source FS.
    sourceFS := os.DirFS(".") // or the repo root

    page, err := docs.Render(slug, data, loaded, sourceFS)
    if err != nil {
        return fmt.Errorf("render doc: %w", err)
    }

    // Extract frontmatter if present.
    frontmatter := "{}"
    // (parse YAML frontmatter from data, or use a helper)

    // Insert into review_docs.
    res, err := store.DB().ExecContext(ctx, `
        INSERT INTO review_docs (slug, title, path, content, frontmatter_json, indexed_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(slug) DO UPDATE SET
            title = excluded.title,
            path = excluded.path,
            content = excluded.content,
            frontmatter_json = excluded.frontmatter_json,
            indexed_at = excluded.indexed_at
    `, slug, page.Title, path, string(data), frontmatter, time.Now().Unix())
    if err != nil {
        return fmt.Errorf("insert review doc: %w", err)
    }

    docID, _ := res.LastInsertId()

    // Insert snippets.
    for _, snip := range page.Snippets {
        paramsJSON, _ := json.Marshal(snip.Params)
        _, err := store.DB().ExecContext(ctx, `
            INSERT INTO review_doc_snippets
                (doc_id, stub_id, directive, symbol_id, file_path, kind, language,
                 text, params_json, start_line, end_line, commit_hash)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `, docID, snip.StubID, snip.Directive, snip.SymbolID, snip.FilePath,
            snip.Kind, snip.Language, snip.Text, string(paramsJSON),
            snip.StartLine, snip.EndLine, snip.CommitHash)
        if err != nil {
            return fmt.Errorf("insert snippet: %w", err)
        }
    }

    return nil
}
```

### 5.6 The review serve command

`codebase-browser review serve` starts an HTTP server that serves the review docs and provides the history API.

```go
// cmd/codebase-browser/cmds/review/serve.go (simplified)

type ServeSettings struct {
    DBPath   string `glazed:"db"`
    Addr     string `glazed:"addr"`
    RepoRoot string `glazed:"repo-root"`
}

func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
    s := &ServeSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
        return err
    }

    // Open the review database.
    store, err := review.Open(s.DBPath)
    if err != nil {
        return fmt.Errorf("open review db: %w", err)
    }
    defer store.Close()

    // Load the latest snapshot as the static index.
    loaded, err := review.LoadLatestSnapshot(ctx, store)
    if err != nil {
        return fmt.Errorf("load snapshot: %w", err)
    }

    // The source FS is the repo root.
    sourceFS := os.DirFS(s.RepoRoot)

    // Construct a server. We reuse the existing Server but add review routes.
    srv := server.New(loaded, sourceFS, web.FS(), nil, nil)
    srv.History = store.History
    srv.RepoRoot = s.RepoRoot

    // Register review-specific routes.
    h := srv.Handler()
    // But we need to mount review routes on a sub-mux.
    // Better: extend Server.Handler() to accept review handlers,
    // or create a ReviewServer wrapper.
}
```

**Better approach:** Create a `ReviewServer` in `internal/review/server.go` that wraps `server.Server` and adds `/api/review/*` routes.

```go
// internal/review/server.go

package review

import (
    "net/http"
    "strings"

    "github.com/wesen/codebase-browser/internal/server"
)

// ReviewServer extends the base server with review-specific routes.
type ReviewServer struct {
    *server.Server
    Store *Store
}

// Handler returns an http.Handler with review routes mounted.
func (rs *ReviewServer) Handler() http.Handler {
    mux := http.NewServeMux()

    // Base server routes.
    base := rs.Server.Handler()

    // Review-specific routes take precedence.
    mux.HandleFunc("/api/review/docs", rs.handleReviewDocList)
    mux.HandleFunc("/api/review/docs/", rs.handleReviewDocPage)
    mux.HandleFunc("/api/review/commits", rs.handleReviewCommits)
    mux.HandleFunc("/api/review/stats", rs.handleReviewStats)

    // Everything else goes to the base server.
    mux.Handle("/", base)

    return mux
}

func (rs *ReviewServer) handleReviewDocList(w http.ResponseWriter, r *http.Request) {
    rows, err := rs.Store.DB().QueryContext(r.Context(), `
        SELECT slug, title, path, indexed_at FROM review_docs ORDER BY slug
    `)
    // ... scan into []DocMeta, writeJSON
}

func (rs *ReviewServer) handleReviewDocPage(w http.ResponseWriter, r *http.Request) {
    slug := strings.TrimPrefix(r.URL.Path, "/api/review/docs/")
    if slug == "" {
        http.Error(w, "missing slug", http.StatusBadRequest)
        return
    }

    // Read doc content from DB.
    var content string
    err := rs.Store.DB().QueryRowContext(r.Context(), `
        SELECT content FROM review_docs WHERE slug = ?
    `, slug).Scan(&content)
    if err != nil {
        http.Error(w, "doc not found", http.StatusNotFound)
        return
    }

    // Render using the existing pipeline.
    page, err := docs.Render(slug, []byte(content), rs.Server.Loaded, rs.Server.SourceFS)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    writeJSON(w, page)
}
```

### 5.7 The review db create command

`codebase-browser review db create` is identical to `review index` except it skips the markdown doc phase. It produces a SQLite DB containing only the history tables (commits, snapshots, file contents). This is the artifact you hand to an LLM.

```go
// cmd/codebase-browser/cmds/review/db.go (simplified)

func (c *DBCreateCommand) RunIntoGlazeProcessor(
    ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
    s := &DBCreateSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
        return err
    }

    store, err := review.Create(s.DBPath)
    if err != nil {
        return err
    }
    defer store.Close()

    opts := review.IndexOptions{
        RepoRoot:     s.RepoRoot,
        CommitRange:  s.CommitRange,
        Patterns:     s.Patterns,
        IncludeTests: s.IncludeTests,
        Parallelism:  s.Parallelism,
    }

    result, err := review.IndexReview(ctx, store, opts)
    if err != nil {
        return err
    }

    // Output summary row for Glazed.
    row := types.NewRow(
        types.MRP("db", s.DBPath),
        types.MRP("commits", result.CommitsIndexed),
        types.MRP("duration", result.Duration.String()),
    )
    return gp.AddRow(ctx, row)
}
```

Wait — `IndexReview` also indexes docs. For `db create`, we need a variant that skips docs. Let's call it `IndexCommitsOnly`:

```go
func IndexCommitsOnly(ctx context.Context, store *Store, opts IndexOptions) (*IndexResult, error) {
    // Same as IndexReview but skip Phases 3 and 4.
}
```

Or simpler: add a `SkipDocs bool` field to `IndexOptions`.

### 5.8 Glazed help entries

We need two help entries. They live in `docs/help/` and are embedded into the binary.

**Entry 1: Reference guide** (`docs/help/review-reference.md`)

```yaml
---
Title: "Review Database Reference"
Slug: "review-db-reference"
Short: "Schema, tables, and query patterns for the code-review SQLite database."
Topics:
- code-review
- sqlite
- reference
Commands:
- review index
- review serve
- review db create
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---
```

Content sections:
- Overview of the review database
- History tables (commits, snapshot_symbols, snapshot_files, snapshot_refs, file_contents)
- Review doc tables (review_docs, review_doc_snippets)
- Common SQL queries for LLMs
- Symbol ID scheme
- Commit range syntax

**Entry 2: User guide** (`docs/help/review-user-guide.md`)

```yaml
---
Title: "Writing Code Review Guides"
Slug: "review-user-guide"
Short: "How to write markdown review guides and serve them with codebase-browser review."
Topics:
- code-review
- markdown
- tutorial
Commands:
- review index
- review serve
- review db create
Flags:
- commits
- docs
- db
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---
```

Content sections:
- Quick start: index and serve a review
- Writing review markdown files
- Available directives (codebase-diff, codebase-impact, etc.)
- Commit range syntax
- Sharing review databases
- Querying the DB with an LLM

**Embedding the docs:**

```go
// docs/help/embed.go
package help

import (
    "embed"
    "github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *.md
var docFS embed.FS

func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
    return helpSystem.LoadSectionsFromFS(docFS, ".")
}
```

And in `cmd/codebase-browser/main.go`:

```go
import reviewhelp "github.com/wesen/codebase-browser/docs/help"

func main() {
    // ... existing setup ...
    cobra.CheckErr(reviewhelp.AddDocToHelpSystem(helpSystem))
    // ...
}
```

## 6. Pseudocode and key flows

### 6.1 Flow: review index

```
User runs: codebase-browser review index --commits HEAD~5..HEAD --docs ./reviews/ --db pr-42.db

1. Parse flags
   CommitRange = "HEAD~5..HEAD"
   DocsPaths   = ["./reviews/"]
   DBPath      = "pr-42.db"

2. review.Create(DBPath)
   a. sql.Open("sqlite3", "pr-42.db?_journal_mode=WAL")
   b. configure(db) — enable foreign keys, WAL
   c. ResetSchema(ctx)
      i.   exec history.dropSchemaSQL
      ii.  exec history.createSchemaSQL
      iii. exec review.dropSchemaSQL
      iv.  exec review.createSchemaSQL

3. review.IndexReview(ctx, store, opts)
   a. Phase 1: gitutil.ParseCommitRange(repoRoot, "HEAD~5..HEAD")
      → []gitutil.Commit{...5 commits...}

   b. Phase 2: history.IndexCommits(ctx, store.History, histOpts)
      For each commit:
        i.   git worktree add /tmp/wt-<sha> <sha>
        ii.  indexer.Extract(moduleRoot=/tmp/wt-<sha>)
             → *indexer.Index
        iii. store.History.LoadSnapshot(ctx, commit, idx, wt)
             → INSERT INTO commits, snapshot_*, file_contents
        iv.  git worktree remove /tmp/wt-<sha>

   c. Phase 3: discoverDocs(["./reviews/"])
      → []string{"./reviews/pr-42.md", "./reviews/checklist.md"}

   d. Phase 4: For each doc path:
        i.   os.ReadFile(path)
        ii.  review.loadLatestSnapshot(ctx, store)
             → SELECT latest commit hash
             → Reconstruct *indexer.Index from snapshot_* tables
             → browser.LoadFromBytes
        iii. docs.Render(slug, data, loaded, sourceFS)
             → preprocess directives → []SnippetRef
             → goldmark.Convert → HTML
        iv.  INSERT INTO review_docs (slug, title, path, content, ...)
        v.   INSERT INTO review_doc_snippets (doc_id, stub_id, ...)

4. Output result:
   CommitsIndexed: 5
   DocsIndexed: 2
   SnippetsIndexed: 12
   Duration: 45s
```

### 6.2 Flow: review serve

```
User runs: codebase-browser review serve --db pr-42.db --addr :3002

1. Parse flags
   DBPath = "pr-42.db"
   Addr   = ":3002"

2. review.Open(DBPath)
   → *review.Store (with embedded *history.Store)

3. review.LoadLatestSnapshot(ctx, store)
   → *browser.Loaded (from latest commit's snapshot)

4. Construct ReviewServer
   a. server.New(loaded, sourceFS, spaFS, nil, nil)
   b. srv.History = store.History
   c. rs := &review.ReviewServer{Server: srv, Store: store}

5. rs.Handler()
   mux.HandleFunc("/api/review/docs",      rs.handleReviewDocList)
   mux.HandleFunc("/api/review/docs/",     rs.handleReviewDocPage)
   mux.HandleFunc("/api/review/commits",   rs.handleReviewCommits)
   mux.HandleFunc("/api/review/stats",     rs.handleReviewStats)
   mux.Handle("/", rs.Server.Handler())  // base routes + history routes

6. http.ListenAndServe(Addr, h)

7. Browser opens http://localhost:3002
   a. SPA loads from /
   b. React router sees /review/pr-42 (or similar)
   c. Frontend fetches /api/review/docs/pr-42
   d. Server: SELECT content FROM review_docs WHERE slug = 'pr-42'
   e. docs.Render(slug, content, loaded, sourceFS)
   f. JSON response: {slug, title, html, snippets, errors}
   g. React DocPage renders HTML, DocSnippet hydrates widgets
```

### 6.3 Flow: LLM querying the DB

```
User runs: codebase-browser review db create --commits HEAD~5..HEAD --db pr-42.db

Then prompts an LLM:
"Tell me which functions had signature changes between the first and last commit in pr-42.db."

The LLM connects to the SQLite file and runs:

SELECT
    old.name,
    old.signature AS old_sig,
    new.signature AS new_sig,
    c.short_hash,
    c.message
FROM snapshot_symbols old
JOIN snapshot_symbols new ON old.id = new.id
JOIN commits c ON c.hash = new.commit_hash
WHERE old.commit_hash = (SELECT hash FROM commits ORDER BY author_time ASC LIMIT 1)
  AND new.commit_hash = (SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1)
  AND old.signature != new.signature;
```

## 7. Implementation phases

### Phase 1: Schema and store (1–2 days)

**Goal:** Create the review schema and store, and refactor history.Store to support external DB connections.

**Files to create:**
- `internal/review/schema.go` — SQL constants for review_docs and review_doc_snippets
- `internal/review/store.go` — Store type with Open, Create, ResetSchema, Close

**Files to modify:**
- `internal/history/store.go` — Add `NewFromDB(*sql.DB) (*Store, error)`

**Validation:**
```bash
go test ./internal/review/... -v
```

### Phase 2: Review indexer (2–3 days)

**Goal:** Implement `review.IndexReview` with commit indexing and doc indexing.

**Files to create:**
- `internal/review/indexer.go` — IndexOptions, IndexResult, IndexReview
- `internal/review/loader.go` — loadLatestSnapshot helper

**Files to modify:**
- `internal/history/indexer.go` — Ensure it works with a Store that has an external DB

**Validation:**
```bash
codebase-browser review db create --commits HEAD~2..HEAD --db /tmp/test.db
sqlite3 /tmp/test.db ".tables"
sqlite3 /tmp/test.db "SELECT COUNT(*) FROM commits"
```

### Phase 3: Review server (2 days)

**Goal:** Implement ReviewServer with doc list and doc page handlers.

**Files to create:**
- `internal/review/server.go` — ReviewServer, handler methods

**Validation:**
```bash
codebase-browser review index --commits HEAD~2..HEAD --docs ./testdata/reviews/ --db /tmp/test.db
codebase-browser review serve --db /tmp/test.db --addr :3002 &
curl http://localhost:3002/api/review/docs
curl http://localhost:3002/api/review/docs/my-review
```

### Phase 4: CLI commands (2 days)

**Goal:** Wire up `review index`, `review serve`, and `review db create` commands.

**Files to create:**
- `cmd/codebase-browser/cmds/review/root.go` — Register command tree
- `cmd/codebase-browser/cmds/review/index.go` — index subcommand
- `cmd/codebase-browser/cmds/review/serve.go` — serve subcommand
- `cmd/codebase-browser/cmds/review/db.go` — db create subcommand

**Files to modify:**
- `cmd/codebase-browser/main.go` — Add `review.Register(rootCmd)`

**Validation:**
```bash
codebase-browser review --help
codebase-browser review index --help
codebase-browser review serve --help
codebase-browser review db create --help
```

### Phase 5: Glazed help entries (1–2 days)

**Goal:** Write and embed reference and user guide help entries.

**Files to create:**
- `docs/help/embed.go` — Go embedding and registration
- `docs/help/review-reference.md` — Reference guide
- `docs/help/review-user-guide.md` — User guide

**Files to modify:**
- `cmd/codebase-browser/main.go` — Add `reviewhelp.AddDocToHelpSystem(helpSystem)`

**Validation:**
```bash
codebase-browser help review-db-reference
codebase-browser help review-user-guide
```

### Phase 6: Integration and end-to-end testing (2 days)

**Goal:** Full workflow test, fix edge cases, write integration tests.

**Validation:**
```bash
make test
make build
# Full workflow:
codebase-browser review index --commits HEAD~5..HEAD --docs ./reviews/ --db /tmp/e2e.db
codebase-browser review serve --db /tmp/e2e.db --addr :3003 &
curl -s http://localhost:3003/api/review/docs | jq .
curl -s http://localhost:3003/api/review/stats | jq .
```

## 8. API reference

### 8.1 New CLI commands

```
codebase-browser review index
  --commits RANGE          Git commit range (e.g. HEAD~10..HEAD)
  --docs PATHS...          Markdown files or directories
  --db PATH                Output SQLite database path
  --repo-root PATH         Git repository root (default: .)
  --patterns PATTERNS...   Go package patterns (default: ./...)
  --include-tests          Include test files (default: true)
  --parallelism N          Max concurrent worktrees (default: 1)

codebase-browser review serve
  --db PATH                Review SQLite database path
  --addr ADDRESS           Bind address (default: :3002)
  --repo-root PATH         Git repository root for source file reads

codebase-browser review db create
  --commits RANGE          Git commit range
  --db PATH                Output SQLite database path
  --repo-root PATH         Git repository root (default: .)
  --patterns PATTERNS...   Go package patterns
  --include-tests          Include test files
  --parallelism N          Max concurrent worktrees
```

### 8.2 New HTTP endpoints

```
GET /api/review/docs
→ [{slug, title, path, indexed_at}, ...]

GET /api/review/docs/:slug
→ {slug, title, html, snippets, errors}
  (Same shape as existing /api/doc/:slug)

GET /api/review/commits
→ [{hash, short_hash, message, author_name, author_time}, ...]

GET /api/review/stats
→ {commits, docs, snippets, symbols, files, refs}
```

### 8.3 SQLite schema reference

**History tables** (unchanged from GCB-009):
- `commits` — commit metadata
- `snapshot_packages` — per-commit packages
- `snapshot_files` — per-commit files
- `snapshot_symbols` — per-commit symbols
- `snapshot_refs` — per-commit cross-references
- `file_contents` — deduplicated file content blobs

**Review tables** (new):
- `review_docs` — markdown document content and metadata
- `review_doc_snippets` — resolved snippet references within docs

## 9. Testing strategy

### 9.1 Unit tests

- `internal/review/store_test.go` — Test Open, Create, ResetSchema, Close
- `internal/review/indexer_test.go` — Test discoverDocs, loadLatestSnapshot with a fixture DB

### 9.2 Integration tests

- `cmd/codebase-browser/cmds/review/review_test.go` — Full workflow:
  1. Create a temp git repo with 3 commits
  2. Write a markdown file with a `codebase-snippet` directive
  3. Run `review index`
  4. Assert DB contains commits, symbols, docs, and snippets
  5. Run `review serve`, hit `/api/review/docs`, assert HTML contains stub divs

### 9.3 Manual validation checklist

- [ ] `review db create` produces a valid SQLite file queryable by sqlite3 CLI
- [ ] `review index` with `--docs` produces a DB with review_docs and review_doc_snippets
- [ ] `review serve` renders markdown with correct HTML and snippet metadata
- [ ] Widgets (codebase-diff, codebase-impact) hydrate correctly in the browser
- [ ] `codebase-browser help review-db-reference` shows the reference guide
- [ ] `codebase-browser help review-user-guide` shows the user guide

## 10. Risks, alternatives, and open questions

### 10.1 Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Loading latest snapshot is slow for large repos | High | Add caching, or allow `--snapshot-commit` override |
| Review DB becomes large | Medium | Compress file_contents, or store deltas |
| Markdown files reference symbols not in indexed commits | Medium | Validate during indexing and warn |
| Concurrent worktree creation fails on resource limits | Low | Respect `--parallelism`, default to 1 |

### 10.2 Alternatives considered

**Alternative 1: Separate review DB and history DB.**
Instead of a unified DB, keep history.db separate and store review docs in a second SQLite file. Rejected because it complicates distribution — users would need to share two files.

**Alternative 2: Store rendered HTML in the DB instead of raw markdown.**
Store the goldmark output so serve is faster. Rejected because it bloats the DB and makes editing impossible. Raw markdown + on-the-fly rendering is fast enough.

**Alternative 3: Use embedded docs instead of DB-stored docs.**
Continue using `internal/docs/embed/pages/` and rebuild the binary for each review. Rejected because it defeats the purpose of a lightweight review tool — reviews should be data, not code.

### 10.3 Open questions

1. **Should `review index` also build the static index.json?** Currently the review server reconstructs a Loaded from the latest commit. Should we also store a flat `index.json` in the DB for faster loading?
2. **How should we handle TypeScript in review docs?** The existing indexer supports both Go and TS. The review indexer should too, but we need to decide if TS extraction runs in worktrees (which may not have node_modules).
3. **Should review docs support frontmatter-driven metadata?** E.g. YAML frontmatter with `title:`, `author:`, `pr:` fields. This is mentioned in the schema but needs a parser decision.
4. **What is the URL structure for review docs in the SPA?** Should they be at `/review/:slug`, `/docs/:slug`, or something else?

## 11. References

### 11.1 Key files in this repo

| File | What it contains |
|------|-----------------|
| `cmd/codebase-browser/main.go` | Root command, help system setup |
| `cmd/codebase-browser/cmds/serve/run.go` | Existing serve command — model for review serve |
| `cmd/codebase-browser/cmds/index/build.go` | Index builder — model for review db create |
| `cmd/codebase-browser/cmds/history/scan.go` | Commit range scanner |
| `internal/indexer/types.go` | Canonical Index, Symbol, Ref types |
| `internal/indexer/id.go` | Stable symbol ID generation |
| `internal/browser/index.go` | Loaded index with lookup maps |
| `internal/docs/renderer.go` | Markdown directive pipeline |
| `internal/docs/pages.go` | Doc page listing |
| `internal/server/server.go` | HTTP server and route registration |
| `internal/server/api_doc.go` | Doc API handlers |
| `internal/history/schema.go` | History SQLite schema |
| `internal/history/store.go` | History store operations |
| `internal/history/indexer.go` | Per-commit indexing pipeline |
| `internal/sqlite/schema.go` | Codebase SQLite schema |
| `ui/src/features/doc/DocPage.tsx` | React doc page renderer |
| `ui/src/features/doc/DocSnippet.tsx` | Widget hydration dispatcher |

### 11.2 Related tickets

| Ticket | Topic |
|--------|-------|
| GCB-001 | Original codebase-browser design |
| GCB-005 | Semantic PR review architecture |
| GCB-009 | Git-aware indexing (history subsystem) |
| GCB-010 | Embeddable semantic diff widgets |
| GCB-011 | diffs library adoption |

### 11.3 External documentation

- Glazed help authoring: run `glaze help how-to-write-good-documentation-pages`
- Glazed command framework: `github.com/go-go-golems/glazed/pkg/cmds`
- Goldmark markdown renderer: `github.com/yuin/goldmark`
- SQLite3 Go driver: `github.com/mattn/go-sqlite3`
