---
Title: Structured Query Concepts Implementation Guide
Ticket: GCB-008
Status: active
Topics:
    - sqlite
    - cli
    - query-catalog
    - concepts
DocType: design-doc
Intent: implementation-guide
Owners: []
RelatedFiles:
    - Path: cmd/codebase-browser/cmds/query/query.go
      Note: Existing raw SQL query command that will gain a `commands` subcommand
    - Path: internal/sqlite/schema.go
      Note: Database schema queried by concepts
    - Path: queries/symbols/exported-functions.sql
      Note: Existing plain SQL file that will become a structured concept
ExternalSources:
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd
      Note: Reference catalog implementation
Summary: "Implementation guide for adding SQL-only structured query concepts to codebase-browser."
---

# Structured Query Concepts Implementation Guide

## Executive summary

GCB-008 turns reusable SQLite queries into **concepts**: named, typed, documented questions about a codebase. A concept is implemented as a SQL template, but it is exposed as a CLI verb and later as a web UI form.

This ticket deliberately starts smaller than go-minitrace. go-minitrace supports SQL commands, JavaScript commands, aliases, external repositories, web APIs, and generated React forms. For codebase-browser we start with:

- SQL concept files only
- typed parameters in a YAML preamble
- a Go catalog loader
- SQL template rendering
- generated CLI verbs under `codebase-browser query commands`
- render-only SQL preview

Aliases, JavaScript concepts, and website forms come later.

## Why this matters

GCB-007 gave us this command:

```bash
codebase-browser query "SELECT COUNT(*) FROM symbols"
```

That is useful, but it is raw SQL. Raw SQL is excellent for exploration and poor as a product interface. Once a query becomes reusable, we need a name, help text, parameters, defaults, and validation.

For example, this should become possible:

```bash
codebase-browser query commands symbols exported-functions \
  --package internal/server \
  --limit 50
```

And this should be possible before execution:

```bash
codebase-browser query commands symbols exported-functions \
  --package internal/server \
  --limit 50 \
  --render-only
```

The same metadata can later produce a web form with fields for `package` and `limit`.

## Concept file format

Concepts live under `concepts/`.

A SQL concept file starts with a comment preamble:

```sql
/* codebase-browser concept
name: exported-functions
short: List exported functions and methods
long: |
  Shows exported functions and methods, optionally restricted by package
  import path substring.
tags: [symbols, exported, api]
params:
  - name: package
    type: string
    help: Optional package import path substring
    default: ""
  - name: limit
    type: int
    help: Maximum number of rows
    default: 100
*/
SELECT ...
```

The SQL body is a Go `text/template` with helper functions:

- `value "name"` returns a hydrated parameter value
- `sqlString value` returns a quoted SQL string literal
- `sqlLike value` returns a quoted `%value%` LIKE pattern
- `sqlStringIn value` returns a comma-separated quoted string list
- `sqlIntIn value` returns a comma-separated integer list

Recommended style:

```sql
AND ({{ sqlString (value "package") }} = ''
     OR p.import_path LIKE {{ sqlLike (value "package") }})
LIMIT {{ value "limit" }};
```

## Package design

Create `internal/concepts/`.

Suggested files:

```text
internal/concepts/types.go
internal/concepts/parse_sql.go
internal/concepts/catalog.go
internal/concepts/render.go
internal/concepts/types_test.go
internal/concepts/parse_sql_test.go
internal/concepts/catalog_test.go
internal/concepts/render_test.go
```

### Core types

```go
type Param struct {
    Name      string
    Type      string
    Help      string
    Required  bool
    Default   any
    Choices   []string
    ShortFlag string
}

type ConceptSpec struct {
    Name   string
    Short  string
    Long   string
    Tags   []string
    Params []Param
    Query  string
}

type Concept struct {
    Name       string
    Folder     string
    Path       string
    Short      string
    Long       string
    Tags       []string
    Params     []Param
    Query      string
    SourcePath string
}

type Catalog struct {
    Concepts []*Concept
    ByPath   map[string]*Concept
    ByName   map[string]*Concept
}
```

## CLI integration

Keep the existing raw SQL command. Add a `commands` subcommand below it.

```text
codebase-browser query "SELECT ..."              # still raw SQL
codebase-browser query -f queries/foo.sql         # still raw SQL file
codebase-browser query commands symbols exported-functions --limit 20
```

Implementation location:

```text
cmd/codebase-browser/cmds/query/query.go
cmd/codebase-browser/cmds/query/commands.go
```

The dynamic command generator should:

1. load `concepts/`
2. create nested Cobra groups from concept folders
3. create a leaf command for each concept
4. create typed flags from concept params
5. add `--render-only`
6. on execution, hydrate defaults + flag values
7. render SQL
8. if render-only, print SQL
9. otherwise open SQLite and execute through the existing raw SQL runner

## First concepts

Implement these first:

- `concepts/packages/package-counts.sql`
- `concepts/symbols/exported-functions.sql`
- `concepts/symbols/most-referenced.sql`
- `concepts/refs/refs-for-symbol.sql`

## Validation commands

After implementation:

```bash
go test ./internal/concepts -count=1
go test ./cmd/codebase-browser/cmds/query ./internal/concepts ./internal/sqlite -count=1
go test ./... -count=1
go generate ./internal/sqlite

go run ./cmd/codebase-browser query commands packages package-counts --render-only
go run ./cmd/codebase-browser query commands packages package-counts

go run ./cmd/codebase-browser query commands symbols exported-functions --package internal/server --limit 10
go run ./cmd/codebase-browser query commands symbols most-referenced --limit 10
go run ./cmd/codebase-browser query commands refs refs-for-symbol --symbol-id 'sym:...' --render-only
```

## Implementation sequence

1. Ticket docs and guide.
2. `internal/concepts` parser/catalog/rendering.
3. Concept SQL files.
4. Dynamic CLI commands.
5. Validation and diary.

## Review risks

- SQL quoting helpers must be safe enough for generated SQL templates.
- Typed parameter hydration must be deterministic and unsurprising.
- Dynamic CLI command paths must avoid collisions.
- The raw SQL command must keep working.
- The design should not accidentally commit generated `codebase.db`.
