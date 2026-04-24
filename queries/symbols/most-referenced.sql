-- Show symbols with the most incoming references.
SELECT
  target.name,
  target.kind,
  p.import_path AS package,
  COUNT(*) AS incoming_refs
FROM refs r
JOIN symbols target ON target.id = r.to_symbol_id
JOIN packages p ON p.id = target.package_id
GROUP BY target.id, target.name, target.kind, p.import_path
ORDER BY incoming_refs DESC, target.name
LIMIT 50;
