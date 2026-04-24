# Changelog

## 2026-04-24

- Created GCB-008 ticket workspace for structured SQLite query concepts.
- Step 2: Added `internal/concepts` SQL concept catalog package with parsing, catalog loading, typed value hydration, template rendering, and tests. Code commit: `7cb3b381a8169be97df80865be9eca99296d51bc`.
- Step 3: Added the initial structured SQL concept files under `concepts/` and a catalog loading test. Code commit: `17272ea3f0b4d2fdf59bc6f7ef3f5495d548269d`.
- Step 4: Added dynamic `codebase-browser query commands ...` concept CLI verbs with typed flags, `--render-only`, SQLite execution, and preserved raw SQL behavior. Code commit: `620002d8cf26ac7a52f0cc37e968915c4c3513c6`.
