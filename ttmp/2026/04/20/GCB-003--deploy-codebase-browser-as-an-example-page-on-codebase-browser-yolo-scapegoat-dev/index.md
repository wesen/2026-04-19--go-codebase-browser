---
Title: Deploy codebase-browser as an example page on codebase-browser.yolo.scapegoat.dev
Ticket: GCB-003
Status: active
Topics:
    - codebase-browser
    - embedded-web
    - deployment
    - github-actions
    - ghcr
    - gitops
    - argocd
    - kubernetes
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Deploy the self-documenting codebase browser as a public GHCR-backed GitOps app and expose /doc/03-meta on codebase-browser.yolo.scapegoat.dev.
LastUpdated: 2026-04-20T16:30:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Deploy codebase-browser as an example page on codebase-browser.yolo.scapegoat.dev

## Overview

This ticket documents the path from the current local single-binary codebase browser to a public deployment on `codebase-browser.yolo.scapegoat.dev` using the same GitHub Actions -> GHCR -> GitOps PR -> Argo CD pattern already used in the Hetzner K3s repo.

The key point is that the example page already exists in-tree. The deployment task is to package the browser cleanly and expose the current docs route, especially `/doc/03-meta`, as the public example page.

## Key Links

- **Design doc**: [Implementation guide](./design-doc/01-implementation-guide-deploy-codebase-browser-to-codebase-browser-yolo-scapegoat-dev.md)
- **Investigation diary**: [Reference log](./reference/01-investigation-diary.md)
- **App repo**: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser`
- **Infra repo**: `/home/manuel/code/wesen/2026-03-27--hetzner-k3s`
- **Example page**: `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/03-meta.md`

## Status

Current status: **active**

## Topics

- codebase-browser
- embedded-web
- deployment
- github-actions
- ghcr
- gitops
- argocd
- kubernetes

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` - architecture and implementation guidance
- `reference/` - investigation diary and reusable context
- `playbooks/` - command sequences and test procedures
- `scripts/` - temporary code and tooling
- `various/` - working notes and research
- `archive/` - deprecated or reference-only artifacts
