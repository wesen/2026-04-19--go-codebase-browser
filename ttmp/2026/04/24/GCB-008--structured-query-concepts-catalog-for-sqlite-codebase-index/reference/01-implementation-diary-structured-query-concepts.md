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

## Step 3: Add the first structured concept files

This step created the first user-facing `concepts/` catalog. The files are no longer plain SQL snippets; each one now has a `codebase-browser concept` preamble with name, help text, tags, and typed parameters.

These files are the bridge between raw SQL exploration and generated CLI/web interfaces. They can be loaded by `internal/concepts` today and will become dynamic CLI verbs in the next step.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementing GCB-008 by adding concrete query concepts after the parser/catalog package.

**Inferred user intent:** Turn the useful SQL examples into durable named analyses that can be invoked and validated consistently.

**Commit (code):** 17272ea3f0b4d2fdf59bc6f7ef3f5495d548269d — "Add initial SQL concept files"

### What I did

- Created `concepts/packages/package-counts.sql`.
- Created `concepts/symbols/exported-functions.sql`.
- Created `concepts/symbols/most-referenced.sql`.
- Created `concepts/refs/refs-for-symbol.sql`.
- Added `internal/concepts/catalog_files_test.go` to verify the repo catalog loads those files.

### Why

The catalog package needs real source files before the CLI can expose useful verbs. These concepts cover the first practical navigation and inspection questions: package size, exported API, central symbols, and symbol refs.

### What worked

```bash
gofmt -w internal/concepts/catalog_files_test.go
go test ./internal/concepts -count=1
```

### What didn't work

N/A

### What I learned

The concept file format is readable enough for manual editing. The `value` helper keeps parameter access explicit, especially for names like `symbol-id`.

### What was tricky to build

The `refs-for-symbol` concept needed `LEFT JOIN` for source/target symbols because refs can point to external symbols that are not local rows in `symbols`.

### What warrants a second pair of eyes

- Whether `refs-for-symbol` should show raw external symbol IDs more prominently when the target/source name is null.
- Whether `kind` in `most-referenced` should be a `choice` instead of free string.

### What should be done in the future

- Add more concepts for undocumented exported symbols, largest files, and package dependency edges.
- Add aliases once alias support exists.

### Code review instructions

Review:

- `concepts/`
- `internal/concepts/catalog_files_test.go`

Validate with:

```bash
go test ./internal/concepts -count=1
```

## Step 4: Add dynamic concept CLI commands

This step connected the concept catalog to the application CLI. `codebase-browser query` still supports raw SQL, but it now also has a `commands` subtree generated from the `concepts/` catalog.

The important result is that reusable SQL concepts are now executable as typed commands. They also support `--render-only`, which gives us the same validation loop that made the go-minitrace system useful before web UI forms existed.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the initial implementation by exposing concepts as CLI verbs and validating them against the generated SQLite DB.

**Inferred user intent:** Make parameterized SQL concepts usable from the command line now, so later website work can consume the same abstraction.

**Commit (code):** 620002d8cf26ac7a52f0cc37e968915c4c3513c6 — "Add concept query CLI commands"

### What I did

- Added `cmd/codebase-browser/cmds/query/commands.go`.
- Added `codebase-browser query commands`.
- Loaded the `concepts/` catalog at CLI registration time.
- Generated nested Cobra command groups from concept folders.
- Added one leaf command per concept.
- Mapped concept params to CLI flags.
- Added `--render-only` per concept command.
- Reused the existing SQLite DB opening and SQL result rendering path.
- Changed `--db` and `--format` to persistent query flags so concept subcommands inherit them.
- Preserved existing raw SQL behavior.

### Why

The CLI is the fastest place to validate concept metadata and SQL templates. Once these commands are stable, the website can expose the same concepts as generated forms instead of hand-writing each query UI.

### What worked

The following commands succeeded:

```bash
gofmt -w cmd/codebase-browser/cmds/query
go test ./cmd/codebase-browser/cmds/query ./cmd/codebase-browser ./internal/concepts -count=1
go generate ./internal/sqlite
go run ./cmd/codebase-browser query commands packages package-counts --render-only | head -30
go run ./cmd/codebase-browser query commands packages package-counts | head -8
go run ./cmd/codebase-browser query commands symbols exported-functions --package internal/server --limit 5
go run ./cmd/codebase-browser query commands symbols most-referenced --limit 5
go run ./cmd/codebase-browser query commands refs refs-for-symbol --symbol-id 'sym:github.com/wesen/codebase-browser/internal/indexer.func.SymbolID' --limit 5 --render-only | head -30
go run ./cmd/codebase-browser query commands refs refs-for-symbol --symbol-id 'sym:github.com/wesen/codebase-browser/internal/indexer.func.SymbolID' --limit 5
go test ./... -count=1
go run ./cmd/codebase-browser query "SELECT COUNT(*) AS symbols FROM symbols"
```

### What didn't work

The first implementation attempt used `StringVar`, `IntVar`, and related pflag helpers with custom map-backed value types. That was the wrong API because those helpers require pointers to concrete Go primitives, not custom `pflag.Value` implementations. I rewrote the flag registration to use `flags.Var` / `flags.VarP` with explicit `pflag.Value` implementations.

One smoke command used `head` to truncate output and produced an expected broken-pipe signal after the consumer closed the pipe:

```text
signal: broken pipe
```

The command itself worked; the signal was from truncating output for display.

### What I learned

Cobra/pflag supports dynamic typed flags well, but custom storage should use the `pflag.Value` interface directly. That makes it easy to hydrate a `map[string]any` for concept rendering.

### What was tricky to build

The biggest sharp edge was preserving raw SQL behavior while adding a subcommand tree. `codebase-browser query "SELECT ..."` still works, while `codebase-browser query commands ...` dispatches to the generated subtree. Shared flags such as `--db` and `--format` had to become persistent query flags so subcommands inherit them.

### What warrants a second pair of eyes

- The dynamic flag implementation is intentionally simple and should be reviewed for edge cases around required ints and lists.
- The catalog currently loads from the working-tree `concepts/` directory, not embedded assets. That is fine for the CLI implementation slice but should be revisited for released binaries.
- Choice validation happens during concept render hydration rather than at Cobra flag parse time.

### What should be done in the future

- Add embedded concept assets for released binaries.
- Add alias support after the SQL-only path is stable.
- Add API endpoints and browser-generated forms after the website work resumes.

### Code review instructions

Review:

- `cmd/codebase-browser/cmds/query/query.go`
- `cmd/codebase-browser/cmds/query/commands.go`

Validate with:

```bash
go generate ./internal/sqlite
go run ./cmd/codebase-browser query commands symbols exported-functions --package internal/server --limit 5
go run ./cmd/codebase-browser query commands symbols exported-functions --package internal/server --limit 5 --render-only
go run ./cmd/codebase-browser query "SELECT COUNT(*) AS symbols FROM symbols"
go test ./... -count=1
```

## Step 5: Embed built-in concepts and add external concept repositories

This step made the concept system usable outside the repository checkout and aligned its repository loading behavior with the go-minitrace pattern. Built-in SQL concepts are now embedded into the binary, while external concept repositories can be layered on top through a repeated CLI flag or an environment variable.

The important behavior change is precedence: external repositories are mounted first, embedded concepts are mounted last, and the first concept path wins. That lets a team override a built-in concept by shipping a file with the same relative path in its own repository.

### Prompt Context

**User prompt (verbatim):** "do embedded and external concept repositories. no goja." and "look at how loupedeck/go-minitrace/jsverbs does it"

**Assistant interpretation:** Follow go-minitrace’s repository-mounting pattern for SQL concepts only: add embedded built-ins, external repository roots, env/flag discovery, and override precedence, but do not add any JavaScript runtime.

**Inferred user intent:** Make concept commands shippable in released binaries and extensible by teams/projects without reopening the Goja scope.

**Commit (code):** 2ad27ddb0aabbef9699cdec337d375c41427ad29 — "Add embedded and external concept repositories"

### What I did

- Added `concepts/embed.go` so the checked-in built-in SQL concept files are compiled into the binary with `go:embed`.
- Added `internal/concepts/assets.go` with `EmbeddedSourceRoot()` and `LoadEmbeddedCatalog()`.
- Added `internal/concepts/repositories.go`.
- Added repository discovery constants:
  - `CODEBASE_BROWSER_CONCEPT_REPOSITORIES`
  - `--concept-repository`
- Added repository path normalization, env loading, CLI flag extraction from raw `os.Args`, and source-root construction.
- Mounted external repositories before the embedded catalog.
- Changed catalog loading so duplicate concept paths no longer fail; the first repository wins and later duplicates are ignored.
- Generalized `SourceRoot` to work with `fs.FS` so both embedded and on-disk repositories use the same loader.
- Recorded `SourceRoot` on compiled concepts for debugging and tests.
- Updated the query CLI to load concepts through `LoadConfiguredCatalog(...)` instead of directly reading `./concepts`.
- Updated `cmd/codebase-browser/main.go` to pre-extract `--concept-repository` values from raw args before building the dynamic command tree, mirroring go-minitrace’s pattern.
- Added tests for:
  - embedded catalog loading
  - repository root ordering
  - repository flag parsing
  - env repository discovery
  - override precedence

### Why

Without embedding, released binaries would only work when run beside a checked-out `concepts/` directory. Without external repositories, teams would have to patch the main repo just to add or override concepts. This step removes both limitations while keeping the system SQL-only.

### What worked

The following commands succeeded:

```bash
gofmt -w concepts/embed.go internal/concepts/*.go cmd/codebase-browser/main.go cmd/codebase-browser/cmds/query/*.go
go test ./internal/concepts ./cmd/codebase-browser/cmds/query ./cmd/codebase-browser -count=1
go test ./... -count=1
go generate ./internal/sqlite
```

Embedded built-ins rendered correctly:

```bash
go run ./cmd/codebase-browser query commands symbols exported-functions --package internal/server --limit 2 --render-only
```

An external repository overrode the built-in `symbols/exported-functions` concept:

```bash
go run ./cmd/codebase-browser --concept-repository "$tmp_repo" query commands symbols exported-functions --limit 2 --render-only
```

Environment-based repository discovery also worked:

```bash
CODEBASE_BROWSER_CONCEPT_REPOSITORIES="$tmp_repo" \
  go run ./cmd/codebase-browser query commands team team-check --render-only
```

### What didn't work

Nothing failed in the final implementation slice, but one design point changed while coding: the existing catalog loader treated duplicate concept paths as hard errors. That would prevent the override model used by go-minitrace. I changed the loader so the first mounted repository wins and later duplicates are ignored.

### What I learned

The repository behavior from go-minitrace ports cleanly even without any JS runtime. The important parts are not the JavaScript engine; they are the mounting order, embedded defaults, and the early extraction of repository flags before Cobra builds the command tree.

### What was tricky to build

The subtle part was timing. Dynamic command trees are built before Cobra finishes parsing flags, so the query CLI cannot wait until normal flag parsing to discover `--concept-repository`. The fix is the same pattern go-minitrace uses: scan `os.Args[1:]` early, then build the command tree with those paths already known.

### What warrants a second pair of eyes

- Whether `CODEBASE_BROWSER_CONCEPT_REPOSITORIES` and `--concept-repository` are the final names we want long term.
- Whether later duplicate concepts should emit debug logs when they are ignored.
- Whether we eventually want app-config support in addition to flags and env, as go-minitrace does.

### What should be done in the future

- Add alias repositories on top of this same loading model.
- Reuse the same repository loader for future HTTP/API exposure of concepts.
- Keep JS/Goja out unless SQL concepts prove insufficient.

### Code review instructions

Review:

- `concepts/embed.go`
- `internal/concepts/assets.go`
- `internal/concepts/repositories.go`
- `internal/concepts/catalog.go`
- `internal/concepts/repositories_test.go`
- `cmd/codebase-browser/main.go`
- `cmd/codebase-browser/cmds/query/commands.go`
- `cmd/codebase-browser/cmds/query/query.go`

Validate with:

```bash
go test ./internal/concepts -count=1
go test ./... -count=1
go run ./cmd/codebase-browser query commands symbols exported-functions --package internal/server --limit 2 --render-only

TMP_REPO=$(mktemp -d)
mkdir -p "$TMP_REPO/symbols"
cat > "$TMP_REPO/symbols/exported-functions.sql" <<'EOSQL'
/* codebase-browser concept
name: exported-functions
short: Override exported functions
*/
SELECT 'override' AS source;
EOSQL

go run ./cmd/codebase-browser --concept-repository "$TMP_REPO" query commands symbols exported-functions --render-only
CODEBASE_BROWSER_CONCEPT_REPOSITORIES="$TMP_REPO" go run ./cmd/codebase-browser query commands symbols exported-functions --render-only
```
