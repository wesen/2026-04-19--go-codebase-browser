---
Title: 'Structured Query Concepts Catalog for SQLite Codebase Index'
Ticket: GCB-008
Status: active
Topics:
    - sqlite
    - cli
    - query-catalog
    - concepts
    - web-ui
DocType: index
Intent: implementation
Owners: []
RelatedFiles: []
ExternalSources:
    - Path: /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd
      Note: Reference implementation for structured query catalog loading and typed commands
    - Path: /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/23/GCB-007--sqlite-codebase-index-query-symbols-files-and-xrefs-with-sql/reference/02-structured-query-concepts-report.md
      Note: Research report that motivates this ticket
Summary: "Implement SQL-only structured query concepts for codebase-browser's SQLite index."
LastUpdated: 2026-04-24T00:00:00Z
WhatFor: "Track work to turn reusable SQLite queries into named, typed CLI concepts that can later power web UI forms."
WhenToUse: "Use while implementing or reviewing the concept catalog, dynamic CLI verbs, render-only SQL preview, and concept query files."
---

# Structured Query Concepts Catalog for SQLite Codebase Index

## Overview

GCB-008 adds a go-minitrace-inspired structured query catalog to `codebase-browser`.

The previous ticket, GCB-007, added the SQLite database and raw SQL CLI. This ticket adds the next layer: named query concepts with typed parameters, render-only validation, generated CLI verbs, and a metadata shape that can later be exposed to the website as generated forms.

## Key Links

- [Implementation guide](./design-doc/01-structured-query-concepts-implementation-guide.md)
- [Tasks](./tasks.md)
- [Diary](./reference/01-implementation-diary-structured-query-concepts.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**

## Topics

- sqlite
- cli
- query-catalog
- concepts
- web-ui

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).
