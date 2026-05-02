# Symbol History and Impact Analysis

This example demonstrates symbol history and impact analysis widgets.

## Symbol history

Track how the `AddRenderedReviewDocs` function evolved:

```codebase-symbol-history sym=staticapp.AddRenderedReviewDocs limit=5
```

## Impact: who calls AddRenderedReviewDocs?

```codebase-impact sym=staticapp.AddRenderedReviewDocs dir=usedby depth=1
```

## Diff stats across recent commits

```codebase-diff-stats from=HEAD~5 to=HEAD
```

## Changed files in recent range

```codebase-changed-files from=HEAD~5 to=HEAD
```

## Notes

- Symbol history is per-commit; the DB stores a snapshot per commit.
- Impact analysis traces references from the `snapshot_refs` table.
