-- Template: show outgoing refs for a symbol ID.
-- Replace the value in the WHERE clause with the symbol ID you care about.
SELECT
  source.name AS source,
  r.kind,
  target.name AS target,
  target.kind AS target_kind,
  f.path AS file,
  r.start_line
FROM refs r
JOIN symbols source ON source.id = r.from_symbol_id
JOIN symbols target ON target.id = r.to_symbol_id
JOIN files f ON f.id = r.file_id
WHERE source.id = 'sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleIndex'
ORDER BY f.path, r.start_line;
