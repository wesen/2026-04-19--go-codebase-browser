---
Title: 'Git-Aware Codebase Index: Track Symbol Locations Across Commits for Per-Function Diff and Change History'
Ticket: GCB-009
Status: active
Topics:
    - git
    - indexing
    - sqlite
    - symbols
    - diff
    - history
    - concepts
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/concepts/types.go
      Note: Concept types — will be extended with history concepts
    - Path: internal/indexer/extractor.go
      Note: Go AST extractor that produces Index — will run once per commit
    - Path: internal/indexer/id.go
      Note: Stable symbol ID scheme — foundation of cross-commit tracking
    - Path: internal/indexer/types.go
      Note: Core index types (Index
    - Path: internal/sqlite/loader.go
      Note: Bulk load pattern — will be adapted for snapshot loading
    - Path: internal/sqlite/schema.go
      Note: Single-commit SQLite schema — basis for the snapshot schema
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-24T19:29:11.421812089-04:00
WhatFor: ""
WhenToUse: ""
---







# Git-Aware Codebase Index: Track Symbol Locations Across Commits for Per-Function Diff and Change History

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- git
- indexing
- sqlite
- symbols
- diff
- history
- concepts

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
