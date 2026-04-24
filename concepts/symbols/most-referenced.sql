/* codebase-browser concept
name: most-referenced
short: Show symbols with the most incoming references
long: |
  Counts incoming refs per local target symbol. Optionally restrict by symbol
  kind to find central functions, methods, types, or variables.
tags: [symbols, refs, centrality]
params:
  - name: kind
    type: string
    help: Optional symbol kind filter
    default: ""
  - name: limit
    type: int
    help: Maximum number of rows
    default: 50
*/
SELECT
  target.name,
  target.kind,
  p.import_path AS package,
  COUNT(*) AS incoming_refs
FROM refs r
JOIN symbols target ON target.id = r.to_symbol_id
JOIN packages p ON p.id = target.package_id
WHERE ({{ sqlString (value "kind") }} = '' OR target.kind = {{ sqlString (value "kind") }})
GROUP BY target.id, target.name, target.kind, p.import_path
ORDER BY incoming_refs DESC, target.name
LIMIT {{ value "limit" }};
