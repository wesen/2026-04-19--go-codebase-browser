# Slice 3 Demo: Inline impact analysis

This page demonstrates `codebase-impact`, an inline caller/callee widget backed
by `/api/history/impact`. It walks `snapshot_refs` in the history database and
groups related symbols by graph depth.

## Who uses `writeJSON`?

`writeJSON` is a small server helper with many direct callers. This is a useful
example of a `usedby` impact query.

```codebase-impact sym=sym:github.com/wesen/codebase-browser/internal/server.func.writeJSON dir=usedby depth=2
```

## What does `handleSnippet` use?

`handleSnippet` fans out to several helpers and lookup methods. This shows the
opposite direction (`uses`).

```codebase-impact sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet dir=uses depth=2
```

The first implementation is intentionally a compact list rather than a graph.
Depth 1 means direct edges; depth 2 means one more hop away from the root.
