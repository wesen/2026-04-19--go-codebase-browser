---
Title: Adopt diffs library for nicer semantic diff rendering
Ticket: GCB-011
Status: active
Topics:
    - codebase-browser
    - react-frontend
    - semantic-diff
    - ui-design
    - documentation-tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ui/package.json
      Note: UI package where @pierre/diffs dependency will be added
    - Path: ui/src/features/doc/widgets/CommitWalkWidget.tsx
      Note: Commit walk composes diff widgets and should benefit from the shared wrapper
    - Path: ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx
      Note: Primary embedded semantic diff widget to upgrade to @pierre/diffs
    - Path: ui/src/features/history/HistoryPage.tsx
      Note: Focused symbol-history body diff renderer to replace with shared Diffs wrapper
ExternalSources:
    - https://diffs.com/docs
Summary: Adopt @pierre/diffs for nicer, more maintainable diff rendering across embedded semantic diff widgets and focused symbol-history pages.
LastUpdated: 2026-04-25T12:35:00-04:00
WhatFor: Plan and track replacing hand-rendered diffs with the Diffs library.
WhenToUse: Use before implementing or reviewing the Diffs integration.
---


# Adopt diffs library for nicer semantic diff rendering

## Overview

GCB-011 tracks integrating [`@pierre/diffs`](https://diffs.com/docs) for nicer diff rendering across codebase-browser. The immediate target is to replace hand-rendered unified diff blocks in embedded semantic widgets and the focused symbol-history page while keeping the history-backed data model from GCB-010.

The upstream Diffs documentation has been downloaded into [`sources/`](./sources/) with Defuddle for offline/reference use.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- codebase-browser
- react-frontend
- semantic-diff
- ui-design
- documentation-tooling

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
