# Changelog

## 2026-04-23

- Initial workspace created


## 2026-04-23

Created design document for SQLite codebase index. 47KB doc covering schema, Go package design, browser-side sql.js, CLI query command, build pipeline changes, migration strategy.

## 2026-04-24

- Step 1: Added the first real `internal/sqlite/` store package with schema creation, index loading, predicate symbol queries, optional FTS5 setup, `go generate` plumbing, and integration tests. Code commit: `d70eaee166094303d93c5a7a076be9aae9a9b4c3`.
- Updated `tasks.md` to remove the backwards-compatibility migration phase and reflect SQLite as the sole Go-side index path.
- Step 2: Added `codebase-browser query`, reusable `.sql` files, generated/smoke-tested `codebase.db`, and adjusted refs to allow external symbol IDs. Code commit: `dc5718614ccfe97b1213317ff73eef930756dc66`.
- Step 3: Added build-tagged FTS5 verification for `EnableFTS5` and `MATCH` queries. Code commit: `2c5e7525e06ea91a6c78d9254b4d86dc9b355f83`.
- Step 4: Completed package, file, and ref query helper APIs. Code commit: `71995295cab28970b521b4222ae0d7c3f823ea3a`.

