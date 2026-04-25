# Slice 0 Demo: Snapshot at a commit

This page demonstrates the `commit=` parameter on existing `codebase-snippet`
and `codebase-signature` directives (GCB-010 Slice 0). It shows the same
symbol rendered at two different commits, so a reviewer can see what changed
without leaving the doc page.

## `stubHTML` — before and after

The `stubHTML` function in `internal/docs/renderer.go` was modified in this
PR to support the `data-commit` attribute. Here it is at the old commit:

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/docs.func.stubHTML commit=c913257
```

And here it is at the latest commit (HEAD), where it gained the `data-commit`
attribute support:

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/docs.func.stubHTML commit=e457069
```

## `handleSnippet` — before and after

The server-side snippet handler was also modified. Here's the signature at
the old commit:

```codebase-signature sym=github.com/wesen/codebase-browser/internal/server.func.handleSnippet commit=c913257
```

And the signature at HEAD:

```codebase-signature sym=github.com/wesen/codebase-browser/internal/server.func.handleSnippet commit=e457069
```

The full function body at HEAD (showing the new `commit=` branch):

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/server.func.handleSnippet commit=e457069
```

## `resolveDirective` — at the old commit

For comparison, here's `resolveDirective` before the commit= changes:

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/docs.func.resolveDirective commit=c913257
```

## How this works

When the author writes `commit=<hash>`, the server emits a `data-commit`
attribute on the stub `<div>`. The React frontend reads this attribute and
fetches the snippet from `/api/snippet?sym=...&commit=<hash>` instead of
from the static index. The server resolves the symbol from the per-commit
snapshot in the history SQLite database.

No new directives were needed — just an optional parameter on existing ones.
This is the foundation for the diff, history, and impact widgets coming in
subsequent slices.
