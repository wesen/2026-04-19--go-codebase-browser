# API surface

The HTTP server mounts `/api/*` routes before the SPA handler so static
assets never shadow an API call.

## /api/snippet

Slicing is O(1): the handler looks up the symbol's byte range in the
index and returns exactly those bytes from the embedded source FS. No
re-parsing, no tokenisation.

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/server.Server.handleSnippet
```

Path hygiene is mandatory for `/api/source` because the `path` query
parameter is author-controlled. Whitelisting against the index's files
table is the primary sandbox; `safePath` is a belt-and-braces second
check:

```codebase-snippet sym=github.com/wesen/codebase-browser/internal/server.safePath
```
