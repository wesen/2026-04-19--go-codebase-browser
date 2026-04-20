---
Title: Investigation diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-20T12:13:36.644336168-04:00
WhatFor: ""
WhenToUse: ""
---


# Investigation diary

## Goal

Figure out how to deploy `codebase-browser` as a public example page on `codebase-browser.yolo.scapegoat.dev` using the same GitHub Actions -> GHCR -> GitOps PR -> Argo CD pattern used in the Hetzner K3s repo.

## Context

The app repo already looks like a deployable single-binary web service. The remaining unknowns were the release packaging contract, the GitOps manifest shape, and whether the existing self-documenting page already satisfied the "example page" requirement without extra code.

## Quick reference

### Core files inspected

- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/README.md`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/Makefile`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/serve/run.go`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/server.go`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/spa.go`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/pages.go`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_doc.go`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/app/App.tsx`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocPage.tsx`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/03-meta.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/source-app-deployment-infrastructure-playbook.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/public-repo-ghcr-argocd-deployment-playbook.md`
- `/home/manuel/code/wesen/obsidian-vault/Projects/2026/03/29/PROJ - Serve Artifacts - Deploying to K3s with GitOps.md`

### Commands run and what they showed

| Command | What it told me |
| --- | --- |
| `docmgr status --summary-only` | The repo already had GCB-001 and GCB-002; there was room for a new ticket. |
| `find . -maxdepth 2 -type f \| sort` | The app repo has source, UI, and ticket docs, but no `.github/`, `Dockerfile`, or `deploy/` directory yet. |
| `find /home/manuel/code/wesen/2026-03-27--hetzner-k3s -maxdepth 3 -type f \| sort` | The infra repo already contains the public GHCR + Argo playbooks and the `gitops/kustomize/artifacts` pattern. |
| `nl -ba ...README.md`, `Makefile`, `internal/server/*`, `internal/docs/*`, `ui/src/*` | Confirmed the app already has a single-binary embedded runtime and a `/doc/:slug` route for doc pages. |
| `docmgr vocab add ...` | Added missing deployment vocabulary slugs so the ticket could be tracked cleanly. |
| `docmgr ticket create-ticket ...` | Created GCB-003 and the workspace skeleton. |

## Usage examples

This reference is useful when implementing the ticket because it captures the concrete evidence that should drive the code changes:

- when adding the app repo `Dockerfile`, remember that `make build` is the canonical build contract
- when wiring CI, remember that the existing example page already lives at `/doc/03-meta`
- when writing the GitOps package, mirror `artifacts`/`pretext` rather than inventing a new deployment category
- when reviewing the rollout, verify the public endpoint, the docs page, and the GitOps image pin separately

## Session log

1. Started with repository discovery and `docmgr status`.
2. Read the current app README and build files to understand the runtime and build contract.
3. Inspected the server and docs code to confirm that docs pages are already an embedded runtime feature.
4. Read the Hetzner K3s public deployment playbooks and the live stateless app manifests.
5. Checked the Obsidian vault and found a useful prior writeup for the same GitOps pattern.
6. Added missing vocabulary terms and created ticket GCB-003.
7. Drafted the design guide and this diary.
8. Started implementation in the app repo: added the release packaging tasks and began wiring the Docker/CI release path.
9. Adjusted the build plan after confirming the web generator still needed to be aligned with the Dagger-based build path.
10. Implemented the source-tree snapshot generator, tightened the index build package set, and validated the embedded build path with `make build` and `go test ./...`.
11. Added the matching GitOps package and Argo CD `Application` in the Hetzner K3s repo so the app can now be bootstrapped into cluster reconciliation.
12. Pushed both repos, bootstrapped the live `codebase-browser` Argo CD `Application`, and resolved the initial stale-image bootstrap by recreating the Application so it could pick up the published `sha-e5700d9` image.
13. Validated the live rollout through Argo CD (`Synced Healthy Succeeded`) and a port-forward smoke test against `/api/index` and `/doc/03-meta`.

## What worked

- The repo already had a clear `make build` story and a single embedded runtime binary.
- The docs system already exposes a self-documenting example page; no new example server is needed.
- The Hetzner repo already documents the exact public deployment contract we want to reuse.
- The public stateless app manifests in the infra repo are a close match for this deployment.

## What didn't work

- I initially tried to read Hetzner deployment docs from the current repo path and got `ENOENT`; correcting the path to `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/...` fixed it.
- I also initially probed a nonexistent Obsidian note path before locating the actual `Projects/2026/03/29` file.

## What was tricky to build

- The build is not a plain `go build`; it is `go generate` on the generator packages plus `go build -tags embed`.
- `cmd/build-ts-index` prefers Dagger but falls back to local pnpm when Dagger is unavailable, so CI must make the Docker/Dagger path available.
- The web generator also needed to be aligned with Dagger so the release workflow can avoid host pnpm entirely.
- The source snapshot generator needed explicit exclusion rules so the embedded tree would not recurse into build outputs, nested modules, or test files.
- The example page is already present in source control, so the deployment work is mostly about packaging and release plumbing rather than adding a new feature surface.

## Implementation progress

- Added the first release-scaffolding files in the app repo: `Dockerfile`, `.dockerignore`, `deploy/gitops-targets.json`, `.github/workflows/publish-image.yaml`, and `scripts/open_gitops_pr.py`.
- Refactored the web build generator so `go generate ./internal/web` can use Dagger instead of requiring host pnpm.
- Added `internal/sourcefs/generate.go` + `internal/sourcefs/generate_build.go` so the embedded source tree can be regenerated as part of the build.
- Tightened `codebase-browser index build` defaults so the indexer does not crawl the embedded source snapshot.
- Updated the README and SPA/serve error paths so the build instructions match the new Dagger-first flow.
- Added the matching GitOps package and Argo CD `Application` in the Hetzner repo.
- Validated the local build with `make build`, `go test ./...`, and a smoke test against `/api/index` on the embedded server.
- Pushed both repos, bootstrapped the live Argo CD Application, and fixed the first rollout by deleting/recreating the Application so it could reconcile against the published GHCR image.
- Confirmed the live cluster ended in `Synced Healthy Succeeded` and the public app serves `/doc/03-meta` as expected.

## Code review instructions

When reviewing the eventual implementation, check these things in order:

1. `Makefile` still defines the canonical build path and the CI workflow follows it.
2. The Dockerfile packages only the compiled binary and does not smuggle in extra build dependencies.
3. The GitHub Actions workflow publishes immutable SHA tags and fails loudly if the GitOps PR credential is missing.
4. The GitOps package in the Hetzner repo mirrors the existing `artifacts`/`pretext` shape.
5. The Argo `Application` is bootstrapped once and then left to reconcile from Git.
6. The public browser exposes the example page through `/doc/03-meta` rather than a separate special-case runtime.
7. The live endpoint proves the docs sidebar and snippet resolution still work after deployment.
