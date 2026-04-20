---
Title: Go Codebase Browser — Analysis and Implementation Guide
Ticket: GCB-001
Status: active
Topics:
    - go-ast
    - codebase-browser
    - embedded-web
    - react-frontend
    - storybook
    - glazed
    - rtk-query
    - documentation-tooling
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../.claude/skills/glazed-command-authoring/SKILL.md
      Note: Glazed conventions referenced throughout §8 (CLI)
    - Path: ../../../../../../../../../.claude/skills/go-web-frontend-embed/SKILL.md
      Note: Embed + SPA fallback + go generate pipeline referenced in §5 §7.7 §12
    - Path: ../../../../../../../../../.claude/skills/react-modular-themable-storybook/SKILL.md
      Note: Parts/tokens + Storybook contract referenced in §10
    - Path: ../../../../../../../../../.claude/skills/react-modular-themable-storybook/references/parts-and-tokens.md
      Note: Specific parts/tokens naming guidance in §10.3
ExternalSources: []
Summary: End-to-end design for a Go app that indexes its own source at build time (go/ast, go/packages, go/types, go/analysis), embeds both index and source in a single binary, and exposes a themable React + RTK-Query web UI (with Storybook coverage) that renders a navigable codebase browser and supports documentation pages which embed live source snippets by stable symbol ID.
LastUpdated: 2026-04-20T00:00:00Z
WhatFor: Design and implementation guide for GCB-001 — the self-embedding, self-documenting Go codebase browser.
WhenToUse: Read before implementing the indexer, the Go web server, the REST API, the React/RTK-Query frontend, or the docs-with-embedded-source renderer. Also read when reviewing PRs that touch the index schema, the embedded FS contract, or the themable frontend package.
---




# Go Codebase Browser — Analysis and Implementation Guide

## 1. Executive summary

This ticket defines a Go application that **indexes its own source code at build time** and **ships both the index and the source** inside a single binary. The binary then **serves a themable web UI** (React + Vite + RTK-Query, with Storybook) that renders:

1. a navigable tree of packages, files, and symbols,
2. rich symbol pages (signature, doc comment, source snippet, cross-references), and
3. long-form documentation pages that can **embed live source constructs by stable symbol ID**, guaranteeing the rendered snippet always corresponds to the exact source shipped in the binary.

The design has four pillars:

1. **Build-time indexing** using `go/packages` (+ `go/ast`, `go/types`, and optional `go/analysis` passes) to produce a deterministic JSON index keyed by stable symbol IDs.
2. **A single-binary distribution**: the index JSON, the original source tree, and the compiled SPA are packaged via `go:embed` so there is no runtime filesystem dependency.
3. **A Glazed CLI** (`codebase-browser index`, `codebase-browser serve`, `codebase-browser symbol`, `codebase-browser doc render`) matching the `glazed-command-authoring` conventions.
4. **A themable React package** (`@codebase-browser/ui`) styled via `data-part` attributes + CSS variables and documented with Storybook stories for default, themed, and unstyled variants, with all data access flowing through RTK-Query.

The "self-embedding" property is load-bearing: because the binary is the single source of truth for both the source and the index, the snippet shown in a doc page cannot drift from the code actually compiled into that binary.

## 2. Problem statement and scope

### 2.1 What we want to solve

Engineering documentation degrades fast when it references code by name or by inline copy-paste. Names get renamed, signatures change, and examples become subtly wrong. Existing tools (`godoc`, GitHub file views, `pkg.go.dev`) solve *published API reference* but not:

1. **In-tree exploration of the exact binary you are running** (binary → source → docs round trip).
2. **Doc pages that embed specific functions, methods, type declarations, or even sub-ranges of code** and that remain trustworthy across refactors.
3. **A themable, embeddable UI** that another Go application can use to ship its own "about/source" surface without a full IDE.

### 2.2 In scope

1. Build-time extraction of Go symbols and relationships into a JSON index.
2. Bundling source + index + SPA into one binary.
3. A REST API over the index + source.
4. A React frontend that renders trees, files, symbols, and docs with embedded snippets.
5. Theming via CSS variables + `data-part` selectors and Storybook coverage.
6. Glazed-based CLI for indexing, serving, and offline symbol queries.

### 2.3 Out of scope (initial phases)

1. Multi-language support (TypeScript, Python, etc.) — the index schema is language-agnostic, but only a Go extractor ships in phase 1.
2. Authenticated multi-tenant hosting — single-binary, local/LAN use is the default.
3. Write/edit capabilities — this is a read-only browser.
4. Incremental re-indexing — phase 1 rebuilds the full index each `go generate`.

## 3. Current-state analysis

This repository is new. At the time of writing it contains only `ttmp/` scaffolding and a `.ttmp.yaml` config. There is no Go code, no `go.mod`, and no frontend. That is a useful starting constraint: there is no legacy tooling to migrate, but also no shared conventions to piggy-back on, so this doc is the authoritative source of truth until code lands.

Evidence:

```
$ ls /home/manuel/code/wesen/2026-04-19--go-codebase-browser
.git  .ttmp.yaml  ttmp
```

The surrounding ecosystem we are building against:

1. **Glazed** — the project will follow the conventions in the `glazed-command-authoring` skill. Notable constraints: import paths live under `github.com/go-go-golems/glazed/pkg/...`; commands embed `*cmds.CommandDescription`; decode via `vals.DecodeSectionInto(schema.DefaultSlug, settings)`; root commands wire logging + embedded help via `logging.AddLoggingSectionToRootCommand` and `help_cmd.SetupCobraRootCommand`.
2. **Go web + embed** — the project will follow the `go-web-frontend-embed` skill pattern: Vite on `:3000`, Go on `:3001` in dev; single-binary with `go:embed` + `//go:build embed` in prod; `go generate ./internal/web` copies the Vite output into `internal/web/embed/public`.
3. **React modular themable** — we will follow `react-modular-themable-storybook`: `data-widget` root, `data-part`/`data-role`/`data-state` for styling hooks, CSS variables for tokens, Storybook for default/themed/unstyled variants.

## 4. Gap analysis

Given the greenfield state, the gap is essentially "everything." The useful framing is which sub-problems are genuinely hard versus already-solved:

| Concern | Already solved (by what) | Novel work in this project |
|---|---|---|
| Parse Go source | `go/parser`, `go/ast` | — |
| Load packages with types | `golang.org/x/tools/go/packages` | — |
| Type information | `go/types` | — |
| Analysis passes | `golang.org/x/tools/go/analysis` | Write a small `indexer` analyzer that emits records |
| Embed static tree | `embed.FS`, build tags | — |
| Serve SPA | `http.ServeMux` + SPA fallback | — |
| RTK-Query | `@reduxjs/toolkit/query/react` | API surface design |
| Theming contract | `data-part` + CSS vars | Mapping the symbol/file views to parts/tokens |
| Live-source embedding in docs | — | **Core novel contribution** — stable symbol IDs + MDX-ish directives |
| Stable symbol IDs across refactors | Partial (`go/types.Object`) | IDs that survive file moves (see §6.3) |
| Build-time determinism | `go generate`, content-addressable snapshots | Deterministic traversal + stable ordering |

## 5. Proposed architecture

### 5.1 High-level topology

```
┌────────────────────────────────────────────────────────────────┐
│                      Single Go binary                           │
│                                                                 │
│  ┌───────────────┐    ┌──────────────────┐    ┌─────────────┐  │
│  │ embedded FS   │    │ embedded FS      │    │ embedded FS │  │
│  │ source tree   │    │ index.json       │    │ SPA (ui/)   │  │
│  │ (go:embed)    │    │ (go:embed)       │    │ (go:embed)  │  │
│  └───────┬───────┘    └─────────┬────────┘    └──────┬──────┘  │
│          │                      │                    │          │
│          │     ┌────────────────┴────────┐           │          │
│          └────►│ internal/browser (svc)  │◄──────────┘          │
│                └────────┬────────────────┘                      │
│                         │                                        │
│               ┌─────────┴──────────┐                            │
│               │ internal/server    │ (http.ServeMux)            │
│               │   /api/*           │                            │
│               │   /                │ (SPA fallback)             │
│               └─────────┬──────────┘                            │
│                         │                                        │
│                         ▼                                        │
│               ┌────────────────────┐                            │
│               │ cmd/<bin>/main.go  │ (Glazed + Cobra)           │
│               └────────────────────┘                            │
└────────────────────────────────────────────────────────────────┘

Browser ──► GET /api/index → JSON (all packages/symbols)
Browser ──► GET /api/source?path=… → text (raw bytes)
Browser ──► GET /api/snippet?sym=…&kind=body → text (exact byte range)
Browser ──► GET /api/doc/<slug> → pre-rendered HTML or MDX AST
Browser ──► GET /           → index.html (SPA fallback)
```

### 5.2 Repository layout

```
<repo-root>/
  cmd/
    codebase-browser/
      main.go                       # root cobra wiring, log + help
      cmds/
        index/
          root.go
          build.go                  # `codebase-browser index build`
          stats.go                  # `codebase-browser index stats`
        symbol/
          root.go
          show.go                   # `codebase-browser symbol show`
          find.go                   # `codebase-browser symbol find`
        serve/
          root.go
          run.go                    # `codebase-browser serve`
        doc/
          root.go
          render.go                 # `codebase-browser doc render`
  internal/
    browser/                        # domain logic
      index.go                      # in-memory index type + queries
      loader.go                     # ingest index.json + embedded source
      snippet.go                    # byte-range extraction
      xref.go                       # caller graphs
    indexer/                        # build-time only
      analyzer.go                   # go/analysis pass
      extractor.go                  # go/ast visitor
      id.go                         # stable symbol ID scheme
      write.go                      # index.json emitter
    server/
      server.go                     # http.ServeMux routes
      api_index.go                  # GET /api/index
      api_source.go                 # GET /api/source, /api/snippet
      api_doc.go                    # GET /api/doc/<slug>
      api_search.go                 # GET /api/search
      spa.go                        # SPA fallback (from skill template)
    web/
      embed.go                      # //go:build embed  → SPA FS
      embed_none.go                 # //go:build !embed → os.DirFS fallback
      embed/public/                 # Vite output, written by go generate
      generate.go                   # //go:generate directives
      generate_build.go             # program run by go generate
    sourcefs/
      embed.go                      # //go:build embed  → source FS
      embed_none.go                 # //go:build !embed → os.DirFS
      embed/source/                 # copied source tree, written by generator
    indexfs/
      embed.go                      # //go:build embed  → index.json
      embed_none.go                 # //go:build !embed → disk read
      embed/index.json              # written by `codebase-browser index build`
    docs/
      embed.go                      # //go:build embed  → MDX-ish corpus
      embed/pages/*.md              # hand-written pages with snippet refs
  ui/                                # Vite React SPA
    package.json
    vite.config.ts
    src/
      app/                          # app shell (routes, providers)
      api/                          # RTK-Query slices (indexApi, sourceApi, docApi)
      features/
        tree/                       # PackageTree, FileList
        symbol/                     # SymbolPage, SignatureBox, DocBlock
        source/                     # SourceView, Snippet
        doc/                        # DocPage, MDX renderer + <Snippet/>
      packages/                     # publishable module
        ui/                         # @codebase-browser/ui (themable widgets)
          src/
            parts.ts
            theme/
              light.css
              dark.css
            Widget.tsx
            SymbolCard.tsx
            Snippet.tsx
            TreeNav.tsx
          index.ts
    .storybook/
      main.ts
      preview.tsx
  ttmp/                              # docmgr workspace (this ticket lives here)
  Makefile
  go.mod
```

### 5.3 Data flow at a glance

1. `go generate ./...` runs two generators:
   a. `internal/web/generate.go` → builds Vite SPA, copies into `internal/web/embed/public`.
   b. `internal/indexfs/generate.go` → runs `go run ./cmd/codebase-browser index build`, writes `internal/indexfs/embed/index.json` and mirrors the Go source tree into `internal/sourcefs/embed/source`.
2. `go build -tags embed ./cmd/codebase-browser` produces the single binary with all three embedded FS branches.
3. At runtime:
   a. `serve` loads the index via `internal/browser`, mounts `/api/*`, and serves the SPA on `/`.
   b. The SPA fetches `/api/index` once on boot (RTK-Query cached), then resolves symbols, snippets, and doc pages on demand.

## 6. Index schema

The index is the pivot between extraction and rendering. Its stability across refactors determines whether doc pages keep working.

### 6.1 JSON shape (top level)

```json
{
  "version": "1",
  "generatedAt": "2026-04-20T00:00:00Z",
  "module": "github.com/example/codebase-browser",
  "goVersion": "go1.22",
  "packages": [ { /* Package */ } ],
  "symbols":  [ { /* Symbol  */ } ],
  "files":    [ { /* File    */ } ],
  "refs":     [ { /* Ref     */ } ]
}
```

Separation rationale: the **symbols** array is the primary keyed table (by `id`), while `packages`, `files`, and `refs` are secondary indexes. Keeping them flat (not nested) keeps JSON parsing predictable in both Go and the browser, and lets RTK-Query build selectors per collection.

### 6.2 Record shapes

```ts
// Package
{
  id: "pkg:<importPath>",          // e.g. "pkg:github.com/foo/bar/internal/baz"
  importPath: string,
  name: string,                    // identifier, not import path
  doc: string,                     // package-level doc comment (markdown-safe)
  fileIds: string[],               // -> File.id
  symbolIds: string[]              // top-level only; nested live under Symbol.children
}

// File
{
  id: "file:<relPath>",            // e.g. "file:internal/browser/index.go"
  path: string,                    // relative to module root
  packageId: string,
  size: number,                    // bytes
  lineCount: number,
  buildTags: string[],             // parsed build constraints
  sha256: string                   // of the bytes that shipped
}

// Symbol
{
  id: "sym:<pkgImportPath>.<Kind>.<Name>[#<signatureHash>]",
  kind: "func"|"method"|"type"|"const"|"var"|"iface"|"struct"|"field"|"alias",
  name: string,                    // identifier only; receiver elsewhere
  packageId: string,
  fileId: string,
  range: { startLine: number, startCol: number, endLine: number, endCol: number,
           startOffset: number, endOffset: number },  // byte offsets are authoritative
  doc: string,                     // doc comment (raw)
  signature: string,               // pretty-printed
  receiver?: { typeName: string, pointer: boolean },
  typeParams?: string[],           // generics
  exported: boolean,
  children?: Symbol[],             // methods on a type, fields on a struct
  tags?: string[]                  // e.g. ["test", "main", "example"]
}

// Ref (cross-reference)
{
  fromSymbolId: string,
  toSymbolId: string,
  kind: "call"|"implements"|"embeds"|"uses-type"|"reads"|"writes"|"returns",
  fileId: string,
  range: Range
}
```

### 6.3 Stable symbol IDs

Refactors (renaming files, moving packages) should not invalidate doc snippets en masse. The ID scheme is therefore:

1. **Primary key**: `sym:<importPath>.<Kind>.<Name>` (+ `#<signatureHash>` only to disambiguate overloads like test helpers with identical names). Using `importPath` rather than file path lets a file move without changing the ID.
2. **Secondary key**: `file:<relPath>` + byte range, cached in the index for fast lookup but never quoted by authors.
3. **Writer-facing key**: `pkg/foo.Bar` short form, resolved at render time to the full ID. If the short form is ambiguous (two methods named `Bar` on different types), the renderer emits a hard error at `doc render` build.

This mirrors how `go/types.Object.Id()` works but extends it for methods and for unexported identifiers visible only within a package.

### 6.4 Size and format considerations

For a mid-sized project (say 300 Go files, 5k symbols), index.json is typically under 3–5 MB uncompressed. Observations that shaped the format:

1. Flat arrays parse faster in the browser than deeply nested trees.
2. Byte offsets are preferred over (line, col) for snippet extraction because they are O(1) to slice and do not require re-reading the file.
3. The `sha256` per file lets the API prove "the bytes I serve are the bytes I indexed" — a cheap audit hook.
4. In the binary, `index.json` is embedded as-is; at runtime it is decoded into a Go struct once, then kept in memory. Decoding a 5 MB JSON is ~30–60 ms on modern hardware, which is fine for `serve` startup.

## 7. Extraction pipeline

### 7.1 Choosing the right `golang.org/x/tools` entrypoint

1. `go/parser` alone is too low-level; we need type info.
2. `go/packages.Load` with `NeedName | NeedFiles | NeedTypes | NeedSyntax | NeedTypesInfo | NeedDeps` gives us fully typed ASTs and transitively loaded deps.
3. `go/analysis` gives us a uniform `pass.Pkg`, `pass.TypesInfo`, and `pass.Files` — useful if we want the indexer composable with other analyzers later.

**Chosen approach**: wrap extraction as a `go/analysis.Analyzer` (`indexer.Analyzer`). The driver is `singlechecker.Main` under the hood, but we call it programmatically from a Glazed command so we can stream progress and emit JSON on a known path.

### 7.2 Extraction steps

```go
// Pseudocode: internal/indexer/extractor.go

func Extract(modRoot string, patterns []string) (*Index, error) {
    cfg := &packages.Config{
        Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
              packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps |
              packages.NeedModule | packages.NeedImports,
        Dir:  modRoot,
        Fset: token.NewFileSet(),
        Tests: true, // include _test.go files
    }

    pkgs, err := packages.Load(cfg, patterns...)
    if err != nil { return nil, err }

    idx := NewIndex(cfg.Fset)
    for _, p := range pkgs {
        idx.addPackage(p)
        for _, f := range p.Syntax {
            ast.Inspect(f, func(n ast.Node) bool {
                switch d := n.(type) {
                case *ast.FuncDecl:
                    idx.addFunc(p, f, d)
                case *ast.GenDecl:
                    idx.addGenDecl(p, f, d) // type/const/var/alias
                }
                return true
            })
        }
        idx.addRefsFromTypesInfo(p.TypesInfo, p.Fset)
    }
    return idx, nil
}
```

### 7.3 Byte ranges, not line ranges

Every snippet operation reads `source[Symbol.range.startOffset:Symbol.range.endOffset]`. Line numbers are computed once at extraction time and cached for display, but never used for slicing. This removes a whole class of "CRLF vs LF" and "tab width" bugs.

### 7.4 Doc comments

`ast.CommentGroup.Text()` yields the raw comment text. We preserve the raw form (no HTML rendering at index time) and let the frontend render markdown/godoc conventions (`//`, `// Deprecated:`, etc.). Keeping the index markdown-free means re-skinning the renderer later does not require re-indexing.

### 7.5 Cross-references (phase 2)

`types.Info.Uses` gives us `Object` for every identifier use. Walk each file, and for each `*ast.Ident` with a `types.Object` whose `Pkg()` resolves to a known package, emit a `Ref` with the enclosing function/method as `fromSymbolId`. This covers call graphs, type uses, and embedded types. Interface satisfaction (`implements`) is computed separately via `types.Implements`.

### 7.6 Deterministic output

1. Sort packages by import path.
2. Sort symbols by (packageId, fileId, startOffset).
3. Sort refs by (fromSymbolId, toSymbolId, fileId, startOffset).
4. Marshal with `encoding/json` using a custom encoder that writes keys alphabetically (or use `jsoniter` with `SortMapKeys: true`). Deterministic output makes `git diff` on the committed index readable.

### 7.7 Embedding the source tree

The source tree is copied, not referenced, so the binary stays self-contained:

1. `internal/indexfs/generate_source.go` walks the module root, **filtering out `.git`, `node_modules`, `ui/dist`, and vendor**, and writes the filtered tree under `internal/sourcefs/embed/source/`.
2. Large non-Go assets are excluded by default (configurable via `.codebase-browser.yaml`).
3. The generator refuses to run if any file `path` is not clean (`filepath.Clean(p) != p`) — a guard against path traversal surfacing via the API.

## 8. Glazed CLI

All commands follow the `glazed-command-authoring` skill conventions. Each command is a struct embedding `*cmds.CommandDescription`, implements `RunIntoGlazeProcessor`, and is wired through `cli.BuildCobraCommandFromCommand`. The root follows the canonical pattern (logging section + embedded help + `help_cmd.SetupCobraRootCommand`).

### 8.1 Command tree

```
codebase-browser
├── index
│   ├── build     Build index.json from Go source.
│   └── stats     Print counts by kind / package.
├── symbol
│   ├── show      Print a single symbol (signature, doc, snippet).
│   └── find      Query by name/kind/package with Glazed output.
├── serve         Run the embedded web server.
└── doc
    └── render    Render embedded docs to HTML (pre-render for SSG).
```

### 8.2 Canonical skeleton — `index build`

```go
// cmd/codebase-browser/cmds/index/build.go

type BuildCommand struct {
    *cmds.CommandDescription
}

type BuildSettings struct {
    ModuleRoot string   `glazed:"module-root"`
    Patterns   []string `glazed:"patterns"`
    Output     string   `glazed:"output"`
    Pretty     bool     `glazed:"pretty"`
}

func NewBuildCommand() (*BuildCommand, error) {
    glazedSection, _ := settings.NewGlazedSchema()
    cmdSettingsSection, _ := cli.NewCommandSettingsSection()

    desc := cmds.NewCommandDescription(
        "build",
        cmds.WithShort("Build index.json from Go source"),
        cmds.WithLong(`Walk the Go module at --module-root, load packages matching --patterns,
and emit a JSON index to --output.

Examples:
  codebase-browser index build --module-root . --patterns ./... --output internal/indexfs/embed/index.json
`),
        cmds.WithFlags(
            fields.New("module-root", fields.TypeString,
                fields.WithDefault("."),
                fields.WithHelp("Path to the module root (contains go.mod)")),
            fields.New("patterns", fields.TypeStringList,
                fields.WithDefault([]string{"./..."}),
                fields.WithHelp("Package patterns passed to go/packages.Load")),
            fields.New("output", fields.TypeString,
                fields.WithDefault("internal/indexfs/embed/index.json"),
                fields.WithHelp("Output path")),
            fields.New("pretty", fields.TypeBool,
                fields.WithDefault(true),
                fields.WithHelp("Indent JSON output")),
        ),
        cmds.WithSections(glazedSection, cmdSettingsSection),
    )
    return &BuildCommand{CommandDescription: desc}, nil
}

func (c *BuildCommand) RunIntoGlazeProcessor(
    ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
    s := &BuildSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil { return err }

    idx, err := indexer.Extract(s.ModuleRoot, s.Patterns)
    if err != nil { return err }
    if err := indexer.Write(idx, s.Output, s.Pretty); err != nil { return err }

    row := types.NewRow(
        types.MRP("output", s.Output),
        types.MRP("packages", len(idx.Packages)),
        types.MRP("symbols", len(idx.Symbols)),
        types.MRP("files", len(idx.Files)),
    )
    return gp.AddRow(ctx, row)
}
```

### 8.3 `symbol find` — Glazed output as first-class UX

Because Glazed gives us `--output json|yaml|table|csv` for free, `symbol find` returns a row per matching symbol and users can pipe it into `jq` or a spreadsheet. This is the pattern to extend for every "list" surface — **do not build ad hoc CLI output**; let Glazed do it.

### 8.4 `serve` — production wiring

`serve` must **not** rebuild the index. It reads `internal/indexfs` at startup and trusts what the build produced. This is also what makes `go build -tags embed` produce a truly self-contained binary.

```go
// pseudocode
func (c *ServeCommand) RunIntoGlazeProcessor(...) error {
    idx, err := browser.LoadEmbedded(indexfs.FS, sourcefs.FS)
    if err != nil { return err }

    srv := server.New(idx, web.FS /* SPA */)
    addr := s.Addr
    return srv.Run(ctx, addr)
}
```

## 9. HTTP API

### 9.1 Endpoints

| Method | Path | Purpose | Response |
|---|---|---|---|
| GET | `/api/index` | Whole index (cached once per server lifetime) | `application/json` |
| GET | `/api/packages` | Lightweight package list (no symbols) | JSON |
| GET | `/api/symbol/{id}` | One symbol + its children | JSON |
| GET | `/api/source?path=<relPath>` | Raw file bytes | `text/plain; charset=utf-8` |
| GET | `/api/snippet?sym=<id>&kind=<body\|signature\|declaration>` | Exact byte range for a symbol part | `text/plain` |
| GET | `/api/search?q=<query>&kind=<filter>` | Name-prefix + kind filter | JSON, streamed |
| GET | `/api/doc` | List doc pages | JSON |
| GET | `/api/doc/{slug}` | Doc page MDX-AST + resolved snippet IDs | JSON |
| GET | `/api/xref/{id}` | Callers/implementers (phase 2) | JSON |

All endpoints set `Cache-Control: public, max-age=31536000, immutable` because the binary is immutable. Browser caching is aggressive by design.

### 9.2 Path hygiene

`/api/source?path=...` is the dangerous one. It must:

1. Reject paths containing `..` or absolute paths (`filepath.IsAbs`).
2. Only serve paths present in the index `files` table (index-backed whitelist).
3. Use `fs.ReadFile(sourcefs.FS, path)` — the embedded FS is itself the sandbox.

### 9.3 `/api/snippet` semantics

Given `sym=sym:pkg/foo.func.Bar` and `kind=body`:

1. Look up `Symbol.range`.
2. Slice `source[startOffset:endOffset]`.
3. Optionally dedent by the common-prefix whitespace of non-empty lines (`kind=body-dedented`).
4. Return bytes with `X-Codebase-Symbol-Id`, `X-Codebase-File-Sha256`, `X-Codebase-Range` response headers so downstream tools can audit.

The three `kind` values cover the typical doc needs: signature line only, full declaration, or body only. Callers that need something weirder fall back to `/api/source` + explicit byte range.

## 10. Frontend — React + Vite + RTK-Query + Storybook

### 10.1 Package layout

Two TS packages in the same Vite workspace:

1. **App** (`ui/src/app`) — the runnable SPA, includes routes, providers, and layout.
2. **Widgets** (`ui/src/packages/ui`, published as `@codebase-browser/ui`) — presentation-only, themable, Storybook-covered. This is the package another Go application could vendor the React source of if they want a custom surface.

This split matters because the widgets package must not depend on `react-router-dom`, RTK-Query, or any app concerns. Data flows in as props; events flow out via callbacks. That is what makes it themable and testable in Storybook without a backend.

### 10.2 RTK-Query slices

Following the `rtk-query` conventions, we define three API slices (one per REST resource family):

```ts
// ui/src/api/indexApi.ts
export const indexApi = createApi({
  reducerPath: 'indexApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api' }),
  tagTypes: ['Index', 'Package', 'Symbol'],
  endpoints: (b) => ({
    getIndex:    b.query<IndexSummary, void>({ query: () => '/index',
                                                providesTags: ['Index'] }),
    getPackage:  b.query<Package, string>({ query: (id) => `/packages/${id}`,
                                             providesTags: (r, e, id) => [{ type: 'Package', id }] }),
    getSymbol:   b.query<Symbol, string>({ query: (id) => `/symbol/${encodeURIComponent(id)}`,
                                            providesTags: (r, e, id) => [{ type: 'Symbol', id }] }),
    searchSymbols: b.query<SymbolHit[], { q: string; kind?: string }>(
                   { query: ({q, kind}) => `/search?q=${encodeURIComponent(q)}&kind=${kind ?? ''}` }),
  }),
});

// ui/src/api/sourceApi.ts
export const sourceApi = createApi({ ...snippet+source endpoints... });

// ui/src/api/docApi.ts
export const docApi = createApi({ ...doc list + page endpoints... });
```

Because the backend is immutable per-binary, we set long `keepUnusedDataFor` (e.g. `3600` s) and rely on RTK-Query's cache as the primary read path.

### 10.3 Theming: parts + tokens

The widgets package uses the `react-modular-themable-storybook` contract. Every themable widget has a `data-widget` root, `data-part` children, and optional `data-state`:

```tsx
// ui/src/packages/ui/src/SymbolCard.tsx
export function SymbolCard({ symbol, snippet, unstyled }: SymbolCardProps) {
  return (
    <article data-widget="codebase-browser" data-part="symbol-card"
             data-state={snippet ? 'with-snippet' : 'no-snippet'}>
      <header data-part="symbol-header">
        <span data-part="symbol-kind" data-role="kind">{symbol.kind}</span>
        <code data-part="symbol-name" data-role="name">{symbol.name}</code>
        <code data-part="symbol-signature" data-role="signature">{symbol.signature}</code>
      </header>
      {symbol.doc && <div data-part="symbol-doc" data-role="doc">{renderGodoc(symbol.doc)}</div>}
      {snippet && <pre data-part="symbol-snippet" data-role="code"><code>{snippet}</code></pre>}
    </article>
  );
}
```

The `parts.ts` file exports the canonical list:

```ts
// ui/src/packages/ui/src/parts.ts
export const PARTS = {
  root: 'codebase-browser',
  treeNav: 'tree-nav',
  treeNode: 'tree-node',
  symbolCard: 'symbol-card',
  symbolHeader: 'symbol-header',
  symbolKind: 'symbol-kind',
  symbolName: 'symbol-name',
  symbolSignature: 'symbol-signature',
  symbolDoc: 'symbol-doc',
  symbolSnippet: 'symbol-snippet',
  sourceView: 'source-view',
  sourceLine: 'source-line',
  sourceGutter: 'source-gutter',
  docPage: 'doc-page',
  snippetEmbed: 'snippet-embed',
} as const;
```

Tokens are a small set of CSS variables (mirroring `parts-and-tokens.md`):

```css
/* ui/src/packages/ui/src/theme/base.css — optional default theme */
[data-widget="codebase-browser"] {
  --cb-color-bg:      #ffffff;
  --cb-color-text:    #1a1a1a;
  --cb-color-muted:   #6b7280;
  --cb-color-accent:  #2563eb;
  --cb-color-code-bg: #f6f8fa;
  --cb-font-family:   ui-sans-serif, system-ui, sans-serif;
  --cb-font-mono:     ui-monospace, SFMono-Regular, Menlo, monospace;
  --cb-space-1: 4px; --cb-space-2: 8px; --cb-space-3: 12px;
  --cb-space-4: 16px; --cb-space-5: 24px; --cb-space-6: 32px;
  --cb-radius-1: 4px; --cb-radius-2: 8px;
}
```

An `unstyled` variant is literally "do not import base.css" — the components still render with correct semantics, just without visuals. This is the same pattern documented in the themable skill.

### 10.4 Storybook coverage

The skill says "copious amounts" — concretely, every widget has at minimum these stories:

1. `Default` — with realistic fixture data.
2. `Empty` — no data / no match.
3. `Loading` — skeleton variant.
4. `Error` — with a representative error message.
5. `Themed — Light`, `Themed — Dark`, `Themed — HighContrast` — three theme overrides via CSS vars.
6. `Unstyled` — base CSS not imported.
7. `Slot override` — demonstrating a custom renderer for one part.

Fixtures live in `ui/src/packages/ui/src/__fixtures__/` and are hand-authored JSON shaped exactly like the real API responses. The same fixtures power RTK-Query tests in the app.

Minimum widget set with stories in phase 1:

1. `<TreeNav/>` (packages + files).
2. `<SymbolCard/>`.
3. `<SourceView/>` (syntax-highlighted file with anchor jumps).
4. `<Snippet/>` (the doc-embed primitive — §11).
5. `<DocPage/>` (markdown + snippets).
6. `<SearchBox/>`.

### 10.5 Storybook wiring

`.storybook/preview.tsx` wraps every story with:

1. An RTK-Query mock provider (MSW or a tiny in-memory `fetchBaseQuery` replacement) so stories can demonstrate real hooks (`useGetSymbolQuery(...)`) without a live server.
2. A theme toolbar decorator that swaps a `className` or attribute on `#root` to toggle themes.

## 11. Documentation pages with embedded source

This is the most novel part and deserves its own section.

### 11.1 Authoring model

Doc pages live under `internal/docs/embed/pages/*.md` and are written in an MDX-ish dialect (markdown with a small set of custom fenced blocks). The canonical embedding primitive is:

~~~
```codebase-snippet sym=pkg/foo.Bar kind=body dedent=true highlight=3-5
```
~~~

Additional directive types:

1. `codebase-signature sym=...` — one-liner signature box.
2. `codebase-doc sym=...` — render the godoc comment only.
3. `codebase-file path=... range=10-40` — arbitrary range from a file, when the symbol boundary is too narrow.

### 11.2 Rendering pipeline

Doc rendering is a function of `(docText, index, sourceFS)`. It runs in two places:

1. **At `doc render` build time** (`codebase-browser doc render`) — produces a cached JSON AST so the frontend does not re-parse markdown on every page load.
2. **At runtime** (in `serve`) — if docs change (only really useful in dev), the server re-parses on demand.

```
markdown text
   │
   ▼
parse (goldmark) with custom fenced-block extension
   │
   ▼
walk AST → for each snippet directive:
   1) resolve `sym` via index (error if ambiguous/missing)
   2) slice bytes via `Symbol.range`
   3) dedent if asked
   4) attach resolved snippet payload to the AST node
   │
   ▼
serialize to JSON AST (for the frontend), or HTML (for pre-render / print)
```

The resolved AST nodes carry both the **resolved snippet text** and the **symbol ID** so the frontend's `<Snippet/>` widget can link "jump to source" back to the symbol page.

### 11.3 Drift guarantee

Because:

1. the index is built from the exact source being embedded,
2. the source FS is embedded in the same binary, and
3. the doc renderer resolves snippets *against the embedded index*,

it is impossible for a deployed binary to show a snippet that differs from its source. The worst failure mode is a build-time error ("symbol `pkg/foo.Bar` not found"), which fails CI loudly, which is what we want. That is the entire point of the project.

### 11.4 Authoring ergonomics

To keep authors productive:

1. `codebase-browser doc render --check` runs without writing anything and prints every doc page that references a missing/ambiguous symbol, with file/line, so it can gate PRs.
2. `codebase-browser symbol find --name <prefix>` helps authors discover valid IDs before writing them down.
3. A Storybook story (`DocPage / AuthoringErrors`) renders the error variants so writers can recognize them.

## 12. Implementation plan (phased)

### Phase 0 — Scaffolding (0.5 day)

1. `go mod init github.com/.../codebase-browser`.
2. Create directory layout from §5.2.
3. Add Makefile (dev-backend, dev-frontend, frontend-check, build, generate).
4. Add `ui/` with Vite + React + TS template. Configure dev proxy per `go-web-frontend-embed` skill.

### Phase 1 — Indexer + CLI (2–3 days)

1. Implement `internal/indexer/{extractor,id,write}.go` for packages, files, and top-level symbols only (no xref yet).
2. Implement `cmd/codebase-browser/cmds/index/{build,stats}.go` per Glazed conventions.
3. Implement `cmd/codebase-browser/cmds/symbol/{show,find}.go` backed by a loader that reads `index.json` from disk.
4. Add a golden-file test: run `index build` on a tiny fixture module in `testdata/`, assert byte-stable JSON output.
5. Wire root `main.go` with logging + embedded help (skill pattern).

### Phase 2 — Server + embed (2 days)

1. Implement `internal/indexfs/`, `internal/sourcefs/`, `internal/web/` following the `go-web-frontend-embed` pattern (with `//go:build embed` / `!embed` pairs).
2. Implement `internal/server/{server,api_index,api_source,api_doc,api_search}.go`.
3. Implement `cmd/codebase-browser/cmds/serve/run.go` (reads index + source FS, mounts routes, starts server).
4. Regression test (`server_static_test.go`): in-memory `fstest.MapFS` with `index.html`, assert `GET /` returns 200 HTML.
5. Regression test (`api_source_test.go`): `..` traversal attempt returns 400; path not in index returns 404.

### Phase 3 — Frontend shell (2–3 days)

1. Scaffold `ui/src/app` with `@reduxjs/toolkit` store, RTK-Query slices (`indexApi`, `sourceApi`, `docApi`), react-router, and a layout with tree-nav + main content.
2. Implement the minimum feature pages: `/packages`, `/packages/:id`, `/symbol/:id`, `/source/*`.
3. Fetch the whole index once on boot into RTK-Query; derive local selectors for tree and search.
4. `pnpm -C ui run build` produces `ui/dist/public/*`; `go generate ./internal/web` copies to `internal/web/embed/public/`.

### Phase 4 — Themable widget package + Storybook (2–3 days)

1. Move presentation-only components into `ui/src/packages/ui`.
2. Introduce `parts.ts`, `theme/base.css`, optional theme presets.
3. Storybook set up under `ui/.storybook/`; stories for each widget per §10.4.
4. Add MSW-based RTK-Query mocking in Storybook decorator.
5. Typecheck, lint, Storybook build in CI.

### Phase 5 — Doc renderer + live snippets (2 days)

1. Implement `internal/docs` + goldmark custom fenced-block extension.
2. Implement `cmd/codebase-browser/cmds/doc/render.go` (build-time renderer, emits AST JSON).
3. Implement `/api/doc` endpoints.
4. Implement frontend `DocPage` + `<Snippet/>` widget + "jump to source" link.
5. Ship 2–3 real doc pages that embed snippets from the project itself (dogfood).

### Phase 6 — Cross-references (optional, 2 days)

1. Extend extractor to emit `refs` using `types.Info.Uses`.
2. Add `/api/xref/{id}` with callers / implementers / users.
3. Frontend "Called by" and "Uses" panels on symbol pages.

### Phase 7 — Polish

1. Dark theme + one "custom theme" example.
2. Docs-of-docs: authoring-a-doc-page page that itself embeds doc-renderer source.
3. `README.md` with animated screenshot.
4. GitHub Actions: frozen pnpm lockfile → `go generate ./...` → `go build -tags embed ./...`.

## 13. Testing strategy

| Layer | Tests | Tooling |
|---|---|---|
| Indexer | Golden JSON on fixture module; determinism (run twice, diff zero) | Go test |
| Index schema | JSON schema validation of emitted files | Go test + `jsonschema` |
| ID stability | Move a file in fixture, assert symbol IDs unchanged | Go test |
| Path hygiene | `..`, absolute, unknown path rejected on `/api/source` | Go `httptest` |
| SPA fallback | `GET /` returns HTML | Go `httptest` + `fstest.MapFS` (per skill) |
| Doc renderer | Missing/ambiguous symbol fails loudly; happy path resolves snippet | Go test with fixture index |
| Frontend unit | RTK-Query hooks with MSW | Vitest |
| Frontend visual | Storybook stories build cleanly; optional Chromatic | Storybook |
| E2E smoke | Start `serve`, curl `/api/index`, `/api/symbol/...`, `/` | Makefile `smoke` target |

The golden tests are the single most important set — they protect the index schema from accidental drift.

## 14. Risks, alternatives, and open questions

### 14.1 Risks

1. **Index size growth**. For a ~2k-symbol project, 3–5 MB JSON is fine. For a 50k-symbol monorepo, we would likely split into `index.json` (packages + files only) and lazy-loaded per-package JSON. Mitigation: keep the schema shape flat so splitting later does not require a rewrite.
2. **`go/packages` load cost in CI**. Loading with `NeedDeps` + `NeedTypes` for large modules can take minutes. Mitigation: make `patterns` configurable, default to `./...` but allow narrowing.
3. **Generic type parameters**. Pretty-printing signatures for generics is fiddly. Mitigation: use `types.TypeString` with a `Qualifier` that shortens the local package, and golden-test a handful of tricky cases.
4. **Storybook + MSW in CI**. MSW service workers can be flaky in headless CI. Mitigation: use MSW's "node" adapter in Storybook interaction tests rather than the browser service worker.
5. **Circular generators**: the `codebase-browser` binary is needed to build its own index. Bootstrap by using `go run ./cmd/codebase-browser index build` (not the built binary) in `go generate`, which sidesteps the chicken-and-egg.
6. **Source tree privacy**: embedding the source ships everything in the tree. If a user ships the binary to customers, they ship source. Mitigation: `.codebase-browser.yaml` `exclude:` globs, honored by the source-tree generator. Default **opt-out** of `internal/indexfs/embed/index.json` itself (we don't need to re-embed our own output).

### 14.2 Alternatives considered

1. **Runtime indexing** (the binary parses its own source at startup): rejected because it requires shipping a Go toolchain-equivalent parser set, increases startup time, and makes the binary non-self-contained if source is read from disk.
2. **`gopls` JSON-RPC as the index source**: rejected for external-dependency reasons and because `gopls` is optimized for incremental, not batch, extraction.
3. **SQLite on disk instead of JSON**: attractive for search and xref, but breaks "one immutable file" mental model and adds a cgo dependency. Revisit in phase 6 if xref query perf demands it.
4. **Server-rendered (Go templates) instead of React**: rejected because the user explicitly asked for themable React + RTK-Query + Storybook. The split between app and widgets package also directly enables third-party theming.
5. **Tree-sitter instead of go/ast**: multi-language future-proof, but loses Go type information. Keep `go/packages` for Go; if we ever add TS/Python, add a tree-sitter-based extractor per language that writes to the same `Symbol` schema.

### 14.3 Open questions

1. Do we commit the generated `index.json` to the repo, or regenerate in CI? Committing makes diffs readable; regenerating avoids merge noise. Default: **regenerate**, do not commit.
2. Do we publish `@codebase-browser/ui` to npm, or keep it in-tree only? Recommendation: keep in-tree for phase 1; publish after phase 4 if there is real reuse.
3. Search ranking: prefix match only, or add fuzzy (`fzf`-style)? Phase 1: prefix + substring; revisit.
4. Should `<Snippet/>` lazy-load snippet bytes, or embed them in the doc-page AST? Embedding simplifies offline reading; lazy-loading keeps the doc AST small. Default: **embed for snippets ≤ 8 KB, lazy for larger**.
5. How do we attribute license/authorship when embedding upstream code? If the indexer ever runs on deps, the embedded source must carry the original LICENSE files. Phase 1 scope is **module-root only**, so this defers until multi-module support is added.

## 15. References

1. `glazed-command-authoring` skill — `/home/manuel/.claude/skills/glazed-command-authoring/SKILL.md` (import paths, canonical skeleton, root init pattern).
2. `go-web-frontend-embed` skill — `/home/manuel/.claude/skills/go-web-frontend-embed/SKILL.md` (embed + SPA fallback, `go generate` pipeline, Makefile).
3. `react-modular-themable-storybook` skill — `/home/manuel/.claude/skills/react-modular-themable-storybook/SKILL.md` + `references/parts-and-tokens.md` (parts/tokens contract).
4. `go/packages` — https://pkg.go.dev/golang.org/x/tools/go/packages (Load modes).
5. `go/analysis` — https://pkg.go.dev/golang.org/x/tools/go/analysis (Analyzer, singlechecker).
6. `go/types` — https://pkg.go.dev/go/types (Object, TypeString, Implements).
7. RTK-Query — https://redux-toolkit.js.org/rtk-query/overview (createApi, cache lifetimes).
8. Storybook — https://storybook.js.org (stories + decorators).
9. goldmark — https://pkg.go.dev/github.com/yuin/goldmark (markdown parser with extensions).
10. Glazed repo — `/home/manuel/code/wesen/corporate-headquarters/glazed` (reference implementation for Glazed patterns).
