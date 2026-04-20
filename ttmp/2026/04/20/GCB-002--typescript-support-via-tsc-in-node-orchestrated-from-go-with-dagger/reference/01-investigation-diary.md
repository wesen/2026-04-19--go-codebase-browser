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
    - Path: cmd/build-ts-index/main.go
      Note: Dagger + local fallback orchestrator (Step 5)
    - Path: internal/indexer/multi.go
      Note: Merge + Extractor interface (Step 5)
    - Path: tools/ts-indexer/src/extract.ts
      Note: 'Two-pass extractor: symbols + refs via TypeChecker (Steps 5 and 6)'
    - Path: tools/ts-indexer/test/extract.test.ts
      Note: |-
        Vitest assertions for symbols and xref (Steps 5
        Vitest assertions across symbol + xref phases
    - Path: ttmp/2026/04/20/GCB-002--typescript-support-via-tsc-in-node-orchestrated-from-go-with-dagger/design-doc/01-typescript-extractor-design-and-implementation-guide.md
      Note: The primary deliverable this diary narrates
    - Path: ttmp/2026/04/20/GCB-002--typescript-support-via-tsc-in-node-orchestrated-from-go-with-dagger/scripts/extract.ts
      Note: Prototype extractor validated in Step 2
    - Path: ui/src/packages/ui/src/highlight/ts.ts
      Note: TS+JSX tokenizer (Step 5 + JSX polish Step 7)
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

1. **`pnpm install` in `scripts/` initially failed with ERR_PNPM_EROFS.** Error verbatim (historical — resolved in Step 4):

   ```
   ERR_PNPM_EROFS  Failed to add tarball from "https://registry.npmjs.org/get-tsconfig/-/get-tsconfig-4.14.0.tgz"
     to store: EROFS: read-only file system, open '/home/manuel/.local/share/pnpm/store/v10/files/f7/...'
   ```

   The pnpm store (`~/.local/share/pnpm/store/v10/`) was mounted read-only during this session, so any tarball not already cached couldn't land. See Step 4 for the cleanup once rw access was restored.

2. **`npx -p typescript@5 -p tsx@4 tsx extract.ts`** failed with `Cannot find module 'typescript'` — tsx's sandboxed `npx` context didn't expose the ephemeral typescript install to the script's module resolution. Worked around with a symlink to `ui/node_modules`; the real fix (`pnpm install` in `scripts/`) lands in Step 4.

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

## Step 4: Clean-up — pnpm store writable, workaround removed

The pnpm store's read-only state was a session artifact; it's writable now. Redid the scripts setup the normal way — plain `pnpm install` works, tarballs land in `~/.local/share/pnpm/store/v10/` without drama, and `pnpm run extract` runs the extractor via tsx without any symlink trick. The prototype's observable behaviour is identical (1 package / 2 files / 7 symbols).

### Prompt Context

**User prompt (verbatim):**

```
You should have rw access now, does that allow you to simplify the setup / remove the workarounds?
```

**Assistant interpretation:** Confirm the sandbox's pnpm store is now writable, remove the symlink + tsc-compile workaround from the prototype, and commit the pnpm-lock.yaml so the scripts dir reproduces cleanly.

**Inferred user intent:** Leave the prototype in a shape that a reviewer can `pnpm install && pnpm run extract fixture-ts` on, not a shape that requires knowing about a sandbox quirk.

### What I did

1. Probed: `touch ~/.local/share/pnpm/store/v10/.probe` succeeded — writable.
2. Removed the `scripts/node_modules` symlink and the `scripts/.build/` tsc output.
3. Ran `pnpm install` in `scripts/` cleanly (`Done in 1.5s`, no EROFS).
4. Ran `./node_modules/.bin/tsx extract.ts fixture-ts | jq '{module, packages, files, symbols, kinds}'` — output matches Step 2 byte-for-byte (modulo `generatedAt`).
5. Added `packageManager: "pnpm@10.13.1"` to `scripts/package.json` per the `go-web-dagger-pnpm-build` skill's reproducibility convention.
6. Replaced the `scripts/.gitignore`'s EROFS-era contents (`.build/`, `pnpm-lock.yaml`) with the normal `node_modules/` entry — we now commit the lockfile.
7. Added `scripts/README.md` with the clean run instructions.
8. Marked Step 2's EROFS notes as historical and pointed at Step 4.

### Why

The diary is append-only (we don't rewrite history), but the README + prototype layout should reflect the actual supported invocation. Keeping `scripts/.gitignore` with the old entries and the symlink-directory-tree in the diary would have mislead anyone opening the ticket from scratch.

### What worked

1. `pnpm install` in `scripts/` — 1.5 s.
2. `./node_modules/.bin/tsx extract.ts fixture-ts` — produces identical JSON to the Step 2 prototype.
3. Committing `pnpm-lock.yaml` gives deterministic installs for reviewers.

### What didn't work

Nothing. The cleanup was straightforward.

### What I learned

The skill's `packageManager` field convention matters not just for Vite builds — it belongs on any `package.json` in the repo so `corepack` / pnpm picks a consistent version. Added it to `scripts/package.json` as a small additional improvement over what the Step 2 setup had.

### What was tricky to build

N/A — cleanup step.

### What warrants a second pair of eyes

Whether `scripts/pnpm-lock.yaml` should be committed or not. Convention says yes (lockfile determinism); alternative is to rely on the `packageManager` pin + `package.json` semver ranges. Going with "commit" matches what we did for `ui/pnpm-lock.yaml` in GCB-001.

### What should be done in the future

Nothing specific to this cleanup. The Phase 2 `tools/ts-indexer/` package will inherit the clean layout (proper `pnpm install`, committed `pnpm-lock.yaml`, `packageManager` pinned).

### Code review instructions

1. `cd scripts/` then `pnpm install && pnpm run extract fixture-ts` — should print the fixture index JSON.
2. `git log --diff-filter=D -- '*scripts/.build*'` — should show nothing (artifact was never committed).
3. `grep -n 'symlink\|EROFS' reference/01-investigation-diary.md` — Step 2 retains historical mentions, Step 4 closes them.

### Technical details

Final `scripts/` layout:

```
scripts/
├── .gitignore              # only: node_modules/
├── README.md               # run instructions
├── package.json            # packageManager pin + tsx + typescript
├── pnpm-lock.yaml          # committed
├── extract.ts              # prototype extractor (~200 lines)
├── node_modules/           # gitignored
└── fixture-ts/
    ├── src/
    │   ├── greeter.ts
    │   └── main.ts
    └── tsconfig.json
```

Verification:

```bash
$ cd scripts
$ pnpm install
Done in 1.5s using pnpm v10.13.1
$ ./node_modules/.bin/tsx extract.ts fixture-ts | jq '{packages:.packages|length, files:.files|length, symbols:.symbols|length}'
{ "packages": 1, "files": 2, "symbols": 7 }
```

## Step 5: Phases 1–5 shipped — extractor, Dagger program, merge, frontend

With the design doc and prototype settled, rolled through Phases 1–5 in
five focused commits (`569dc4c` → `7bf0370`). Phase 1 added the Go
`Extractor` interface, `Language` field on `Package`/`File`/`Symbol`, and
`Merge()` with duplicate-ID detection. Phase 2 lifted the prototype into
`tools/ts-indexer/` as a proper pnpm package (`typescript` + `tsx` +
`vitest`, `packageManager: "pnpm@10.13.1"`). Phase 3 added
`cmd/build-ts-index` with a Dagger container (`node:22-bookworm`, pnpm
`CacheVolume`) and a `BUILD_TS_LOCAL=1` fallback. Phase 4 wired
`codebase-browser index build --lang go|ts|auto` to shell out to the
Dagger program and merge the two JSON parts. Phase 5 added
`highlight/ts.ts`, a language dispatch in `highlight/index.ts`, a `class`
kind color, `--path-prefix` (so `File.path` stays repo-rooted while symbol
IDs stay prefix-free), and a TypeScript story.

### Prompt Context

**User prompt (verbatim):** "do GCB-002"

**Assistant interpretation:** Execute the phased plan in
`design-doc/01-typescript-extractor-design-and-implementation-guide.md`
§10: land Phases 1 through 5 (the "shipping" phases), keep 6 and 7 as
follow-ups.

**Inferred user intent:** End the session with a working
`index build --lang auto` that emits a merged JSON the server already
loads, and with TS syntax highlighting rendering in the UI — enough to
browse the TS side of this repo alongside the Go side.

**Commits (code):**
- `569dc4c` — "GCB-002 Phase 1: Extractor interface + Language field + Merge"
- `60a9adc` — "GCB-002 Phase 2: tools/ts-indexer/ package"
- `ce69ff3` — "GCB-002 Phase 3: cmd/build-ts-index Dagger program"
- `ca47795` — "GCB-002 Phase 4: --lang go|ts|auto on index build"
- `7bf0370` — "GCB-002 Phase 5: frontend TS highlighter + class kind + path-prefix fix"

### What I did

1. Phase 1: added `Language` to `internal/indexer/types.go`, introduced
   `Extractor` interface + `Merge()` in `internal/indexer/multi.go`,
   stamped `"go"` on Go-extracted records, wrote four merge tests
   covering duplicate-ID detection, nil parts, and language stamping.
2. Phase 2: created `tools/ts-indexer/` with `package.json`,
   `tsconfig.json` (NodeNext, strict), `src/types.ts`, `src/ids.ts`,
   `src/extract.ts` (the prototype, cleaned), `src/cli.ts`, plus a
   vitest fixture and 6 initial tests.
3. Phase 3: `cmd/build-ts-index/main.go` connects via `dagger.Connect`,
   mounts `tools/ts-indexer` + the module root, `WithMountedCache` on
   the pnpm store, `corepack enable && prepare pnpm@10.13.1`, builds the
   extractor in the container, runs the CLI, exports
   `/out/index-ts.json` to the host. `BUILD_TS_LOCAL=1` branch shells
   to local `pnpm + node` for machines without Docker.
4. Phase 4: `--lang` glazed flag (`go`|`ts`|`auto`), `--ts-module-root`,
   `--ts-index-path`; `auto` shells the Dagger program, reads both
   parts, runs `indexer.Merge`, writes the combined file plus a glazed
   table summarising go/ts/total counts.
5. Phase 5: `highlight/ts.ts` with ~60 keywords, builtins, template
   literal handling with `${…}` depth tracking; dispatcher in
   `highlight/index.ts`; `--cb-color-kind-class` + `data-role='class'`
   rule in `base.css` + `dark.css`; `language` prop threaded from
   `SymbolCard` → `Code` / `LinkedCode` / `SourceView`; file paths now
   flow from `--path-prefix` (so `ui/src/...` works against the Go
   server's source FS rooted at repo root); Storybook stories for
   `class`/`method`/`iface`/`alias` kinds.

### Why

The phased split was deliberate: each phase leaves HEAD building, each
commit stops at a natural review boundary (schema → tool → container →
CLI flag → UI). The last-mile `--path-prefix` change was a late
correction after `/api/snippet` came back empty for TS symbols because
`File.path` was `src/api/...` (relative to `ui/`) rather than
`ui/src/api/...` (resolvable against the Go server's source FS).

### What worked

1. The prototype → `tools/ts-indexer/` lift was mechanical: the API
   shape in `src/types.ts` is 1:1 with Go's `types.go`, so no schema
   reconciliation was needed.
2. Indexer `Merge` duplicate detection caught a real TS bug on first
   merge: Storybook's `const meta` collided across `*.stories.tsx`
   files. Fix: file-scoped symbol IDs (`sym:<mod>/<relfile-stem>.kind.name`)
   in the TS extractor.
3. `BUILD_TS_LOCAL=1` fallback is the path actually used during dev —
   Dagger is code-complete but end-to-end is gated on Docker.
4. End-to-end: `go run ./cmd/codebase-browser index build --lang auto`
   prints `134 go + 163 ts = 297 symbols` and the server resolves
   `sym:ui/src/packages/ui/src/TreeNav.iface.TreeNavProps` via
   `/api/snippet`.

### What didn't work

1. Initial TS IDs collided on `const meta` — see above; fixed by making
   the symbol-scope string file-scoped (`moduleName + '/' + relPath -
   extension`).
2. `/api/snippet` returned empty for TS symbols before the
   `--path-prefix` plumbing — `File.path` didn't match the Go server's
   source FS rooting. Fixed by adding the flag and threading it from
   `cmd/build-ts-index`.
3. A stretch of edits to `extract.ts` during Phase 5 silently didn't
   land (two `Edit` attempts failed with "file has not been read yet"
   without surfacing); caught by `grep -n pathPrefix extract.ts`
   returning empty, re-`Read` + re-apply fixed it.

### What I learned

1. TypeScript doesn't have Go's package-is-a-directory rule; two files
   in the same directory can both declare `const meta` as top-level
   exports (each file is its own module). Symbol IDs therefore have to
   be file-scoped even when the package grouping is directory-scoped.
2. Splitting `File.path` (prefix-rooted for the server's source FS)
   from symbol-ID scope (prefix-free for stability across repo moves)
   is the right factoring — don't conflate them.

### What was tricky to build

`--path-prefix`: the constraint is two conflicting needs on the same
string. The server's source FS is rooted at repo-root and reads files
by `File.path`, so TS `File.path` must include `ui/`. But symbol IDs
are used in doc snippets and cross-references; prepending `ui/` to the
ID scope means moving the TS project from `ui/` to `web/` invalidates
every `codebase-snippet` directive. Resolution: `pathPrefix` flows
through to `File.path` only; `symScope` stays based on the un-prefixed
`relNative`.

### What warrants a second pair of eyes

1. The `Merge` duplicate-ID detection short-circuits on first collision;
   someone reviewing should confirm that's the desired behaviour
   (alternative: keep extracting, report a list).
2. Dagger path is code-complete but has not been exercised end-to-end in
   this session — the local-pnpm fallback is what actually emits
   `index-ts.json` today.

### What should be done in the future

1. Dagger-path smoke test (requires a live engine).
2. TS xref (follow-up Phase 6).
3. JSX tokenizer polish (follow-up Phase 7).

### Code review instructions

1. `BUILD_TS_LOCAL=1 go run ./cmd/codebase-browser index build --lang auto`
   — should produce a glazed table with `go`, `ts`, and merged totals.
2. `go test ./internal/indexer` — merge + dup-ID tests must pass.
3. `cd tools/ts-indexer && pnpm test` — 6 vitest cases must pass.
4. `pnpm run build` in `ui/` — no TypeScript errors.
5. Spot-check: `curl /api/snippet?sym=sym:ui/src/packages/ui/src/TreeNav.iface.TreeNavProps`
   should return the TS interface body, not empty.

### Technical details

Final counts from the merged index (both Go and TS emitted by the same
`index build --lang auto` invocation):

```
packages: 27 (16 go + 11 ts)
files:    71
symbols: 297 (134 go + 163 ts)
```

Symbol-ID scheme across languages:

| Lang | Scope format                            | Example                                                 |
|-----:|:----------------------------------------|:--------------------------------------------------------|
| Go   | `<importPath>.<kind>.<name>`            | `sym:github.com/wesen/codebase-browser/internal/indexer.func.Merge` |
| TS   | `<module>/<relfile-no-ext>.<kind>.<name>` | `sym:ui/src/packages/ui/src/TreeNav.iface.TreeNavProps` |

## Step 6: Phase 6 — TS xref via TypeChecker

Made the TS extractor two-pass. Pass 1 is unchanged except that every
emitted declaration now registers into a `Map<ts.Declaration, string>`.
Pass 2 walks function bodies and method bodies only (same reach as Go's
`addRefsForFile`), calling `checker.getSymbolAtLocation` on every
identifier and, when the symbol is an import alias (named import),
following it through `checker.getAliasedSymbol` so refs target the real
exported declaration rather than the local binding. Ref kind mirrors
Go: `call` for Function/Method, `uses-type` for Class/Interface/Alias/Enum,
`reads` for Variable/BlockScopedVariable. 99 refs emitted on the ui/
tree; `/api/xref/{id}` and `/api/snippet-refs` both show them.

### Prompt Context

**User prompt (verbatim):**

```
Continue working on GCB-002 (read the docmgr skill), keeping a diary (read the skill), committing at appropriate intervals a nd checking off the tasks.
```

**Assistant interpretation:** Pick up the two remaining follow-up
phases (Phase 6 TS xref, Phase 7 JSX/`.tsx` polish) from the design
doc §10, commit each as its own logical unit, update the diary and the
docmgr task list.

**Inferred user intent:** Close out GCB-002. The "shipping" phases
(1–5) were already in; the follow-ups gate things like linkified TS
identifiers in the UI that depend on refs existing in the index.

**Commit (code):** `5eb25d0` — "GCB-002 Phase 6: TS xref via TypeChecker"

### What I did

1. Added `declToSymbolID: Map<ts.Declaration, string>` threaded through
   `collectTopLevel`; each of the six symbol emitters (`Function`,
   `Class` + methods, `Interface`, `TypeAlias`, `VariableStatement`
   declarations) registers its node.
   - `VariableStatement` registers each `VariableDeclaration` (not the
     containing statement), because the checker returns the
     `VariableDeclaration` as `sym.declarations[0]` for identifier uses.
2. Split the top-level source-file loop into two passes: the existing
   symbol pass, plus a new `for (const sf of projectFiles)` that calls
   `collectRefs(node, sf, fid, declToSymbolID, checker, idx)`.
3. `collectRefs` matches the Go behaviour's shallowness — only
   `FunctionDeclaration` bodies and `ClassDeclaration` method bodies.
4. `emitBodyRefs` walks with `ts.forEachChild`, skipping identifier
   occurrences that are in binding positions (`Parameter.name`,
   `VariableDeclaration.name`, `BindingElement.name`) so a function
   doesn't emit refs from its body to its own parameters.
5. For every other identifier: `checker.getSymbolAtLocation` → if the
   result has `SymbolFlags.Alias`, try `getAliasedSymbol` (wrapped in
   try/catch — it throws on unresolvable aliases) → look up the first
   declaration in `declToSymbolID`.
6. `refKindFor` maps `SymbolFlags.Function|Method` → `call`,
   `Class|Interface|TypeAlias|Enum` → `uses-type`,
   `Variable|BlockScopedVariable` → `reads`, else `use`.
7. Fixture: added `tools/ts-indexer/test/fixture/src/main.ts` with a
   `run()` function that calls `greet`, constructs `new Greeter`,
   calls `g.hello`, and reads `MaxRetries`.
8. Tests: updated the `7 symbols` expectation (6 → 7), added assertions
   that the four expected targets appear as `from=run`, and a third
   test that checks the ref `kind` classification.
9. Determinism test extended to also compare ref order across runs.

### Why

The server already has a generic `/api/xref/{id}` endpoint (loops over
`index.Refs` regardless of language) and a `/api/snippet-refs`
endpoint (for the linkified-identifier rendering). Both worked for Go
out of the box; the only missing piece was that TS files had no refs.
Adding the refs pass was the smallest possible change that lights up
both endpoints for TS symbols.

### What worked

1. The first full run on the `ui/` tree emitted 99 refs, split
   `reads: 51 / call: 39 / uses-type: 9` — reasonable for a codebase
   dominated by RTK-Query hooks + JSX composition.
2. JSX component usage (`<Header />` etc.) is parsed as a call
   internally, so refs to components show up as `call`-kind out of the
   box — no JSX-specific handling needed.
3. `/api/xref/sym:ui/src/app/App.func.App` returned 10 `uses` entries
   on first try (Header, SearchPanel, DocList, PackageList, and five
   route pages + `widgetRootAttrs`).

### What didn't work

1. First test run emitted only 1 ref out of the 4 expected. Diagnostic:
   only `g.hello` resolved; `greet`, `Greeter`, `MaxRetries` (all
   imported names) returned `undefined`. Cause: the checker returns
   an *alias symbol* for a named import. `sym.declarations[0]` is the
   `ImportSpecifier`, which isn't in `declToSymbolID`.
2. Fixed by detecting `sym.flags & ts.SymbolFlags.Alias` and calling
   `checker.getAliasedSymbol(sym)` to follow the alias to the exported
   declaration. Wrapped in try/catch because
   `getAliasedSymbol` throws on unresolvable aliases (e.g. circular
   or module-not-found).

### What I learned

1. `checker.getSymbolAtLocation` returns the *alias* for imported
   names, not the exported declaration. `getAliasedSymbol` is the
   follow-through; there's no automatic flag to skip this step.
2. Multi-declaration symbols exist in TS (e.g. an interface + a
   const with the same name). The fixture doesn't exercise that
   path, but `resolveSymbolID` walks `sym.declarations` in order and
   returns the first match, so it's consistent with "whatever the
   extractor emitted first for this name."
3. Binding-position skip matters more than I initially thought: a
   `for (const x of xs)` body emits an ident `x` that's bound in the
   VariableDeclaration; without the skip, `x` would generate a self-ref
   to its own binding, which is noise.

### What was tricky to build

The extractor needs to run the symbol pass to completion *before* the
refs pass starts (not interleaved per-file) because TS lets you
reference a symbol declared later in the same file or in another file.
Resolution: pre-filter `projectFiles` once, run the symbol pass over
the whole list, then run the refs pass over the same list. Cost is a
second traversal per file (~40 files in `ui/`) — negligible vs the
existing `getSourceFiles` filtering cost.

### What warrants a second pair of eyes

1. `refKindFor` uses bitwise AND on `SymbolFlags`. TS flags are
   composed (e.g. a class's constructor has both `Method` and
   `Class` bits), so an ambiguous call ordering matters. Current
   order: Function/Method (`call`) tested before Class/Interface (`uses-type`).
   For a constructor call (`new Foo()`), the identifier `Foo` has flag
   `Class` (not `Method`), so it falls through to `uses-type`. That
   matches what I'd expect from Go (`types.TypeName` → `uses-type`).
2. `getAliasedSymbol`'s throw-on-error behaviour: the try/catch
   defaults to "skip this ref", which is the same outcome as the
   symbol not being in our map. Worth confirming that no real import
   shape (`import * as foo` + re-export chains) produces useful misses.

### What should be done in the future

1. Emit refs from the TS side for JSX attribute values referencing
   exports (e.g. `<Foo style={baseStyle}>` should ref `baseStyle`).
   Spot check suggests these are already picked up via
   `ts.forEachChild` recursion — the JSX attribute walk includes the
   `{baseStyle}` child. Not explicitly tested.
2. Class-field references (`this.prefix` inside a method) don't
   emit refs today because the TS extractor doesn't index class fields
   as symbols. Matches Go's behaviour for struct fields. Could be
   expanded if someone wants field-level xref.

### Code review instructions

1. `cd tools/ts-indexer && pnpm test` — 8 cases pass, including two
   new xref assertions.
2. `BUILD_TS_LOCAL=1 go run ./cmd/build-ts-index` —
   `index-ts.json` contains `refs` with ~99 entries on `ui/`.
3. `go run ./cmd/codebase-browser serve --addr 127.0.0.1:47056` then
   `curl /api/xref/sym:ui/src/app/App.func.App` — should return `uses`
   of length 10.
4. Grep check: `grep -c '"kind":"call"' internal/indexfs/embed/index-ts.json`
   should be in the 30s–40s range.

### Technical details

The decl-to-ID map's key type (`ts.Declaration`) is a node identity
comparison — the same AST node registered in pass 1 must be the node
returned by `sym.declarations[0]` in pass 2. TS's symbol table returns
the same node instances across `getSymbolAtLocation` calls within one
`Program`, so the identity holds. If we ever re-parse (incremental), we
lose this guarantee and have to fall back to `(fileName, pos)` keys.

Ref kind classification:

```ts
function refKindFor(sym: ts.Symbol): string {
  const f = sym.flags;
  if (f & (ts.SymbolFlags.Function | ts.SymbolFlags.Method)) return 'call';
  if (f & (ts.SymbolFlags.Class | ts.SymbolFlags.Interface |
           ts.SymbolFlags.TypeAlias | ts.SymbolFlags.Enum)) return 'uses-type';
  if (f & (ts.SymbolFlags.Variable | ts.SymbolFlags.BlockScopedVariable)) return 'reads';
  return 'use';
}
```

## Step 7: Phase 7 — .tsx + JSX tokenizer polish

Closed out the two explicit follow-ups from design-doc §10. First,
confirmed the extractor already handles `.tsx` + `export default
function` by adding a `card.tsx` fixture and asserting 3 symbols +
a `Card → greet` ref flow through the same two-pass pipeline. Second,
polished `highlight/ts.ts` so capitalized identifiers immediately after
`<` or `</` tokenize as `type` — React components render with the
builtins colour, DOM tags keep the neutral `id` colour. Generic type
args (`Array<number>`) are untouched because the heuristic requires
uppercase + zero-whitespace adjacency, and generic args usually have
whitespace or use existing BUILTINS entries anyway.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Phase 7 from the design doc. Verify the
`.tsx` path and add a small JSX highlighting improvement.

**Inferred user intent:** Finish the "follow-ups" column in §10 so the
ticket can close.

**Commit (code):** `39290c9` — "GCB-002 Phase 7: .tsx support + JSX tokenizer polish"

### What I did

1. Added `tools/ts-indexer/test/fixture/src/card.tsx` with an
   `export default function Card(props: CardProps)` that returns JSX and
   calls `greet(name)`. Also `export function Footer()`.
2. Updated `tools/ts-indexer/test/fixture/tsconfig.json` with
   `jsx: "preserve"` so the compiler doesn't reject the tag syntax.
3. Updated `test/extract.test.ts`: 7 → 10 symbols, new assertions for
   `Card` (default-exported, exported=true), `Footer`, `CardProps`, and
   a `Card → greet` ref inside the JSX body.
4. Added `isJsxComponentRef(out, word)` helper to
   `ui/src/packages/ui/src/highlight/ts.ts`: returns `true` when `word`
   starts with a capital letter and the last non-ws emitted token is
   `<` (optionally preceded by `/`). The identifier emitter calls the
   helper after checking KEYWORDS/BUILTINS, so those remain
   authoritative.
5. Added `ui/src/packages/ui/src/highlight/ts.test.ts` mirroring
   `go.test.ts`: roundtrip assertions + four JSX-specific cases
   (`<Button/>` capitalizes, `<div />` stays id, `</Button>` matches
   close, `a < B` with whitespace does not trigger).
6. Added a `CodeBlockJSX` story to `TypeScript.stories.tsx` with a
   realistic JSX snippet that exercises nested components, DOM tags,
   JSX children, interpolation, and closing tags.

### Why

The Phase 7 line in the design doc is short ("1 day") and mostly
verification — `.tsx` already loaded through `ts.createProgram`, and
the prior JSX tokenizer already handled the characters without
choking. The missing thing was the visual distinction between React
components and DOM tags, which is the single most-used heuristic in
production JSX highlighters.

### What worked

1. `.tsx` "just worked" through the existing extractor: `Card` and
   `Footer` both emit cleanly; the ref from `Card` → `greet` shows up
   on the first run.
2. The narrow heuristic (capital + zero-whitespace adjacency) is
   surprisingly robust. On the real `ui/` tree it correctly tags
   `<Header>`, `<SearchPanel>`, `<TreeNav>` as components while
   leaving `<div>`, `<section>`, `<button>` as id tokens.
3. `a < B` (comparison with whitespace) correctly falls through
   because the last emitted non-ws-separated token is a `ws`, not `<`.

### What didn't work

Nothing major. The heuristic walks back one `/` to cover `</Name`,
which is the only two-token prefix we have to care about. `< Name`
(JSX with space, unusual) would not be recognized — acceptable, since
it's both rare and Prettier removes the space.

### What I learned

1. TS's `ts.createProgram` parses `.tsx` automatically; no `jsx` flag
   is strictly required to get the AST, but emitting types fails
   without one (which we don't care about — we only walk the AST).
   Adding `jsx: preserve` to the fixture tsconfig silences the
   diagnostic.
2. `ts.ModifierFlags.Export` covers both `export` and `export default`
   — the `exported: true` flag on the `Card` symbol was already
   correct without any Phase-7 change.
3. Uppercase-is-component is the JSX spec (not just convention): the
   React runtime *treats* `<foo>` as a DOM element and `<Foo>` as a
   component, so aligning highlighter semantics with runtime semantics
   is more than cosmetic.

### What was tricky to build

Keeping the heuristic from firing on `x < B` (where `B` is e.g. a
capitalized local variable). The fix is "last non-ws emitted token
must be `<` directly" — because `<` as comparison operator always has
whitespace in formatted code, and our tokenizer emits that whitespace
as a `ws` token that breaks the adjacency check.

### What warrants a second pair of eyes

Whether `<Foo.Bar>` should highlight `Bar` too. Current implementation
lexes `Foo`, `.`, `Bar` and only `Foo` gets the `type` tag. Fine for
a phase-1 highlighter; a proper fix would track "am I currently inside
a JSX tag name?" across `.` tokens.

### What should be done in the future

1. `<Foo.Bar>` nested access highlighting (cosmetic).
2. JSX attribute highlighting (distinguish `className=` from other
   identifiers). Deferred.
3. Plain JavaScript (`.js`) extraction — still out of scope per
   design-doc §2.3. Revisit when someone asks.

### Code review instructions

1. `cd tools/ts-indexer && pnpm test` — 10 cases pass.
2. Visit Storybook → `Widgets/TypeScript` → `CodeBlockJSX` (run
   `pnpm storybook` in `ui/`) and confirm `HelloCard`, `Greeter`,
   `Icon`, `Label` render with the type color; `div`, `p`, `button`
   stay neutral.
3. `BUILD_TS_LOCAL=1 go run ./cmd/codebase-browser index build --lang auto`
   — rebuilds cleanly with the expanded fixture.

### Technical details

Key function in `ui/src/packages/ui/src/highlight/ts.ts`:

```ts
function isJsxComponentRef(out: Token[], word: string): boolean {
  if (!/^[A-Z]/.test(word)) return false;
  let k = out.length - 1;
  if (k >= 0 && out[k].type === 'punct' && out[k].text === '/') k--;
  if (k < 0) return false;
  const prev = out[k];
  return prev.type === 'punct' && prev.text === '<';
}
```

Called from the identifier emitter after KEYWORDS/BUILTINS are ruled
out, so those remain authoritative.
