# Changelog

## 2026-04-20

- Initial workspace created


## 2026-04-20

Step 2: server emits data-codebase-snippet stubs (commit 5d40b01)

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer.go — stubHTML helper + preprocess integration
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/internal/docs/renderer_test.go — TestRender_EmitsHydrationStubs coverage


## 2026-04-20

Step 3: frontend hydrates stubs with LinkedCode/Link/blockquote (commit 87af108)

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocPage.tsx — useEffect + createPortal hydration
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/DocSnippet.tsx — new component dispatching per directive

