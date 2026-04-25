---
Title: Future assessment of Diffs-powered review UX improvements
Ticket: GCB-012
Status: active
Topics:
    - codebase-browser
    - react-frontend
    - semantic-diff
    - ui-design
    - performance
    - documentation-tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ui/src/features/diff/DiffsUnifiedDiff.tsx
      Note: Lazy-loading shell and fallback for Diffs review rendering
    - Path: ui/src/features/diff/DiffsUnifiedDiffRenderer.tsx
      Note: Current Diffs renderer
    - Path: ui/src/features/doc/widgets/AnnotationWidget.tsx
      Note: Future candidate for Diffs-native line annotations
    - Path: ui/src/features/doc/widgets/CommitWalkWidget.tsx
      Note: Future candidate for richer authoring/deep links/preferences
    - Path: ui/src/features/history/HistoryPage.tsx
      Note: Focused symbol-history UX and diff surface
ExternalSources:
    - https://diffs.com/docs
Summary: Roadmap and assessment workspace for future improvements after adopting @pierre/diffs for semantic review widgets.
LastUpdated: 2026-04-25T13:05:00-04:00
WhatFor: Collect and prioritize future work without overloading GCB-011.
WhenToUse: Use when planning follow-up tickets for Diffs-powered review UX, performance, annotations, theming, accessibility, or patch support.
---


# Future assessment of Diffs-powered review UX improvements

## Overview

GCB-012 is a future assessment workspace for everything we should, can, or could address after GCB-010/GCB-011. It is intentionally a planning ticket: read the design guide, decide what matters next, then split implementation into smaller tickets.

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
- performance
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
