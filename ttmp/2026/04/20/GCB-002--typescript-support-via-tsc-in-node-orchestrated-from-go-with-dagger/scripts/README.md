# GCB-002 prototype scripts

Validating prototype for the TypeScript extractor described in
[`../design-doc/01-typescript-extractor-design-and-implementation-guide.md`](../design-doc/01-typescript-extractor-design-and-implementation-guide.md).

## Run

```bash
pnpm install
pnpm run extract fixture-ts
```

Expected output: a JSON document with `packages: 1`, `files: 2`,
`symbols: 7` (class/method/const/func/iface/alias/const), deterministic
sort order.

Pipe to `jq` for a quick summary:

```bash
./node_modules/.bin/tsx extract.ts fixture-ts | jq '{
  module,
  packages: (.packages|length),
  files: (.files|length),
  symbols: (.symbols|length),
  kinds: [.symbols[].kind]
}'
```

## Files

- `extract.ts` — prototype extractor using the TypeScript Compiler API.
- `fixture-ts/` — tiny TS module with class/method/const/func/iface/alias
  declarations covering the phase-1 kind vocabulary.
- `package.json` — declares `tsx` + `typescript` devDependencies.

## Scope

This is a design-validation prototype only. The implementation in
`tools/ts-indexer/` (phase 2 of GCB-002) will promote this extractor into
a versioned package with vitest coverage and a CLI that matches the
`cmd/build-ts-index` Dagger program's expectations.
