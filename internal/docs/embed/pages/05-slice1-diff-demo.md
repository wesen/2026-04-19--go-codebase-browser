# Slice 1 Demo: Inline semantic diff

This page demonstrates the first new history-backed widget: `codebase-diff`.
It renders a symbol body diff inline in markdown using the history database.

## `stubHTML` diff

The `stubHTML` function changed when Slice 0 added support for carrying the
`commit=` parameter through rendered markdown stubs.

```codebase-diff sym=sym:github.com/wesen/codebase-browser/internal/docs.func.stubHTML from=c9132579687ca9b334ff81f9161ba058bffc52c4 to=e457069b75f87ddc0a07a58d5608e96334a7fcf0
```

## `handleSnippet` diff

The `handleSnippet` method changed to route commit-aware snippet requests to
the history database.

```codebase-diff sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet from=c9132579687ca9b334ff81f9161ba058bffc52c4 to=e457069b75f87ddc0a07a58d5608e96334a7fcf0
```

This widget is intentionally simple for Slice 1: it uses the existing
`/api/history/symbol-body-diff` endpoint and renders a unified diff with
red/green highlights. Later slices can add side-by-side layout, xref links,
and richer per-line controls.
