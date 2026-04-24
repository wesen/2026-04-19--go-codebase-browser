---
Title: Structured Query Concepts for Codebase Browser
Ticket: GCB-007
Status: active
Topics:
    - sqlite
    - cli
    - query-catalog
    - concepts
    - web-ui
    - go-minitrace
DocType: reference
Intent: design
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/query/query.go
      Note: Current raw SQL query command; useful but not yet a structured catalog
    - Path: queries/packages/package-counts.sql
      Note: Existing reusable SQL query without metadata or typed parameters
    - Path: queries/symbols/exported-functions.sql
      Note: Existing reusable SQL query that could become a first structured concept
    - Path: queries/symbols/most-referenced.sql
      Note: Existing reusable SQL query that could become a named CLI/web concept
    - Path: internal/sqlite/schema.go
      Note: SQLite schema queried by future concept commands
    - Path: internal/sqlite/query.go
      Note: Typed Go query helpers; complements but does not replace a SQL concept catalog
ExternalSources:
    - Path: /home/manuel/code/wesen/obsidian-vault/Projects/2026/04/21/PROJ - go-minitrace - JS Commands and Structured Query Catalog PR #6.md
      Note: Project report describing go-minitrace PR #6 query catalog architecture
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/catalog.go
      Note: Source-root catalog loader for SQL, JS, and alias command sources
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/types.go
      Note: Command spec model with verb/alias and SQL/JS runtime kinds
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/parse_sql.go
      Note: Sqleton-style SQL command parser with YAML preamble
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/parse_javascript.go
      Note: Scanner-first JS command parser using go-go-goja jsverbs
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/query/commands.go
      Note: Dynamically builds nested CLI verbs from the query catalog
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go
      Note: Hydrates typed parameters, renders SQL, loads runtime data, and executes commands
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go
      Note: HTTP API that exposes query commands and executes/render-previews them for the web UI
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/QueryEditorPage.tsx
      Note: Web UI consumes command metadata to render forms and run/preview commands
Summary: "Study report on adapting go-minitrace's structured query catalog/concept system to codebase-browser's SQLite backend."
WhatFor: "Use this document when deciding how to evolve codebase-browser from raw SQL CLI queries to named, typed query concepts that can become CLI verbs and web forms."
WhenToUse: "Before implementing a codebase-browser query catalog, command metadata parser, CLI command generator, or browser-side query form system."
---

# Structured Query Concepts for Codebase Browser

## Executive summary

Yes: the go-minitrace structured query-command system maps very well to `codebase-browser`, and it would materially improve the SQLite migration.

The current GCB-007 implementation gives us the necessary foundation: a generated `codebase.db`, a Go-side SQLite store, reusable SQL files, and a raw `codebase-browser query` command. That is enough for ad-hoc validation, but it is not enough for a durable query vocabulary. A raw SQL command answers, "Can I run this query?" A structured concept system answers, "What named questions can this application answer, what parameters do they accept, how do I discover them, and how can the same definition become both a CLI verb and a web form?"

The pattern from go-minitrace should therefore be adopted, with one important simplification: start with **SQL-only structured concepts** for codebase-browser. JavaScript-backed commands are powerful in go-minitrace because transcript analysis sometimes needs procedural post-processing. For codebase-browser, the first wave of concepts should be SQL templates over SQLite. That gives us typed parameters, render-only validation, aliases, discoverability, and web-form generation without introducing Goja or a second command runtime.

Recommended direction:

1. Keep the existing raw SQL command as `codebase-browser query sql` or `codebase-browser query raw`.
2. Add a structured catalog package, probably `internal/querycatalog/` or `internal/concepts/`.
3. Move reusable queries into a metadata-bearing catalog such as `concepts/` or `queries/commands/`.
4. Generate nested CLI verbs from that catalog: `codebase-browser query commands symbols exported-functions --package internal/server`.
5. Add `render-only` support so each concept can be validated on the CLI before execution.
6. Later expose the same command metadata to the web UI so forms can be generated from the exact same source definitions.

The short answer is: **we do not already cover this fully**. We cover raw SQL execution and plain `.sql` files. We do **not** yet cover named concepts, typed argument metadata, aliases, generated CLI verbs, render-only SQL preview, or web-form metadata. Those are the pieces to borrow from go-minitrace.

---

## What go-minitrace has

The go-minitrace system is best understood as a layer above SQL. It does not merely store query files; it turns query files into typed commands.

A command in go-minitrace has:

- a stable name
- a folder/path in a query catalog
- short and long help text
- typed flags and positional arguments
- tags and metadata
- a runtime kind: SQL or JavaScript
- optional aliases with prefilled defaults
- a rendered SQL preview path for the web UI
- a CLI command generated from the metadata
- a web form generated from the same metadata

That is the key idea: the query is not just text; it is a **command definition**.

### SQL commands

SQL command files use a sqleton-style YAML preamble inside a SQL comment. The parser lives in:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/parse_sql.go
```

The shape is:

```sql
/* sqleton
name: session-list
short: List sessions
flags:
  - name: limit
    type: int
    default: 10
*/
SELECT id, title, framework, created_at
FROM {{TABLE_NAME}}
LIMIT {{limit}};
```

The preamble is metadata. The body is a template. go-minitrace parses both into a `MinitraceCommandSpec`.

Important code paths:

- `LooksLikeSqletonSQLCommand(contents)` detects whether a `.sql` file is a structured command.
- `ParseSQLCommandSpec(path, contents)` splits the comment preamble from the body.
- YAML is decoded into `MinitraceCommandSpec`.
- The SQL body becomes `spec.Query`.
- `spec.Validate()` enforces required fields and runtime shape.

### JavaScript commands

go-minitrace also supports JS-backed commands using go-go-goja's scanner-first `jsverbs` system. The parser lives in:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/parse_javascript.go
```

JS commands declare sections and verbs:

```javascript
__section__("filters", {
  fields: {
    limit: { type: "int", default: 10 }
  }
});

function sessionList(filters) {
  const mt = require("minitrace");
  return mt.query(`
    SELECT id, title
    FROM ${mt.tableName}
    LIMIT ${filters.limit}
  `);
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions",
  fields: {
    filters: { bind: "filters" }
  }
});
```

The scanner extracts the command spec before runtime execution. That means the CLI and web UI can know the command name and fields without running arbitrary JS.

This is powerful, but codebase-browser does not need it as the first step. We can get 80% of the value from SQL-only command definitions.

### Aliases

Alias files are YAML files ending in `.alias.yaml` or `.alias.yml`. They wrap an existing command with a new name and prefilled defaults. The parser lives in:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/parse_alias.go
```

Conceptually:

```yaml
name: server-exported-functions
short: Exported funcs in internal/server
aliasFor: exported-functions
flags:
  package: github.com/wesen/codebase-browser/internal/server
```

Aliases are especially relevant for codebase-browser because many useful queries are parameterized versions of a generic concept:

- exported symbols in one package
- refs to one symbol
- public API surface for one package
- undocumented exported declarations in one package
- dependency edges touching one package

### Catalog loading

The catalog loader lives in:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/catalog.go
```

It walks one or more `SourceRoot`s, detects source kind by extension, parses command specs, compiles them into commands, and indexes them by path and name.

Important design points:

- `.sql` files become SQL commands if they have a sqleton preamble.
- `.js` / `.cjs` files become JS commands if they declare verbs.
- `.alias.yaml` files become aliases.
- multiple source roots are supported.
- first source root wins for duplicate source paths.
- duplicate logical command paths are rejected.
- commands are sorted by path.
- aliases are resolved after all commands load.

This is exactly the pattern codebase-browser should adopt, but with names adjusted away from "minitrace".

### CLI verb generation

The dynamic CLI generation lives in:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/query/commands.go
```

The important behavior is this:

- every command in the catalog becomes a Cobra command
- folder paths become nested command groups
- typed fields become CLI flags or arguments through Glazed
- command help comes from the query metadata

Example mapping:

```text
queries/overview/session-list.sql
  → go-minitrace query commands overview session-list
```

For codebase-browser, that could become:

```text
concepts/symbols/exported-functions.sql
  → codebase-browser query commands symbols exported-functions
```

or, if we choose the word `concepts` as the user-facing term:

```text
codebase-browser concepts symbols exported-functions
```

### Runtime execution

The command runtime lives in:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go
```

For SQL commands, it:

1. parses runtime settings
2. collects command values from Glazed fields
3. resolves aliases
4. loads data into DuckDB
5. renders SQL from the command template and values
6. validates read-only SQL
7. executes the query into a Glazed processor

For codebase-browser, steps 4 and 6 are simpler:

- no transcript archive has to be loaded at command runtime if `codebase.db` already exists
- read-only validation is still useful for web/API safety
- the runtime opens the SQLite database directly

### Web UI exposure

The web API and frontend are the most important reason to copy the pattern. The server exposes query commands over an API, and the frontend renders forms from command metadata.

Relevant files:

```text
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/proto/go_go_golems/minitrace/api/v1/query_commands.proto
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/types/queryCommand.ts
/home/manuel/code/wesen/corporate-headquarters/go-minitrace/web/src/pages/QueryEditorPage.tsx
```

The API exposes:

- `ListQueryCommandsResponse`
- `QueryCommand`
- `QueryCommandParam`
- `ExecuteQueryCommandRequest`
- `ExecuteQueryCommandResponse`

The frontend uses those to build:

- a command sidebar
- generated argument/flag forms
- a "Preview SQL" action
- a "Run command" action
- a results table

This is directly applicable to codebase-browser's future web UI. It means a query can be validated on the CLI first, then appear as a form in the browser without rewriting it.

---

## What codebase-browser already covers

As of the current GCB-007 implementation, codebase-browser has the lower-level pieces:

- `internal/sqlite/` creates and loads `codebase.db`
- `go generate ./internal/sqlite` generates the database
- `codebase-browser query` runs raw SQL against the DB
- `codebase-browser query -f <file.sql>` runs SQL from a file
- `--format json` provides machine-readable output
- `queries/` contains a few reusable SQL files

This is a good base, but it is intentionally primitive. The current query files are plain SQL; they have no metadata, no typed parameters, no defaults, no aliases, and no automatic UI representation.

The current command answers:

```text
Can I run this SQL text against codebase.db?
```

A structured concept catalog would answer:

```text
What named analyses does codebase-browser know about, what parameters do they accept, and how can those analyses be exposed consistently in the CLI and web UI?
```

Those are different layers.

---

## What is missing

The missing layer is a **query concept catalog**.

The term "concept" is a good fit for codebase-browser because the query definitions are not merely technical SQL files. They represent domain-level questions about a codebase:

- symbols
- packages
- files
- references
- public API surface
- dependency edges
- docs coverage
- dead or weakly referenced code
- test coverage hints
- architectural boundaries

A concept should have a stable name and typed inputs. SQL is just its implementation.

Missing capabilities:

1. **Metadata-bearing SQL files**
   - name
   - short description
   - long description
   - flags
   - arguments
   - tags
   - output expectations

2. **Typed parameter rendering**
   - string
   - int
   - bool
   - choice
   - string list
   - optional values

3. **Catalog loading**
   - embedded source roots
   - external source roots
   - duplicate path detection
   - aliases

4. **Generated CLI verbs**
   - nested folders become command groups
   - concept metadata becomes help text
   - fields become flags/arguments

5. **Render-only mode**
   - validate template expansion without executing
   - show final SQL
   - useful for debugging and web preview

6. **Read-only validation**
   - important if commands are exposed in a web UI
   - raw CLI can stay flexible, but concept execution should be read-only by default

7. **Web API shape**
   - list concepts
   - execute concept
   - preview/render concept

8. **Generated browser forms**
   - same field metadata as CLI
   - no duplicated form definitions in React

---

## Recommended codebase-browser shape

I recommend adding a new package, not overloading `internal/sqlite`.

Suggested package:

```text
internal/querycatalog/
```

or, if we want domain language throughout:

```text
internal/concepts/
```

I prefer `internal/concepts/` for product language and `querycatalog` for code-internal clarity. A compromise is:

```text
internal/concepts/       # public domain package name inside this repo
concepts/                # source files
```

### Proposed source tree

```text
concepts/
├── symbols/
│   ├── exported-functions.sql
│   ├── most-referenced.sql
│   ├── undocumented-exported.sql
│   └── refs-for-symbol.sql
├── packages/
│   ├── package-counts.sql
│   ├── public-api.sql
│   └── dependency-edges.sql
├── files/
│   ├── largest-files.sql
│   └── files-by-language.sql
└── aliases/
    └── server-exported-functions.alias.yaml
```

The existing plain `queries/` directory can either become this catalog or remain as a raw SQL scratch area. If we keep both, I would use:

```text
queries/       # raw SQL snippets and scratch files
concepts/      # structured, user-facing commands
```

### SQL concept file format

A codebase-browser concept can use the same sqleton-style preamble, but with a project-specific marker to avoid confusion:

```sql
/* codebase-browser concept
name: exported-functions
short: List exported functions and methods
long: |
  Shows exported functions and methods, optionally restricted to a package
  import path. Useful for reviewing public API surface.
tags: [symbols, api, exported]
flags:
  - name: package
    type: string
    help: Optional package import path substring
    default: ""
  - name: limit
    type: int
    help: Maximum number of rows
    default: 100
*/
SELECT
  s.name,
  s.kind,
  p.import_path AS package,
  f.path AS file,
  s.start_line
FROM symbols s
JOIN packages p ON p.id = s.package_id
JOIN files f ON f.id = s.file_id
WHERE s.exported = 1
  AND s.kind IN ('func', 'method')
  AND ({{ sqlString .package }} = '' OR p.import_path LIKE {{ sqlLike .package }})
ORDER BY p.import_path, s.name
LIMIT {{ .limit }};
```

This is intentionally close to go-minitrace so we can reuse ideas and maybe code patterns. The main differences are:

- no `{{TABLE_NAME}}` is needed; codebase-browser has stable tables
- SQLite is the engine, not DuckDB
- concepts should be read-only by default

### Generated CLI

A concept catalog should produce commands like:

```bash
codebase-browser query commands symbols exported-functions --package internal/server --limit 50
codebase-browser query commands symbols most-referenced --limit 25
codebase-browser query commands packages package-counts
codebase-browser query commands refs refs-for-symbol --symbol-id sym:...
```

Alternative product language:

```bash
codebase-browser concepts symbols exported-functions --package internal/server
```

I recommend keeping it under `query commands` initially because we already introduced `codebase-browser query`. Once stable, a top-level `concepts` alias can be added for discoverability.

### Render-only mode

Every concept should support render-only:

```bash
codebase-browser query commands symbols exported-functions \
  --package internal/server \
  --render-only
```

Output:

```sql
SELECT ...
WHERE s.exported = 1
  AND p.import_path LIKE '%internal/server%'
LIMIT 50;
```

This is crucial. It lets us validate the command metadata and template expansion before executing the query. It also maps directly to the web UI's "Preview SQL" button.

### Raw SQL command remains useful

The current raw command should stay, but be repositioned:

```bash
codebase-browser query sql "SELECT COUNT(*) FROM symbols"
codebase-browser query sql -f queries/my-queries/scratch.sql
```

Raw SQL is for exploration. Concepts are for durable, named workflows.

---

## How this supports the web UI later

The most important reason to add a catalog now is that it prevents divergence between CLI workflows and browser workflows.

Without a concept catalog, the web UI will eventually need handwritten forms:

- a form for symbol search
- a form for most-referenced symbols
- a form for refs by symbol
- a form for package counts
- a form for docs coverage

That duplicates metadata and validation. It also means a query that works on the CLI has to be manually ported into React.

With a catalog, the web UI asks the backend or local browser runtime:

```text
List concepts.
```

It receives:

```json
{
  "name": "exported-functions",
  "folder": "symbols",
  "path": "symbols/exported-functions",
  "shortDescription": "List exported functions and methods",
  "flags": [
    { "name": "package", "type": "string", "defaultJson": "\"\"" },
    { "name": "limit", "type": "int", "defaultJson": "100" }
  ],
  "rawSql": "SELECT ..."
}
```

Then the UI renders the form automatically. This is exactly what go-minitrace does in `QueryEditorPage.tsx`.

For codebase-browser's static browser build, there are two possible execution modes:

1. **Server mode**: the Go server exposes `/api/query-concepts` and `/api/query-concepts/{path}/execute` against SQLite.
2. **Static mode**: the browser loads concept metadata JSON plus SQL templates, renders SQL locally, and executes against `sql.js`.

The same catalog source files can feed both modes.

---

## SQL-only first, JS later if needed

go-minitrace supports both SQL and JS. Should codebase-browser copy both immediately?

Recommendation: **no**. Start SQL-only.

Reasons:

- SQLite already handles most codebase navigation queries.
- SQL concepts can run in Go and in the browser via `sql.js`.
- JS commands require a Goja runtime on the server/CLI side.
- JS commands do not automatically run in the static browser unless we design a separate browser-side JS sandbox.
- The current need is typed parameterization and query validation, not procedural analysis.

JS commands might become useful later for:

- multi-query workflows
- summarizing result sets
- graph shaping for visualizations
- preparing nested JSON structures for rich widgets
- calling host helpers that SQL cannot express cleanly

But adding JS now would increase surface area before the SQL concept layer is proven.

---

## Proposed implementation plan

### Phase 1: SQL concept catalog package

Create:

```text
internal/concepts/types.go
internal/concepts/source_kind.go
internal/concepts/parse_sql.go
internal/concepts/parse_alias.go
internal/concepts/catalog.go
internal/concepts/compiler.go
internal/concepts/render.go
```

Model after go-minitrace's `pkg/minitracecmd`, but simplify:

- `ConceptKind`: `verb`, `alias`
- `Runtime`: SQL only for now
- `ConceptSpec`: metadata + SQL body
- `Concept`: compiled runtime form
- `Catalog`: `Commands`, `ByPath`, `ByName`, `SourceRoots`

Acceptance test:

```go
catalog, err := concepts.LoadCatalog(...)
// should load symbols/exported-functions.sql
// should expose name, flags, path, tags
```

### Phase 2: render parameterized SQL

Add:

```go
func RenderConcept(concept *Concept, values map[string]any) (string, error)
```

Use a small template helper set:

- `sqlString`
- `sqlLike`
- `sqlStringIn`
- `sqlIntIn`

Acceptance test:

```go
sql, err := RenderConcept(exportedFunctions, map[string]any{
    "package": "internal/server",
    "limit": 50,
})
// SQL contains LIKE '%internal/server%'
// SQL contains LIMIT 50
```

### Phase 3: generated CLI verbs

Add:

```text
cmd/codebase-browser/cmds/query/commands.go
cmd/codebase-browser/cmds/query/command_runtime.go
```

Then register under the existing query command:

```bash
codebase-browser query commands symbols exported-functions --package internal/server
```

Acceptance tests:

- catalog command appears in `--help`
- flags are parsed as typed values
- `--render-only` prints SQL
- execution returns rows from generated `codebase.db`

### Phase 4: migrate existing `.sql` files into concepts

Turn these into structured concepts:

- `queries/packages/package-counts.sql`
- `queries/symbols/exported-functions.sql`
- `queries/symbols/most-referenced.sql`
- `queries/refs/refs-for-symbol.sql`

The old files can either move to `concepts/` or remain as raw examples. I recommend moving user-facing ones to `concepts/` and leaving `queries/my-queries/` for scratch SQL.

### Phase 5: API shape for server/web mode

Later add endpoints similar to go-minitrace:

```text
GET  /api/query-concepts
POST /api/query-concepts/{path...}/execute
```

Request:

```json
{
  "values": { "package": "internal/server", "limit": 50 },
  "renderOnly": false
}
```

Response:

```json
{
  "renderedSql": "SELECT ...",
  "columns": ["name", "kind", "package"],
  "rows": [{ "name": "handleIndex", "kind": "method" }],
  "durationMs": 2,
  "rowCount": 1
}
```

### Phase 6: static browser mode

For the static SQLite/browser path, emit a catalog artifact:

```text
dist/concepts.json
```

or store concept metadata in SQLite:

```sql
concepts(path, name, folder, short, long, flags_json, raw_sql)
```

The browser can then:

1. load concepts metadata
2. render a form
3. render SQL from values
4. execute SQL against `sql.js`
5. show results

The server-side API and browser-side static mode should use the same concept source files.

---

## Proposed first codebase-browser concepts

The first catalog should be small but useful.

### `symbols/exported-functions`

Parameters:

- `package` string, optional
- `limit` int, default 100

Answers:

- what is the public function/method surface?

### `symbols/most-referenced`

Parameters:

- `kind` choice or string, optional
- `limit` int, default 50

Answers:

- which symbols are most central?

### `symbols/undocumented-exported`

Parameters:

- `package` string, optional
- `limit` int, default 100

Answers:

- which exported declarations need docs?

### `refs/for-symbol`

Parameters:

- `symbol-id` string, required
- `direction` choice: `incoming`, `outgoing`, `both`

Answers:

- what depends on this symbol and what does it depend on?

### `packages/package-counts`

Parameters:

- `language` choice: empty, `go`, `ts`

Answers:

- how large is each package by file/symbol count?

### `files/largest-files`

Parameters:

- `language` choice: empty, `go`, `ts`
- `limit` int, default 50

Answers:

- which files are largest and likely worth splitting?

---

## Design decisions for codebase-browser

### Decision 1: call them concepts or commands?

Use **concepts** in user-facing prose and maybe in the web UI. Use **commands** in the CLI path if it helps fit the existing `query` namespace.

Recommended CLI:

```text
codebase-browser query commands ...
```

Recommended UI label:

```text
Concepts
```

This mirrors the idea that a concept is a named question, while a command is how you invoke it.

### Decision 2: SQL templates or prepared statements?

For raw execution, prepared statements are ideal. For concept rendering and preview, SQL templates are useful because users want to see the final SQL.

Use templates, but keep helper functions safe and explicit. Avoid arbitrary string interpolation where possible. Prefer helpers like:

```text
{{ sqlString .package }}
{{ sqlLike .package }}
{{ sqlStringIn .kinds }}
{{ sqlIntIn .lines }}
```

### Decision 3: read-only validation?

Yes for concepts. Maybe not for raw CLI.

- `codebase-browser query sql` can remain a power-user command.
- `codebase-browser query commands ...` should be read-only by default.
- web execution must be read-only.

### Decision 4: JS runtime?

Not now. SQL-only first.

### Decision 5: embed concepts or load from disk?

Both eventually.

Initial implementation can load from the working tree. The durable design should support:

1. embedded built-in concepts
2. external concept repositories
3. maybe a local `concepts/my-concepts/` directory

This mirrors go-minitrace's repository precedence model.

---

## Comparison table

| Capability | go-minitrace | codebase-browser today | Recommended codebase-browser next |
|---|---|---|---|
| Raw SQL CLI | Yes, DuckDB | Yes, SQLite | Keep, rename/namespace as raw SQL if needed |
| Reusable SQL files | Yes | Yes, plain SQL | Convert user-facing ones to concepts |
| Metadata preamble | Yes, sqleton | No | Add SQL concept preamble |
| Typed parameters | Yes | No | Add Glazed field definitions from metadata |
| Dynamic CLI verbs | Yes | No | Add `query commands ...` |
| Aliases | Yes | No | Add after base concept loading |
| JS command runtime | Yes | No | Defer |
| Render-only preview | Yes | No | Add early |
| Web command list API | Yes | No | Add when server/web integration resumes |
| Web generated forms | Yes | No | Add later from same metadata |
| Static browser execution | Not the main focus | Planned via sql.js | Use concept metadata + sql.js |

---

## Answer to the original question

> Is that something we can add (or maybe already cover) in this application?

We can definitely add it, and it is a natural next layer for GCB-007. We only partially cover it today.

What we already cover:

- SQLite database generation
- raw SQL execution from CLI
- plain SQL files
- typed Go helper methods for some query patterns

What we do not yet cover:

- structured query metadata
- typed CLI parameters for SQL files
- concepts as named domain questions
- aliases/prefilled variants
- generated CLI verbs from query definitions
- render-only SQL preview
- web-form generation from the same definitions

The recommended next implementation slice is therefore:

```text
Add a SQL-only concept catalog for codebase-browser, modeled after go-minitrace's minitracecmd package, and expose it through generated CLI verbs under `codebase-browser query commands`.
```

That would let us validate SQL queries easily on the CLI with argument parametrization now, and later use the same metadata to build forms in the web UI.

---

## Suggested task additions

Add a new phase to GCB-007:

### Phase 5 — Structured query concepts

- [ ] Create `internal/concepts/` with SQL concept parsing and catalog loading.
- [ ] Define the codebase-browser SQL concept preamble format.
- [ ] Add render helpers for `sqlString`, `sqlLike`, `sqlStringIn`, and `sqlIntIn`.
- [ ] Add `codebase-browser query commands` dynamic CLI generation.
- [ ] Add `--render-only` to concept commands.
- [ ] Convert existing reusable SQL files into concept files.
- [ ] Add aliases for common package/symbol views.
- [ ] Add tests for catalog loading, rendering, aliases, CLI execution, and render-only preview.
- [ ] Later: expose the catalog over API for web-generated forms.

This phase should happen before browser-side SQLite integration, because it will give the browser a better abstraction to consume than raw SQL files.
