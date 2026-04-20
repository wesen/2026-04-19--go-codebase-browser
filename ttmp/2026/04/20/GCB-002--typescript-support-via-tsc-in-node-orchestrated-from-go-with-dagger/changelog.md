# Changelog

## 2026-04-20

- Initial workspace created


## 2026-04-20

Created ticket + design-doc + diary. Seeded vocabulary (typescript/dagger/node-tooling/multi-language). Validated approach with a working tsx prototype in scripts/ that extracts 1 package / 2 files / 7 symbols from a fixture TS module matching the Go Index schema (byte offsets + SHA256 + sorted deterministic output). Documented a sandbox-specific pnpm EROFS workaround (symlink to ui/node_modules). Ran docmgr doctor: passes.


## 2026-04-20

pnpm store rw access restored; removed the symlink + tsc-compile workaround from scripts/ and reran prototype via the normal 'pnpm install && pnpm run extract' path (1 package / 2 files / 7 symbols). Added packageManager pin and scripts/README.md. Committed scripts/pnpm-lock.yaml for reviewer determinism. Diary Step 4 documents the cleanup; Step 2 keeps the historical context.

