-- List exported functions and methods with their package and file.
SELECT
  s.name,
  s.kind,
  p.import_path AS package,
  f.path AS file,
  s.start_line
FROM symbols s
JOIN packages p ON p.id = s.package_id
JOIN files f ON f.id = s.file_id
WHERE s.exported = 1
  AND s.kind IN ('func', 'method')
ORDER BY p.import_path, s.name
LIMIT 100;
