---
Title: Investigation Diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../.claude/skills/glazed-command-authoring/SKILL.md
      Note: Loaded during Step 2 to ensure idiomatic Glazed usage
    - Path: ../../../../../../../../../.claude/skills/go-web-frontend-embed/SKILL.md
      Note: Loaded during Step 2 for embed + SPA topology
    - Path: ../../../../../../../../../.claude/skills/react-modular-themable-storybook/SKILL.md
      Note: Loaded during Step 2 for theming + Storybook contract
    - Path: ttmp/2026/04/20/GCB-001--go-codebase-browser-ast-indexed-embedded-themed/design-doc/01-go-codebase-browser-analysis-and-implementation-guide.md
      Note: The primary deliverable whose decisions this diary narrates
ExternalSources: []
Summary: Chronological diary of the design phase for GCB-001 — covers ticket scaffolding, the decisions that shaped the design doc (build-time indexing, stable symbol IDs, source-embedding, frontend/widget split), and the follow-ups the design leaves open.
LastUpdated: 2026-04-20T00:00:00Z
WhatFor: Onboarding context for whoever implements GCB-001; captures why the design looks the way it does and what we considered along the way.
WhenToUse: Read alongside the design doc; consult when a design decision needs to be revisited or when picking up implementation.
---


# Investigation Diary

## Goal

Record the design-phase reasoning for GCB-001 (Go codebase browser: AST-indexed, embedded, themed) so the implementation phase can continue without re-deriving decisions.

## Step 1: Ticket creation and vocabulary setup

During this step I initialized the docmgr workspace for GCB-001 and seeded vocabulary terms so the ticket creation would not emit warnings. This unlocks the rest of the work: design doc, diary, bookkeeping, and reMarkable upload all expect a valid ticket with a valid topic list.

### Prompt Context

**User prompt (verbatim):**

```
Create a new docmgr ticket and create ana anlysis and implementation guide for the following idea:

- it's a go app (with glazed commands, see skill)
- at build time, use go/ast / go/analysis to create an index of the codebase and store it in a json or so
- build a web server that embeds the source code (of itself, in this case), and uses the index that was computed to create a nice navigationable view of the codebase, and say can embed individual functions or other linguistic contructs that were parsed in the analysis step.

THe UI should be themable react css with rtk-query (see skill), with copious amounts of storybook.

That way, a program can embed a rich reprsentation of its source code and allow the creation of rich documentation which embeds its own source code or its APIs and make sure they always corresond to the right sourcecode.

Create the analysis / implementation guide in the ticket and keep a diary. Upload to remarkable (the design doc) when you're done
```

**Assistant interpretation:** The user wants a full docmgr ticket with a long-form analysis/implementation design doc for a self-embedding Go codebase browser (build-time AST index, embedded source, themable React+RTK-Query UI, extensive Storybook, live-snippet docs) plus a diary and a reMarkable upload at the end.

**Inferred user intent:** Produce an exhaustive design artifact that an engineer could start implementing from directly, while leaving durable context (diary + ticket bookkeeping) and a review-ready PDF on the reMarkable for offline reading.

### What I did

1. `docmgr status --summary-only` to confirm the docs root and that the ticket count was zero.
2. Inspected `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/.ttmp.yaml` and the (empty) repo contents.
3. Added vocabulary topics needed by the ticket: `go-ast`, `codebase-browser`, `embedded-web`, `react-frontend`, `storybook`, `glazed`, `rtk-query`, `documentation-tooling` (all via `docmgr vocab add --category topics`).
4. Created the ticket:
   ```bash
   docmgr ticket create-ticket --ticket GCB-001 \
     --title "Go codebase browser: AST-indexed, embedded, themed" \
     --topics go-ast,codebase-browser,embedded-web,react-frontend,storybook,glazed,rtk-query,documentation-tooling
   ```
5. Added the two docs that anchor the ticket:
   ```bash
   docmgr doc add --ticket GCB-001 --doc-type design-doc --title "Go Codebase Browser — Analysis and Implementation Guide"
   docmgr doc add --ticket GCB-001 --doc-type reference --title "Investigation Diary"
   ```

### Why

The skill flow (`ticket-research-docmgr-remarkable`) is explicit that the ticket must exist first and that design-doc + diary are the two load-bearing artifacts. Adding vocabulary upfront avoids `docmgr doctor` warnings later.

### What worked

1. `docmgr vocab add` is idempotent-ish: each call prints the added entry without complaining about existing terms.
2. `docmgr ticket create-ticket` created the expected tree under `ttmp/2026/04/20/GCB-001--go-codebase-browser-ast-indexed-embedded-themed/` including `index.md`, `tasks.md`, `changelog.md`, and doc subdirectories.
3. The auto-generated scaffold for the design doc has valid frontmatter and a clean section skeleton that matched the writing-style.md guidance closely enough to keep and extend.

### What didn't work

Nothing failed at this step. The only mildly annoying piece was that each `vocab add` prints the full config header, which clutters the terminal; grouping with `&&` is fine but noisy.

### What I learned

1. `docmgr` writes docs into `<ticket>/design-doc/NN-<slug>.md` rather than `<ticket>/design/NN-<slug>.md`, so the directories listed in `index.md` (`design/`, `reference/`, `playbooks/`) and the actual doc locations (`design-doc/`, `reference/`) are slightly different. Not a bug — just worth knowing when writing `docmgr doc relate` commands.

### What was tricky to build

Not applicable in this step (no code).

### What warrants a second pair of eyes

The choice of ticket ID (`GCB-001`) was mine; if there is a pre-existing ticket numbering system I don't know about, this is the place it might collide. The repo is empty, so I defaulted to "Go Codebase Browser" initials.

### What should be done in the future

Add a follow-up vocabulary sweep if later steps introduce new domains (for example: `goldmark` for the markdown parser, `go-packages` for the analysis loader).

### Code review instructions

1. Start with `ttmp/2026/04/20/GCB-001--go-codebase-browser-ast-indexed-embedded-themed/index.md` (frontmatter lists all topics).
2. Verify vocab by `docmgr vocab list | grep topics`.
3. Verify the ticket by `docmgr list tickets`.

### Technical details

Ticket path: `ttmp/2026/04/20/GCB-001--go-codebase-browser-ast-indexed-embedded-themed/`. Design doc path: `.../design-doc/01-go-codebase-browser-analysis-and-implementation-guide.md`. Diary path (this file): `.../reference/01-investigation-diary.md`.

## Step 2: Drafting the design doc

I wrote the full design doc in one pass, optimizing for someone who has never seen the repo before. The structure follows the `writing-style.md` ordering (executive summary, problem/scope, current-state, gap analysis, proposed architecture, API sketches, pseudocode, phased plan, tests, risks, references). Major design decisions were made explicitly in the doc rather than buried in the diary, but this step captures the reasoning I had while writing.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Convert the four-bullet idea in the user's prompt into a complete design-doc including architecture, file layout, API, Glazed commands, frontend package split, and a phased plan.

**Inferred user intent:** A document sharp enough to start implementation on a Monday morning without needing a kickoff conversation.

### What I did

1. Read the three most relevant skills in full: `glazed-command-authoring/SKILL.md`, `go-web-frontend-embed/SKILL.md`, `react-modular-themable-storybook/SKILL.md`, plus `parts-and-tokens.md`.
2. Wrote the design doc in one pass: 15 sections, ~1,000 lines.
3. Sequenced the major decisions explicitly in the doc (§6.3 Stable IDs, §7.1 extractor entrypoint choice, §10.1 app vs widgets split, §11.3 drift guarantee).

### Why

Loading the skills first ensures the design uses the exact import paths, file-layout conventions, and theming contracts the user has standardized on. It also prevents the doc from being a generic "design a code browser" doc — it is specifically a Glazed-idiomatic, embed-idiomatic, parts/tokens-idiomatic doc.

### What worked

1. Anchoring the design on three existing skills makes most of the scaffolding "free" — the doc mostly customizes a handful of decisions on top of known-good templates.
2. Splitting the frontend into an **app** and a **themable widgets** package up front makes the theming/Storybook asks from the user cleanly addressable: the widgets package is presentational + Storybook-only, the app is RTK-Query-and-router land.
3. The "drift guarantee" is a clear, quotable property: *because the index is built from the exact source being embedded, and the doc renderer resolves snippets against that index, no deployed binary can show a stale snippet.* This is the design's thesis.

### What didn't work

I initially considered making the index computation happen *at compile time via `go:generate`* versus *via the binary indexing itself on startup*. The startup option is tempting because it avoids committing a generated artifact, but it:

1. Requires shipping a Go-toolchain-equivalent parser set (every `go/packages.Load` dep) into the runtime binary, bloating it.
2. Delays first byte on `serve` startup by hundreds of ms.
3. Breaks "one immutable file" mental model — the binary would not be bit-identical across runs even with identical source.

I rejected it in §14.2.

### What I learned

1. **Stable symbol IDs** are the single most consequential design decision. If IDs depend on file paths, a file move invalidates every doc snippet referencing symbols in that file. Keying by `importPath + Kind + Name` (with `signatureHash` only for disambiguation) makes the IDs survive file moves.
2. **Byte offsets beat line/column** for snippet slicing. Once byte offsets are cached in the index, snippet extraction is O(1) `fs.ReadFile` + slice, with zero exposure to line-ending / tab-width issues.
3. **Flat JSON arrays** are friendlier than nested trees for both Go decoding and RTK-Query normalization. "Normalize at the shape layer, not at the transport layer" — the frontend builds the tree view by walking packages/files/symbols locally.

### What was tricky to build

Not applicable (design-only step). The conceptually tricky piece — stable IDs surviving refactors — is documented in §6.3 as the scheme the implementer should follow. The underlying cause of the trickiness is that Go's `types.Object.Id()` is file-oblivious and receiver-oblivious for methods; so the scheme pads it with `Kind` and an explicit `#signatureHash` segment for disambiguation.

### What warrants a second pair of eyes

1. **Index size claim** (§6.4): I estimated 3–5 MB for ~300 files / ~5k symbols. That is an informed estimate, not a measurement. Phase 1 should measure on a real project (ideally this one once it has code) before committing to the "load whole index on boot" strategy. If it blows up to 30 MB on a larger project, the split-load story in §14.1 kicks in.
2. **Path hygiene on `/api/source`** (§9.2): embedded FS is the primary sandbox, but I want an explicit test asserting that `..`, absolute paths, and paths not present in the index table are all rejected with 400/404.
3. **`sha256` per file**: I added it to the `File` record, but have not defined whether it's computed over raw file bytes or over the embedded copy. It must be the embedded copy, otherwise the "I can prove I serve what I indexed" audit is hollow.
4. **Doc authoring ergonomics** (§11.4): `codebase-browser doc render --check` failing in CI is the right gate, but we should make sure its output points authors back at the exact fenced block and offers the closest-match symbol IDs to reduce the friction of the first error they hit.

### What should be done in the future

1. Implement phase 0 (scaffolding) first, then phase 1 (indexer + CLI). Everything downstream depends on the index shape being real.
2. Once the indexer is working on this repo, **dogfood immediately** — write one real doc page that embeds a live snippet of the indexer. This is the smallest possible end-to-end validation of the drift guarantee.
3. Measure `index.json` size, server startup time, and SPA bundle size after phase 3 and recalibrate the split-load / lazy-load decisions.
4. Revisit search ranking after phase 3; the initial "prefix + substring" approach is likely not good enough for discovery.
5. Decide the `@codebase-browser/ui` publishing story after phase 4 based on whether there is a second consumer.

### Code review instructions

1. Read the design doc top to bottom. The ordering is intentional — each section depends on earlier ones.
2. Pay particular attention to:
   - §6.2 (record shapes) — every downstream consumer assumes this.
   - §6.3 (stable IDs) — changing this later is expensive.
   - §7.6 (deterministic output) — tests will golden-file this, so the rules need to be followed.
   - §11 (doc rendering) — this is the novel contribution.
3. Stress-test the phased plan: can you imagine a PR per phase? Each phase in §12 is sized to be one or two PRs.
4. Check §14.1 (risks) for anything that should block phase 1 vs. things we can address in phase 6+.

### Technical details

Doc sections and their weight (lines, approximate):

```
§1 Executive summary          ~20
§2 Problem statement/scope    ~30
§3 Current-state              ~20
§4 Gap analysis               ~20
§5 Proposed architecture      ~120 (topology + repo layout)
§6 Index schema               ~90
§7 Extraction pipeline        ~70
§8 Glazed CLI                 ~90
§9 HTTP API                   ~50
§10 Frontend                  ~140
§11 Doc pages with snippets   ~60
§12 Implementation plan       ~80
§13 Testing                   ~30
§14 Risks/alternatives/open   ~60
§15 References                ~15
```

Key cross-references:

- §5.2 (repo layout) → §12 (phase files map 1:1).
- §6.3 (IDs) → §11.1 (author-facing short form relies on ID scheme).
- §9.3 (`/api/snippet`) → §10.4 (frontend `<Snippet/>` semantics).
- §10.1 (app vs widgets split) → §14.3 (open question on publishing `@codebase-browser/ui`).

## Step 3: Pending — ticket bookkeeping and reMarkable delivery

This step will (a) relate the two docs to the key skill files that informed them, (b) update `tasks.md` and `changelog.md`, (c) run `docmgr doctor`, and (d) dry-run + upload to reMarkable. The diary will be appended with verification evidence (remote listing output) once upload is confirmed.
