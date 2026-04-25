---
Title: Embeddable semantic diff widgets for literate PR code review in markdown
Ticket: GCB-010
Status: active
Topics:
    - codebase-browser
    - pr-review
    - semantic-diff
    - embeddable-widgets
    - markdown-directives
    - history-index
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for embedding semantic diff, history, impact, and code-browsing widgets into markdown doc pages, enabling literate PR review guides where reviewers navigate source and explore changes inline."
LastUpdated: 2026-04-25T12:00:00Z
WhatFor: Guide implementation of embeddable review widgets
WhenToUse: Read before implementing any new codebase-* directive or review widget
---

# GCB-010: Embeddable semantic diff widgets for literate PR code review

## Overview

This ticket designs and implements embeddable widgets that turn markdown documents into interactive code review workspaces. Reviewers can navigate source code, explore semantic diffs across commits, trace impact, and follow guided commit walks — all inline in a single markdown document, without switching windows.

## Key Links

- **Design doc**: `design-doc/01-embeddable-semantic-diff-widgets-design-affordances-and-architecture-for-literate-pr-review.md` (67KB, 10 sections, self-contained onboarding reference)
- **Investigation diary**: `reference/01-investigation-diary.md`
- **Tasks**: `tasks.md`
- **Changelog**: `changelog.md`

## Predecessor tickets

- **GCB-005** — Semantic PR review architecture and widget catalog (design only, no implementation)
- **GCB-009** — Git-aware indexing: per-commit snapshots, history SQLite store, symbol diff engine (implemented)

## Core idea

The codebase-browser has three pillars:
1. **Semantic index** — stable symbol IDs, cross-references, byte-accurate ranges
2. **Git history** — per-commit symbol snapshots in SQLite, diff engine, body diff
3. **Directive pipeline** — `codebase-*` fenced blocks → hydrated React widgets in markdown

GCB-010 connects them with 7 new directives: `codebase-diff`, `codebase-symbol-history`, `codebase-impact`, `codebase-commit-walk`, `codebase-annotation`, `codebase-changed-files`, `codebase-diff-stats`.

## Status

Current status: **active** — design complete, implementation pending.

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` — Architecture and design documents
- `reference/` — Investigation diary, prompt packs, API contracts
- `playbooks/` — Command sequences and test procedures
- `scripts/` — Temporary code and tooling
