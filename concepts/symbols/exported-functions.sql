/* codebase-browser concept
name: exported-functions
short: List exported functions and methods
long: |
  Shows exported functions and methods, optionally restricted by package import
  path substring. Useful for reviewing public API surface.
tags: [symbols, exported, api]
params:
  - name: package
    type: string
    help: Optional package import path substring
    default: ""
  - name: limit
    type: int
    help: Maximum number of rows
    default: 100
*/
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
  AND ({{ sqlString (value "package") }} = '' OR p.import_path LIKE {{ sqlLike (value "package") }})
ORDER BY p.import_path, s.name
LIMIT {{ value "limit" }};
