# Changelog

## 2026-04-20

- Initial workspace created


## 2026-04-20

Created ticket + design-doc + diary. Seeded vocabulary (typescript/dagger/node-tooling/multi-language). Validated approach with a working tsx prototype in scripts/ that extracts 1 package / 2 files / 7 symbols from a fixture TS module matching the Go Index schema (byte offsets + SHA256 + sorted deterministic output). Documented a sandbox-specific pnpm EROFS workaround (symlink to ui/node_modules). Ran docmgr doctor: passes.


## 2026-04-20

pnpm store rw access restored; removed the symlink + tsc-compile workaround from scripts/ and reran prototype via the normal 'pnpm install && pnpm run extract' path (1 package / 2 files / 7 symbols). Added packageManager pin and scripts/README.md. Committed scripts/pnpm-lock.yaml for reviewer determinism. Diary Step 4 documents the cleanup; Step 2 keeps the historical context.


## 2026-04-20

Phase 1 landed: Extractor interface in internal/indexer/multi.go (Language, Extract methods), GoExtractor wrapping existing Extract(), Merge([]*Index) with duplicate-id detection + stable sort. Added Language field on Package/File/Symbol (omitempty, empty-means-go). stampLanguage helper keeps each extractor from having to thread the field through every constructor. Tests cover merge, dup detection, nil handling, and language stamping. Real build stamps 'go' on 15 pkg / 30 files / 121 symbols.


## 2026-04-20

Phase 2 landed: tools/ts-indexer/ promoted from scripts prototype. src/extract.ts + src/cli.ts + src/ids.ts + src/types.ts (schema mirror). Vitest fixture test (6 tests, all green) covers language stamping, symbol counts, method receiver IDs, byte-offset roundtrip, and determinism. Compiled bin/cli.js runs via 'node bin/cli.js --module-root <path>'.


## 2026-04-20

Phase 3 landed: cmd/build-ts-index Dagger program with BUILD_TS_LOCAL=1 fallback. Local smoke on ui/ frontend: 12 packages / 38 files / 145 symbols (func/const/iface/alias) extracted from real TypeScript code. Dagger path mounts tools/ts-indexer + ui/ narrowly with CacheVolume for the pnpm store, corepack-activated pnpm, frozen-lockfile install, then runs the compiled bin/cli.js. Local path skips Dagger entirely via 'node tools/ts-indexer/bin/cli.js ...'.


## 2026-04-20

Phase 4 landed: --lang go|ts|auto on 'index build'. Shells out to cmd/build-ts-index for ts/auto; Merge() combines Go + TS indexes. Output table reports go/ts symbol counts. Discovered + fixed a real TS ID collision: 'const meta' appeared in multiple story files in the same directory. Fix: scope TS symbol IDs to the file path (sym:<module>/<rel-file-stem>.<kind>.<name>) so intra-package cross-file collisions are impossible. Go IDs unchanged. --lang auto on this repo: 28 pkg / 69 files / 278 symbols (133 Go + 145 TS).

