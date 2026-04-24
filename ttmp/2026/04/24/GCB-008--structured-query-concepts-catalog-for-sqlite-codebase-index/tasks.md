# Tasks

## Goal

Add SQL-only structured query concepts to `codebase-browser`: named SQL templates with typed parameters, generated CLI verbs, render-only validation, aliases later, and a metadata shape suitable for future web UI forms.

## Phase 1 — Concept catalog package

- [ ] Create `internal/concepts/`.
- [ ] Define `Concept`, `ConceptSpec`, `Param`, `Catalog`, and source-root types.
- [ ] Detect structured SQL concept files by preamble marker.
- [ ] Parse SQL concept files into specs.
- [ ] Compile specs into catalog entries indexed by path and name.
- [ ] Add tests for parsing, validation, duplicate detection, and catalog loading.

## Phase 2 — SQL rendering

- [ ] Add `RenderConcept(concept, values)`.
- [ ] Add template helpers: `value`, `sqlString`, `sqlLike`, `sqlStringIn`, `sqlIntIn`.
- [ ] Add typed default hydration for concept parameters.
- [ ] Add render-only tests with parameterized SQL.

## Phase 3 — First concept files

- [ ] Create `concepts/` as the user-facing structured query catalog.
- [ ] Convert `package-counts` into a concept.
- [ ] Convert `exported-functions` into a concept.
- [ ] Convert `most-referenced` into a concept.
- [ ] Convert `refs-for-symbol` into a parameterized concept.

## Phase 4 — Dynamic CLI verbs

- [ ] Add `codebase-browser query commands`.
- [ ] Generate nested Cobra commands from concept folders and names.
- [ ] Map concept parameters to typed CLI flags.
- [ ] Add `--render-only` to print rendered SQL without executing.
- [ ] Execute rendered SQL against `codebase.db`.
- [ ] Preserve the existing raw SQL `codebase-browser query ...` behavior.

## Phase 5 — Validation and docs

- [ ] Run `go generate ./internal/sqlite`.
- [ ] Validate concept commands against the generated DB.
- [ ] Run `go test ./internal/concepts ./cmd/codebase-browser/cmds/query ./...`.
- [ ] Update implementation diary and changelog after each slice.
- [ ] Commit code and docs at appropriate intervals.

## Future phases, explicitly out of scope for this ticket

- [ ] Alias files with prefilled defaults.
- [ ] JavaScript concept runtime.
- [ ] HTTP API for concept listing/execution.
- [ ] Generated browser forms.
