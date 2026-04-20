---
Title: Investigation Diary
Ticket: GCB-002
Status: active
Topics:
    - typescript
    - dagger
    - node-tooling
    - multi-language
    - go-ast
    - codebase-browser
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/20/GCB-002--typescript-support-via-tsc-in-node-orchestrated-from-go-with-dagger/design-doc/01-typescript-extractor-design-and-implementation-guide.md
      Note: The primary deliverable this diary narrates
    - Path: ttmp/2026/04/20/GCB-002--typescript-support-via-tsc-in-node-orchestrated-from-go-with-dagger/scripts/extract.ts
      Note: Prototype extractor validated in Step 2
ExternalSources: []
Summary: 'Diary for GCB-002: scaffolding a Node/TypeScript Compiler API extractor and a Dagger orchestration path for it, validated via a working prototype in scripts/.'
LastUpdated: 2026-04-20T00:00:00Z
WhatFor: Design-phase reasoning + prototype validation for GCB-002.
WhenToUse: Read alongside the design doc before starting Phase 1 of implementation.
---


# Investigation Diary

## Goal

Validate that a Node-based TypeScript extractor can emit JSON in the exact shape the Go indexer emits, and that Dagger is a suitable orchestrator for running it from Go. Capture failures + workarounds so the implementer doesn't re-derive them.

## Step 1: Ticket scaffold + toolchain probe

Created the GCB-002 ticket, seeded vocabulary (`typescript`, `dagger`, `node-tooling`, `multi-language`), added the design-doc and diary scaffolds, and confirmed the local toolchain has what we need.

### Prompt Context

**User prompt (verbatim):**

```
Create a new docmgr ticket to add typescript support using node tsc, run from go using dagger. See the existing skill about building a web bundle using dagger as a starting point.

Create an implementation and design guide and store it in the ticket. You can run tests in the scripts/ folder of the ticket to verify assumptios if needed. keep a diary as you work.
```

**Assistant interpretation:** Design (and validate via a scripts/ prototype) how to add TypeScript language support to the existing GCB-001 codebase-browser: use the TypeScript Compiler API in Node, invoke it from Go via a Dagger container (matching the go-web-dagger-pnpm-build skill pattern), and produce an Index JSON that plugs into the existing server/frontend untouched.

**Inferred user intent:** Confirm TS support is feasible without breaking the "single Go binary" shipping story, and produce a design sharp enough for an engineer to start on Monday.

### What I did

1. Read `go-web-dagger-pnpm-build/SKILL.md` and peeked at Smailnail's reference `cmd/build-web/main.go` for the canonical CacheVolume + corepack + `pnpm install --frozen-lockfile` pattern.
2. `docmgr vocab add` for four new topics (typescript, dagger, node-tooling, multi-language).
3. `docmgr ticket create-ticket GCB-002`, then `docmgr doc add` for design-doc + diary.
4. Probed the toolchain: `node v22.22.1`, `pnpm 10.15.1`, `dagger v0.20.0`, `typescript v5.9.3` via `npx -p typescript@5 tsc --version`.

### Why

Before writing a design I wanted evidence the approach works end-to-end, not just in theory. The skill pattern + Smailnail reference gave me the Dagger half; a `scripts/` prototype validates the Node/TS half.

### What worked

All of the above — single-shot. The Smailnail skeleton in particular mapped cleanly onto the TS extraction case: the only differences are the image workdir (`/src/tools/ts-indexer` instead of `/src/ui`), the final exec (`node bin/ts-indexer.js` instead of `pnpm run build`), and the exported file path.

### What didn't work

Nothing yet; this is a probe step.

### What I learned

Dagger v0.20.0 is available on this host. The `typescript` npm package is already cached in `ui/node_modules/` (our existing frontend dev dep), which I reused later for the prototype when the sandbox pnpm store refused new downloads.

### What was tricky to build

N/A (probe step, no code).

### What warrants a second pair of eyes

N/A.

### What should be done in the future

Nothing from this step alone; Steps 2+3 produce the actionable follow-ups.

### Code review instructions

1. `docmgr task list --ticket GCB-002` should show no tasks yet (we defer task ingest until the design is drafted).
2. `ttmp/.../GCB-002--.../` should have `index.md`, `tasks.md`, `changelog.md`, `design-doc/01-...md`, `reference/01-...md` — confirmed.

### Technical details

Ticket path: `ttmp/2026/04/20/GCB-002--typescript-support-via-tsc-in-node-orchestrated-from-go-with-dagger/`. Design doc: `design-doc/01-typescript-extractor-design-and-implementation-guide.md`. Diary: this file.

## Step 2: Validating prototype in `scripts/`

Wrote a self-contained TypeScript extractor at `scripts/extract.ts`, a fixture TS module at `scripts/fixture-ts/`, and ran the extractor against the fixture. Output matched the expected Go-schema shape (packages/files/symbols/refs with byte offsets + SHA256), which is the empirical proof the design rests on.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** (see Step 1)

**Inferred user intent:** (see Step 1)

### What I did

1. Created `scripts/fixture-ts/src/greeter.ts` (class + interface + type alias + const + function) and `scripts/fixture-ts/src/main.ts` (importer).
2. Wrote `scripts/extract.ts` using `ts.createProgram` + `ts.forEachChild` to emit records: 1 Package per directory, 1 File per source file (with SHA256 + line count), 1 Symbol per top-level declaration (class expanded into its methods).
3. Tried to install `typescript` + `tsx` in `scripts/` via `pnpm install`. **EROFS on the pnpm store** — the sandbox's `~/.local/share/pnpm/store` is read-only, so new tarballs couldn't be cached. This is the same constraint I hit in GCB-001 Phase 4 (Storybook addon-themes).
4. Worked around by symlinking `scripts/node_modules` to the main project's `ui/node_modules` (which already has `typescript` cached). Compiled `extract.ts` → `.build/extract.js` with `ui/node_modules/typescript/bin/tsc` and ran the result with plain `node`.
5. Verified output matches expectations:

```json
{
  "version": "1", "module": "fixture-ts", "language": "ts",
  "packages": [{ "id": "pkg:fixture-ts/src", "fileIds": [2], "symbolIds": [7] }],
  "files": [{ "sha256": "9a27...", ... }, { "sha256": "4e72...", ... }],
  "symbols": [
    {"kind":"class",  "name":"Greeter",    "id":"sym:fixture-ts/src.class.Greeter"},
    {"kind":"method", "name":"hello",      "id":"sym:fixture-ts/src.method.Greeter.hello"},
    {"kind":"const",  "name":"MaxRetries", "id":"sym:fixture-ts/src.const.MaxRetries"},
    {"kind":"func",   "name":"greet",      "id":"sym:fixture-ts/src.func.greet"},
    {"kind":"iface",  "name":"Greetable",  "id":"sym:fixture-ts/src.iface.Greetable"},
    {"kind":"alias",  "name":"Prefix",     "id":"sym:fixture-ts/src.alias.Prefix"},
    {"kind":"const",  "name":"g",          "id":"sym:fixture-ts/src.const.g"}
  ]
}
```

Byte offsets and line numbers read back cleanly via `ts.Node.getStart(sf)` / `getEnd()` — matches what `/api/snippet` expects.

### Why

Before committing to the design doc I wanted to empirically verify:

1. The Compiler API exposes byte offsets on nodes (yes, via `getStart()` / `getEnd()`).
2. JSDoc is accessible (yes, via `(node as any).jsDoc`; undocumented-but-stable).
3. Method extraction from `ClassDeclaration.members` gives us the same receiver-qualified method IDs the Go indexer uses.
4. `tsconfig.json` parsing via `ts.parseJsonConfigFileContent` works without a full `tsc` invocation.

All four confirmed. The design now rests on working code, not speculation.

### What worked

1. Compiler API is straightforward. ~200 lines of TS covers function/class/method/const/var/interface/alias extraction plus deterministic sort.
2. `ts.ModifierFlags.Export` via `ts.getCombinedModifierFlags(decl)` gives a reliable exported/private flag (mirroring Go's `ast.IsExported`).
3. `node.getText(sf)` → slice up to first `{` is a good-enough phase-1 signature renderer. Preserves generics, type unions, conditional types verbatim.
4. Reusing the host's cached `typescript` via symlink sidestepped the pnpm EROFS and let me compile + run the prototype without network access.

### What didn't work

1. **`pnpm install` in `scripts/` failed with ERR_PNPM_EROFS.** Error verbatim:

   ```
   ERR_PNPM_EROFS  Failed to add tarball from "https://registry.npmjs.org/get-tsconfig/-/get-tsconfig-4.14.0.tgz"
     to store: EROFS: read-only file system, open '/home/manuel/.local/share/pnpm/store/v10/files/f7/25181f1820b9bc37bd1dc279474a11d5b85227dc0195a97a806c1cc4b5fe756d8f24595c2ac66cc7446eb40a009d9a2a8efb0294614d2048b51aa5ab75a93ax13036213'
   ```

   Not an issue in the real implementation — Dagger's own cache volume is writable; this is a sandbox-only constraint. Documented here so future-me doesn't chase it again.

2. **`npx -p typescript@5 -p tsx@4 tsx extract.ts`** failed with `Cannot find module 'typescript'` — tsx's sandboxed `npx` context didn't expose the ephemeral typescript install to the script's module resolution. Symlink-to-ui-node_modules sidestepped it.

3. **Type error on `ts.getCombinedModifierFlags(VariableStatement)`** — the API expects a `Declaration`, not a `Statement`. Fixed by passing `node.declarationList.declarations[0] as ts.Declaration` instead. Semantically identical; the modifier flags propagate from the statement to all its declarators so using the first is fine.

### What I learned

1. `ts.SourceFile.getLineAndCharacterOfPosition(pos)` returns 0-based line/col. I add +1 to match the Go indexer's 1-based convention. Mixing these would silently offset every error message.
2. `ts.findConfigFile + ts.parseJsonConfigFileContent` is the correct two-step for loading a tsconfig + resolving `extends` + expanding `include`/`exclude` globs. Using `JSON.parse` alone would miss all of that.
3. `ts.createProgram` with `{ rootNames: parsed.fileNames, options: parsed.options }` is the minimal boilerplate. All downstream walks go through `program.getSourceFiles()` filtered by `sf.isDeclarationFile`.
4. Method IDs should embed the class name as receiver to match GCB-001's method-ID scheme — `sym:<pkg>.method.<Recv>.<name>`. Without this, TS methods would collide across classes that share a method name.

### What was tricky to build

**Mapping "package" onto TypeScript.** TS doesn't have Go's import-path-as-identity notion; modules are files. The design (§5 of the design doc) picks "directory under the module root" as the package granularity, emitting `pkg:<module>/<rel-dir>`. This makes the tree-nav and search UIs work the same way they do for Go, at the cost of treating nested subdirectories as separate packages even when TS considers them just part of the same module. Phase 1 accepts this; if it chafes, a flatter grouping (per npm-package-boundary) is a tsconfig.json-aware follow-up.

**Signature text vs. fully-typed signature.** `ts.TypeChecker.typeToString()` gives richer, fully-resolved signatures (including inferred return types), but it renders synthetically — not byte-identical to source. Phase 1 uses raw source bytes up to the first `{` because snippet fidelity > inferred-type richness; the `/api/snippet` endpoint's byte-slice already gives us the exact declaration.

### What warrants a second pair of eyes

1. **Package identity collisions**. `pkg:fixture-ts/src` for `scripts/fixture-ts/` makes the package ID depend on the module name chosen at extract time. If the same TS project is extracted under two different `module` values, the same package gets two IDs. Keeping the `--module-root` flag stable across runs is a contract we should document.
2. **`(node as any).jsDoc`** is undocumented. If TypeScript ever refactors its internal JSDoc model, the extractor breaks silently (returns no docs). A defensive change: prefer `ts.getJSDocTags(node)` + `ts.getJSDocCommentsAndTags(node)` when available, fall back to the cast only when those return nothing.
3. **Bypassing `pnpm install` by symlinking `node_modules`** worked for the prototype but is sandbox-specific. The real implementation uses a Dagger container where pnpm writes freely to the mounted cache, so this path is not production-reachable. Flagged so no one copies the symlink trick into `tools/ts-indexer/`.

### What should be done in the future

1. Phase 1 implementation: promote `scripts/extract.ts` to `tools/ts-indexer/src/extract.ts` with the CLI split described in design §6.1.
2. Phase 3: write `cmd/build-ts-index/main.go` from the Smailnail skeleton + design §7.2.
3. Phase 4: `--lang auto` flag on `codebase-browser index build`.
4. Write a vitest fixture test inside `tools/ts-indexer/` that reproduces the `scripts/fixture-ts/` scenario as an ongoing regression.
5. Revisit JSDoc access (use the public API) if TypeScript major-versions roll over.

### Code review instructions

1. Open `scripts/fixture-ts/` and `scripts/extract.ts` side-by-side; verify the emitted records match each declaration visually.
2. Run the reproduction:

   ```bash
   cd ttmp/2026/04/20/GCB-002--.../scripts
   ln -sfn /path/to/2026-04-19--go-codebase-browser/ui/node_modules node_modules
   UI_TSC=../../../../../../ui/node_modules/typescript/bin/tsc
   node $UI_TSC --target es2022 --module node16 --moduleResolution node16 --skipLibCheck --outDir .build --esModuleInterop extract.ts
   node .build/extract.js fixture-ts | jq '.packages, .files[0], .symbols | length'
   ```

3. Read `design-doc/01-...md` §3.3 (which embeds the same output) + §12.1 (risks) — they are the bridge from prototype to plan.

### Technical details

```
scripts/
├── extract.ts              # prototype extractor (~200 lines TS)
├── package.json            # (unused — pnpm install EROFS'd)
├── pnpm-lock.yaml          # (broken due to partial install; gitignore)
├── node_modules -> ../../../../../../../ui/node_modules  # sandbox workaround
├── .build/extract.js       # tsc output
└── fixture-ts/
    ├── src/
    │   ├── greeter.ts
    │   └── main.ts
    └── tsconfig.json
```

Prototype output stats: `packages=1, files=2, symbols=7, refs=0` (phase-1 prototype doesn't emit refs yet).

## Step 3: Design doc drafted

Wrote the full design doc at `design-doc/01-typescript-extractor-design-and-implementation-guide.md`. 13 sections, optimised for someone who hasn't read GCB-001 recently — §3.3 embeds the validating prototype output so the design's "it works" claim is inline with the evidence.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** (see Step 1)

**Inferred user intent:** (see Step 1)

### What I did

1. Wrote all 13 sections top-to-bottom: executive summary, problem/scope, current-state (with prototype evidence embedded), gap analysis, topology + repo layout, extractor details, Dagger orchestration, Go-side pluggability refactor, frontend changes, phased plan, testing, risks/alternatives, references.
2. Cross-referenced GCB-001's design doc where relevant (especially §5 for the topology diagram style).

### Why

Keeping the design within a single document (no loose planning notes) makes onboarding one-click and minimises drift between the design and what gets implemented.

### What worked

The Smailnail reference gave us a ready-made `cmd/build-ts-index/main.go` skeleton (just swap dir names + final exec).

### What didn't work

N/A — this step is pure writing.

### What I learned

GCB-001's `Index` schema was even more language-agnostic than I expected. The only field I had to add is `Language` (optional, defaults to "go"). Everything else — `Range`, `Symbol`, `File`, `Package`, `Ref`, stable ID scheme — carries over verbatim.

### What was tricky to build

**Reconciling "package" semantics** across two languages is the subtle piece. Documented in §5.1 and §3.3. Settled on "directory" for TS, which maps cleanly to the tree-nav widget.

### What warrants a second pair of eyes

1. Design §5.4 (`Language` field) — deliberately additive and optional so old clients keep working. If we ever decide Go is a second-class citizen we can invert the default.
2. Design §8.3 merge rules — collision detection on IDs is declared as "error, not silent drop". Reviewers should argue one way or the other before implementation.
3. Design §12.3 question 5 — should `tools/ts-indexer/bin/ts-indexer.js` be committed? Committing means `go build -tags embed` works without Node installed; not committing keeps the repo clean. Recommendation is "don't commit, Dagger rebuilds"; open to overruling.

### What should be done in the future

1. Implement phases 1–5 (1 week total per the estimate in §10).
2. Phase 6 xref and Phase 7 JSX polish as follow-ups.
3. Revisit the tree-sitter alternative (design §12.2) if Dagger ever becomes a blocker.

### Code review instructions

1. Read design §1 (executive summary) and §10 (phased plan) first.
2. Read §3.3 (current-state prototype evidence) + §6.1 (extractor entrypoint) — these are the load-bearing sections for implementation.
3. Skim §12 risks + alternatives. The tree-sitter alternative was considered and rejected; the reasoning is written down.

### Technical details

Design doc word-weight by section (approximate):

```
§1  Executive summary        ~180 words
§2  Problem / scope          ~220 words
§3  Current-state            ~320 words   (includes prototype output)
§4  Gap analysis             ~160 words
§5  Proposed architecture    ~520 words   (topology + layout + flow + kind vocab)
§6  TS extractor details     ~340 words
§7  Dagger orchestration     ~500 words   (incl. main.go skeleton)
§8  Go-side refactor         ~330 words
§9  Frontend changes         ~210 words
§10 Phased plan              ~320 words   (6 phases)
§11 Testing strategy         ~130 words
§12 Risks/alternatives/Qs    ~300 words
§13 References               ~70 words
```
