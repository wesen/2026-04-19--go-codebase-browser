-- Count files and symbols per package.
SELECT
  p.import_path,
  p.language,
  COUNT(DISTINCT f.id) AS files,
  COUNT(DISTINCT s.id) AS symbols
FROM packages p
LEFT JOIN files f ON f.package_id = p.id
LEFT JOIN symbols s ON s.package_id = p.id
GROUP BY p.id, p.import_path, p.language
ORDER BY symbols DESC, files DESC, p.import_path;
