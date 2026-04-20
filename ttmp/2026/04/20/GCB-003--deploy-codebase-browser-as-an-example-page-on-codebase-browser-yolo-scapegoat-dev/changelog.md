# Changelog

## 2026-04-20

- Initial workspace created
- Added the deployment analysis, investigation diary, and evidence-backed references for the codebase-browser public deployment plan
- Added ticket-level related files for the main source and infrastructure references
- Uploaded the design doc + diary bundle to reMarkable at `/ai/2026/04/20/GCB-003` and verified the remote listing
- Implemented the app-side release packaging and build plumbing: Dockerfile, GHCR workflow, GitOps PR helper, Dagger-based web build, and source snapshot generator
- Validated the local build path with `make build`, `go test ./...`, and an embedded-server smoke test against `/api/index`
