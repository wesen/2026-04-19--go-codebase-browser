# Tasks

## Open

- [ ] Validate the public rollout after Argo CD syncs the new Application

## Completed

- [x] Gather evidence from the app repo, the Hetzner K3s repo, and the Obsidian vault
- [x] Create the GCB-003 ticket workspace
- [x] Write the design document
- [x] Write the investigation diary
- [x] Relate the most important source files to the docs
- [x] Upload the GCB-003 design doc and diary bundle to reMarkable and verify the remote listing

## Notes

- The current ticket documents focus on the deployment contract and rollout plan, not on the code changes themselves.
- The public example page should use the existing `/doc/03-meta` route rather than introducing a separate page system.
- [x] Validate the local build and deployment contract, then update the diary and changelog
- [x] Add deployment target metadata and GitOps PR helper in the app repo
- [x] Add app repo release packaging: Dockerfile, .dockerignore, and a minimal runtime image for codebase-browser
- [x] Add the matching GitOps package and Argo CD Application in the Hetzner K3s repo
- [x] Add GitHub Actions workflow to test, build, and publish immutable GHCR images
