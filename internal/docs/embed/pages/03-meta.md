# The browser, browsed by itself

This page is a small demonstration of what the codebase-browser is *for*:
it embeds its own source via the same `codebase-snippet` directive it
exposes to third-party doc authors. Every fenced block below is resolved
from the live index — click any linked identifier to jump into the tree.

## Two languages, one index

The Go indexer and the TypeScript indexer emit JSON in the same shape;
`indexer.Merge` combines them into the single file that the server
embeds. Duplicate-ID detection short-circuits the merge rather than
silently dropping records, which turned out to catch a real bug during
GCB-002 (Storybook `const meta` colliding across `*.stories.tsx`
files).

```codebase-signature sym=github.com/wesen/codebase-browser/internal/indexer.Merge
```

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/indexer.Merge
```

## How TypeScript refs show up

The TS extractor is two-pass. Pass 1 registers every emitted declaration
into a `Map<ts.Declaration, string>`. Pass 2 walks function/method
bodies, calling `TypeChecker.getSymbolAtLocation` on each identifier and
following named-import aliases via `getAliasedSymbol` so refs target the
real exported declaration, not the local import binding.

```codebase-snippet sym=sym:ui/src/packages/ui/src/highlight/ts.func.tokenize
```

The same ref records flow into the same `/api/xref/{id}` endpoint that
Go symbols use — the server is language-agnostic.

```codebase-signature sym=github.com/wesen/codebase-browser/internal/server.Server.handleXref
```

## Dagger orchestrates the Node toolchain from Go

`cmd/build-ts-index` is invoked by `go generate ./internal/indexfs`. It
mounts `tools/ts-indexer/` + the module root into a `node:22` container,
uses a pnpm `CacheVolume` for the store, and exports the resulting
`index-ts.json` back to the host. Setting `BUILD_TS_LOCAL=1` bypasses
Dagger and shells out to local `pnpm + node` instead — both paths
produce byte-identical output (sha256-verified).

```codebase-signature sym=github.com/wesen/codebase-browser/cmd/build-ts-index.runDagger
```

## Frontend: language-dispatched highlighting

The React side doesn't need to know a symbol's language at render time —
it threads `language` through to the `<Code>` component, which asks the
`tokenizeForLanguage` dispatcher in `highlight/index.ts` for the right
tokenizer. JSX component names get the small polish of being tagged as
`type` so they stand out from lowercase DOM elements.

```codebase-snippet sym=sym:ui/src/packages/ui/src/highlight/ts.func.isJsxComponentRef
```

## Try it yourself

Every `codebase-snippet` block above is resolved at request time by
`internal/docs/renderer.go`. Write your own markdown under
`internal/docs/embed/pages/` and it'll appear in the Docs list on
startup — no code changes required.
