---
Title: "Writing Code Review Guides"
Slug: "review-user-guide"
Short: "How to write markdown review guides and serve them with codebase-browser review."
Topics:
- code-review
- markdown
- tutorial
Commands:
- review index
- review serve
- review db create
Flags:
- commits
- docs
- db
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

## Quick start

Write a markdown file with embedded code widgets, then index and serve it:

```bash
# 1. Create a review guide
cat > ./reviews/pr-42.md << 'EOF'
# PR #42: Add strict mode to Extract

## Motivation
The `Extract` function needs to support build tag filtering.

## Changes

### 1. New parameter
```codebase-diff sym=indexer.Extract from=HEAD~1 to=HEAD
```

### 2. Updated callers
```codebase-impact sym=indexer.Extract dir=usedby depth=2
```
EOF

# 2. Index commits and docs into a review database
codebase-browser review index \
  --commits HEAD~5..HEAD \
  --docs ./reviews/pr-42.md \
  --db ./reviews/pr-42.db

# 3. Serve it
codebase-browser review serve --db ./reviews/pr-42.db --addr :3002

# 4. Open http://localhost:3002 in a browser
```

## Writing review markdown files

Review guides are regular markdown files with special fenced code blocks that the renderer replaces with interactive widgets.

### Available directives

| Directive | Purpose | Example |
|-----------|---------|---------|
| `codebase-snippet` | Full symbol body | `` ```codebase-snippet sym=indexer.Extract``` `` |
| `codebase-signature` | Just the signature | `` ```codebase-signature sym=indexer.Extract``` `` |
| `codebase-doc` | Godoc/TSDoc comment | `` ```codebase-doc sym=indexer.Extract``` `` |
| `codebase-file` | Whole or partial file | `` ```codebase-file path=internal/server/server.go range=28-44``` `` |
| `codebase-diff` | Symbol body diff between commits | `` ```codebase-diff sym=indexer.Extract from=HEAD~1 to=HEAD``` `` |
| `codebase-symbol-history` | Timeline of commits touching a symbol | `` ```codebase-symbol-history sym=indexer.Merge limit=8``` `` |
| `codebase-impact` | Transitive caller/callee list | `` ```codebase-impact sym=indexer.Extract dir=usedby depth=2``` `` |
| `codebase-commit-walk` | Guided narrative through commits | `` ```codebase-commit-walk from=HEAD~4 to=HEAD``` `` |
| `codebase-annotation` | Inline highlights and notes | `` ```codebase-annotation sym=indexer.Extract commit=HEAD``` `` |
| `codebase-changed-files` | File-level diff summary | `` ```codebase-changed-files from=main to=HEAD``` `` |
| `codebase-diff-stats` | Compact numeric summary | `` ```codebase-diff-stats from=main to=HEAD``` `` |

### Symbol references

Use full `sym:` IDs or short forms:

```markdown
Full:  sym:github.com/wesen/codebase-browser/internal/indexer.func.Extract
Short: indexer.Extract                    # unambiguous
Short: indexer.Store.LoadSnapshot         # method: pkg.Recv.Method
```

Short forms fail if ambiguous (two symbols with the same name in the same package).

### Commit parameters

Most directives accept an optional `commit=` parameter to show the symbol at a specific commit:

```markdown
Before this PR:
```codebase-snippet sym=indexer.Extract commit=HEAD~3
```

After this PR:
```codebase-snippet sym=indexer.Extract
```
```

When `commit=` is present, the renderer queries the history database for that commit's snapshot instead of the latest index.

## Commit range syntax

The `--commits` flag accepts any git log range:

| Example | Meaning |
|---------|---------|
| `HEAD~10..HEAD` | Last 10 commits |
| `main..feature` | Commits on `feature` not on `main` |
| `abc123..def456` | Between two SHAs |
| `HEAD` | Just the current commit |
| `--all` | All reachable commits |

For PR reviews, `HEAD~N..HEAD` is usually what you want, where `N` is the number of commits in the PR.

## Sharing review databases

A review database (`.db` file) is a single SQLite file. You can:

- **Email it:** Attach `pr-42.db` to an email. The recipient runs `codebase-browser review serve --db pr-42.db`.
- **Store it in CI:** Generate `review.db` in CI and upload it as an artifact.
- **Query it with an LLM:** Give the `.db` file to an LLM with instructions to run SQL against it. The schema is documented in `review-db-reference`.

## Querying the DB with an LLM

After running `review db create --commits HEAD~10..HEAD --db review.db`, you have a queryable SQLite file. Here are example prompts for an LLM:

**Prompt:** "Which functions had signature changes in this PR?"

```sql
SELECT s1.name, s1.signature AS old, s2.signature AS new
FROM snapshot_symbols s1
JOIN snapshot_symbols s2 ON s1.id = s2.id
WHERE s1.commit_hash = (SELECT hash FROM commits ORDER BY author_time ASC LIMIT 1)
  AND s2.commit_hash = (SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1)
  AND s1.signature != s2.signature;
```

**Prompt:** "Which symbols were added in this PR?"

```sql
SELECT s.name, s.kind, s.signature
FROM snapshot_symbols s
WHERE s.commit_hash = (SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1)
  AND s.id NOT IN (
    SELECT id FROM snapshot_symbols
    WHERE commit_hash = (SELECT hash FROM commits ORDER BY author_time ASC LIMIT 1)
  );
```

**Prompt:** "Show me the impact graph for `indexer.Extract` — who calls it?"

```sql
SELECT r.from_symbol_id, s.name, s.signature, r.kind
FROM snapshot_refs r
JOIN snapshot_symbols s ON s.id = r.from_symbol_id
  AND s.commit_hash = r.commit_hash
WHERE r.to_symbol_id = 'sym:github.com/wesen/codebase-browser/internal/indexer.func.Extract'
  AND r.commit_hash = (SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1);
```

## Workflow tips

### Iterative review writing

1. Write the markdown guide with placeholder text.
2. Run `review index --commits RANGE --docs ./reviews/ --db review.db`.
3. Run `review serve --db review.db` and open the browser.
4. Edit the markdown, re-run `review index` (it overwrites existing docs).
5. Refresh the browser to see changes.

### Team reviews

For team review meetings:

1. The PR author writes the review guide.
2. They run `review index` and share the `.db` file.
3. Reviewers run `review serve --db shared.db` locally.
4. Everyone sees the same interactive widgets with no network dependency.

### Large commit ranges

For large PRs (50+ commits), the review DB can grow large because each commit stores a full snapshot. Strategies:

- Use `--worktrees` for accurate per-commit extraction (slower but correct).
- Use a smaller range: `HEAD~20..HEAD` instead of `main..feature`.
- The `file_contents` table deduplicates identical files across commits, so content bloat is bounded.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Symbol not found in rendered doc | Commit range doesn't include the symbol | Widen `--commits` range |
| Widget shows "doc error" | Ambiguous short ref or missing symbol | Use full `sym:` ID |
| Serve shows blank page | No docs in DB | Run `review index` with `--docs` |
| Diff widget shows no changes | `from` and `to` commits have same `body_hash` | Check commit range |
| Large `.db` file | Many commits indexed | Use narrower range or delete old `.db` |

## See Also

- `review-db-reference` — Complete schema reference and SQL query patterns
- `codebase-browser help history` — History subsystem documentation
- GCB-010 design doc — Embeddable widget catalog and wireframes
