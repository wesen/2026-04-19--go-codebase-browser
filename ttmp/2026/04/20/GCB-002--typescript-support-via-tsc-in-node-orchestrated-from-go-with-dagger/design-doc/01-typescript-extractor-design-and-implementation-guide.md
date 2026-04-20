---
Title: TypeScript Extractor — Design and Implementation Guide
Ticket: GCB-002
Status: active
Topics:
    - typescript
    - dagger
    - node-tooling
    - multi-language
    - go-ast
    - codebase-browser
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../.claude/skills/go-web-dagger-pnpm-build/SKILL.md
      Note: Canonical Dagger/pnpm/CacheVolume pattern followed in §7
    - Path: ../../../../../../../corporate-headquarters/smailnail/cmd/build-web/main.go
      Note: Reference Smailnail skeleton mirrored in §7.2
    - Path: internal/indexer/extractor.go
      Note: Extractor function refactored into interface in §8
    - Path: internal/indexer/types.go
      Note: GCB-001 schema that §5.4 extends with Language
    - Path: ui/src/packages/ui/src/Code.tsx
      Note: Where the tokenizer dispatch in §9.1 plugs in
ExternalSources: []
Summary: Design for indexing TypeScript codebases in the codebase-browser. A Node-based extractor uses the TypeScript Compiler API to emit Index JSON in the same shape Go emits; a Dagger container orchestrated from Go (following the go-web-dagger-pnpm-build skill) runs it hermetically. Go stays the single source of truth for serve/embed; the build pipeline becomes Go+Node. Validated with a working tsx prototype that extracted 7 symbols / 2 files from a fixture.
LastUpdated: 2026-04-20T00:00:00Z
WhatFor: Add TypeScript to GCB-001's language support without forking the index schema or the server, and without introducing a runtime Node dependency.
WhenToUse: Read before implementing the Extractor interface refactor, the Node extractor package, the build-ts-index Dagger command, or any PR that changes Index records to be language-scoped.
---


# TypeScript Extractor — Design and Implementation Guide

## 1. Executive summary

This ticket defines how the codebase-browser (GCB-001) gains **TypeScript support** without forking its index schema, without doubling its runtime surface, and without regressing the "single self-contained binary" property.

The design has three pillars:

1. **TypeScript Compiler API in Node emits the same `Index` JSON shape the Go indexer already emits.** A small `tools/ts-indexer/` Node program (standalone TypeScript file + `package.json`) runs `ts.createProgram` + `TypeChecker` and walks the AST emitting `Package`, `File`, `Symbol`, and `Ref` records with `language: "ts"`. The schema gets one additive field (`Symbol.language`) but is otherwise unchanged.
2. **Dagger orchestration from Go.** Following the `go-web-dagger-pnpm-build` skill pattern (Smailnail/Glazed/Mento), a new `cmd/build-ts-index/main.go` mounts the target source tree into a `node:22` container, `pnpm install`s the TS toolchain (with a `CacheVolume` for the store), runs the Node extractor, and exports the resulting `index-ts.json` back to the host. A local-pnpm fallback lets developers run without Dagger if the engine is unavailable.
3. **Index-merge at extraction time.** `codebase-browser index build` grows a `--lang` flag (`go` | `ts` | `auto`). In `auto` mode it runs the Go extractor and the TS extractor in parallel, merges the two `Index` outputs into one JSON, and writes the combined file. Server, frontend, doc renderer, and xref all keep working without language awareness — they just see more symbols.

The validating prototype in `scripts/extract.ts` (this ticket) already produces correctly-shaped records on a fixture module: 1 package (`src`), 2 files, 7 symbols (class, method, const, alias, func, iface). The Go side needs three changes: a new command, a small schema addition, and an `Extractor` interface refactor.

## 2. Problem statement and scope

### 2.1 Why

GCB-001 shipped Go-only extraction. Three use cases break:

1. **Polyglot repositories** — e.g. this repo already has a Go backend and a TS frontend (`ui/`). The browser shows the Go side but the TS side is invisible.
2. **TS-only projects** — libraries/apps that want a self-documenting binary distribution can't use the browser at all.
3. **Cross-language docs** — doc pages that want to embed both `indexer.Extract` (Go) and `<SymbolCard>` (TS) via `codebase-snippet` directives are one-sided.

### 2.2 In scope

1. Node-based TypeScript extractor that emits `Index` JSON matching the existing schema plus a `Symbol.language` field.
2. Dagger-based orchestration from Go (`cmd/build-ts-index`) following the `go-web-dagger-pnpm-build` skill pattern.
3. `Extractor` interface refactor in `internal/indexer/` so Go and TS extractors are pluggable.
4. Index merge pass for `--lang auto`.
5. Frontend: tokenizer dispatch on `language` and one new `highlight/ts.ts`.

### 2.3 Out of scope

1. **TypeScript xref for non-call references.** The initial TS extractor emits symbols only; xref parity (types, generics, JSX props) is a follow-up.
2. **Incremental re-indexing.** Still a full rebuild per `go generate`, same as Go.
3. **Server-side language-aware ranking.** Search stays purely name-based; language filtering can be a frontend-only query parameter.
4. **JSX-heavy analysis.** JSX elements parse fine in `ts.createProgram`, but extracting prop/component relationships is deferred.
5. **Non-TypeScript JavaScript (`.js`) files.** The extractor accepts `.ts`/`.tsx` only; `.js` is a future extension.

## 3. Current-state analysis

### 3.1 What GCB-001 gives us

1. A stable `Index` schema at `internal/indexer/types.go` (`Package`, `File`, `Symbol`, `Ref`, `Range`) that is already language-agnostic in shape.
2. A `browser.Loaded` loader at `internal/browser/index.go` that is also language-agnostic.
3. A set of `/api/*` endpoints (`/api/packages`, `/api/symbol/{id}`, `/api/source`, `/api/snippet`, `/api/search`, `/api/xref/{id}`) that all operate on opaque IDs.
4. A React frontend with a tokenizer dispatch in `Code.tsx` (currently `language === 'go'` → tokenize, otherwise raw text).
5. The Vite build pipeline already uses `ui/node_modules/typescript` as a dev dep — TypeScript tooling is already in the repo for frontend reasons.

### 3.2 What it doesn't give us

1. No extractor abstraction. `internal/indexer/extractor.go:Extract` is hard-coded to Go.
2. No language field on `Symbol`. Symbol kinds like `struct`/`iface` happen to work for both Go and TS, but `func` vs `method` need to be consistent across languages.
3. No Dagger integration. `go generate ./internal/web` runs Vite locally today (`ui/src/...` → `internal/web/embed/public`) via a plain `os/exec` wrapper.
4. No merge logic. The indexer writes one JSON file from one extractor run.

### 3.3 Evidence: validating prototype

The ticket's `scripts/extract.ts` was built and run against `scripts/fixture-ts/`:

```
scripts/fixture-ts/
├── src/
│   ├── greeter.ts    # class Greeter, const MaxRetries, func greet, interface Greetable, type Prefix
│   └── main.ts       # imports + usage
└── tsconfig.json
```

Extractor output (abbreviated):

```json
{
  "version": "1",
  "module": "fixture-ts",
  "language": "ts",
  "packages": [{
    "id": "pkg:fixture-ts/src",
    "importPath": "fixture-ts/src",
    "name": "src",
    "fileIds": ["file:src/greeter.ts", "file:src/main.ts"],
    "symbolIds": ["sym:fixture-ts/src.alias.Prefix", "sym:fixture-ts/src.class.Greeter",
                  "sym:fixture-ts/src.const.MaxRetries", "sym:fixture-ts/src.func.greet",
                  "sym:fixture-ts/src.iface.Greetable", "sym:fixture-ts/src.method.Greeter.hello", ...]
  }],
  "files": [{ "id": "file:src/greeter.ts", "path": "src/greeter.ts", "sha256": "9a27...", ... }, ...],
  "symbols": [{
    "id": "sym:fixture-ts/src.class.Greeter",
    "kind": "class", "name": "Greeter",
    "range": { "startLine": 2, "startCol": 1, "endLine": 9, "endCol": 2,
               "startOffset": 62, "endOffset": 244 },
    ...
  }, ...]
}
```

Byte offsets match `ts.Node.getStart/getEnd()` precisely, so the existing `/api/snippet` byte-slice logic works unchanged. SHA256 matches a recompute of the file bytes. Record shape is a superset of what the Go indexer emits — the only addition is `language: "ts"` on each record.

## 4. Gap analysis

| Concern | Already handled | TS-specific work needed |
|---|---|---|
| Parse TS + type info | `typescript` npm package (Compiler API) | — |
| Package concept | Go: import path | TS: directory path under the project root; emit as `pkg:<module>/<rel-dir>` |
| Stable IDs | `sym:<importPath>.<Kind>.<Name>` | Works as-is; `Kind` gets `class` and method IDs use receiver = class name |
| Byte ranges | `node.getStart(sf)` / `getEnd()` | Same idea as `token.FileSet.Position(...)` — no mapping needed |
| Doc comments | JSDoc via `(node as any).jsDoc` | Separate from Go's `*ast.CommentGroup` but equivalent semantic |
| Xref (phase 2) | `ts.TypeChecker.getSymbolAtLocation` + resolved symbol path | More work than Go's `types.Info.Uses`, but feasible |
| Orchestration | Dagger for web build (skill) | New `cmd/build-ts-index` that reuses the same CacheVolume pattern |
| Frontend highlighter | Go tokenizer at `ui/src/packages/ui/src/highlight/go.ts` | New `highlight/ts.ts` (same shape, TS keywords + JSX) |
| Merge two indexes | Not implemented | Deterministic merge that preserves ID ordering |
| Schema addition | None | `Symbol.language` + `File.language` + `Package.language` (all optional, default `"go"`) |

## 5. Proposed architecture

### 5.1 Topology

```
                              ┌──────────────────────────────────┐
                              │  codebase-browser index build    │
                              │    --lang auto --module-root .   │
                              └──────────────────┬───────────────┘
                                                 │
                                 ┌───────────────┴───────────────┐
                                 │                               │
                                 ▼                               ▼
                ┌────────────────────────┐        ┌──────────────────────────────┐
                │ Go extractor (in-proc) │        │ Dagger pipeline               │
                │ internal/indexer/go.go │        │ cmd/build-ts-index/main.go    │
                │ → *indexer.Index (Go)  │        │   node:22 + pnpm + typescript │
                │                        │        │   → tools/ts-indexer/         │
                │                        │        │     runs extract.ts           │
                │                        │        │     → /out/index-ts.json      │
                │                        │        │     export to host            │
                └────────────┬───────────┘        └────────────────┬─────────────┘
                             │                                     │
                             └─────────────────┬───────────────────┘
                                               ▼
                                  ┌─────────────────────────┐
                                  │ Merge (internal/indexer/│
                                  │        merge.go)        │
                                  │ → internal/indexfs/     │
                                  │   embed/index.json      │
                                  └─────────────────────────┘
```

### 5.2 Repository layout (additions)

```
cmd/
  build-ts-index/              # NEW — Dagger program
    main.go                    # connect, mount ui dir, run Node extractor, export
tools/
  ts-indexer/                  # NEW — Node extractor, versioned in repo
    package.json               # "packageManager": "pnpm@10.15.1", deps: typescript, zod(optional)
    pnpm-lock.yaml
    tsconfig.json
    src/
      extract.ts               # TypeScript Compiler API walk → Index JSON
      types.ts                 # shared shape with Go (mirror of internal/indexer/types.go)
      refs.ts                  # xref pass (phase 2)
      cli.ts                   # argv parsing, stdout/file output
    bin/
      ts-indexer.js            # compiled entrypoint, generated by tsc
internal/
  indexer/
    extractor.go               # refactored — defines Extractor interface + goExtractor
    go_extractor.go            # NEW — existing Extract() renamed into method receiver
    ts_extractor.go            # NEW — shells to build-ts-index (Dagger) or local-pnpm fallback
    merge.go                   # NEW — Merge([]Index) Index with stable ordering
    types.go                   # +Language field on Package/File/Symbol (optional, default "go")
ui/src/packages/ui/src/highlight/
  ts.ts                        # NEW — TS tokenizer (mirrors go.ts)
```

### 5.3 Data flow

1. **Go build-time path** (unchanged): `internal/indexer/go_extractor.go:Extract(modRoot, patterns)` → `*indexer.Index` (with `language: "go"` stamped on every record).
2. **TS build-time path** (new): `cmd/build-ts-index/main.go` runs Dagger:
   - `client.Container().From("node:22-bookworm")`
   - `WithMountedCache("/pnpm/store", pnpmStore)` (named `<module>-ts-indexer-pnpm-store`)
   - `WithDirectory("/src", <target-tree>)` filtered via `Exclude`
   - `WithWorkdir("/src/tools/ts-indexer")`
   - `WithExec corepack + pnpm@<version>; pnpm install --frozen-lockfile`
   - `WithExec node bin/ts-indexer.js --module-root /src --tsconfig /src/ui/tsconfig.json --out /out/index-ts.json`
   - `Container.File("/out/index-ts.json").Export(ctx, localPath)`
3. **Merge**: `internal/indexer/merge.go:Merge(go, ts)` concatenates `Packages`, `Files`, `Symbols`, `Refs`; re-sorts per the existing rules in `sortIndex`; dedupes IDs (collision is an error, not a silent drop).
4. **Write**: single `index.json` consumed by server unchanged.

### 5.4 Language stamping

```go
// internal/indexer/types.go (additive)
type Package struct {
    ID         string   `json:"id"`
    ImportPath string   `json:"importPath"`
    Name       string   `json:"name"`
    Language   string   `json:"language,omitempty"` // "go" | "ts" — empty defaults to "go" for rollback compatibility
    // ... rest unchanged
}

type File struct {
    ID        string `json:"id"`
    Path      string `json:"path"`
    Language  string `json:"language,omitempty"`
    // ... rest unchanged
}

type Symbol struct {
    ID        string `json:"id"`
    Kind      string `json:"kind"`
    Language  string `json:"language,omitempty"`
    // ... rest unchanged
}
```

The `,omitempty` + "empty means Go" convention means existing indexes keep working (rollback-safe).

### 5.5 Kind vocabulary

| Kind | Go | TS | Notes |
|---|---|---|---|
| `func` | top-level function | top-level `function` / exported arrow const | |
| `method` | `func (r *T) Foo(...)` | `class X { foo() {} }` | ID scheme: `sym:<pkg>.method.<recv>.<name>` — same in both languages |
| `type` | `type T = ...` | would be `alias`, see below | |
| `iface` | `type T interface {...}` | `interface T {...}` | |
| `struct` | `type T struct {...}` | N/A for TS | TS uses `class` |
| `class` | N/A for Go | `class X {...}` | **new kind** |
| `alias` | `type T = otherType` | `type T = ...` | |
| `const` / `var` | `const`/`var` declaration | `const`/`let`/`var` declaration | |

`class` is a new Kind added to the vocabulary. The frontend's `data-role="class"` CSS rule gets a new color token (`--cb-color-kind-class`); everything else works unchanged.

## 6. TypeScript extractor details

### 6.1 Node entrypoint

```ts
// tools/ts-indexer/src/cli.ts
import { extract } from './extract';
import * as fs from 'fs';

const args = parseArgs(process.argv.slice(2));
const idx = extract({
  moduleRoot: args.moduleRoot,
  tsconfig: args.tsconfig ?? 'tsconfig.json',
});
if (args.out === '-') {
  process.stdout.write(JSON.stringify(idx, null, 2) + '\n');
} else {
  fs.writeFileSync(args.out, JSON.stringify(idx, null, 2) + '\n');
}
```

The extractor core is the prototype at `scripts/extract.ts` in this ticket, promoted and packaged:

1. Parse `tsconfig.json` via `ts.parseJsonConfigFileContent`.
2. `ts.createProgram({rootNames, options})`.
3. For each non-declaration source file inside `moduleRoot`:
   - Emit a `Package` keyed by directory.
   - Emit a `File` with SHA256 of the on-disk bytes.
   - Walk top-level nodes via `ts.forEachChild(sf, …)`.
   - Handle `FunctionDeclaration`, `ClassDeclaration` (+ its members), `InterfaceDeclaration`, `TypeAliasDeclaration`, `VariableStatement`, `EnumDeclaration`.
   - For each symbol, capture byte range + line/col + JSDoc + signature-prefix (text up to first `{`).
4. Emit the Index with deterministic sort (same rules as Go: by `importPath` / `path` / `(packageId, fileId, startOffset)`).

### 6.2 Signature rendering

Unlike Go where we use `printer.Config.Fprint`, TS has `ts.TypeChecker.typeToString()` for full types. For phase 1 we keep it simple: signature = source text up to first `{`. This works for:

1. `function foo<T>(x: T): T` → `function foo<T>(x: T): T`
2. `class Greeter {` → `class Greeter`
3. `const MaxRetries = 3` → `const MaxRetries = 3`

Generics, union types, conditional types all pass through because we're slicing bytes, not re-printing.

### 6.3 JSDoc extraction

TypeScript parses JSDoc comments into `node.jsDoc` (non-public API, accessed via `(node as any).jsDoc`). Phase 1 concatenates all `jsDoc[].comment` strings. `@deprecated` tags aren't parsed out separately yet — the frontend's existing `detectLeadingAnnotation` already detects leading `Deprecated:` prefixes, and we can follow up by extracting `@deprecated` into a prefix line.

### 6.4 Xref (phase 2)

The TS equivalent of `types.Info.Uses` is `ts.TypeChecker.getSymbolAtLocation(ident).declarations[0]`:

```ts
function walkRefs(sf: ts.SourceFile, checker: ts.TypeChecker, fromSymbolId: string): Ref[] {
  const out: Ref[] = [];
  ts.forEachChild(sf, function visit(n: ts.Node) {
    if (ts.isIdentifier(n) && !isDeclarationName(n)) {
      const sym = checker.getSymbolAtLocation(n);
      const decl = sym?.declarations?.[0];
      if (decl && decl.getSourceFile() !== sf /* or same-file ref */) {
        const toSymbolId = resolveToSymbolId(decl);
        if (toSymbolId) out.push({ fromSymbolId, toSymbolId, kind: 'call',
                                   fileId: fileIdOf(sf), range: rangeFrom(sf, n.getStart(), n.getEnd()) });
      }
    }
    ts.forEachChild(n, visit);
  });
  return out;
}
```

The hard part is `resolveToSymbolId`: given a `ts.Declaration`, reproduce the same ID the extractor stamped on it. Either (a) keep a `Map<ts.Declaration, string>` built during symbol extraction, or (b) re-derive the ID from the declaration's shape. Option (a) is simpler and robust. Phase 1 skips xref; phase 2 adds it.

## 7. Dagger orchestration

### 7.1 Why Dagger

1. **Hermetic builds**: matches the existing `go-web-dagger-pnpm-build` pattern we already use for Vite — same CacheVolume, same `corepack prepare pnpm@<version> --activate`, same output-export idiom.
2. **CI parity**: any machine with `dagger` + Docker produces bit-identical `index-ts.json`, independent of the host's Node version.
3. **Cache reuse**: the pnpm store is per-project-per-scope, so consecutive builds of the same project reuse the same cache volume.
4. **Fallback simplicity**: if Dagger isn't available, shell out to local `pnpm` with the same arguments.

### 7.2 `cmd/build-ts-index/main.go`

Modelled directly on Smailnail's `cmd/build-web/main.go` (see skill §5.1). Skeleton:

```go
// cmd/build-ts-index/main.go
package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "dagger.io/dagger"
    "github.com/pkg/errors"
)

const defaultPNPMVersion = "10.15.1"

func main() {
    ctx := context.Background()
    if err := run(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run(ctx context.Context) error {
    modRoot, _ := filepath.Abs(envDefault("MODULE_ROOT", "."))
    outPath, _ := filepath.Abs(envDefault("TS_INDEX_OUT", "internal/indexfs/embed/index-ts.json"))
    tsconfig := envDefault("TS_TSCONFIG", "ui/tsconfig.json")

    // Opportunistic local fallback — skip Dagger if the engine isn't reachable.
    if os.Getenv("BUILD_TS_LOCAL") == "1" {
        return runLocal(ctx, modRoot, outPath, tsconfig)
    }

    client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
    if err != nil {
        // Second-chance fallback if the daemon is unreachable.
        fmt.Fprintln(os.Stderr, "dagger unavailable, falling back to local pnpm")
        return runLocal(ctx, modRoot, outPath, tsconfig)
    }
    defer func() { _ = client.Close() }()

    pnpmVer := envDefault("WEB_PNPM_VERSION", defaultPNPMVersion)
    src := client.Host().Directory(modRoot, dagger.HostDirectoryOpts{
        Exclude: []string{
            "**/node_modules",
            "**/.git",
            "ui/dist",
            "internal/indexfs/embed",
            "internal/web/embed",
            "bin",
            "storybook-static",
        },
    })
    pnpmStore := client.CacheVolume(cacheName(modRoot))

    ctr := client.Container().
        From("node:22-bookworm").
        WithEnvVariable("PNPM_HOME", "/pnpm").
        WithEnvVariable("PATH", "/pnpm:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin").
        WithMountedCache("/pnpm/store", pnpmStore).
        WithDirectory("/src", src).
        WithWorkdir("/src/tools/ts-indexer").
        WithExec([]string{"sh", "-lc", "corepack enable && corepack prepare pnpm@" + pnpmVer + " --activate"}).
        WithExec([]string{"pnpm", "install", "--frozen-lockfile", "--prefer-offline"}).
        WithExec([]string{"pnpm", "run", "build"}).
        WithExec([]string{"node", "bin/ts-indexer.js",
            "--module-root", "/src",
            "--tsconfig", "/src/" + strings.TrimPrefix(tsconfig, "./"),
            "--out", "/out/index-ts.json",
        })

    if _, err := ctr.File("/out/index-ts.json").Export(ctx, outPath); err != nil {
        return errors.Wrap(err, "export index-ts.json")
    }
    return nil
}

func runLocal(ctx context.Context, modRoot, outPath, tsconfig string) error { /* exec pnpm install && node bin/ts-indexer.js ... */ }

func cacheName(modRoot string) string { return "codebase-browser-ts-indexer-" + filepath.Base(modRoot) }

func envDefault(k, fallback string) string { if v := os.Getenv(k); v != "" { return v }; return fallback }
```

Key alignment points with the skill's checklist:

1. ✅ `dagger.WithLogOutput(os.Stdout)`.
2. ✅ `CacheVolume("codebase-browser-ts-indexer-<proj>")`.
3. ✅ `WithMountedCache("/pnpm/store", ...)`.
4. ✅ `--frozen-lockfile --prefer-offline`.
5. ✅ pnpm version configurable via `WEB_PNPM_VERSION`.
6. ✅ Local fallback via `BUILD_TS_LOCAL=1`.
7. ✅ `go:generate` wiring (see §7.3).

### 7.3 `go:generate` wiring

```go
// internal/indexer/generate.go
package indexer

//go:generate go run ../../cmd/build-ts-index
```

And a new Makefile target:

```make
generate-ts:
	go run ./cmd/build-ts-index

generate: generate-web generate-ts
	go run ./cmd/codebase-browser index build --lang auto
```

### 7.4 Index-path contract

- `internal/indexfs/embed/index.json` — the merged, canonical index. Consumed by server.
- `internal/indexfs/embed/index-ts.json` — intermediate Node extractor output. Committed? **No**, gitignore it. The merger reads it + the in-process Go index and writes the canonical file.

### 7.5 First-run bootstrap

The ts-indexer's compiled `bin/ts-indexer.js` is written into `tools/ts-indexer/bin/` by `pnpm run build` inside Dagger. Since the container runs on the mounted source and exports only `/out/index-ts.json`, the compiled `bin/` is ephemeral unless we choose to export it too. Recommended: don't export — keep `tools/ts-indexer/bin/` gitignored, rebuild per-run. The cache volume makes repeat `pnpm install` cheap; `tsc` compilation of ~500 lines is fast.

## 8. Go-side refactor: pluggable extractors

### 8.1 Interface

```go
// internal/indexer/extractor.go
package indexer

import "context"

type Extractor interface {
    Language() string
    Extract(ctx context.Context, opts ExtractOptions) (*Index, error)
}
```

### 8.2 Implementations

```go
// internal/indexer/go_extractor.go — existing Extract() renamed into a receiver method
type goExtractor struct{}
func NewGoExtractor() Extractor { return goExtractor{} }
func (goExtractor) Language() string { return "go" }
func (goExtractor) Extract(ctx context.Context, opts ExtractOptions) (*Index, error) {
    return extractGo(opts) // the existing function, moved verbatim
}

// internal/indexer/ts_extractor.go — thin wrapper around build-ts-index
type tsExtractor struct{ BuilderCmd string }
func NewTSExtractor() Extractor { return &tsExtractor{BuilderCmd: "go run ./cmd/build-ts-index"} }
func (e *tsExtractor) Language() string { return "ts" }
func (e *tsExtractor) Extract(ctx context.Context, opts ExtractOptions) (*Index, error) {
    // Invoke build-ts-index with MODULE_ROOT=opts.ModuleRoot, TS_INDEX_OUT=/tmp/... ,
    // then read and unmarshal the resulting JSON.
}
```

### 8.3 Composition

```go
// internal/indexer/multi.go
func Extract(ctx context.Context, langs []string, opts ExtractOptions) (*Index, error) {
    var parts []*Index
    for _, l := range langs {
        ext := lookup(l)
        idx, err := ext.Extract(ctx, opts)
        if err != nil {
            return nil, fmt.Errorf("extractor %s: %w", l, err)
        }
        parts = append(parts, idx)
    }
    return Merge(parts), nil
}
```

### 8.4 Merge rules

1. `Packages` — concatenate, dedupe by `ID` (collision is an error; TS and Go packages live in disjoint namespaces by import path).
2. `Files` — concatenate, dedupe by `ID` (no expected collision).
3. `Symbols` — concatenate; `sym:` IDs are unique across languages because they include the package import path which embeds the `module` name (Go import path vs. `module/<dir>`).
4. `Refs` — concatenate; no dedupe needed.
5. Re-sort the merged slices with the existing `sortIndex` rules so output stays deterministic.
6. Single top-level `Module`: if all parts share the same `module`, keep it; otherwise set to `"<module-a>+<module-b>"` (fine-for-now mixed-module placeholder, revisit if needed).

## 9. Frontend changes

### 9.1 Tokenizer dispatch

```ts
// ui/src/packages/ui/src/Code.tsx — current
const tokens = language === 'go' ? tokenize(text) : [{ type: 'id', text }];

// → new
const tokens = tokenizeForLanguage(language, text);

// ui/src/packages/ui/src/highlight/index.ts — new dispatch
import { tokenize as tokenizeGo } from './go';
import { tokenize as tokenizeTS } from './ts';
export function tokenizeForLanguage(lang: string, src: string): Token[] {
  switch (lang) {
    case 'go': return tokenizeGo(src);
    case 'ts':
    case 'tsx':
    case 'typescript': return tokenizeTS(src);
    default: return [{ type: 'id', text: src }];
  }
}
```

### 9.2 TypeScript tokenizer

`highlight/ts.ts` mirrors `highlight/go.ts`:

1. Keyword list: ~60 TS/JS keywords (`class`, `interface`, `function`, `const`, `let`, `var`, `if`, `else`, `for`, `while`, `do`, `switch`, `case`, `break`, `continue`, `return`, `throw`, `try`, `catch`, `finally`, `new`, `delete`, `typeof`, `instanceof`, `in`, `of`, `yield`, `await`, `async`, `export`, `import`, `from`, `as`, `type`, `enum`, `extends`, `implements`, `public`, `private`, `protected`, `readonly`, `abstract`, `static`, `override`, `declare`, `namespace`, `module`, `void`, `null`, `undefined`, `this`, `super`).
2. Builtins: `string`, `number`, `boolean`, `any`, `unknown`, `never`, `object`, `symbol`, `bigint`, `true`, `false`, `Array`, `Promise`, `Map`, `Set`.
3. Strings: `"..."`, `'...'`, `` `...` `` (template literals — handle `${}` interpolation as nested context; phase 1 treats interpolation as plain string with inner dollar-braces visually distinct via a `data-tok="punct"` on `${}`).
4. Numbers: same as Go tokenizer (`0x`, `0b`, `0o`, decimal, float, `1n` bigint).
5. Line comment `//`; block comment `/* */`; JSDoc `/** */` stays as `com` but the annotation post-processor picks up `@deprecated` (analogous to Go's `Deprecated:`).
6. Identifiers: same rules.
7. JSX: tokenize `<Name>` and `</Name>` as punct + id sequences in phase 1. Full JSX lexing (attribute values, whitespace inside tags) is a follow-up.

### 9.3 Kind vocabulary update

Add CSS var + rule for `class`:

```css
[data-widget='codebase-browser'] {
  --cb-color-kind-class: #8b5cf6;
}
[data-widget='codebase-browser'] [data-part='symbol-kind'][data-role='class'] {
  background: var(--cb-color-kind-class);
}
```

## 10. Implementation plan (phased)

### Phase 1 — Go-side pluggability (1 day)

1. Refactor `internal/indexer/extractor.go`: extract the current body into `extractGo()` in `go_extractor.go`, expose `Extractor` interface.
2. Add `Language` field to `Package`/`File`/`Symbol` with `json:",omitempty"`. Stamp `"go"` in the Go extractor.
3. Add `Merge([]*Index) *Index` in `merge.go` with re-application of `sortIndex`.
4. Unit tests: merge two fixture indexes; assert counts + deterministic sort.

### Phase 2 — Node extractor (1-2 days)

1. Create `tools/ts-indexer/` with `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `src/extract.ts`, `src/cli.ts`.
2. Migrate `scripts/extract.ts` → `tools/ts-indexer/src/extract.ts` with the interface split.
3. Add `pnpm run build` producing `bin/ts-indexer.js`.
4. Add a small fixture test: extract the fixture module, assert record count + known IDs (equivalent to Go's golden test).

### Phase 3 — Dagger orchestration (1 day)

1. Create `cmd/build-ts-index/main.go` per §7.2 skeleton.
2. Validate: run `go run ./cmd/build-ts-index` locally (with Dagger engine); check `internal/indexfs/embed/index-ts.json` appears.
3. Fallback: run `BUILD_TS_LOCAL=1 go run ./cmd/build-ts-index` on a machine without Dagger; assert local-pnpm path works.
4. Add `go generate` directive + Makefile target.

### Phase 4 — Glazed CLI wiring (0.5 day)

1. Add `--lang` flag to `codebase-browser index build`. Values: `go` (default, unchanged), `ts`, `auto`.
2. `auto` = run both in parallel; Merge; write canonical `index.json`.
3. Log counts by language (`go=107 ts=23`) in the emitted row.

### Phase 5 — Frontend (1 day)

1. Add `highlight/ts.ts` mirroring `highlight/go.ts`.
2. Add `tokenizeForLanguage` dispatch in `highlight/index.ts`.
3. Extend CSS: `--cb-color-kind-class` + `data-role='class'` rule in `base.css` and `dark.css`.
4. Storybook: `Code` story `TypeScript` variant; `SymbolCard` story for `class` kind.

### Phase 6 — TS xref (follow-up, 2 days)

1. Build a `Map<ts.Declaration, string>` of declaration-to-symbolID during extraction.
2. Walk each function body + class method body emitting `Ref`s via `getSymbolAtLocation`.
3. Unify with Go's xref on the server side — the `/api/xref/{id}` endpoint already loops over all refs regardless of language.

### Phase 7 — JSX + `.tsx` polish (follow-up, 1 day)

1. Handle `.tsx` files in the extractor (they already load; make sure symbol extraction works on `export default function MyComponent() {}`).
2. JSX tokenizer.

## 11. Testing strategy

| Layer | Test | Tooling |
|---|---|---|
| TS extractor | Fixture module → expected symbol list | Vitest (inside `tools/ts-indexer/`), run in Dagger container |
| TS extractor determinism | Run extract twice, diff-free | Vitest |
| Merge | Two fixture indexes (Go + TS) → merged totals + sort stability | `go test ./internal/indexer` |
| Dagger program | Smoke: `go run ./cmd/build-ts-index` writes `index-ts.json` | Manual + CI `go run` |
| Local fallback | `BUILD_TS_LOCAL=1` path produces the same JSON | CI with `dagger` uninstalled |
| End-to-end | `go run ./cmd/codebase-browser index build --lang auto`; server starts; `/api/symbol/sym:<ts-pkg>.class.Greeter` resolves | `go test` with httptest |
| Tokenizer | TS fixture → expected token stream | Existing `highlight/go.test.ts` pattern; mirror for `ts.test.ts` |

## 12. Risks, alternatives, and open questions

### 12.1 Risks

1. **TypeScript Compiler API is a non-trivial dep** (~10 MB). Mitigation: pinned version, Dagger cache volume keeps install cheap.
2. **JSDoc access uses `(node as any).jsDoc`** — undocumented-but-stable for years; if it breaks we move to `ts.getJSDocTags()` per node.
3. **Two-process orchestration** — CI now needs Docker for Dagger. Mitigation: local-pnpm fallback (`BUILD_TS_LOCAL=1`) keeps ci-lite paths viable.
4. **Index size doubling** — a polyglot repo has roughly `|go symbols| + |ts symbols|` entries. For current scope (this repo: 112 Go + ~40 TS), still small (≤ 200 KB).
5. **`class` Kind not visually distinct from `struct`** — same color family is fine; the `data-role` selector keeps them separately themable.

### 12.2 Alternatives considered

1. **tree-sitter-typescript in-process**: pure-Go, no Node dep. Rejected for phase 1 because it loses type info (no reliable xref) and struct-similar heuristics on TS generics get noisy. Revisit if Dagger ever becomes a blocker.
2. **Running `tsc --emitDeclarationOnly` and parsing `.d.ts`**: narrower than Compiler API but loses function/class bodies we need for snippets. Rejected.
3. **`ts-morph`**: higher-level wrapper around the Compiler API. Nicer DX, more deps. Neutral; fine to adopt later if phase 1 `extract.ts` proves too verbose.
4. **Serverside live indexing (no build step)**: shell `tsc` on every `serve` start. Rejected — too slow, breaks the immutable-binary property.
5. **Separate server for TS**: reject — doubles the surface and breaks the "one binary" thesis.

### 12.3 Open questions

1. **Commit `tools/ts-indexer/pnpm-lock.yaml`?** Yes (determinism).
2. **Commit `tools/ts-indexer/bin/ts-indexer.js`?** No (build artifact). The Dagger container builds it from source each run; local dev can `cd tools/ts-indexer && pnpm run build` once.
3. **`extract` with `includes`/`excludes`?** Phase 1: respect `tsconfig.json` includes only. Phase 2: add `--exclude` pattern to the CLI.
4. **What about `.js` files (plain JavaScript)?** Phase 1 skips. `tsconfig.json allowJs: true` would surface them; defer until someone asks.
5. **Incremental?** `tsc --incremental`'s `.tsbuildinfo` output could speed up repeat runs. Phase 1 ignores; re-evaluate if index build takes > 5 s.
6. **How to show language in the UI?** Options: small chip next to the symbol kind (`go` / `ts`), or filter dropdown on `SearchPanel`. Recommendation: filter dropdown, skip the chip (kind color already encodes most useful info).

## 13. References

1. GCB-001 design doc — `ttmp/2026/04/20/GCB-001--go-codebase-browser-ast-indexed-embedded-themed/design-doc/01-go-codebase-browser-analysis-and-implementation-guide.md`
2. `go-web-dagger-pnpm-build` skill — `/home/manuel/.claude/skills/go-web-dagger-pnpm-build/SKILL.md`
3. Smailnail reference — `/home/manuel/code/wesen/corporate-headquarters/smailnail/cmd/build-web/main.go`
4. TypeScript Compiler API docs — https://github.com/microsoft/TypeScript/wiki/Using-the-Compiler-API
5. Dagger Go SDK — https://docs.dagger.io/sdk/go
6. Validating prototype — `scripts/extract.ts` + `scripts/fixture-ts/` in this ticket
