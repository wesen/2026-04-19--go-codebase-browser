---
Title: Implementation Diary Structured Query Concepts
Ticket: GCB-008
Status: active
Topics:
    - sqlite
    - cli
    - concepts
    - query-catalog
DocType: reference
Intent: diary
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation diary for GCB-008 structured query concepts."
WhatFor: "Record each implementation step, validation commands, failures, and review notes."
WhenToUse: "Use when continuing or reviewing GCB-008."
---

# Diary

## Goal

Record the task-by-task implementation of SQL-only structured query concepts for `codebase-browser`.

## Step 1: Create ticket, guide, tasks, and diary scaffold

This step created the GCB-008 workspace and turned the go-minitrace concept-catalog research into an implementation plan. The ticket scopes the first implementation to SQL-only concepts: metadata-bearing SQL files, typed parameters, generated CLI verbs, and render-only SQL preview.

The guide intentionally excludes aliases, JavaScript commands, HTTP APIs, and generated web forms from the first pass. Those remain future phases after the CLI concept path is proven.

### Prompt Context

**User prompt (verbatim):** "Ok, make a new ticket that tackles that and write an implementation guide, and a set of tasks for building it. Then build task by task, committing at appropriate intervals, keeping a diary. once we're all done, we'll get back to this ticket and continue building out our website"

**Assistant interpretation:** Create a new ticket for go-minitrace-style structured query concepts, write docs/tasks, then implement the feature incrementally with commits and diary updates.

**Inferred user intent:** Build a durable CLI-validated concept layer now so the later website can consume typed query metadata rather than raw SQL.

**Commit (docs):** pending

### What I did

- Created `ttmp/2026/04/24/GCB-008--structured-query-concepts-catalog-for-sqlite-codebase-index/`.
- Added `index.md`, `README.md`, `changelog.md`, and `tasks.md`.
- Added `design-doc/01-structured-query-concepts-implementation-guide.md`.
- Added this diary scaffold.

### Why

The concept catalog is a new scope of work. Keeping it separate from GCB-007 lets us return to website work later with a cleaner boundary: GCB-007 owns SQLite database support; GCB-008 owns structured query concepts over that database.

### What worked

- Ticket workspace and implementation guide were created.

### What didn't work

N/A

### What I learned

The smallest useful slice is SQL-only concepts. That gives us CLI validation and future web forms without adopting go-minitrace's JS runtime yet.

### What was tricky to build

The main scoping issue was deciding what *not* to include. Aliases and JS commands are useful, but they would make the first pass too broad.

### What warrants a second pair of eyes

- Whether the concept file marker should be `codebase-browser concept` or a more generic `sqleton` marker.
- Whether the user-facing CLI should be `query commands` or top-level `concepts`.

### What should be done in the future

- Implement `internal/concepts/`.
- Convert the first SQL files into concept files.
- Add dynamic CLI command generation.

### Code review instructions

Review the ticket docs first:

- `tasks.md`
- `design-doc/01-structured-query-concepts-implementation-guide.md`

### Technical details

The planned command shape is:

```bash
codebase-browser query commands symbols exported-functions --package internal/server --limit 50
codebase-browser query commands symbols exported-functions --package internal/server --limit 50 --render-only
```

## Step 2: Add the SQL concept catalog package

This step added the first code for GCB-008: an `internal/concepts` package that can parse metadata-bearing SQL files, compile them into catalog entries, hydrate typed parameter values, and render SQL templates with safe-ish helper functions.

The implementation is SQL-only by design. It establishes the catalog layer that dynamic CLI verbs will use in later steps.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Begin implementing the structured query concepts feature in focused code slices.

**Inferred user intent:** Build the reusable concept abstraction before wiring it into the CLI or website.

**Commit (code):** 7cb3b381a8169be97df80865be9eca99296d51bc — "Add SQL concept catalog package"

### What I did

- Created `internal/concepts/`.
- Added concept types: `Param`, `ConceptSpec`, `Concept`, `SourceRoot`, and `Catalog`.
- Added validation for required concept metadata and supported parameter types.
- Added SQL concept preamble detection for `/* codebase-browser concept ... */`.
- Added SQL concept parsing with YAML metadata and SQL body splitting.
- Added catalog loading from filesystem directories.
- Added concept compilation into `ByPath` and `ByName` indexes.
- Added value hydration and type coercion for string, int, bool, choice, stringList, and intList params.
- Added SQL template rendering with `value`, `sqlString`, `sqlLike`, `sqlStringIn`, and `sqlIntIn` helpers.
- Added tests for parsing, loading, rendering, and required parameter validation.

### Why

The concept catalog needs to exist independently of the CLI so it can later be reused by server APIs and static-browser metadata generation. This slice gives us the core model without coupling it to Cobra or SQLite execution.

### What worked

The following commands passed:

```bash
gofmt -w internal/concepts
go test ./internal/concepts -count=1
go test ./internal/concepts ./internal/sqlite -count=1
go test ./... -count=1
```

### What didn't work

The first focused test run failed because `sqlIntIn` returned only a string in the empty-list case even though the function signature returns `(string, error)`:

```text
internal/concepts/render.go:216:10: not enough return values
	have (string)
	want (string, error)
```

I fixed it by returning `"0", nil`.

### What I learned

Keeping concepts SQL-only makes the implementation small and portable. The package currently has no dependency on SQLite or Cobra, which is good for later reuse in the web path.

### What was tricky to build

The template API needed a stable way to access parameter names like `symbol-id`, which are not convenient as Go template dot fields. The package therefore exposes a `value` helper, so templates can write `{{ value "symbol-id" }}` rather than relying on dot syntax.

### What warrants a second pair of eyes

- The SQL quoting helpers are intentionally simple. They are appropriate for generated templates, but they should be reviewed before exposing concept execution broadly through a web API.
- The `Default any` YAML handling should be reviewed with more parameter types and real concept files.
- The catalog currently loads from OS directories only; embedding can be added later.

### What should be done in the future

- Add concept files under `concepts/`.
- Wire concepts into `codebase-browser query commands`.
- Add dedicated duplicate path tests.

### Code review instructions

Start with:

- `internal/concepts/types.go`
- `internal/concepts/parse_sql.go`
- `internal/concepts/catalog.go`
- `internal/concepts/render.go`
- `internal/concepts/concepts_test.go`

Validate with:

```bash
go test ./internal/concepts -count=1
go test ./... -count=1
```
