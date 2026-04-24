---
title: "GCB-009: Git-Aware Codebase Index — Design and Implementation Guide"
doc_type: design-doc
topics:
  - git
  - indexing
  - sqlite
  - symbols
  - diff
  - history
  - concepts
related_files:
  - "internal/indexer/types.go:Core index types (Index, Package, File, Symbol, Ref, Range)"
  - "internal/indexer/extractor.go:Go AST extractor that walks packages and emits symbols"
  - "internal/indexer/id.go:Stable ID scheme for symbols, files, packages"
  - "internal/indexer/xref.go:Cross-reference extraction from Go AST"
  - "internal/indexer/multi.go:Multi-language extractor interface and merge logic"
  - "internal/indexer/write.go:Index JSON serialization"
  - "internal/sqlite/schema.go:SQLite schema (packages, files, symbols, refs, meta)"
  - "internal/sqlite/loader.go:Bulk loads Index into SQLite"
  - "internal/sqlite/store.go:SQLite Store (Open, Create, DB, Close)"
  - "internal/sqlite/generate_build.go:Build-time DB generation from index.json"
  - "internal/concepts/types.go:Concept spec and param types"
  - "cmd/codebase-browser/cmds/index/build.go:CLI index build command"
  - "cmd/codebase-browser/cmds/query/query.go:CLI query command"
  - "cmd/codebase-browser/cmds/query/commands.go:Dynamic concept CLI commands"
  - "internal/server/api_concepts.go:Server-side concept execution API"
  - "ui/src/features/query/QueryConceptsPage.tsx:React structured query page"
---

# GCB-009: Git-Aware Codebase Index

## Track Symbol Locations Across Commits for Per-Function Diff and Change History

---

# Part 1: Why This Exists

## The Problem Today

Right now, `codebase-browser` produces a **single snapshot** of your codebase. When you run:

```bash
codebase-browser index build
```

the Go AST extractor walks every `.go` file in the module, builds an `Index` containing packages, files, symbols (functions, methods, types, constants, variables), and cross-references, and writes it to `index.json`. That JSON is then loaded into a SQLite database (`codebase.db`) via `go generate ./internal/sqlite`.

The result is powerful: you can query the index with SQL, run structured concepts, browse symbols in the web UI, and navigate cross-references. But it has a fundamental limitation:

**It only knows about one point in time.**

If you want to answer questions like:

- "What did function `Extract` look like 10 commits ago?"
- "Which functions changed in PR #42?"
- "Show me the diff of just `funcSymbol` across the main branch."
- "When was `MethodID` last modified, and what changed?"
- "What symbols were added or removed between v1.0 and v2.0?"

…you can't. The index is a flat snapshot. Git history is invisible to it.

This ticket changes that.

## The Goal

**Build a git-aware codebase index that tracks symbol locations across commits**, so that every commit gets its own index, and you can:

1. **Look up any symbol at any commit** — "Where was `SymbolID` defined in commit `abc1234`?"
2. **Diff a single function across commits** — "Show me how `funcSymbol` changed between these two commits."
3. **Track symbol history** — "When was `MethodID` introduced? When was it last modified?"
4. **Summarize changes in a PR** — "List every function that changed between `main` and this branch."
5. **Query the full history** — "Show me all commits that modified exported functions in `internal/server`."

The key insight is that the existing indexing pipeline already produces everything we need for a single commit. We just need to run it for **many commits** and store the results in a way that supports time-travel queries.

---

# Part 2: How the System Works Today

Before designing the git-aware extension, let's walk through every component that participates in indexing today. If you are a new intern, this section will give you a complete mental model.

## 2.1 The Index Data Model

The core types live in `internal/indexer/types.go`.

```
Index
├── Version      "1"
├── GeneratedAt  "2026-04-24T19:01:49Z"
├── Module       "github.com/wesen/codebase-browser"
├── GoVersion    "go1.22.0"
├── Packages[]   ──►  Package
├── Files[]      ──►  File
├── Symbols[]    ──►  Symbol
└── Refs[]       ──►  Ref
```

### Package

A Go package (or TS module directory). Identified by import path.

```
Package
├── ID          "pkg:github.com/wesen/codebase-browser/internal/indexer"
├── ImportPath  "github.com/wesen/codebase-browser/internal/indexer"
├── Name        "indexer"
├── Doc         "Package indexer extracts..."
├── FileIDs[]   ["file:extractor.go", "file:types.go", ...]
├── SymbolIDs[] ["sym:github.com/.../func.Extract", ...]
└── Language    "go"
```

### File

A single source file. Identified by module-relative path.

```
File
├── ID         "file:internal/indexer/types.go"
├── Path       "internal/indexer/types.go"
├── PackageID  "pkg:github.com/wesen/codebase-browser/internal/indexer"
├── Size       4200
├── LineCount  120
├── SHA256     "a1b2c3..."
├── BuildTags  []
└── Language   "go"
```

### Symbol

A named declaration: function, method, type, interface, struct, constant, variable, or alias. This is the central entity we want to track across commits.

```
Symbol
├── ID          "sym:github.com/.../func.Extract"
├── Kind        "func"
├── Name        "Extract"
├── PackageID   "pkg:github.com/.../indexer"
├── FileID      "file:internal/indexer/extractor.go"
├── Range       {StartLine:45, StartCol:0, EndLine:102, EndCol:1,
│                StartOffset:1400, EndOffset:3200}
├── Doc         "Extract loads Go packages..."
├── Signature   "func Extract(opts ExtractOptions) (*Index, error)"
├── Receiver    nil  (or {TypeName:"Store", Pointer:false} for methods)
├── TypeParams  nil
├── Exported    true
├── Children[]  nil  (nested symbols, unused today)
├── Tags        []
└── Language    "go"
```

The `Range` is critical: it tells us the exact byte range and line/column of the symbol's body in the file. This is what lets us extract the source code of any function from the file content.

### Ref (Cross-Reference)

A reference from one symbol to another.

```
Ref
├── FromSymbolID  "sym:github.com/.../func.Extract"
├── ToSymbolID    "sym:github.com/.../func.SymbolID"
├── Kind          "call"
├── FileID        "file:internal/indexer/extractor.go"
└── Range         {StartLine:62, StartCol:14, ...}
```

## 2.2 The Extraction Pipeline

The extraction pipeline is a three-phase process: **load → walk → emit**.

```
┌─────────────────────────────────────────────────────┐
│  Phase 1: Load                                      │
│  packages.Load(cfg, "./...")                        │
│  │                                                  │
│  │ Reads go.mod, resolves dependencies,             │
│  │ parses every .go file into an AST                │
│  │                                                  │
│  ▼                                                  │
│  Phase 2: Walk                                      │
│  For each package → for each file → for each decl   │
│  │                                                  │
│  │ extractDecl() → funcSymbol(), typeSymbol(),      │
│  │                 valueSymbol()                    │
│  │                                                  │
│  │ addRefsForFile() → ast.Inspect(fn.Body, ...)     │
│  │                                                  │
│  ▼                                                  │
│  Phase 3: Emit                                      │
│  sortIndex() → stampLanguage() → return &Index{}    │
│                                                     │
│  Caller writes to index.json via indexer.Write()    │
└─────────────────────────────────────────────────────┘
```

**File:** `internal/indexer/extractor.go` — the `Extract()` function

The extraction happens in-process using Go's `go/packages` and `go/ast` standard library packages. No external tools are needed. The key steps are:

1. **Configure the loader:** `packages.Config` with `NeedName | NeedFiles | NeedSyntax | NeedTypes | NeedTypesInfo | NeedImports | NeedModule`.

2. **Load packages:** `packages.Load(cfg, patterns...)` — this is the slow part (typically 2-10 seconds for a medium project).

3. **Walk each package's AST:** For every `ast.FuncDecl` we emit a function or method symbol. For every `ast.GenDecl` we emit types, constants, and variables.

4. **Extract cross-references:** `addRefsForFile` walks function bodies looking for identifier uses (`types.Info.Uses`) that resolve to known symbols.

5. **Sort and return:** The index is sorted deterministically so the output is diff-stable.

## 2.3 The ID Scheme

Stable IDs are the foundation of cross-commit tracking. Without them, we can't match the same function across different commits because line numbers and even file paths change.

**File:** `internal/indexer/id.go`

### Symbol ID

```
sym:<importPath>.<kind>.<name>[#<signatureHash>]
```

Examples:
- `sym:github.com/wesen/codebase-browser/internal/indexer.func.Extract`
- `sym:github.com/wesen/codebase-browser/internal/indexer.method.Store.ResetSchema`
- `sym:github.com/wesen/codebase-browser/internal/indexer.func.helper#ab12cd34`

**Why import path, not file path?** If you rename a file or move it to a different directory within the same package, the import path doesn't change. The symbol ID stays stable.

**The `#hash` suffix** disambiguates cases where two symbols in the same package would collide (e.g., two test helpers named `helper` in different files). It uses a SHA-256 truncated hash of the function signature.

### Method ID

```
sym:<importPath>.method.<recvType>.<name>
```

Example: `sym:github.com/.../sqlite.method.Store.ResetSchema`

### File ID

```
file:<module-relative-path>
```

Example: `file:internal/indexer/types.go`

### Package ID

```
pkg:<import-path>
```

Example: `pkg:github.com/wesen/codebase-browser/internal/indexer`

## 2.4 The SQLite Schema

The flat `index.json` is converted into a relational SQLite database at build time.

**File:** `internal/sqlite/schema.go`

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  meta     │    │ packages │    │  files   │    │ symbols  │
│──────────│    │──────────│    │──────────│    │──────────│
│ key (PK) │    │ id (PK)  │◄───│ pkg_id   │◄───│ pkg_id   │
│ value    │    │ import_  │    │ id (PK)  │    │ id (PK)  │
│          │    │ path     │    │ path     │    │ kind     │
│          │    │ name     │    │ size     │    │ name     │
│          │    │ doc      │    │ line_cnt │    │ file_id  │
│          │    │ language │    │ sha256   │    │ range... │
│          │    │          │    │ language │    │ doc      │
└──────────┘    └──────────┘    └──────────┘    │ signature│
                                                │ exported │
                                                │ language │
                                                └────┬─────┘
                                                     │
                                                ┌────▼─────┐
                                                │   refs   │
                                                │──────────│
                                                │ from_sym │
                                                │ to_sym   │
                                                │ kind     │
                                                │ file_id  │
                                                │ range... │
                                                └──────────┘
```

Key design decisions:
- `refs.from_symbol_id` and `refs.to_symbol_id` are **text columns**, not strict foreign keys, because refs can target external symbols (e.g., `sym:os.func.Getenv`).
- The `symbols` table stores range as flat columns (`start_line`, `start_col`, `end_line`, `end_col`, `start_offset`, `end_offset`) for easy SQL querying.
- FTS5 (full-text search) is optional and enabled via a build tag.

## 2.5 The Build Pipeline

The full build pipeline, from source to browser, looks like this:

```
┌──────────────────────────────────────────────────────────────────────┐
│  Step 1: Extract Go index                                           │
│  codebase-browser index build                                       │
│  │                                                                  │
│  │ Reads: ./cmd/..././internal/... Go source files                  │
│  │ Writes: internal/indexfs/embed/index.json                        │
│  ▼                                                                  │
│  Step 2: Extract TS index (optional)                                │
│  codebase-browser index build --lang auto --ts-module-root ui       │
│  │                                                                  │
│  │ Runs: go run ./cmd/build-ts-index                                │
│  │ Writes: internal/indexfs/embed/index-ts.json (intermediate)      │
│  │ Merges into: internal/indexfs/embed/index.json                   │
│  ▼                                                                  │
│  Step 3: Generate SQLite DB                                         │
│  go generate ./internal/sqlite                                      │
│  │                                                                  │
│  │ Reads: internal/indexfs/embed/index.json                         │
│  │ Writes: internal/sqlite/embed/codebase.db                        │
│  ▼                                                                  │
│  Step 4: Generate embedded web assets                               │
│  BUILD_WEB_LOCAL=1 go generate ./internal/web                       │
│  │                                                                  │
│  │ Reads: ui/dist/public/ (Vite build output)                      │
│  │ Writes: internal/web/embed/public/                               │
│  ▼                                                                  │
│  Step 5: Build the Go binary                                        │
│  go build ./cmd/codebase-browser                                    │
│  │                                                                  │
│  │ Embeds: index.json, codebase.db, web assets, source snapshot     │
│  ▼                                                                  │
│  Step 6: Serve                                                      │
│  codebase-browser serve --addr :3011                                │
│                                                                     │
│  Browser: /api/* → JSON APIs, /* → SPA                             │
│  CLI: codebase-browser query, codebase-browser query commands ...   │
└──────────────────────────────────────────────────────────────────────┘
```

**File:** `cmd/codebase-browser/cmds/index/build.go` — the CLI command that orchestrates steps 1-2.

**File:** `internal/sqlite/generate_build.go` — the `go generate` helper for step 3.

## 2.6 The Concept System

Structured query concepts (from GCB-008) are named, parameterized SQL queries stored as `.sql` files with YAML metadata preambles.

**File:** `concepts/symbols/exported-functions.sql`

```sql
/* codebase-browser concept
name: exported-functions
short: Exported functions in a package
params:
  - name: package
    type: string
    help: Package import path prefix (e.g. internal/server)
  - name: limit
    type: int
    default: 50
    help: Max results
tags: [symbols, exported]
*/
SELECT id, name, kind, package_id, file_id,
       start_line, end_line, doc, signature
FROM   symbols
WHERE  exported = 1
  AND  kind IN ('func', 'method')
  AND  package_id LIKE '%' || {{sqlString .package}} || '%'
ORDER BY name
LIMIT  {{.limit}};
```

The concept system provides:
- **CLI:** `codebase-browser query commands symbols exported-functions --package internal/server --limit 5`
- **Web API:** `POST /api/query-concepts/symbols/exported-functions/execute`
- **Web UI:** `/#/queries/symbols/exported-functions?p.package=internal%2Fserver`

---

# Part 3: What We Need to Change

## 3.1 The Core Idea: Per-Commit Indexing

The fundamental change is to run the existing extraction pipeline **once per commit** instead of once total.

```
Before (single snapshot):

  HEAD ──► Extract() ──► index.json ──► codebase.db

After (per-commit history):

  commit a1b2c3d ──► Extract() ──► snapshot
  commit e4f5a6b ──► Extract() ──► snapshot
  commit c7d8e9f ──► Extract() ──► snapshot
  ...                                   │
  HEAD ─────────────────────────────────┤
                                       ▼
                              history.db (unified)
```

Each "snapshot" contains the same data as today's index — packages, files, symbols, refs — but tagged with a commit hash.

The unified `history.db` stores all snapshots in one SQLite database with an additional `commit_id` dimension on every table. This lets you query:

- "What symbols exist at commit X?"
- "What changed between commit X and commit Y?"
- "Show me the history of symbol S."

## 3.2 Why This Is Hard

Several challenges make this non-trivial:

### Challenge 1: Checking Out Each Commit

To extract the index for a commit, we need the source code as it existed at that commit. There are three approaches:

| Approach | How | Pros | Cons |
|----------|-----|------|------|
| **Checkout** | `git checkout <hash>` for each commit | Simple, exact | Destructive, slow, requires clean working tree |
| **Worktree** | `git worktree add <dir> <hash>` | Non-destructive, parallelizable | Disk space, cleanup |
| **git show** | `git show <hash>:<path>` to read individual files | No checkout needed | Can't run `go/packages.Load` without files on disk |

Since the extractor needs real files on disk (it uses `go/packages.Load` which parses Go source files), we need either checkout or worktree. **Worktree is the right choice** because it's non-destructive and can be parallelized.

### Challenge 2: Handling File Moves and Renames

When a file moves from `internal/old/file.go` to `internal/new/file.go`, the `file_id` changes from `file:internal/old/file.go` to `file:internal/new/file.go`. But the symbols inside might be identical.

The existing symbol ID scheme already handles this gracefully: `sym:<importPath>.<kind>.<name>` uses the package import path, not the file path. If the package import path stays the same (which it usually does when files are moved within a package), the symbol ID is stable.

However, if a package is renamed, the symbol IDs all change. We need a **symbol identity heuristic** that goes beyond exact ID matching.

### Challenge 3: Symbol Matching Across Commits

Not all changes are renames. Sometimes:
- A function gets split into two.
- A method is extracted from one type to another.
- A symbol is deleted and a new one with the same name is created.
- A function's signature changes.

For the MVP, we'll use **exact ID matching**: if `sym:pkg.func.Foo` exists in both commits, it's the same symbol. This covers the vast majority of cases. More sophisticated matching (fuzzy signature matching, structural similarity) can come later.

### Challenge 4: Performance

Indexing a medium project takes 2-10 seconds. Indexing 1000 commits would take 30-170 minutes. We need:
- **Incremental indexing:** Only index commits that haven't been indexed yet.
- **Parallelism:** Index multiple commits simultaneously.
- **Narrowed commit ranges:** Only index commits that touch tracked files.

### Challenge 5: Storage

Each snapshot contains ~30 packages, ~76 files, ~329 symbols, and ~1005 refs (as of today's codebase). Storing 1000 snapshots naively would be ~1.4M rows in the symbols table alone. SQLite handles this fine, but the schema needs to be designed for it.

---

# Part 4: Architecture

## 4.1 System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Git-Aware Indexing System                    │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Commit Scanner                                                │  │
│  │  • Lists commits in a range                                    │  │
│  │  • Filters to commits that touch tracked files                 │  │
│  │  • Detects already-indexed commits                             │  │
│  └──────────────────────┬─────────────────────────────────────────┘  │
│                         │                                            │
│                         ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Worktree Manager                                              │  │
│  │  • Creates temporary git worktrees                             │  │
│  │  • One worktree per parallel slot                              │  │
│  │  • Cleans up after indexing                                    │  │
│  └──────────────────────┬─────────────────────────────────────────┘  │
│                         │                                            │
│                         ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Per-Commit Extractor                                          │  │
│  │  • Runs existing Extract() on each worktree                    │  │
│  │  • Tags resulting Index with commit metadata                   │  │
│  │  • Returns (commit, Index) pairs                               │  │
│  └──────────────────────┬─────────────────────────────────────────┘  │
│                         │                                            │
│                         ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  History Store (SQLite)                                        │  │
│  │  • Commits table: hash, message, author, timestamp             │  │
│  │  • Snapshot tables: packages, files, symbols, refs             │  │
│  │    (same as today, plus commit_hash FK)                        │  │
│  │  • Symbol snapshots: tracks symbol existence per commit        │  │
│  │  • Diff engine: compares snapshots between commits             │  │
│  └──────────────────────┬─────────────────────────────────────────┘  │
│                         │                                            │
│                         ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Query & Concept Layer                                         │  │
│  │  • Time-travel queries: "symbols at commit X"                  │  │
│  │  • History concepts: "function history", "PR changes"          │  │
│  │  • Diff API: "diff symbol S between commits A and B"          │  │
│  └──────────────────────┬─────────────────────────────────────────┘  │
│                         │                                            │
│                         ▼                                            │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  CLI & Web                                                     │  │
│  │  • CLI: codebase-browser history scan / diff / show            │  │
│  │  • Web: /api/history/* endpoints                               │  │
│  │  • UI: history timeline, symbol diff viewer                    │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

## 4.2 The Commits Table

A new top-level table stores commit metadata:

```sql
CREATE TABLE commits (
    hash TEXT PRIMARY KEY,          -- Full SHA-256 hash
    short_hash TEXT NOT NULL,       -- First 7 chars for display
    message TEXT NOT NULL,          -- First line of commit message
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_time INTEGER NOT NULL,   -- Unix timestamp
    parent_hashes TEXT NOT NULL,    -- JSON array of parent hashes
    tree_hash TEXT NOT NULL,        -- Git tree hash
    indexed_at INTEGER NOT NULL,    -- When we indexed this commit
    branch TEXT NOT NULL DEFAULT '' -- Branch name at scan time
);

CREATE INDEX idx_commits_author_time ON commits(author_time);
CREATE INDEX idx_commits_branch ON commits(branch);
```

This table is populated by the commit scanner before any extraction happens.

## 4.3 The Snapshot Schema

Every existing table gets a `commit_hash` column. The snapshot tables are separate from the current tables to keep the single-commit path fast and simple.

```sql
-- Snapshot of packages at a specific commit
CREATE TABLE snapshot_packages (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id TEXT NOT NULL,
    import_path TEXT NOT NULL,
    name TEXT NOT NULL,
    doc TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'go',
    PRIMARY KEY (commit_hash, id)
);

-- Snapshot of files at a specific commit
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
    content_hash TEXT NOT NULL DEFAULT '',  -- NEW: git blob hash for fast change detection
    PRIMARY KEY (commit_hash, id)
);

-- Snapshot of symbols at a specific commit
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
    -- NEW: body_hash for fast change detection
    body_hash TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (commit_hash, id)
);

-- Snapshot of refs at a specific commit
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

-- Indexes for common queries
CREATE INDEX idx_snap_pkg_commit ON snapshot_packages(commit_hash);
CREATE INDEX idx_snap_file_commit ON snapshot_files(commit_hash);
CREATE INDEX idx_snap_file_sha ON snapshot_files(sha256);
CREATE INDEX idx_snap_sym_commit ON snapshot_symbols(commit_hash);
CREATE INDEX idx_snap_sym_name ON snapshot_symbols(name);
CREATE INDEX idx_snap_sym_kind ON snapshot_symbols(kind);
CREATE INDEX idx_snap_sym_pkg ON snapshot_symbols(package_id);
CREATE INDEX idx_snap_sym_body ON snapshot_symbols(body_hash);
CREATE INDEX idx_snap_ref_commit ON snapshot_refs(commit_hash);
CREATE INDEX idx_snap_ref_from ON snapshot_refs(from_symbol_id);
CREATE INDEX idx_snap_ref_to ON snapshot_refs(to_symbol_id);
```

### Why `body_hash`?

Instead of comparing full symbol definitions to detect changes, we compute a SHA-256 hash of the symbol body at extraction time. When a symbol has the same `body_hash` across commits, its definition hasn't changed, and we can skip expensive diff computation.

```go
// Pseudocode for body_hash computation
func computeBodyHash(fileContent string, sym Symbol) string {
    body := fileContent[sym.Range.StartOffset:sym.Range.EndOffset]
    h := sha256.Sum256([]byte(body))
    return hex.EncodeToString(h[:])
}
```

### Why `content_hash` on files?

Similarly, the git blob hash of each file lets us skip re-indexing files that haven't changed between commits. If `content_hash` is the same, the file is byte-for-byte identical, and we can skip extraction for that file.

## 4.4 The Symbol History View

A convenience view that summarizes a symbol's lifetime:

```sql
CREATE VIEW symbol_history AS
SELECT
    s.id AS symbol_id,
    s.name,
    s.kind,
    s.package_id,
    c.hash AS commit_hash,
    c.short_hash,
    c.message AS commit_message,
    c.author_time,
    s.body_hash,
    s.start_line,
    s.end_line,
    s.signature
FROM snapshot_symbols s
JOIN commits c ON c.hash = s.commit_hash
ORDER BY s.id, c.author_time DESC;
```

This lets you write queries like:

```sql
-- History of a specific symbol
SELECT * FROM symbol_history
WHERE symbol_id = 'sym:github.com/.../func.Extract'
ORDER BY author_time DESC;

-- Symbols introduced in the last 10 commits
SELECT DISTINCT s.id, s.name, c.short_hash, c.message
FROM snapshot_symbols s
JOIN commits c ON c.hash = s.commit_hash
WHERE c.author_time > (SELECT MAX(author_time) - 864000 FROM commits)
  AND s.id NOT IN (
    SELECT id FROM snapshot_symbols
    WHERE commit_hash IN (
      SELECT hash FROM commits
      WHERE author_time < c.author_time
      ORDER BY author_time DESC LIMIT 1
    )
  );
```

## 4.5 The Diff Engine

The diff engine compares two snapshots and produces a structured diff. It operates at three levels:

### File-Level Diff

```sql
-- Files changed between two commits
SELECT
    COALESCE(a.id, b.id) AS file_id,
    COALESCE(a.path, b.path) AS path,
    CASE
        WHEN a.id IS NULL THEN 'added'
        WHEN b.id IS NULL THEN 'removed'
        WHEN a.sha256 != b.sha256 THEN 'modified'
        ELSE 'unchanged'
    END AS change_type
FROM snapshot_files a
FULL OUTER JOIN snapshot_files b ON a.id = b.id
WHERE a.commit_hash = ? AND b.commit_hash = ?;
```

### Symbol-Level Diff

```sql
-- Symbols changed between two commits
SELECT
    COALESCE(a.id, b.id) AS symbol_id,
    COALESCE(a.name, b.name) AS name,
    COALESCE(a.kind, b.kind) AS kind,
    CASE
        WHEN a.id IS NULL THEN 'added'
        WHEN b.id IS NULL THEN 'removed'
        WHEN a.body_hash != b.body_hash THEN 'modified'
        WHEN a.signature != b.signature THEN 'signature-changed'
        WHEN a.start_line != b.start_line OR a.end_line != b.end_line THEN 'moved'
        ELSE 'unchanged'
    END AS change_type,
    a.start_line AS old_start_line,
    a.end_line AS old_end_line,
    b.start_line AS new_start_line,
    b.end_line AS new_end_line,
    a.body_hash AS old_body_hash,
    b.body_hash AS new_body_hash,
    a.signature AS old_signature,
    b.signature AS new_signature
FROM snapshot_symbols a
FULL OUTER JOIN snapshot_symbols b ON a.id = b.id
WHERE a.commit_hash = ? AND b.commit_hash = ?;
```

### Symbol Body Diff

For symbols marked as `modified`, we extract the actual body text from both commits and compute a unified diff:

```go
// Pseudocode for per-symbol body diff
func DiffSymbolBody(oldCommit, newCommit, symbolID string) (string, error) {
    // 1. Look up symbol in both commits
    oldSym := lookupSymbol(oldCommit, symbolID)
    newSym := lookupSymbol(newCommit, symbolID)
    
    // 2. Read file contents at both commits
    oldContent := readFileAtCommit(oldCommit, oldSym.FileID)
    newContent := readFileAtCommit(newCommit, newSym.FileID)
    
    // 3. Extract symbol bodies using byte ranges
    oldBody := oldContent[oldSym.Range.StartOffset:oldSym.Range.EndOffset]
    newBody := newContent[newSym.Range.StartOffset:newSym.Range.EndOffset]
    
    // 4. Compute unified diff
    return unifieddiff(oldBody, newBody), nil
}
```

The file content can be read from the worktree during extraction and cached, or read on-demand from git using `git show <hash>:<path>`.

## 4.6 File Content Cache

To support body diffs without re-checking out old commits, we cache file contents in the database:

```sql
CREATE TABLE file_contents (
    content_hash TEXT PRIMARY KEY,  -- git blob hash or SHA-256
    content BLOB NOT NULL,         -- compressed file content
    compressed INTEGER NOT NULL DEFAULT 1
);
```

**Storage estimate:** The codebase-browser repo has ~76 files averaging ~4KB each. Per commit, that's ~300KB. For 1000 commits, that's ~300MB raw. With zlib compression and deduplication (most files don't change between commits), the real storage would be ~30-50MB.

**Optimization:** Only store files that actually changed. If `content_hash` matches the parent commit, skip it.

---

# Part 5: Package Design

## 5.1 New Go Packages

```
internal/
├── gitutil/          NEW: Git operations (log, worktree, show)
│   ├── log.go        List commits, parse log output
│   ├── worktree.go   Create/remove worktrees
│   └── show.go       Read file content at a specific commit
│
├── history/          NEW: Per-commit indexing orchestration
│   ├── scanner.go    Scan commits, filter to relevant ones
│   ├── indexer.go    Orchestrate per-commit extraction
│   ├── store.go      Open/create history.db
│   ├── schema.go     History database schema
│   ├── loader.go     Load snapshot into history.db
│   ├── diff.go       Snapshot diff computation
│   ├── bodydiff.go   Per-symbol body diff
│   └── cache.go      File content caching
│
├── indexer/          EXISTING: unchanged, produces per-commit snapshots
├── sqlite/           EXISTING: unchanged, single-commit DB
└── concepts/         EXTENDED: new history/diff concepts
```

## 5.2 `internal/gitutil` — Git Operations

This package wraps the git CLI for the operations we need. It does not use a Go git library (like `go-git`) because the CLI is faster for large repos and handles all edge cases (submodules, LFS, etc.).

### `log.go` — Commit Listing

```go
package gitutil

type Commit struct {
    Hash         string
    ShortHash    string
    Message      string
    AuthorName   string
    AuthorEmail  string
    AuthorTime   time.Time
    ParentHashes []string
    TreeHash     string
}

// LogCommits lists commits in the given range.
// rangeSpec can be:
//   "HEAD~10..HEAD"         — last 10 commits
//   "main..feature-branch"  — commits in feature-branch not in main
//   "--all"                 — all commits
func LogCommits(ctx context.Context, repoRoot, rangeSpec string) ([]Commit, error)

// ChangedFiles returns the list of files changed in a commit.
func ChangedFiles(ctx context.Context, repoRoot, commitHash string) ([]string, error)

// IsAncestor checks if parent is an ancestor of child.
func IsAncestor(ctx context.Context, repoRoot, parent, child string) (bool, error)
```

### `worktree.go` — Worktree Management

```go
// CreateWorktree creates a temporary git worktree at the given commit.
// Returns the worktree directory path. The caller must call RemoveWorktree
// when done.
func CreateWorktree(ctx context.Context, repoRoot, commitHash string) (string, error)

// RemoveWorktree removes a previously created worktree.
func RemoveWorktree(ctx context.Context, repoRoot, worktreeDir string) error

// WorktreePool manages a pool of worktrees for parallel indexing.
type WorktreePool struct { ... }

func NewWorktreePool(repoRoot string, maxSize int) *WorktreePool
func (p *WorktreePool) Acquire(ctx context.Context, commitHash string) (string, error)
func (p *WorktreePool) Release(worktreeDir string) error
func (p *WorktreePool) Close() error
```

### `show.go` — Reading Files at a Commit

```go
// ShowFile reads a file's content at a specific commit.
// Equivalent to: git show <hash>:<path>
func ShowFile(ctx context.Context, repoRoot, commitHash, filePath string) ([]byte, error)

// FileBlobHash returns the git blob hash for a file at a commit.
// Equivalent to: git ls-tree <hash> -- <path>
func FileBlobHash(ctx context.Context, repoRoot, commitHash, filePath string) (string, error)
```

## 5.3 `internal/history` — Per-Commit Indexing

### `scanner.go` — Commit Scanning

```go
package history

type ScanOptions struct {
    RepoRoot    string
    Range       string     // git log range spec
    Branch      string     // branch name (for metadata)
    FileFilter  []string   // only index commits touching these paths
    Incremental bool       // skip already-indexed commits
}

type ScanResult struct {
    Commits     []gitutil.Commit
    Skipped     int        // already indexed
    Filtered    int        // didn't touch tracked files
}

// ScanCommits discovers commits to index.
func ScanCommits(ctx context.Context, store *Store, opts ScanOptions) (*ScanResult, error)
```

### `indexer.go` — Per-Commit Extraction

```go
type IndexOptions struct {
    RepoRoot      string
    Commits       []gitutil.Commit
    Patterns      []string     // Go package patterns
    IncludeTests  bool
    Parallelism   int          // max concurrent worktrees
    ContentCache  bool         // cache file contents for body diffs
    OnProgress    func(done, total int, commit string)
}

type IndexResult struct {
    Indexed     int
    Skipped     int
    Failed      int
    Errors      []error
    Duration    time.Duration
}

// IndexCommits runs the extraction pipeline for each commit.
func IndexCommits(ctx context.Context, store *Store, opts IndexOptions) (*IndexResult, error)
```

The per-commit indexing loop:

```go
// Pseudocode for IndexCommits
func IndexCommits(ctx context.Context, store *Store, opts IndexOptions) (*IndexResult, error) {
    pool := gitutil.NewWorktreePool(opts.RepoRoot, opts.Parallelism)
    defer pool.Close()
    
    sem := make(chan struct{}, opts.Parallelism)
    var wg sync.WaitGroup
    var mu sync.Mutex
    result := &IndexResult{}
    
    for i, commit := range opts.Commits {
        // Skip already-indexed commits (incremental mode)
        if store.HasCommit(ctx, commit.Hash) {
            result.Skipped++
            continue
        }
        
        wg.Add(1)
        sem <- struct{}{} // acquire slot
        
        go func(commit gitutil.Commit) {
            defer wg.Done()
            defer func() { <-sem }() // release slot
            
            // 1. Create worktree at this commit
            wt, err := pool.Acquire(ctx, commit.Hash)
            if err != nil {
                mu.Lock()
                result.Errors = append(result.Errors, err)
                result.Failed++
                mu.Unlock()
                return
            }
            defer pool.Release(wt)
            
            // 2. Extract the index
            idx, err := indexer.Extract(indexer.ExtractOptions{
                ModuleRoot:   wt,
                Patterns:     opts.Patterns,
                IncludeTests: opts.IncludeTests,
            })
            if err != nil {
                mu.Lock()
                result.Errors = append(result.Errors, fmt.Errorf("%s: %w", commit.ShortHash, err))
                result.Failed++
                mu.Unlock()
                return
            }
            
            // 3. Tag with commit metadata
            idx.GeneratedAt = commit.AuthorTime.Format(time.RFC3339)
            
            // 4. Compute body hashes (read file content, hash symbol bodies)
            if err := computeBodyHashes(ctx, wt, idx); err != nil {
                // non-fatal: log and continue without hashes
            }
            
            // 5. Load snapshot into history store
            if err := store.LoadSnapshot(ctx, commit, idx); err != nil {
                mu.Lock()
                result.Errors = append(result.Errors, fmt.Errorf("%s: %w", commit.ShortHash, err))
                result.Failed++
                mu.Unlock()
                return
            }
            
            // 6. Optionally cache file contents
            if opts.ContentCache {
                _ = cacheFileContents(ctx, store, wt, idx)
            }
            
            mu.Lock()
            result.Indexed++
            if opts.OnProgress != nil {
                opts.OnProgress(result.Indexed, len(opts.Commits), commit.ShortHash)
            }
            mu.Unlock()
        }(commit)
    }
    
    wg.Wait()
    return result, nil
}
```

### `store.go` — History Store

```go
type Store struct {
    db *sql.DB
}

func Open(path string) (*Store, error)
func Create(path string) (*Store, error)
func (s *Store) Close() error

// HasCommit checks if a commit has already been indexed.
func (s *Store) HasCommit(ctx context.Context, hash string) (bool, error)

// LoadSnapshot loads a single commit's index into the history database.
func (s *Store) LoadSnapshot(ctx context.Context, commit gitutil.Commit, idx *indexer.Index) error

// SymbolsAtCommit returns all symbols at a specific commit.
func (s *Store) SymbolsAtCommit(ctx context.Context, commitHash string) ([]SnapshotSymbol, error)

// DiffCommits compares two commits and returns the symbol-level diff.
func (s *Store) DiffCommits(ctx context.Context, oldHash, newHash string) (*CommitDiff, error)

// SymbolHistory returns the history of a specific symbol across all indexed commits.
func (s *Store) SymbolHistory(ctx context.Context, symbolID string) ([]SymbolHistoryEntry, error)

// DiffSymbolBody returns the unified diff of a symbol's body between two commits.
func (s *Store) DiffSymbolBody(ctx context.Context, oldHash, newHash, symbolID string) (string, error)
```

### `diff.go` — Snapshot Diff Types

```go
type ChangeType string

const (
    ChangeAdded           ChangeType = "added"
    ChangeRemoved         ChangeType = "removed"
    ChangeModified        ChangeType = "modified"
    ChangeSignatureChanged ChangeType = "signature-changed"
    ChangeMoved           ChangeType = "moved"
    ChangeUnchanged       ChangeType = "unchanged"
)

type FileDiff struct {
    FileID      string
    Path        string
    ChangeType  ChangeType
    OldSHA256   string
    NewSHA256   string
}

type SymbolDiff struct {
    SymbolID       string
    Name           string
    Kind           string
    PackageID      string
    ChangeType     ChangeType
    OldStartLine   int
    OldEndLine     int
    NewStartLine   int
    NewEndLine     int
    OldSignature   string
    NewSignature   string
    OldBodyHash    string
    NewBodyHash    string
}

type CommitDiff struct {
    OldHash    string
    NewHash    string
    Files      []FileDiff
    Symbols    []SymbolDiff
    Stats      DiffStats
}

type DiffStats struct {
    FilesAdded      int
    FilesRemoved    int
    FilesModified   int
    SymbolsAdded    int
    SymbolsRemoved  int
    SymbolsModified int
    SymbolsMoved    int
}
```

---

# Part 6: CLI Design

## 6.1 New Commands

The CLI adds a `history` subcommand group:

```
codebase-browser history
├── scan           Index commits into history.db
├── list           List indexed commits
├── show           Show a commit's snapshot
├── diff           Diff two commits (file and symbol level)
├── symbol-diff    Diff a single symbol between two commits
├── symbol-history Show the history of a symbol
└── stats          Print indexing statistics
```

### `history scan`

```bash
# Index the last 50 commits on the current branch
codebase-browser history scan --range "HEAD~50..HEAD"

# Index all commits that touch internal/server
codebase-browser history scan --range "HEAD~100..HEAD" --filter "internal/server/"

# Index a PR's commits
codebase-browser history scan --range "main..feature-branch"

# Incremental: skip already-indexed commits
codebase-browser history scan --range "HEAD~50..HEAD" --incremental

# Parallel indexing with 4 workers
codebase-browser history scan --range "HEAD~50..HEAD" --parallelism 4

# Custom DB path
codebase-browser history scan --db ./my-history.db --range "HEAD~20..HEAD"
```

### `history diff`

```bash
# Diff two specific commits
codebase-browser history diff abc1234 def5678

# Diff HEAD vs HEAD~1 (last commit's changes)
codebase-browser history diff HEAD~1 HEAD

# Diff a PR
codebase-browser history diff main feature-branch

# JSON output
codebase-browser history diff --format json abc1234 def5678

# Only show modified symbols
codebase-browser history diff abc1234 def5678 --only modified
```

### `history symbol-diff`

```bash
# Diff a specific function between two commits
codebase-browser history symbol-diff \
  --symbol "sym:github.com/.../func.Extract" \
  --from abc1234 --to def5678

# Diff by name (resolves to symbol ID)
codebase-browser history symbol-diff \
  --name Extract \
  --package internal/indexer \
  --from HEAD~5 --to HEAD
```

### `history symbol-history`

```bash
# Show full history of a symbol
codebase-browser history symbol-history \
  --symbol "sym:github.com/.../func.Extract"

# Last 10 commits where this symbol changed
codebase-browser history symbol-history \
  --name Extract \
  --limit 10
```

## 6.2 History Concepts

New structured query concepts for the history database:

```
concepts/
├── history/
│   ├── commits-timeline.sql        List commits chronologically
│   ├── pr-summary.sql              Summarize changes in a PR
│   ├── symbol-changes.sql          Symbols that changed between two commits
│   ├── symbol-history.sql          Full history of a symbol
│   ├── hotspots.sql                Most-changed symbols (by body_hash changes)
│   └── file-changes.sql            Files that changed between two commits
```

### Example: `pr-summary.sql`

```sql
/* codebase-browser concept
name: pr-summary
short: Summarize symbol changes in a PR
params:
  - name: base
    type: string
    help: Base commit hash
  - name: head
    type: string
    help: Head commit hash
tags: [history, diff, pr]
*/
WITH old AS (
    SELECT id, body_hash, signature, name, kind, package_id
    FROM snapshot_symbols WHERE commit_hash = {{sqlString .base}}
),
new AS (
    SELECT id, body_hash, signature, name, kind, package_id
    FROM snapshot_symbols WHERE commit_hash = {{sqlString .head}}
)
SELECT
    COALESCE(old.name, new.name) AS name,
    COALESCE(old.kind, new.kind) AS kind,
    COALESCE(old.package_id, new.package_id) AS package_id,
    CASE
        WHEN old.id IS NULL THEN 'added'
        WHEN new.id IS NULL THEN 'removed'
        WHEN old.body_hash != new.body_hash THEN 'modified'
        WHEN old.signature != new.signature THEN 'signature-changed'
        ELSE 'unchanged'
    END AS change_type
FROM old
FULL OUTER JOIN new ON old.id = new.id
WHERE old.id IS NULL
   OR new.id IS NULL
   OR old.body_hash != new.body_hash
   OR old.signature != new.signature
ORDER BY change_type, name;
```

### Example: `hotspots.sql`

```sql
/* codebase-browser concept
name: hotspots
short: Most frequently changed symbols
params:
  - name: limit
    type: int
    default: 20
    help: Max results
  - name: branch
    type: string
    default: ""
    help: Restrict to a branch
tags: [history, analysis]
*/
SELECT
    s.id AS symbol_id,
    s.name,
    s.kind,
    s.package_id,
    COUNT(DISTINCT s.body_hash) AS distinct_versions,
    COUNT(DISTINCT c.hash) AS commit_count,
    MIN(c.author_time) AS first_seen,
    MAX(c.author_time) AS last_changed
FROM snapshot_symbols s
JOIN commits c ON c.hash = s.commit_hash
WHERE c.branch LIKE '%' || {{sqlString .branch}} || '%'
GROUP BY s.id
HAVING COUNT(DISTINCT s.body_hash) > 1
ORDER BY distinct_versions DESC, commit_count DESC
LIMIT {{.limit}};
```

---

# Part 7: Server API

## 7.1 New Endpoints

```
GET  /api/history/commits                  List indexed commits
GET  /api/history/commits/:hash            Get commit details
GET  /api/history/commits/:hash/symbols    Symbols at a commit
GET  /api/history/commits/:hash/files      Files at a commit

GET  /api/history/diff?from=X&to=Y         Diff two commits
GET  /api/history/diff/:symbolId?from=X&to=Y  Per-symbol body diff

GET  /api/history/symbols/:id/history      Symbol change history
```

### `GET /api/history/diff?from=abc1234&to=def5678`

Response:
```json
{
  "from": "abc1234",
  "to": "def5678",
  "stats": {
    "filesAdded": 2,
    "filesRemoved": 0,
    "filesModified": 5,
    "symbolsAdded": 8,
    "symbolsRemoved": 1,
    "symbolsModified": 12,
    "symbolsMoved": 3
  },
  "files": [
    {
      "fileId": "file:internal/server/server.go",
      "path": "internal/server/server.go",
      "changeType": "modified"
    }
  ],
  "symbols": [
    {
      "symbolId": "sym:github.com/.../func.New",
      "name": "New",
      "kind": "func",
      "changeType": "modified",
      "oldStartLine": 15,
      "oldEndLine": 32,
      "newStartLine": 15,
      "newEndLine": 38,
      "oldSignature": "func New(l, srcFS, spaFS) *Server",
      "newSignature": "func New(l, srcFS, spaFS, sqlite, catalog) *Server"
    }
  ]
}
```

### `GET /api/history/diff/:symbolId?from=abc1234&to=def5678`

Response:
```json
{
  "symbolId": "sym:github.com/.../func.New",
  "name": "New",
  "from": "abc1234",
  "to": "def5678",
  "diff": "--- a/internal/server/server.go\n+++ b/internal/server/server.go\n@@ -15,10 +15,16 @@\n func New(\n \tloaded *browser.Loaded,\n \tsrcFS, spaFS fs.FS,\n+\tsqliteStore *cbsqlite.Store,\n+\tconceptCatalog *concepts.Catalog,\n ) *Server {\n \ts := &Server{\n \t\tloaded: loaded,\n+\t\tsqlite:  sqliteStore,\n+\t\tcatalog: conceptCatalog,\n \t}\n"
}
```

---

# Part 8: Web UI

## 8.1 New Pages

### History Timeline Page

Route: `/#/history`

Shows:
- List of indexed commits with timestamps
- Filter by branch, author, date range
- Click to see commit snapshot
- Select two commits to diff

### Symbol History Page

Route: `/#/symbol/:id/history` (or `/#/history/symbol/:id`)

Shows:
- Timeline of all commits where this symbol changed
- Body hash at each commit
- Click to diff between any two versions

### Symbol Diff Viewer

Route: `/#/history/diff?symbol=...&from=...&to=...`

Shows:
- Side-by-side or unified diff of a single function
- Syntax-highlighted
- With surrounding context

### PR Summary Page

Route: `/#/history/pr?base=main&head=feature-branch`

Shows:
- List of changed symbols with change type badges
- Expandable diff per symbol
- File change list

## 8.2 Integration with Existing Pages

### Package Page Enhancements

On a package page, add:
- "History" tab showing recent commits touching this package
- "Hotspots" showing most-changed symbols in this package

### Symbol Page Enhancements

On a symbol page, add:
- "History" link → navigates to symbol history timeline
- "Last modified" badge showing the most recent commit that changed this symbol
- "Diff with previous" button showing the last change

---

# Part 9: Implementation Plan

## Phase 1: Foundation (1-2 days)

**Goal:** Per-commit extraction that produces a history database.

### Step 1.1: Create `internal/gitutil` package

Implement:
- `LogCommits()` — parse `git log --format=<custom>` output
- `ChangedFiles()` — parse `git diff-tree --no-commit-id -r <hash>`
- `CreateWorktree()` / `RemoveWorktree()` — `git worktree add/remove`
- `ShowFile()` — `git show <hash>:<path>`

### Step 1.2: Create `internal/history` package with schema

Implement:
- `schema.go` — `commits`, `snapshot_*`, `file_contents` tables
- `store.go` — `Open()`, `Create()`, `HasCommit()`
- `loader.go` — `LoadSnapshot()` to bulk-insert a commit's index

### Step 1.3: Create `history scan` CLI command

Implement:
- `cmd/codebase-browser/cmds/history/scan.go`
- Wire into main.go

Validation:
```bash
codebase-browser history scan --range "HEAD~5..HEAD" --db /tmp/test-history.db
# Should index 5 commits into the history database
```

### Step 1.4: Create `history list` CLI command

Implement:
- `cmd/codebase-browser/cmds/history/list.go`
- Show commit hash, message, timestamp, symbol count

Validation:
```bash
codebase-browser history list --db /tmp/test-history.db
```

## Phase 2: Diff Engine (1 day)

**Goal:** Symbol-level and body-level diff between two commits.

### Step 2.1: Implement `DiffCommits()`

In `internal/history/diff.go`:
- File-level diff using FULL OUTER JOIN
- Symbol-level diff using FULL OUTER JOIN
- DiffStats computation

### Step 2.2: Implement body hash computation

In `internal/history/bodydiff.go`:
- `computeBodyHashes()` — hash symbol bodies during extraction
- `DiffSymbolBody()` — extract bodies from cache, compute unified diff

### Step 2.3: Implement file content cache

In `internal/history/cache.go`:
- Cache file contents during scan
- Read from cache (or `git show`) during diff

### Step 2.4: Create `history diff` and `history symbol-diff` CLI commands

Validation:
```bash
codebase-browser history diff HEAD~1 HEAD --db /tmp/test-history.db
codebase-browser history symbol-diff \
  --symbol "sym:github.com/.../func.Extract" \
  --from HEAD~1 --to HEAD --db /tmp/test-history.db
```

## Phase 3: History Concepts (0.5 days)

**Goal:** Structured query concepts for the history database.

### Step 3.1: Create history concept files

- `concepts/history/pr-summary.sql`
- `concepts/history/symbol-history.sql`
- `concepts/history/hotspots.sql`
- `concepts/history/commits-timeline.sql`

### Step 3.2: Wire history concepts into the concept catalog

The concept catalog already supports multiple source roots. Add the history DB as an additional source:

```go
// Pseudocode: loading history concepts alongside regular concepts
catalog := concepts.LoadEmbeddedCatalog()
if historyDB != nil {
    historyCatalog := concepts.LoadFromFS(historyConceptsFS, "history")
    catalog.Merge(historyCatalog)
}
```

### Step 3.3: Validate with CLI

```bash
codebase-browser query commands history pr-summary --base abc1234 --head def5678
```

## Phase 4: Server API (1 day)

**Goal:** HTTP endpoints for history queries.

### Step 4.1: Create `internal/server/api_history.go`

Implement:
- `GET /api/history/commits`
- `GET /api/history/commits/:hash/symbols`
- `GET /api/history/diff`
- `GET /api/history/diff/:symbolId`
- `GET /api/history/symbols/:id/history`

### Step 4.2: Wire into server

Add history store to `Server` struct:

```go
type Server struct {
    // ... existing fields ...
    History *history.Store // optional, nil if no history.db
}
```

## Phase 5: Web UI (1-2 days)

**Goal:** Browser UI for history browsing and symbol diffing.

### Step 5.1: History API client

`ui/src/api/historyApi.ts` — RTK Query endpoints for history APIs.

### Step 5.2: History timeline page

`ui/src/features/history/HistoryPage.tsx`

### Step 5.3: Symbol diff viewer

`ui/src/features/history/SymbolDiffPage.tsx`

### Step 5.4: Integration with existing pages

Add history links to package and symbol pages.

---

# Part 10: Performance Considerations

## 10.1 Indexing Speed

| Operation | Time | Notes |
|-----------|------|-------|
| Single commit extraction | 2-10s | Depends on project size |
| 50 commits, sequential | 2-8 min | |
| 50 commits, 4 workers | 30s-2 min | |
| 1000 commits, 4 workers | 10-40 min | One-time cost |

## 10.2 Query Speed

The snapshot tables use composite primary keys `(commit_hash, id)`, so queries like "symbols at commit X" are O(log N) index lookups. The body_hash column enables fast change detection without reading file contents.

## 10.3 Storage

| Component | Size per commit | 1000 commits |
|-----------|----------------|--------------|
| Snapshot symbols | ~100 rows × 200B = ~20KB | ~20MB |
| Snapshot files | ~76 rows × 100B = ~8KB | ~8MB |
| Snapshot refs | ~1000 rows × 80B = ~80KB | ~80MB |
| File contents (compressed) | ~30KB average | ~30MB (deduped) |
| **Total** | ~140KB | ~140MB |

SQLite handles databases up to 281 TB. 140MB is well within comfortable limits.

## 10.4 Incremental Updates

The `--incremental` flag skips commits already in `history.db`. This makes re-running `history scan` cheap: only new commits get indexed.

---

# Part 11: Risks and Open Questions

## Risk 1: Build Tag Sensitivity

Some commits may have different build tags or require different Go versions. The extractor might fail on old commits.

**Mitigation:** Log errors per-commit and continue. Failed commits are recorded in `commits` with an error status.

## Risk 2: Symbol ID Instability

If a package is renamed between commits, all its symbol IDs change. The diff engine will report these as "removed + added" rather than "renamed."

**Mitigation (future):** Add a rename detection heuristic that matches symbols by (name, kind, file proximity) when IDs don't match.

## Risk 3: Very Large Histories

For projects with 10,000+ commits, the full history might be too large to index.

**Mitigation:** The `--range` flag lets users index only relevant commit ranges (e.g., recent N commits, a PR's commits, a release branch).

## Open Question 1: Branch Tracking

Should `history.db` track which branch(es) each commit belongs to? Git commits can be on multiple branches.

**Current plan:** Store `branch` as metadata at scan time. If you scan `main`, commits are tagged `main`. If you later scan `feature-branch`, those commits are tagged `feature-branch`. Overlapping commits get both tags.

## Open Question 2: Merge Commits

Merge commits have two parent hashes. Should the diff compare against both parents?

**Current plan:** Compare against the first parent only (the branch being merged into). This matches `git log --first-parent` semantics.

## Open Question 3: TypeScript Support

The history system works with any `*Index` produced by the extractor. TypeScript indexing via `build-ts-index` would work automatically if the worktree contains TS files.

**Current plan:** Support Go-only initially. TS support is a natural extension.

---

# Part 12: File Reference Map

Below is a complete map of the files involved, both existing and new.

## Existing Files (Read)

| File | Purpose |
|------|---------|
| `internal/indexer/types.go` | Core index types (Index, Package, File, Symbol, Ref, Range) |
| `internal/indexer/extractor.go` | Go AST extractor, produces *Index from source |
| `internal/indexer/id.go` | Stable ID scheme for symbols, files, packages |
| `internal/indexer/xref.go` | Cross-reference extraction from Go AST |
| `internal/indexer/multi.go` | Multi-language Extractor interface and Merge |
| `internal/indexer/write.go` | Index JSON serialization |
| `internal/sqlite/schema.go` | Single-commit SQLite schema |
| `internal/sqlite/loader.go` | Bulk loads Index into single-commit SQLite |
| `internal/sqlite/store.go` | Single-commit Store (Open, Create, DB, Close) |
| `internal/sqlite/generate_build.go` | Build-time single-commit DB generation |
| `internal/concepts/types.go` | Concept spec, param types |
| `internal/concepts/catalog.go` | Concept catalog loading |
| `internal/concepts/render.go` | SQL template rendering |
| `internal/concepts/repositories.go` | External/embedded concept repositories |
| `cmd/codebase-browser/cmds/index/build.go` | CLI index build command |
| `cmd/codebase-browser/cmds/query/query.go` | CLI query command |
| `cmd/codebase-browser/cmds/query/commands.go` | Dynamic concept CLI commands |
| `internal/server/api_concepts.go` | Server-side concept execution API |
| `internal/server/server.go` | HTTP server setup |
| `ui/src/features/query/QueryConceptsPage.tsx` | React structured query page |

## New Files (Create)

| File | Purpose |
|------|---------|
| `internal/gitutil/log.go` | Git commit listing and parsing |
| `internal/gitutil/worktree.go` | Git worktree management |
| `internal/gitutil/show.go` | Read file content at a specific commit |
| `internal/history/scanner.go` | Commit scanning and filtering |
| `internal/history/indexer.go` | Per-commit extraction orchestration |
| `internal/history/store.go` | History store (Open, Create, Close, queries) |
| `internal/history/schema.go` | History database schema |
| `internal/history/loader.go` | Load snapshot into history.db |
| `internal/history/diff.go` | Snapshot diff computation |
| `internal/history/bodydiff.go` | Per-symbol body diff |
| `internal/history/cache.go` | File content caching |
| `cmd/codebase-browser/cmds/history/root.go` | History CLI root |
| `cmd/codebase-browser/cmds/history/scan.go` | History scan command |
| `cmd/codebase-browser/cmds/history/list.go` | History list command |
| `cmd/codebase-browser/cmds/history/diff.go` | History diff command |
| `cmd/codebase-browser/cmds/history/symbol_diff.go` | Symbol diff command |
| `cmd/codebase-browser/cmds/history/symbol_history.go` | Symbol history command |
| `internal/server/api_history.go` | History HTTP endpoints |
| `ui/src/api/historyApi.ts` | RTK Query history API |
| `ui/src/features/history/HistoryPage.tsx` | History timeline UI |
| `ui/src/features/history/SymbolDiffPage.tsx` | Symbol diff viewer |
| `concepts/history/pr-summary.sql` | PR summary concept |
| `concepts/history/symbol-history.sql` | Symbol history concept |
| `concepts/history/hotspots.sql` | Change hotspot concept |
| `concepts/history/commits-timeline.sql` | Commits timeline concept |
| `concepts/history/symbol-changes.sql` | Symbol changes concept |
| `concepts/history/file-changes.sql` | File changes concept |

---

# Part 13: Glossary

| Term | Definition |
|------|------------|
| **Snapshot** | A complete index (packages, files, symbols, refs) for a single commit |
| **History DB** | The SQLite database containing all snapshots and commit metadata |
| **Worktree** | A `git worktree` — a directory containing the repo at a specific commit |
| **Body hash** | SHA-256 hash of a symbol's source code body, for fast change detection |
| **Content hash** | Git blob hash of a file, for fast file-level change detection |
| **Symbol diff** | A structured comparison of a symbol between two commits (added/removed/modified/moved) |
| **Body diff** | A unified text diff of a symbol's source code between two commits |
| **Hotspot** | A symbol that changes frequently across commits — a potential maintenance concern |
| **PR summary** | A structured diff showing all symbol changes between a PR's base and head commits |
| **Incremental scan** | A scan that skips commits already present in the history DB |
| **Commit range** | A git log range spec like `HEAD~50..HEAD` or `main..feature-branch` |
