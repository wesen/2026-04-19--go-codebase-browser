# Changelog

## 2026-04-23

- Initial workspace created


## 2026-04-23

Created design document for SQLite codebase index. 47KB doc covering schema, Go package design, browser-side sql.js, CLI query command, build pipeline changes, migration strategy.

## 2026-04-24

- Step 1: Added the first real `internal/sqlite/` store package with schema creation, index loading, predicate symbol queries, optional FTS5 setup, `go generate` plumbing, and integration tests. Code commit: `d70eaee166094303d93c5a7a076be9aae9a9b4c3`.
- Updated `tasks.md` to remove the backwards-compatibility migration phase and reflect SQLite as the sole Go-side index path.

