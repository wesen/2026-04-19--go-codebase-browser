---
Title: Technical Writer Brief for codebase-browser Website Documentation
Ticket: GCB-016
Status: active
Topics:
    - codebase-browser
    - documentation
    - technical-writing
    - static-export
    - sqlite
    - react-frontend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: |-
        Current public landing page; should become the concise website/README entry point.
        Current landing page to audit/rewrite for static sql.js website documentation
    - Path: internal/history/schema.go
      Note: Commit/history/snapshot schema.
    - Path: internal/review/schema.go
      Note: Review document schema.
    - Path: internal/staticapp/export.go
      Note: |-
        Static export packaging pipeline and manifest writing.
        Static export pipeline to explain and verify in docs
    - Path: internal/staticapp/reviewdocs.go
      Note: |-
        Export-time rendering of review markdown into SQLite.
        Review markdown rendering into SQLite to explain in docs
    - Path: pkg/doc/db-reference.md
      Note: |-
        Existing Glazed schema/reference help page for the SQLite review database.
        Existing Glazed DB reference now using slug db-reference
    - Path: pkg/doc/embed.go
      Note: Glazed help embed registration location for future website docs
    - Path: pkg/doc/user-guide.md
      Note: |-
        Existing Glazed tutorial help page for writing and exporting static review guides.
        Existing Glazed user tutorial now using slug user-guide
    - Path: ui/src/api/sqlJsQueryProvider.ts
      Note: |-
        Main browser data access layer over SQLite.
        Browser semantic query provider source for docs
    - Path: ui/src/api/sqljs/sqlJsDb.ts
      Note: |-
        Browser-side sql.js bootstrap that loads manifest.json and db/codebase.db.
        Browser sql.js bootstrap source for docs
ExternalSources: []
Summary: A non-developer technical writer brief describing what information to gather, where to find it, what concepts to understand, and how to verify every example when writing the public website documentation for codebase-browser.
LastUpdated: 2026-05-02T00:00:00Z
WhatFor: Use this brief before drafting website documentation, README content, help pages, examples, tutorials, or diagrams for codebase-browser.
WhenToUse: Use when assigning documentation work to a writer who is not expected to read the whole codebase unaided but must produce accurate, verified user-facing documentation.
---


# Technical Writer Brief for codebase-browser Website Documentation

This brief is for a technical writer who needs to write the public website documentation for `codebase-browser`. You are not expected to be a Go or React developer, but you do need to understand the product model well enough to explain it accurately, choose good examples, and verify that the examples actually run.

The most important thing to know is this: **codebase-browser produces a static website backed by a SQLite database**. The Go command indexes a repository and review markdown into a database. The export command packages a React application, `manifest.json`, and `db/codebase.db`. The browser opens that database with `sql.js` and answers code navigation, review, history, diff, source, and xref questions locally. There is no Go web server in the exported runtime.

Your job is not to document every implementation detail. Your job is to help a new user understand:

1. what problem the tool solves;
2. what files and commands they use;
3. how the static export works;
4. how to write review markdown blocks;
5. how to query the SQLite artifact;
6. how to verify that their documentation examples are true.

The documentation should be written like a technical website, not like an internal changelog. It should teach the product from first principles and then provide copy-pasteable, tested examples.

## 1. Product explanation in plain language

Start from the user's problem, not from the implementation.

A good first explanation is:

> codebase-browser turns a code review into a static, shareable browser. It indexes a commit range, source files, symbols, references, and markdown review notes into SQLite. Then it exports a static React app that opens the SQLite database in the browser with `sql.js`. Reviewers can read prose, inspect symbol diffs, follow callers and callees, browse source files, and query the same database with SQL or an LLM.

Avoid saying that it is a server. The old server runtime has been removed. The exported artifact is static.

Use these terms consistently:

| Term | Meaning |
|---|---|
| **Review database** | The SQLite file produced by `review db create` or `review index`. It contains commits, source snapshots, symbols, refs, file contents, and review markdown. |
| **Static export** | A directory produced by `review export`. It contains the React app, `manifest.json`, `db/codebase.db`, and sql.js assets. |
| **Review guide** | A markdown file written by a human, usually under `reviews/`, containing prose plus fenced `codebase-*` blocks. |
| **Markdown block / directive** | A fenced code block such as ```` ```codebase-diff sym=... from=... to=... ```` that becomes an interactive widget in the browser. |
| **Symbol** | A named code element such as a function, method, type, const, or var. Symbols have stable IDs. |
| **Snapshot** | What packages, files, symbols, and references looked like at one commit. |
| **sql.js** | SQLite compiled to WebAssembly, used by the browser to open `db/codebase.db` locally. |

## 2. The base concepts you need before writing

You do not need to understand every Go package. You do need these concepts.

### 2.1 Static export, not runtime server

The browser is a static app. It may be served by any static file server, for example:

```bash
python3 -m http.server 8784 --directory /tmp/codebase-browser-export
```

That command serves files. It does not provide application APIs. The browser fetches ordinary static files such as `index.html`, JavaScript assets, `manifest.json`, `db/codebase.db`, and sql.js WASM assets.

The exported browser should not make `/api/*` application requests. If a documentation example suggests running `codebase-browser serve`, it is stale and must be corrected.

### 2.2 The database is the source of truth

The exported browser uses `db/codebase.db` for application data. This same DB can be opened by `sqlite3`, scripts, or LLM tools. This is a core product benefit: the browser artifact and the query artifact are the same file.

Key tables to know:

| Table | What it contains |
|---|---|
| `commits` | Indexed Git commits. |
| `snapshot_packages` | Packages/modules at each commit. |
| `snapshot_files` | Files at each commit. |
| `snapshot_symbols` | Functions, methods, types, consts, and vars at each commit. |
| `snapshot_refs` | References/calls/uses between symbols. |
| `file_contents` | Deduplicated raw file bytes. |
| `review_docs` | Raw markdown review documents. |
| `review_doc_snippets` | Resolved markdown block metadata from review docs. |
| `static_review_rendered_docs` | Export-time rendered review pages used by the static browser. |

You can find the reference help page at:

```bash
codebase-browser help db-reference
```

The markdown source for that page is:

```text
pkg/doc/db-reference.md
```

### 2.3 Review markdown becomes interactive review pages

A review guide is a markdown file. It can contain normal prose and special fenced code blocks. For example:

````markdown
# Review: Static export packaging

The export function is the boundary between the indexer and the static browser.

```codebase-signature sym=staticapp.Export
```

```codebase-diff sym=staticapp.Export from=HEAD~1 to=HEAD
```
````

During indexing/export, these blocks are resolved into metadata and widgets. The browser hydrates them using SQLite-backed provider calls.

The current short tutorial is:

```bash
codebase-browser help user-guide
```

The markdown source is:

```text
pkg/doc/user-guide.md
```

A missing documentation page should be added for the full markdown block reference. Suggested target:

```text
pkg/doc/markdown-block-reference.md
```

Suggested Glazed slug:

```yaml
Slug: "markdown-block-reference"
```

### 2.4 Symbols and commit refs

Many examples need `sym=` and commit refs.

Symbols can be written as full IDs:

```text
sym:github.com/wesen/codebase-browser/internal/staticapp.func.Export
```

or short refs when unambiguous:

```text
staticapp.Export
```

Commit refs can be:

```text
HEAD
HEAD~1
HEAD~5
<full SHA>
<short SHA>
<unique prefix>
```

Do not invent symbol names in examples. Choose symbols that exist in the repository and validate them by running the example.

### 2.5 Byte offsets matter

If you write about source extraction or body diffs, mention this rule:

> Source bodies are sliced by byte offsets from `file_contents.content` before UTF-8 decoding.

This matters because Go/indexer offsets are byte offsets. JavaScript strings use UTF-16 indexing. Documentation should not say that snippets are sliced by character index.

## 3. Information to gather, and where to gather it

Use this section as a research checklist before drafting.

### 3.1 User-facing command behavior

Gather from:

```text
cmd/codebase-browser/cmds/review/*.go
cmd/codebase-browser/cmds/history/*.go
cmd/codebase-browser/cmds/query/*.go
cmd/codebase-browser/main.go
```

Commands to run:

```bash
go run ./cmd/codebase-browser --help
go run ./cmd/codebase-browser review --help
go run ./cmd/codebase-browser review db --help
go run ./cmd/codebase-browser review db create --help
go run ./cmd/codebase-browser review index --help
go run ./cmd/codebase-browser review export --help
go run ./cmd/codebase-browser help --list
go run ./cmd/codebase-browser help user-guide
go run ./cmd/codebase-browser help db-reference
```

What to extract:

- exact command names;
- exact flag names;
- which commands create a DB;
- which commands export a static app;
- what paths are defaults and what paths must be provided;
- whether examples use `--commits` or `--range` for each command. Do not guess; verify help output.

### 3.2 Static export contents

Gather from:

```text
internal/staticapp/export.go
internal/staticapp/manifest.go
internal/staticapp/export_test.go
```

Commands to run:

```bash
go run ./cmd/codebase-browser review export \
  --db /tmp/codebase-browser-docs-smoke.db \
  --out /tmp/codebase-browser-docs-export

find /tmp/codebase-browser-docs-export -maxdepth 2 -type f | sort
cat /tmp/codebase-browser-docs-export/manifest.json
```

What to extract:

- export includes `index.html`, assets, `manifest.json`, `db/codebase.db`, and sql.js assets;
- export should not include `precomputed.json`, `search.wasm`, or `wasm_exec.js`;
- manifest fields: DB path, runtime query engine, and `hasGoRuntimeServer=false`;
- export mutates the copied output DB, not the source DB.

### 3.3 Browser runtime behavior

Gather from:

```text
ui/src/api/sqljs/sqlJsDb.ts
ui/src/api/sqljs/sqlRows.ts
ui/src/api/sqlJsQueryProvider.ts
ui/src/api/historyApi.ts
ui/src/api/docApi.ts
ui/src/api/sourceApi.ts
ui/src/api/indexApi.ts
ui/src/api/xrefApi.ts
```

What to extract:

- browser loads `manifest.json`;
- browser fetches `db/codebase.db` from the manifest path;
- browser initializes `sql.js`;
- RTK Query hooks call provider methods, not HTTP endpoint URLs;
- `SqlJsQueryProvider` is the main semantic data layer;
- query functions cover docs, source, symbols, xrefs, history, diffs, impact, and review docs.

Do not document internal TypeScript implementation as user-facing API unless it helps explain architecture.

### 3.4 Review markdown block behavior

Gather from:

```text
internal/docs/renderer.go
ui/src/features/doc/DocSnippet.tsx
ui/src/features/doc/widgets/*.tsx
pkg/doc/user-guide.md
```

Current directives to document:

```text
codebase-snippet
codebase-signature
codebase-doc
codebase-file
codebase-diff
codebase-symbol-history
codebase-impact
codebase-commit-walk
codebase-annotation
codebase-changed-files
codebase-diff-stats
```

For each directive, gather:

- purpose;
- required params;
- optional params;
- example block;
- how it renders;
- common failure modes;
- whether it needs symbol refs, file paths, commit refs, or a mini DSL body.

The `codebase-commit-walk` block needs special attention because it uses a line-oriented step DSL. Inspect `internal/docs/renderer.go` and the relevant widget code before documenting it.

### 3.5 SQLite schema and SQL cookbook

Gather from:

```text
internal/history/schema.go
internal/review/schema.go
internal/staticapp/reviewdocs.go
pkg/doc/db-reference.md
ui/src/api/sqlJsQueryProvider.ts
```

What to extract:

- tables and views;
- join patterns used by the provider;
- common useful SQL queries;
- byte-offset explanation;
- examples for LLM users;
- difference between raw `review_docs` and rendered `static_review_rendered_docs`.

A public documentation site should include both:

1. a **schema reference**;
2. a **SQL cookbook** with practical questions and queries.

### 3.6 Testing and verification behavior

Gather from:

```text
internal/staticapp/export_test.go
ui/src/api/sqljs/sqlRows.test.ts
ui/src/api/sqlJsQueryProvider.test.ts
Makefile
```

Commands to run:

```bash
go test ./...
pnpm -C ui run test
pnpm -C ui run typecheck
```

What to extract:

- how maintainers know the export still omits legacy runtime files;
- how byte-offset slicing is tested;
- how commit ref resolution is tested;
- what manual browser smoke tests still need Playwright automation.

## 4. Documentation pages to produce

The website documentation should be organized by reader task. Suggested pages:

### 4.1 Landing page / README

Purpose: explain the product quickly and guide users to the next page.

Must include:

- one-paragraph product explanation;
- architecture diagram or compact pipeline;
- quick start;
- links to user guide, markdown block reference, DB reference, examples, and development docs;
- current status note: static sql.js runtime, no Go server runtime.

### 4.2 Quick start

Purpose: get from repository to static review browser in the smallest verified path.

Must include:

- prerequisites;
- create a tiny review markdown file;
- create/index DB;
- export static app;
- serve export with `python3 -m http.server`;
- open browser route;
- expected files in the export directory.

Every command must be smoke-tested.

### 4.3 Review markdown block reference

Purpose: canonical reference for every `codebase-*` fenced block.

Must include:

- syntax;
- required and optional params;
- examples;
- rendered behavior;
- troubleshooting;
- symbol ref syntax;
- commit ref syntax.

This should become a Glazed help entry under `pkg/doc/markdown-block-reference.md` with slug `markdown-block-reference`.

### 4.4 Writing review guides tutorial

Purpose: teach how to write good review prose, not just block syntax.

Must include:

- how to structure a review guide;
- how to combine prose and widgets;
- how to iterate;
- how to choose between snippet, diff, history, and impact blocks;
- how to avoid overloading the reader.

This can build from `pkg/doc/user-guide.md`.

### 4.5 SQLite database reference

Purpose: document the DB as both browser runtime artifact and LLM/script artifact.

Must include:

- schema overview;
- table reference;
- relationship diagram;
- common joins;
- byte-offset warning;
- SQL examples.

This can build from `pkg/doc/db-reference.md`.

### 4.6 SQL cookbook

Purpose: give users useful copy-paste SQL tasks.

Examples:

- list commits;
- list changed symbols;
- find added/removed symbols between first and last commit;
- find callers of a symbol;
- find files with most symbol changes;
- list review docs and render errors;
- inspect `static_review_rendered_docs`.

Every SQL example must be run against a smoke DB.

### 4.7 Architecture page

Purpose: explain the static sql.js architecture to maintainers and evaluators.

Must include:

- why SQLite is the runtime boundary;
- why old TinyGo/precomputed paths were removed;
- why old Go `/api/*` server was removed;
- export pipeline;
- browser provider architecture;
- testing strategy;
- known future extensions such as FTS or `sql.js-httpvfs`.

Use the GCB-015 design doc and Obsidian article as sources, but write a shorter website-facing version.

### 4.8 Development and verification guide

Purpose: help contributors run checks and verify docs.

Must include:

- setup;
- Go tests;
- UI tests;
- typecheck;
- building/exporting smoke DBs;
- static serving;
- browser smoke checklist;
- doc example verification rules.

## 5. Verification rules for every documentation example

This section is mandatory. Documentation examples must be treated as code.

### 5.1 Command examples

For every shell command in the docs:

1. run it in a clean-ish repo checkout;
2. record whether it succeeds;
3. if it depends on a fixture, create that fixture in the docs or examples directory;
4. avoid commands that only work because of local temporary files;
5. update the docs to match real flag names and real output paths.

Recommended smoke sequence:

```bash
# from repo root
go test ./...
pnpm -C ui run test
pnpm -C ui run typecheck

mkdir -p /tmp/codebase-browser-docs-reviews
cat > /tmp/codebase-browser-docs-reviews/static-smoke.md <<'EOF'
# Static smoke review

```codebase-signature sym=staticapp.Export
```
EOF

go run ./cmd/codebase-browser review db create \
  --commits HEAD~5..HEAD \
  --docs /tmp/codebase-browser-docs-reviews \
  --db /tmp/codebase-browser-docs-smoke.db

go run ./cmd/codebase-browser review export \
  --db /tmp/codebase-browser-docs-smoke.db \
  --out /tmp/codebase-browser-docs-export

find /tmp/codebase-browser-docs-export -maxdepth 2 -type f | sort
```

Before publishing, verify the actual flag names with `--help`. If the command uses `--range` instead of `--commits`, update the example. The docs must match the binary, not memory.

### 5.2 Markdown block examples

For every `codebase-*` block:

1. put the block in a real temporary review markdown file;
2. index it;
3. export it;
4. open the rendered review page;
5. check whether the widget appears without a doc error;
6. check browser console/network for failures;
7. if the example references a symbol, confirm the symbol exists in the indexed range.

Do not use placeholder symbols like `foo.Bar` in final docs unless the page explicitly marks them as schematic. Prefer symbols from this repo such as:

```text
staticapp.Export
staticapp.AddRenderedReviewDocs
review.IndexReview
indexer.Extract
```

Validate each chosen symbol against the current DB.

### 5.3 SQL examples

For every SQL query:

1. run it with `sqlite3` against a smoke DB;
2. confirm it returns plausible rows or intentionally explain why it may return none;
3. include `LIMIT` in exploratory examples;
4. avoid hard-coded hashes unless the example first shows how to select them;
5. use `ORDER BY author_time` consistently when referring to oldest/newest commits.

Suggested validation pattern:

```bash
sqlite3 /tmp/codebase-browser-docs-smoke.db <<'SQL'
SELECT short_hash, message
FROM commits
ORDER BY author_time DESC
LIMIT 5;
SQL
```

### 5.4 Browser examples

For browser screenshots or route examples:

1. serve the export directory with a static file server;
2. open the documented route;
3. verify that it works after a hard reload;
4. check that no request URL contains `/api/`;
5. check that the page does not show `STATIC_NOT_PRECOMPUTED`;
6. record the exact route used.

Useful routes:

```text
/#/
/#/review/<slug>
/#/history?symbol=<encoded-symbol-id>
/#/source/<path>
```

### 5.5 Negative checks

The static export should not contain old runtime files:

```bash
test ! -e /tmp/codebase-browser-docs-export/precomputed.json
test ! -e /tmp/codebase-browser-docs-export/search.wasm
test ! -e /tmp/codebase-browser-docs-export/wasm_exec.js
```

If any of those files appear in a new docs smoke, stop and ask a developer to investigate before publishing docs.

## 6. Source map for the technical writer

Use this as your map of where to look.

| Need | Source |
|---|---|
| Product overview | `README.md`, GCB-015 design doc, Obsidian article if available |
| Exact CLI commands | `go run ./cmd/codebase-browser ... --help`, `cmd/codebase-browser/cmds/**` |
| Review guide workflow | `pkg/doc/user-guide.md` |
| Database schema | `pkg/doc/db-reference.md`, `internal/history/schema.go`, `internal/review/schema.go` |
| Static export layout | `internal/staticapp/export.go`, `internal/staticapp/export_test.go` |
| Manifest fields | `internal/staticapp/manifest.go` |
| Rendered review docs | `internal/staticapp/reviewdocs.go` |
| Markdown directives | `internal/docs/renderer.go`, `ui/src/features/doc/DocSnippet.tsx`, `ui/src/features/doc/widgets/*.tsx` |
| Browser DB loading | `ui/src/api/sqljs/sqlJsDb.ts` |
| Browser queries | `ui/src/api/sqlJsQueryProvider.ts` |
| SQL row helpers and byte slicing | `ui/src/api/sqljs/sqlRows.ts`, `ui/src/api/sqljs/sqlRows.test.ts` |
| Commit ref behavior | `ui/src/api/sqlJsQueryProvider.test.ts` |
| Export invariants | `internal/staticapp/export_test.go` |
| Ticket history | `ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/` |

## 7. Style guidance

Write the docs in a teaching style.

Good documentation should:

- explain why the static SQLite model exists before listing commands;
- use concrete examples with real symbols;
- distinguish user-facing concepts from implementation details;
- include diagrams where they reduce cognitive load;
- keep the README short and link to deeper references;
- avoid mentioning removed workflows except in architecture/history notes;
- avoid claiming a feature exists until it has been verified.

Avoid these patterns:

- Do not say "run the server" for the exported browser.
- Do not document `codebase-browser serve`; it was removed.
- Do not mention `precomputed.json`, `search.wasm`, or `wasm_exec.js` as active files except in negative tests or historical architecture notes.
- Do not use fake symbols in examples that readers are expected to run.
- Do not copy old ticket prose into public docs without checking whether it predates the sql.js architecture.

## 8. Definition of done for the documentation project

The website documentation is ready when:

- the README accurately explains the static sql.js product;
- there is a standalone markdown block reference;
- the DB reference is current and includes byte-offset warnings;
- the quick start has been run from scratch;
- every command example has been executed;
- every markdown block example has been rendered at least once;
- every SQL example has been run against a smoke DB;
- browser smoke checks confirm no `/api/*` requests;
- docs link to Glazed help pages with current slugs: `user-guide` and `db-reference`;
- any new help pages live under `pkg/doc` and are embedded by `pkg/doc/embed.go`;
- `go test ./...`, `pnpm -C ui run test`, and `pnpm -C ui run typecheck` pass after doc changes.

The final documentation should make the architecture feel simple: **index into SQLite, export static files, open SQLite in the browser, read and review code without a runtime server**.
