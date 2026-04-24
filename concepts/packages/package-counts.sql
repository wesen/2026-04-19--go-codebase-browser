/* codebase-browser concept
name: package-counts
short: Count files and symbols per package
long: |
  Summarizes package size by counting distinct files and symbols. Optionally
  restricts rows to one language.
tags: [packages, overview]
params:
  - name: language
    type: choice
    help: Optional language filter
    default: ""
    choices: ["", "go", "ts"]
*/
SELECT
  p.import_path,
  p.language,
  COUNT(DISTINCT f.id) AS files,
  COUNT(DISTINCT s.id) AS symbols
FROM packages p
LEFT JOIN files f ON f.package_id = p.id
LEFT JOIN symbols s ON s.package_id = p.id
WHERE ({{ sqlString (value "language") }} = '' OR p.language = {{ sqlString (value "language") }})
GROUP BY p.id, p.import_path, p.language
ORDER BY symbols DESC, files DESC, p.import_path;
