---
Title: Implementation Diary
Ticket: GCB-004
Status: active
Topics:
    - codebase-browser
    - documentation-tooling
    - react-frontend
    - embedded-web
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Diary for GCB-004 — replacing inline <pre><code> source snippets on doc pages with React-hydrated <SymbolCard> widgets via server-emitted data-stub divs.
LastUpdated: 2026-04-20T12:20:00Z
WhatFor: Step-by-step record of the stub-and-hydrate implementation including trade-offs, surprises, and review pointers.
WhenToUse: Read alongside the design doc when modifying the server renderer or the DocPage hydration path.
---

# Implementation Diary

## Goal

Ship the Option A design from the preceding conversation: let doc-page `codebase-snippet` directives render as the same `<SymbolCard>` / `<LinkedCode>` widgets used on symbol pages, via server-emitted placeholder stubs that the React frontend hydrates on mount.

## Step 1: Ticket scaffold + design doc

Created the GCB-004 ticket (GCB-003 was already claimed by an unrelated deployment project), filled in the design-doc with the full phased plan, and seeded five implementation tasks on `tasks.md`.

### Prompt Context

**User prompt (verbatim):** "add a ticket for option A and an implementation guide and then implement it."

**Assistant interpretation:** Create a docmgr ticket for the stub-hydration design described in the previous message, write a design/implementation guide detailing server stub emission + frontend hydration flow, then execute the three implementation phases committing at phase boundaries.

**Inferred user intent:** Close the "doc snippets are inert text" gap so that every `codebase-snippet` directive renders with syntax highlighting, xref links, and godoc annotations — matching the richness of `/symbol/{id}` pages — without any changes to doc-page authoring.

### What I did

1. `docmgr ticket create-ticket --ticket GCB-004 ...` (GCB-003 was taken by a pre-existing deployment ticket).
2. `docmgr doc add` for the design-doc + diary.
3. Wrote `design-doc/01-react-hydrated-snippet-widgets-design-and-implementation-guide.md` covering: problem, current state, target design, phased plan, detailed code changes, risks, and testing strategy.
4. Seeded `tasks.md` with five tasks (server renderer change, stub metadata, frontend hydration, meta-page exercise, fallback).

### Why

Having the design written out up front settles two decisions that would otherwise block mid-implementation: (1) the stub carries its plaintext fallback inside it (so JS-disabled readers still see something) and (2) each stub gets its own `createRoot` rather than sharing a single React tree (keeping goldmark-generated prose intact without trying to reconstruct it).

### What worked

Reusing the existing `SymbolCard` + `LinkedCode` components means the whole "hydration" step is maybe 30 lines of glue — `useEffect` + `querySelectorAll('[data-codebase-snippet]')` + `createRoot` per stub. Everything else is already built.

### What didn't work

N/A (scaffold step).

### What I learned

docmgr errors cleanly when a ticket ID collides; it doesn't overwrite the existing one. Found out by creating GCB-003 and getting an ambiguous-ticket error on follow-up commands. Chose GCB-004 instead and the issue disappeared.

### What should be done in the future

Implementation phases 1 (server stubs) and 2 (frontend hydration), then meta-page verification. If either surfaces a surprise, capture it in a follow-up diary step.

### Code review instructions

1. `cat ttmp/2026/04/20/GCB-004--.../design-doc/01-...md` — should contain §1 (exec summary) through §9 (references).
2. `docmgr task list --ticket GCB-004` — should show 5 open tasks.
3. No code changes in this step; the implementation phases land the actual renderer + frontend changes.

### Technical details

Ticket path: `ttmp/2026/04/20/GCB-004--react-hydrated-snippet-widgets-on-doc-pages/`.
