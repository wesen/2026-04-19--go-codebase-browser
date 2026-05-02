---
Title: Implementation diary
Ticket: GCB-015
Status: active
Topics:
    - codebase-browser
    - static-export
    - sqlite
    - react-frontend
    - review-docs
    - architecture
    - history
    - markdown-directives
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/design-doc/01-sql-js-static-frontend-architecture-and-implementation-guide.md
      Note: Architecture source for implementation decisions
    - Path: ttmp/2026/05/01/GCB-015--implement-sql-js-static-codebase-browser-and-review-renderer/tasks.md
      Note: Task checklist that drives diary steps
ExternalSources: []
Summary: Chronological implementation diary for the static-only sql.js frontend cleanup.
LastUpdated: 2026-05-01T20:15:00-04:00
WhatFor: Use this diary to resume or review GCB-015 implementation work, including what changed, why, commands run, failures, commits, and validation notes.
WhenToUse: Read before continuing GCB-015 implementation or reviewing commits from this ticket.
---


# Implementation diary

## Goal

This diary records the implementation of GCB-015: replacing the prototype Go-server/TinyGo/precomputed-review static runtime paths with a static-only browser runtime backed by `sql.js` over `db/codebase.db`.

The intended final shape is simple: Go creates and packages the SQLite database, then exits. React opens that database in the browser with `sql.js`. There is no Go runtime server, no `ServerQueryProvider`, no `/api/*` fallback, and no dynamic reload path other than rerunning the index/export command and refreshing the browser.
