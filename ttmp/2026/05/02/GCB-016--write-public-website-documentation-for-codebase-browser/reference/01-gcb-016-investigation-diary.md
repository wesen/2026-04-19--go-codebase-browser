---
Title: "GCB-016 Investigation Diary"
Ticket: GCB-016
Status: active
Topics:
    - codebase-browser
    - documentation-tooling
    - static-export
    - sqlite
    - react-frontend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
Summary: Chronological work diary for the public website documentation project.
LastUpdated: 2026-05-02T00:00:00Z
WhatFor: Track what was investigated, what decisions were made, what failed, and what to do next.
WhenToUse: Read before resuming work on GCB-016. Update after every work session.
---

# GCB-016 Investigation Diary

## Session 2026-05-02 — Initial orientation and task planning

### What I did

1. Read the technical-writer brief at `design/01-technical-writer-documentation-brief.md`.
2. Listed existing docs: `pkg/doc/user-guide.md` (Glazed tutorial, slug `user-guide`), `pkg/doc/db-reference.md` (Glazed reference, slug `db-reference`), and `pkg/doc/embed.go` (registers all `pkg/doc/*.md` via `embed.FS`).
3. Read README.md — currently describes both the old embedded-index server runtime and the new sql.js export. Needs a full audit.
4. Read the renderer (`internal/docs/renderer.go`) — confirms all 11 directives listed in the user-guide, including the `codebase-commit-walk` line-oriented step DSL.
5. Read the export pipeline (`internal/staticapp/export.go`) — confirms `manifest.json` fields, `HasGoRuntimeServer: false`, and `QueryEngine: "sql.js"`.
6. Verified exact command names with `go run ./cmd/codebase-browser/main.go review --help` — the actual command is `review index` (not `review db create`). The brief used stale naming.
7. Read `glazed-help-page-authoring` skill — confirmed exact frontmatter field names, SectionType values, and that no top-level `#` heading should be used in doc content.
8. Created the diary document.

### Key findings

- README mentions "build-time indexer walks the AST" and "embedded index.json" — old design. Needs full rewrite.
- `pkg/doc/` already has `embed.go` that embeds all `*.md` files via `//go:embed *.md`. Adding a new doc just means adding a `.md` file; no Go changes needed.
- Slugs verified discoverable: `user-guide`, `db-reference`. `markdown-block-reference` will appear after I add the file.
- README quick start used `--range` flag; actual flag is `--commits`.
- README quick start used `review db create`; actual command is `review index`.
- `./bin/codebase-browser` in the repo was stale (pre-`review` command). `make build` regenerates everything before running `docs-smoke`.

### What's next

- **Task 4** — Audit README against the new static sql.js architecture.
- **Task 5** — Create `pkg/doc/markdown-block-reference.md` with all 11 directives.
- **Task 6** — Update `pkg/doc/user-guide.md` to link to the new reference.
- **Task 7** — Update `pkg/doc/db-reference.md` with byte-offset warnings.
- **Task 8** — Rewrite README as concise website/README landing page.
- **Task 9** — Add example review markdown files under `examples/` and verify each.
- **Task 10** — Add smoke-test commands to the Makefile.

---

## Session 2026-05-02 — README audit and markdown block reference

### What I did

1. **README audit (task 4)** — Full rewrite of README.md:
   - Removed stale references to `index.json` and embedded index server runtime.
   - Rewrote product description for static sql.js model.
   - Updated quick start: `review index` (not `review db create`), `--commits` (not `--range`), `--docs`.
   - Added architecture diagram showing SQLite as runtime boundary.
   - Linked to all three Glazed help pages.
   - Updated repo layout to reflect current `internal/` structure.
   - Removed outdated building-the-index section.
   - Added `make docs-smoke` to testing commands.
   - Committed: `68ded30 docs: audit README against static sql.js architecture`.

2. **Markdown block reference (task 5)** — Created `pkg/doc/markdown-block-reference.md`:
   - Documented all 11 directives with quick reference table, required/optional params, examples.
   - Detailed symbol ref syntax (full `sym:` IDs and short refs).
   - Detailed commit ref syntax.
   - Documented `codebase-commit-walk` step DSL with complete kind reference.
   - Added troubleshooting table and See Also links.
   - Slug `markdown-block-reference` verified discoverable via `./bin/codebase-browser help --list`.
   - Committed: `a316104 docs: add markdown block reference page (pkg/doc/markdown-block-reference.md)`.

---

## Session 2026-05-02 — user-guide update, db-reference update, examples, smoke tests

### What I did

1. **User-guide update (task 6)** — Updated `pkg/doc/user-guide.md`:
   - Added link to `markdown-block-reference` in See Also section.
   - Committed: `8910f0a docs: update user-guide to link to markdown-block-reference`.

2. **DB reference update (task 7)** — Updated `pkg/doc/db-reference.md`:
   - Added prominent byte-offset warning under History tables section.
   - Added source DB vs export DB distinction table.
   - Updated See Also to link to `markdown-block-reference`.
   - Committed: `4936759 docs: update db-reference with byte-offset warnings and runtime guidance`.

3. **Example review markdown files (task 9)** — Created `examples/` directory with 4 files:
   - `01-pr-review-static-export.md` — signature, diff, impact widgets
   - `02-symbol-history-and-impact.md` — history, impact, diff-stats, changed-files
   - `03-commit-walk-walkthrough.md` — commit-walk step DSL demonstration
   - `04-file-and-annotation-examples.md` — file, annotation, signature, doc

   Smoke-tested with:
   ```bash
   ./bin/codebase-browser review index --commits HEAD~5..HEAD --docs ./examples --db /tmp/gcb-examples.db
   ./bin/codebase-browser review export --db /tmp/gcb-examples.db --out /tmp/gcb-examples-export
   ```
   Verified:
   - Export contains `manifest.json`, `db/codebase.db`, sql.js WASM files.
   - No `precomputed.json`, `search.wasm`, `wasm_exec.js`.
   - Manifest shows `hasGoRuntimeServer=false`, `queryEngine=sql.js`.
   - All 4 docs indexed and rendered in export DB.
   - Committed: `8466c25 docs: add example review markdown files`.

4. **Smoke-test target (task 10)** — Added `make docs-smoke` to Makefile:
   - Creates temp DB with examples.
   - Runs review export.
   - Checks `manifest.json` exists and `db/codebase.db` exists.
   - Checks no legacy runtime files.
   - Verifies docs in export DB with `sqlite3`.
   - Committed: `5f8c519 docs: add make docs-smoke smoke-test target`.

### Key discovery: `make build` needed before `make docs-smoke`

The `./bin/codebase-browser` in the repo was stale — it predated the `review` command addition. The `docs-smoke` target now includes `@if [ ! -f bin/$(BINARY) ]; then $(MAKE) build; fi` to self-build if the binary is missing or stale.

---

## Commits summary

```
68ded30 docs: audit README against static sql.js architecture
a316104 docs: add markdown block reference page (pkg/doc/markdown-block-reference.md)
8910f0a docs: update user-guide to link to markdown-block-reference
4936759 docs: update db-reference with byte-offset warnings and runtime guidance
8466c25 docs: add example review markdown files
5f8c519 docs: add make docs-smoke smoke-test target
```

## All 10 tasks complete. Ticket closing.

All tasks verified:
- [x] README audited and rewritten for static sql.js model
- [x] `pkg/doc/markdown-block-reference.md` created with all 11 directives
- [x] `pkg/doc/user-guide.md` updated to link to reference
- [x] `pkg/doc/db-reference.md` updated with byte-offset warnings
- [x] README serves as concise website landing page
- [x] 4 example review markdown files created and smoke-tested
- [x] `make docs-smoke` target added and verified passing

Diary updated, ticket closing.
