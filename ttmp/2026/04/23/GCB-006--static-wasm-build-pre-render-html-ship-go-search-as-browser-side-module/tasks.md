# Tasks

## TODO

- [ ] Add tasks here

- [ ] Phase 1: Create internal/wasm/ package with search.go, exports.go, main.go, generate_build.go. Compile to search.wasm with TinyGo. Verify WASM loads in browser.
- [ ] Phase 5: Integration tests with Playwright. Verify file:// protocol works (no server). Update README. Deprecate serve command with migration notes.
- [ ] Phase 3: Create ui/src/api/wasmClient.ts. Swap RTK-Query baseQuery from HTTP fetch to WASM function calls. Test search, symbol lookup, xref navigation without server.
- [ ] Phase 2: Create internal/static/ package. Implement buildSearchIndex(), buildXrefIndex(), extractSnippets(), renderDocPages(). Write precomputed.json to internal/static/embed/
- [x] Phase 4: Create internal/bundle/ package. Bundle search.wasm, search.js, precomputed.json, source/, index.html into dist/ artifact. Update Makefile build-static target.
- [ ] Phase 1b: Write generate_build.go for internal/wasm/ that compiles to search.wasm and generates JS glue.
- [ ] Phase 1a: Create internal/wasm/ package with index types, search context, and WASM exports. Build with TinyGo.
- [ ] Phase 5: Playwright integration test: open static artifact over file://, verify search, navigation, xref. Update README.
- [ ] Phase 4: Create internal/bundle/ package that generates dist/ artifact (index.html + wasm + precomputed.json + source/).
- [ ] Phase 2: Create internal/static/ package for build-time pre-computation (search index, xref, snippets, doc HTML).
