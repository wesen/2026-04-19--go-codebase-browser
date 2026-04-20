# codebase-browser

A single-binary documentation browser for Go + TypeScript codebases. A
build-time indexer walks the AST, the Go binary embeds the resulting
`index.json` (and a snapshot of the source tree), and a small React SPA
renders packages, files, symbols, cross-references, and doc pages — all
served from the one binary with no runtime dependencies.

The browser can document *itself*: markdown pages under
`internal/docs/embed/pages/` can embed live source via
`codebase-snippet sym=<id>` directives, resolved against the embedded
index at request time. See `03-meta.md` for a worked example.

## Why

- **One binary, one download.** The index, the source, and the SPA are
  all embedded. Ship it with your library so readers can browse without
  cloning.
- **Cross-language in the same index.** The Go extractor
  (`golang.org/x/tools/go/packages`) and the TypeScript extractor
  (TS Compiler API, in Node) emit records in the same shape. They merge
  into one file; the server is language-agnostic.
- **Deterministic.** Stable symbol IDs (`sym:<importPath>.<kind>.<name>`)
  survive file moves; sorted output gives reproducible builds.

## Quick start

Prerequisites: Go 1.22+ and optional Docker for the hermetic Dagger build path.
Node 22+ and pnpm 10.x are only needed if you want to work on the frontend
or the TypeScript indexer directly.

```bash
# 1) Optional: install UI deps for local frontend / indexer work
pnpm -C ui install
pnpm -C tools/ts-indexer install

# 2) Build index + frontend bundle, embed, and compile the binary
make build              # runs `go generate` on the generator packages, then `go build -tags embed`

# 3) Run
./bin/codebase-browser serve --addr :3001
# open http://localhost:3001
```

For the tight dev loop (no embedding, hot reload):

```bash
make dev-backend        # :3001 — serves from disk
make dev-frontend       # :3000 — Vite + proxy to :3001
```

## Building the index

`go generate ./internal/indexfs` runs `codebase-browser index build
--lang auto`, which:

1. Walks the Go module at `--module-root` (default `.`) via
   `golang.org/x/tools/go/packages` and extracts every top-level
   declaration plus cross-references from function bodies.
2. Shells to `cmd/build-ts-index`, which runs the Node extractor
   (`tools/ts-indexer`) on the TypeScript module at `--ts-module-root`
   (default `ui`). Dagger orchestrates a `node:22` container with a
   pnpm `CacheVolume`; set `BUILD_TS_LOCAL=1` to fall back to local
   `pnpm + node` when Docker isn't available.
3. `go generate ./internal/web` builds the Vite SPA in a Dagger
   container and copies the generated `ui/dist/public` assets into
   `internal/web/embed/public/`; set `BUILD_WEB_LOCAL=1` to fall back to
   local `pnpm` when Docker isn't available.
4. Calls `indexer.Merge` to stitch both parts together, detecting
   duplicate IDs rather than silently dropping records.
5. Writes `internal/indexfs/embed/index.json`, picked up by the
   `//go:embed` in `internal/indexfs/embed.go` on the next
   `go build -tags embed`.

Separately, `go generate ./internal/sourcefs` mirrors the repository
source tree into `internal/sourcefs/embed/source/`, excluding build
outputs and local caches so snippet lookups stay deterministic.

Flags on `codebase-browser index build`:

| Flag | Default | Purpose |
|------|---------|---------|
| `--lang` | `go` | `go`, `ts`, or `auto` |
| `--module-root` | `.` | Go module root (contains `go.mod`) |
| `--ts-module-root` | `ui` | TypeScript module root |
| `--index-path` | `internal/indexfs/embed/index.json` | Merged output |
| `--ts-index-path` | `internal/indexfs/embed/index-ts.json` | Intermediate TS JSON |
| `--pretty` | `true` | Indent JSON |

## Adding a doc page

Drop a markdown file under `internal/docs/embed/pages/`. Any fenced
block with an info string of `codebase-snippet`, `codebase-signature`,
or `codebase-doc` is replaced at render time with the named symbol's
body, signature, or godoc respectively:

````markdown
## My component

```codebase-signature sym=sym:ui/src/packages/ui/src/SymbolCard.func.SymbolCard
```

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/indexer.Merge
```
````

Short refs work for unambiguous cases: `github.com/.../indexer.Merge`.
Use full `sym:` IDs when a name collides across files in the same
package (common in TS — multiple `*.stories.tsx` with `const meta`).

## Repo layout

```
cmd/codebase-browser/     Main CLI (glazed commands: serve, index, doc, symbol)
cmd/build-ts-index/       Dagger orchestrator for the Node TS extractor
internal/indexer/         Go AST → Index JSON + Merge
internal/browser/         Index loader shared by server + CLI
internal/server/          /api/* HTTP handlers
internal/web/             Vite build embed (SPA assets)
internal/sourcefs/        Source tree embed (for snippet slicing)
internal/indexfs/         index.json embed + go:generate wiring
internal/docs/            Markdown renderer + embedded doc pages
tools/ts-indexer/         Node + TS Compiler API extractor
ui/                       React SPA (RTK-Query + Storybook)
ttmp/                     Ticket workspaces (docmgr)
```

## Testing

```bash
make test                 # go test ./... (server, indexer, docs, merge)
pnpm -C ui run typecheck  # tsc --noEmit for the SPA
pnpm -C tools/ts-indexer test  # vitest (extractor + xref + JSX fixtures)
make smoke                # build embed binary, curl /api/index
```

## Documentation

Tickets and design docs live under `ttmp/`:

- **GCB-001** — original design, 10-phase implementation plan for the
  Go-only browser.
- **GCB-002** — TypeScript support via Node + Dagger, merge pass,
  frontend dispatcher.

Both tickets carry a diary (`reference/01-investigation-diary.md`) and
a design doc (`design-doc/01-*.md`).
