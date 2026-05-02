# Static sql.js runtime

The review browser is a static application. Exporting a review copies the Vite
bundle, writes `manifest.json`, and places the SQLite artifact at
`db/codebase.db`.

At runtime the browser loads `manifest.json`, opens `db/codebase.db` with
sql.js, and answers navigation, source, xref, history, impact, and review-doc
queries directly in the browser. There is no Go HTTP server in the exported
runtime.

Useful implementation entry points:

```codebase-signature sym=sym:github.com/go-go-golems/codebase-browser/internal/staticapp.func.Export
```

```codebase-signature sym=sym:github.com/go-go-golems/codebase-browser/internal/staticapp.func.AddRenderedReviewDocs
```

```codebase-signature sym=sym:ui/src/api/sqlJsQueryProvider.func.getSqlJsProvider
```
