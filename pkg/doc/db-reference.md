---
Title: "Review Database Reference"
Slug: "db-reference"
Short: "Schema, tables, and query patterns for the code-review SQLite database."
Topics:
- code-review
- sqlite
- reference
Commands:
- review index
- review export
- review db create
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

## Overview

The `codebase-browser review` commands produce a SQLite database containing both indexed commit history and review markdown documents.

Two separate DB paths matter:

| DB | Produced by | Use |
|----|-------------|-----|
| **Source DB** | `review index` or `review db create` | Query with `sqlite3`, hand to an LLM, or use as input to `review export` |
| **Export DB** (`db/codebase.db`) | `review export` (copies and enriches the source DB) | The static browser opens this file with sql.js. Contains `static_review_rendered_docs` and rendered HTML. |

`review export` copies the source DB to `db/codebase.db` in the output directory, then writes `static_review_rendered_docs` rows into the output DB. The source DB is never modified.

## Database structure

The review database is a standard SQLite file with two groups of tables:

1. **History tables** — per-commit snapshots of the codebase (from `internal/history/schema.go`)
2. **Review tables** — markdown documents and their resolved snippet references (from `internal/review/schema.go`)
3. **Static export tables** — export-time browser preparation tables written only to the copied export DB

## History tables

> ⚠️ **Byte offsets, not UTF-8 character offsets.** Source bodies are sliced using `start_offset` and `end_offset` from `snapshot_symbols`, which are byte offsets into `file_contents.content` **before** UTF-8 decoding. JavaScript string indexing uses UTF-16 code units. If you extract bytes in JavaScript and then decode as UTF-8, the character positions will not match the offsets in this schema. Always decode the bytes to a string before indexing by character position.

### `commits`

One row per indexed commit.

| Column | Type | Description |
|--------|------|-------------|
| `hash` | TEXT PK | Full 40-character SHA |
| `short_hash` | TEXT | 7-character abbreviation |
| `message` | TEXT | Commit message |
| `author_name` | TEXT | Author name |
| `author_email` | TEXT | Author email |
| `author_time` | INTEGER | Unix timestamp |
| `parent_hashes` | TEXT | JSON array of parent SHAs |
| `tree_hash` | TEXT | Git tree hash |
| `indexed_at` | INTEGER | When the row was inserted |
| `branch` | TEXT | Branch name (if supplied) |
| `error` | TEXT | Empty unless indexing failed |

### `snapshot_packages`

One row per package per commit.

| Column | Type | Description |
|--------|------|-------------|
| `commit_hash` | TEXT FK | References `commits(hash)` |
| `id` | TEXT | `pkg:<importPath>` |
| `import_path` | TEXT | Go/TS import path |
| `name` | TEXT | Package name |
| `doc` | TEXT | Package comment |
| `language` | TEXT | `"go"` or `"ts"` |

### `snapshot_files`

One row per file per commit.

| Column | Type | Description |
|--------|------|-------------|
| `commit_hash` | TEXT FK | References `commits(hash)` |
| `id` | TEXT | `file:<path>` |
| `path` | TEXT | Relative path |
| `package_id` | TEXT | References `snapshot_packages(id)` |
| `size` | INTEGER | File size in bytes |
| `line_count` | INTEGER | Number of lines |
| `sha256` | TEXT | File content hash |
| `language` | TEXT | `"go"` or `"ts"` |
| `build_tags_json` | TEXT | JSON array of build tags |
| `content_hash` | TEXT | References `file_contents` |

### `snapshot_symbols`

One row per symbol per commit. A "symbol" is any top-level declaration: function, method, type, const, var.

| Column | Type | Description |
|--------|------|-------------|
| `commit_hash` | TEXT FK | References `commits(hash)` |
| `id` | TEXT | `sym:<importPath>.<kind>.<name>` |
| `kind` | TEXT | `func`, `method`, `type`, `var`, `const` |
| `name` | TEXT | Symbol name |
| `package_id` | TEXT | Package ID |
| `file_id` | TEXT | File ID |
| `start_line` / `end_line` | INTEGER | Line range |
| `start_col` / `end_col` | INTEGER | Column range |
| `start_offset` / `end_offset` | INTEGER | **Byte offsets** (authoritative for slicing) |
| `doc` | TEXT | Godoc / TSDoc |
| `signature` | TEXT | e.g. `func Merge(...) (*Index, error)` |
| `receiver_type` | TEXT | For methods: receiver type name |
| `receiver_pointer` | INTEGER | 1 if receiver is a pointer |
| `exported` | INTEGER | 1 if name starts with uppercase |
| `language` | TEXT | `"go"` or `"ts"` |
| `type_params_json` | TEXT | JSON array of type parameters |
| `tags_json` | TEXT | JSON array of struct tags |
| `body_hash` | TEXT | SHA-256 of function body bytes |

### `snapshot_refs`

One row per cross-reference per commit.

| Column | Type | Description |
|--------|------|-------------|
| `commit_hash` | TEXT FK | References `commits(hash)` |
| `id` | INTEGER | Auto-increment within commit |
| `from_symbol_id` | TEXT | Caller |
| `to_symbol_id` | TEXT | Callee |
| `kind` | TEXT | `call`, `uses-type`, `reads`, `use` |
| `file_id` | TEXT | Where the reference occurs |
| `start_line` / `end_line` | INTEGER | Line range |
| `start_col` / `end_col` | INTEGER | Column range |
| `start_offset` / `end_offset` | INTEGER | Byte offsets |

### `file_contents`

Deduplicated file content blobs.

| Column | Type | Description |
|--------|------|-------------|
| `content_hash` | TEXT PK | SHA-256 of content |
| `content` | BLOB | Raw file bytes |

### `symbol_history` (view)

A convenience view joining `snapshot_symbols` with `commits`.

## Review tables

### `review_docs`

One row per markdown review document.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `slug` | TEXT UNIQUE | Derived from filename (`pr-42.md` → `pr-42`) |
| `title` | TEXT | From H1 or frontmatter |
| `path` | TEXT | Original file path |
| `content` | TEXT | Raw markdown |
| `frontmatter_json` | TEXT | JSON object of YAML frontmatter |
| `indexed_at` | INTEGER | Unix timestamp |

### `review_doc_snippets`

One row per resolved `codebase-*` directive in a review doc.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `doc_id` | INTEGER FK | References `review_docs(id)` |
| `stub_id` | TEXT | e.g. `stub-1` |
| `directive` | TEXT | `codebase-snippet`, `codebase-diff`, etc. |
| `symbol_id` | TEXT | Resolved symbol ID |
| `file_path` | TEXT | Source file path |
| `kind` | TEXT | `func`, `declaration`, `diff`, etc. |
| `language` | TEXT | `"go"`, `"ts"`, `"text"` |
| `text` | TEXT | Pre-resolved snippet text |
| `params_json` | TEXT | Directive parameters |
| `start_line` / `end_line` | INTEGER | Line range |
| `commit_hash` | TEXT | If `commit=` was specified |

### `static_review_rendered_docs`

One row per rendered review document in the exported browser database (`db/codebase.db`). This table is populated by `review export`, not by `review index`.

| Column | Type | Description |
|--------|------|-------------|
| `slug` | TEXT PK | Review document slug |
| `title` | TEXT | Rendered document title |
| `html` | TEXT | Export-time rendered HTML with widget placeholders |
| `snippets_json` | TEXT | JSON array of resolved snippet/widget metadata |
| `errors_json` | TEXT | JSON array of render errors; `[]` when clean |
| `rendered_at` | INTEGER | Unix timestamp when export rendered the document |

## Common SQL queries

### List all indexed commits

```sql
SELECT short_hash, message, author_name, datetime(author_time, 'unixepoch') AS date
FROM commits
ORDER BY author_time DESC;
```

### Find symbols whose signatures changed between the first and last commit

```sql
SELECT
    s1.name,
    s1.signature AS old_sig,
    s2.signature AS new_sig,
    c1.short_hash AS old_commit,
    c2.short_hash AS new_commit
FROM snapshot_symbols s1
JOIN snapshot_symbols s2 ON s1.id = s2.id
JOIN commits c1 ON c1.hash = s1.commit_hash
JOIN commits c2 ON c2.hash = s2.commit_hash
WHERE c1.author_time = (SELECT MIN(author_time) FROM commits)
  AND c2.author_time = (SELECT MAX(author_time) FROM commits)
  AND s1.signature != s2.signature;
```

### Count symbols per commit

```sql
SELECT c.short_hash, COUNT(s.id) AS symbol_count
FROM commits c
LEFT JOIN snapshot_symbols s ON s.commit_hash = c.hash
GROUP BY c.hash
ORDER BY c.author_time DESC;
```

### Find all callers of a specific symbol

```sql
SELECT
    r.from_symbol_id,
    s.name AS caller_name,
    s.signature AS caller_sig,
    f.path
FROM snapshot_refs r
JOIN snapshot_symbols s ON s.id = r.from_symbol_id AND s.commit_hash = r.commit_hash
JOIN snapshot_files f ON f.id = s.file_id AND f.commit_hash = s.commit_hash
WHERE r.to_symbol_id = 'sym:github.com/foo/bar.func.Target'
  AND r.commit_hash = (SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1);
```

### List review documents and their snippet counts

```sql
SELECT d.slug, d.title, COUNT(s.id) AS snippet_count
FROM review_docs d
LEFT JOIN review_doc_snippets s ON s.doc_id = d.id
GROUP BY d.id;
```

## Symbol ID scheme

Symbol IDs are stable across file moves. The format is:

```
sym:<importPath>.<kind>.<name>              # top-level declaration
sym:<importPath>.method.<Recv>.<name>       # method
```

Examples:
- `sym:github.com/go-go-golems/codebase-browser/internal/indexer.func.Extract`
- `sym:github.com/go-go-golems/codebase-browser/internal/indexer.method.Store.LoadSnapshot`

Short refs (used in markdown directives) are resolved by matching the last segment against unambiguous symbols in the given package.

## Commit range syntax

The `--commits` flag accepts any git log range specification:

| Example | Meaning |
|---------|---------|
| `HEAD~10..HEAD` | Last 10 commits |
| `main..feature` | Commits on `feature` not on `main` |
| `abc123..def456` | Commits between two SHAs |
| `HEAD` | Just the current commit |
| `--all` | All reachable commits |

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `UNIQUE constraint failed: snapshot_symbols` | Duplicate symbol IDs (e.g. blank identifiers) | Fixed in history loader — first occurrence wins |
| `no commits in review database` | `LoadLatestSnapshot` called on empty DB | Run `review index` or `review db create` first |
| `render doc: symbol not found` | Doc references a symbol not in indexed commits | Ensure the commit range includes the symbol |
| Large `.db` file | File contents duplicated across commits | `file_contents` deduplicates by SHA-256; this is expected for large ranges |

## See Also

- `user-guide` — Tutorial for writing review markdown files
- `markdown-block-reference` — Canonical reference for every `codebase-*` directive
