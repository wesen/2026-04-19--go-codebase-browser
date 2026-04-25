# Slice 2 Demo: Inline symbol history

This page demonstrates `codebase-symbol-history`, a compact history timeline
for a single symbol. The rows are clickable: if the selected commit changed the
symbol body and an indexed predecessor exists, the widget expands an inline diff.

## `stubHTML` history

`stubHTML` changed while Slice 0 added support for `data-commit` and Slice 1
added `data-params` support.

```codebase-symbol-history sym=sym:github.com/wesen/codebase-browser/internal/docs.func.stubHTML limit=8
```

## `handleSnippet` history

`handleSnippet` changed when Slice 0 introduced commit-aware snippet fetching.

```codebase-symbol-history sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet limit=8
```

The compact timeline uses `body_hash` from the history database. A filled dot
means the symbol body changed compared with the previous indexed commit.
