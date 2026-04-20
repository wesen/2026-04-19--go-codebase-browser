---
Title: 'Implementation guide: deploy codebase-browser to codebase-browser.yolo.scapegoat.dev'
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-20T12:13:33.061154814-04:00
WhatFor: ""
WhenToUse: ""
---



# Implementation guide: deploy codebase-browser to codebase-browser.yolo.scapegoat.dev

## Executive summary

`codebase-browser` is already very close to a deployable public example service. The repository already builds a single embedded binary, serves a React SPA plus doc pages from that binary, and contains a self-referential example page at `internal/docs/embed/pages/03-meta.md`. What is missing is not product architecture; what is missing is the packaging and release contract that turns the repo into a public GitHub Actions -> GHCR -> GitOps PR -> Argo CD deployment.

The recommended deployment target is `codebase-browser.yolo.scapegoat.dev`, using the same public-stateless-app pattern already proven in the Hetzner K3s repo for `artifacts` and `pretext`. That means:

- the app repo owns build/test/image publishing
- the GitOps repo owns runtime manifests and the Argo CD `Application`
- Argo CD reconciles the desired state into the cluster
- the live example page is the existing `/doc/03-meta` route, not a separate rendering system

## Problem statement and scope

The problem is to make the browser publicly available in a way that is reproducible, reviewable, and consistent with the platform's existing GitOps practices.

The scope is intentionally narrow:

- deploy the current repo as a public web service
- use the existing single-binary runtime model
- expose the existing example doc page through the current docs route
- publish immutable images to GHCR
- let Argo CD roll the workload from a GitOps manifest change

Out of scope for this ticket:

- redesigning the browser UI
- changing the indexer or doc-rendering model
- adding secrets, databases, or persistent storage
- introducing a new content system for the example page
- adopting an app-of-apps or ApplicationSet layer before the first rollout

## Current-state analysis

### The repo already has the right runtime shape

The root README says the project is a single-binary documentation browser with an embedded index, an embedded source snapshot, and a small React SPA (`README.md:3-12`). The quick-start section already describes the canonical build and run flow: install the UI and TypeScript indexer dependencies, run `make build`, then start `./bin/codebase-browser serve --addr :3001` (`README.md:26-49`).

The build contract is already encoded in `Makefile`: `generate` runs `go generate ./...`, `build` depends on `generate`, and `build` then runs `go build -tags embed -o bin/codebase-browser ./cmd/codebase-browser` (`Makefile:31-35`). That is exactly the kind of deterministic build entry point a CI pipeline wants.

The serve command already wires the embedded runtime together. `cmd/codebase-browser/cmds/serve/run.go` loads `indexfs.Bytes()`, constructs a `browser.LoadFromBytes` view of the embedded index, then mounts `server.New(loaded, sourcefs.FS(), web.FS())` (`cmd/codebase-browser/cmds/serve/run.go:58-80`). The HTTP server itself registers `/api/index`, `/api/packages`, `/api/doc`, `/api/doc/`, and other API routes before the SPA fallback (`internal/server/server.go:25-43`). The SPA handler rejects `/api/*`, serves files if they exist, and otherwise falls back to `index.html` so client-routed pages keep working (`internal/server/spa.go:10-38`).

### The self-documenting example page already exists

The repo already contains a worked example under `internal/docs/embed/pages/03-meta.md`. That page demonstrates the key feature this deployment is meant to showcase: a doc page can embed live snippets and signatures from the live index, including `internal/indexer.Merge`, the TypeScript tokenizer, and `internal/server.Server.handleXref` (`internal/docs/embed/pages/03-meta.md:1-68`). In other words, the example page is already part of the compiled product; it just needs to be served publicly.

The docs runtime is also already implemented. `internal/docs/pages.go` walks every `*.md` file under `internal/docs/embed/pages/` and exposes slug/title/path metadata (`internal/docs/pages.go:17-44`). `internal/docs/renderer.go` resolves `codebase-snippet`, `codebase-signature`, `codebase-doc`, and `codebase-file` fences against the embedded index and source tree, then renders Markdown with Goldmark (`internal/docs/renderer.go:49-210`). The `/api/doc` handler lists pages and `/api/doc/:slug` renders one page on demand (`internal/server/api_doc.go:11-41`).

On the frontend, the docs are already part of the normal browser routes. `ui/src/app/App.tsx` registers `/doc/:slug` alongside `/`, `/packages/:id`, `/symbol/:id`, and `/source/*` (`ui/src/app/App.tsx:14-36`). `ui/src/features/doc/DocPage.tsx` fetches `/api/doc/:slug`, renders the HTML returned by the server, and shows how many live snippet references were resolved (`ui/src/features/doc/DocPage.tsx:5-24`). So the example page contract already exists end-to-end.

### What is missing today

The repo currently has no deployment packaging in-tree. A root scan showed no `.github/` directory, no `Dockerfile`, and no `deploy/` directory. So the browser has a build contract, but not a release contract yet.

That is the key gap: the code exists, but the app is not yet packaged as a deployable artifact, and nothing yet tells the Hetzner GitOps repo which image should run at `codebase-browser.yolo.scapegoat.dev`.

### The target pattern is already proven in the Hetzner K3s repo

The Hetzner repo already documents the exact public-app pattern we want to reuse. The source-app deployment playbook says the app repo should own source code, tests, Docker packaging, image publishing, deployment target metadata, and the workflow that opens GitOps pull requests (`docs/source-app-deployment-infrastructure-playbook.md:29-61, 168-246`). The public-repo playbook says the image should be built in GitHub Actions, published to GHCR, pinned in GitOps with an immutable SHA tag, and deployed by Argo CD (`docs/public-repo-ghcr-argocd-deployment-playbook.md:28-37, 63-83, 125-141, 202-257`).

The live K3s manifests for `artifacts` and `pretext` show the concrete public-stateless-app shape we should mirror: a namespace, a deployment, a service, an ingress, and an Argo CD `Application` that points at a Kustomize package (`gitops/kustomize/artifacts/*.yaml`, `gitops/applications/artifacts.yaml`, `gitops/kustomize/pretext/*.yaml`, `gitops/applications/pretext.yaml`).

## Architecture and data flow

```mermaid
flowchart LR
    A[codebase-browser source repo] --> B[GitHub Actions]
    B --> C[make build\n(go generate + go build -tags embed)]
    C --> D[bin/codebase-browser]
    D --> E[small runtime image]
    E --> F[GHCR image tag\nsha-<git-sha>]
    F --> G[GitOps repo image pin]
    G --> H[Argo CD Application]
    H --> I[Kubernetes Deployment]
    I --> J[codebase-browser.yolo.scapegoat.dev]

    style F fill:#dff7e4,stroke:#1f7a3e
    style G fill:#fff2cc,stroke:#b7791f
    style J fill:#dbeafe,stroke:#2563eb
```

There is a second, smaller request path worth keeping in mind:

```mermaid
flowchart TD
    U[Browser request to /doc/03-meta] --> V[React route /doc/:slug]
    V --> W[/api/doc/03-meta]
    W --> X[internal/docs.Render]
    X --> Y[resolve live snippets from embedded index + source]
    Y --> Z[HTML response]
```

That second flow is why the example page is a good deployment target: the demo content already exercises the real runtime path.

## Gap analysis

The missing pieces fall into four buckets.

1. **Application packaging**
   - There is no `Dockerfile` yet.
   - There is no `.github/workflows/publish-image.yaml` yet.
   - There is no `deploy/gitops-targets.json` yet.
   - There is no PR helper script or reusable workflow wiring yet.

2. **GitOps runtime package**
   - There is no `gitops/kustomize/codebase-browser/` package yet in the Hetzner repo.
   - There is no `gitops/applications/codebase-browser.yaml` yet.
   - There is no ingress host entry for `codebase-browser.yolo.scapegoat.dev` yet.

3. **Cluster bootstrap**
   - Even after the manifests exist in Git, the `Application` object must still be applied once to the cluster because the repo does not auto-create new Argo applications (`docs/source-app-deployment-infrastructure-playbook.md:87-118`).

4. **Operational proof**
   - We still need an end-to-end smoke test proving that the public endpoint serves the browser, the docs sidebar, and `/doc/03-meta`.

## Proposed solution

### 1. Package the app repo as a CI-built public image

The app repo should keep using `make build` as the canonical build contract. That already captures the necessary steps: install dependencies, run `go generate ./...`, then compile the embedded binary (`README.md:26-49`, `Makefile:31-35`). The build workflow should therefore do three things on a clean GitHub runner:

1. install Go, Node, and pnpm
2. run the repo's own test/build commands
3. publish a small runtime image that only copies the compiled binary

A minimal runtime Dockerfile is enough because the binary already embeds the index, the source tree, and the SPA assets. A sensible image shape is:

```dockerfile
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY bin/codebase-browser /app/codebase-browser
EXPOSE 8080
ENTRYPOINT ["/app/codebase-browser"]
CMD ["serve", "--addr", ":8080"]
```

The build step should run `serve` on `:8080` inside the container so the K8s Service can keep its usual `port: 80 -> targetPort: http` shape.

### 2. Publish immutable GHCR tags

The workflow should publish tags like `sha-<shortsha>` and, optionally, convenience tags such as `main` or `latest`. The GitOps deployment must pin the SHA tag, not `latest`, because the whole point is to make rollouts and rollbacks visible in Git (`docs/public-repo-ghcr-argocd-deployment-playbook.md:125-141`).

This matters especially for a public example page: the goal is a reviewable release contract, not a magic floating tag.

### 3. Open GitOps pull requests from CI

The app repo should carry deployment target metadata in `deploy/gitops-targets.json`, mirroring the pattern already used in the Hetzner playbooks (`docs/source-app-deployment-infrastructure-playbook.md:57-61, 173-199`). For this ticket, the single target should point at the Hetzner GitOps repo, a manifest path under `gitops/kustomize/codebase-browser/`, and the `codebase-browser` container name.

The CI job should clone the GitOps repo, patch exactly one container image field, and open a pull request. If the required `GITOPS_PR_TOKEN` is missing, the workflow should fail loudly rather than silently skip the PR step. The Serve Artifacts report showed why: a silent `exit 0` turns a broken release handoff into a false green build.

Pseudocode for the release job:

```text
on pull_request:
  install Go + Node + pnpm
  run tests
  run make build
  build the runtime image
  do not push

on push to main:
  install Go + Node + pnpm
  run tests
  run make build
  build and push ghcr.io/wesen/codebase-browser:sha-<shortsha>
  if GITOPS_PR_TOKEN is missing: fail
  clone the GitOps repo
  patch the codebase-browser image field
  open a GitOps PR
```

Before the first push, bootstrap the GitHub secret from the Hetzner K3s repo's `.envrc`:

```bash
cd /home/manuel/code/wesen/2026-03-27--hetzner-k3s
set -a
source .envrc
set +a
gh secret set GITOPS_PR_TOKEN --repo wesen/2026-04-19--go-codebase-browser
```

### 4. Add the GitOps package in the Hetzner repo

The GitOps repo should get a stateless public-app package that mirrors `artifacts` and `pretext`:

- `gitops/kustomize/codebase-browser/namespace.yaml`
- `gitops/kustomize/codebase-browser/deployment.yaml`
- `gitops/kustomize/codebase-browser/service.yaml`
- `gitops/kustomize/codebase-browser/ingress.yaml`
- `gitops/kustomize/codebase-browser/kustomization.yaml`
- `gitops/applications/codebase-browser.yaml`

The deployment should follow the same hardening defaults as the other public apps:

- `enableServiceLinks: false`
- `imagePullPolicy: IfNotPresent`
- readiness and liveness probes on `/`
- `resources` sized like a tiny static web service
- `sync-wave` ordering for namespace -> deployment -> ingress

A representative deployment skeleton would look like this:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codebase-browser
spec:
  replicas: 1
  template:
    spec:
      enableServiceLinks: false
      containers:
        - name: codebase-browser
          image: ghcr.io/wesen/codebase-browser:sha-<shortsha>
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 8080
              name: http
          readinessProbe:
            httpGet:
              path: /
              port: http
          livenessProbe:
            httpGet:
              path: /
              port: http
```

The ingress should bind `codebase-browser.yolo.scapegoat.dev` and the Application should point at the Kustomize package in the Hetzner repo, exactly like the existing `artifacts` and `pretext` Applications.

### 5. Keep the example page as the existing docs route

No special page server is needed. The example page already lives at `/doc/03-meta`, and the runtime already knows how to render it from the embedded Markdown and live index (`internal/docs/embed/pages/03-meta.md:1-68`, `internal/server/api_doc.go:11-41`, `ui/src/features/doc/DocPage.tsx:5-24`).

If we later decide the public root should deep-link directly to the example, that can be a tiny follow-up. For the first rollout, the better choice is to keep the browser behavior normal and let the sidebar docs list surface the example naturally.

## Design decisions

1. **Treat this as a public stateless app.**
   The runtime has no database, no secrets, and no persistent data plane. That fits the `artifacts`/`pretext` deployment category.

2. **Use the repo's existing build contract.**
   `make build` already captures the correct generation and embedding sequence (`Makefile:31-35`). Reusing it keeps the CI story aligned with local development.

3. **Package a minimal runtime image.**
   The binary already embeds the docs, source snapshot, and SPA, so the image should not carry build tooling.

4. **Pin SHA tags in GitOps.**
   Immutable tags make review, rollback, and debugging straightforward.

5. **Fail loudly on missing PR credentials.**
   Silent skips are acceptable for optional features, not for the required release handoff.

6. **Bootstrap the Argo Application once, then let GitOps own the rest.**
   The Hetzner repo does not auto-discover new `Application` objects, so the first apply is a separate step (`docs/source-app-deployment-infrastructure-playbook.md:87-118`).

7. **Use `/doc/03-meta` as the example page.**
   The example is already in-tree and already exercises the live snippet-resolution path.

## Alternatives considered

### Build everything inside the Dockerfile

This would be the most hermetic packaging model, but it would also force us to recreate the repo's Go + Node + pnpm + Dagger-compatible build path inside the container build. For this repository, that is more complexity than the deployment needs. The build is already expressed cleanly as `make build`, so using the GitHub runner for generation and a tiny runtime image for packaging is a better fit.

### Deploy with a manual node image import

The Hetzner repo has used node-local image import as an emergency bridge before, but the public-repo playbook explicitly treats that as a fallback, not the standard path (`docs/public-repo-ghcr-argocd-deployment-playbook.md:37, 250-257`). For a public example page, we should not start with the bridge.

### Add ApplicationSet or app-of-apps now

That would automate the first `Application` creation, but it also adds another control plane before we have a single app rolling cleanly. The current repo already documents the bootstrap requirement, so the simpler path is to create the `Application` once and keep the model easy to explain.

### Create a separate static example site

Unnecessary. The self-documenting browser already exists, and the docs route already renders the example page from the live index.

## Implementation plan

### Phase 1: app repo packaging

Files to add or update in the browser repo:

- `Dockerfile`
- `.github/workflows/publish-image.yaml`
- `deploy/gitops-targets.json`
- `scripts/open_gitops_pr.py` or the chosen reusable PR helper
- `README.md` deployment section, if we want the public URL documented there

Implementation notes:

- keep `make build` as the canonical pre-image step
- ensure `pnpm -C ui install` and `pnpm -C tools/ts-indexer install` happen on a clean runner
- emit `ghcr.io/wesen/codebase-browser:sha-<shortsha>`
- fail the workflow if `GITOPS_PR_TOKEN` is absent and the PR step is required

### Phase 2: GitOps package in hetzner-k3s

Files to add in the infra repo:

- `gitops/kustomize/codebase-browser/namespace.yaml`
- `gitops/kustomize/codebase-browser/deployment.yaml`
- `gitops/kustomize/codebase-browser/service.yaml`
- `gitops/kustomize/codebase-browser/ingress.yaml`
- `gitops/kustomize/codebase-browser/kustomization.yaml`
- `gitops/applications/codebase-browser.yaml`

Implementation notes:

- mirror the `artifacts` app shape first
- use the hostname `codebase-browser.yolo.scapegoat.dev`
- keep `imagePullPolicy: IfNotPresent`
- use `enableServiceLinks: false`
- expose the pod on container port `8080`

### Phase 3: one-time cluster bootstrap

Run the first `kubectl apply` for the Argo `Application` manually, then force a hard refresh so Argo immediately starts tracking the new package.

```bash
cd /home/manuel/code/wesen/2026-03-27--hetzner-k3s
export KUBECONFIG=$PWD/kubeconfig-<server-ip>.yaml

kubectl apply -f gitops/applications/codebase-browser.yaml
kubectl -n argocd annotate application codebase-browser \
  argocd.argoproj.io/refresh=hard --overwrite
```

After that, the normal GitOps PR flow should own subsequent changes.

### Phase 4: public validation

Smoke-test the rollout after Argo syncs:

- `https://codebase-browser.yolo.scapegoat.dev/`
- `https://codebase-browser.yolo.scapegoat.dev/doc/03-meta`
- `GET /api/index`
- `GET /api/doc`

The page is only successful if the browser, the docs sidebar, and the live snippet resolution all work on the public endpoint.

## Testing and validation strategy

The validation strategy should be layered:

1. **Repo-level checks**
   - `go test ./...`
   - `pnpm -C ui run typecheck`
   - `pnpm -C tools/ts-indexer test`

2. **Build checks**
   - `make build`
   - confirm `bin/codebase-browser` exists
   - confirm the generated docs page still resolves after a clean build

3. **Container checks**
   - build the runtime image
   - run it locally on `:8080`
   - `curl -fsS http://127.0.0.1:8080/api/index`
   - `curl -fsS http://127.0.0.1:8080/api/doc`
   - `curl -fsS http://127.0.0.1:8080/doc/03-meta`

4. **GitOps checks**
   - `kubectl -n argocd get application codebase-browser`
   - verify `Synced` + `Healthy`
   - verify the deployment image matches the GitOps SHA tag
   - verify the ingress resolves over HTTPS

5. **User-visible checks**
   - load the root page
   - open the docs sidebar
   - open the example page
   - confirm the live snippet annotations render and the source links work

## Risks, alternatives, and open questions

- **Will the workflow need Dagger, or will local pnpm fallback be enough?**
  The TypeScript extractor can run locally when Dagger is unavailable, but the workflow still needs Node and pnpm installed. We should pin those versions explicitly in CI.

- **Should GHCR be explicitly public?**
  The deployment goal is public access, so the image should be pullable without cluster credentials. If GHCR visibility becomes a problem, the pull-secret path from HK3S-0014 is the fallback.

- **Should `/` stay as the normal browser home page or deep-link to `/doc/03-meta`?**
  Today the browser already exposes the example page through the docs list and `/doc/:slug`. A redirect would be a UX choice, not a deployment requirement.

- **Do we want one target only or a target array from day one?**
  The standard target metadata already supports multiple entries, but for this deployment the first target should stay singular.

- **What if the public deployment later needs auth or a second environment?**
  Then we can extend the target metadata or the GitOps package without changing the core release model.

## References

Primary current-repo evidence:

- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/README.md:3-12, 16-24, 26-49, 51-68, 81-117, 119-138`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/Makefile:31-45`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/cmd/codebase-browser/cmds/serve/run.go:58-94`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/server.go:25-43`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/spa.go:10-38`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/pages.go:17-44`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go:49-210`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/server/api_doc.go:11-41`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/app/App.tsx:14-36`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocPage.tsx:5-24`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/embed/pages/03-meta.md:1-68`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/indexfs/generate_build.go:1-35`
- `/home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/web/generate_build.go:1-80`

Hetzner K3s deployment references:

- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/source-app-deployment-infrastructure-playbook.md:29-61, 87-146, 168-246`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/public-repo-ghcr-argocd-deployment-playbook.md:28-37, 63-83, 125-141, 145-179, 202-257`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/artifacts/*.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/artifacts.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/pretext/*.yaml`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/pretext.yaml`

Contextual note from the vault:

- `/home/manuel/code/wesen/obsidian-vault/Projects/2026/03/29/PROJ - Serve Artifacts - Deploying to K3s with GitOps.md`
