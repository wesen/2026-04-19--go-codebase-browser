/* codebase-browser concept
name: refs-for-symbol
short: Show references for one symbol
long: |
  Shows incoming, outgoing, or both directions of references for a symbol ID.
  This is the main low-level navigation query for symbol dependency analysis.
tags: [refs, symbols, navigation]
params:
  - name: symbol-id
    type: string
    help: Symbol ID to inspect
    required: true
  - name: direction
    type: choice
    help: Which reference direction to show
    default: both
    choices: [incoming, outgoing, both]
  - name: limit
    type: int
    help: Maximum number of rows
    default: 100
*/
SELECT
  CASE
    WHEN r.from_symbol_id = {{ sqlString (value "symbol-id") }} THEN 'outgoing'
    ELSE 'incoming'
  END AS direction,
  r.kind,
  source.name AS source,
  r.from_symbol_id,
  target.name AS target,
  r.to_symbol_id,
  f.path AS file,
  r.start_line
FROM refs r
LEFT JOIN symbols source ON source.id = r.from_symbol_id
LEFT JOIN symbols target ON target.id = r.to_symbol_id
JOIN files f ON f.id = r.file_id
WHERE (
    ({{ sqlString (value "direction") }} IN ('outgoing', 'both') AND r.from_symbol_id = {{ sqlString (value "symbol-id") }})
    OR
    ({{ sqlString (value "direction") }} IN ('incoming', 'both') AND r.to_symbol_id = {{ sqlString (value "symbol-id") }})
)
ORDER BY direction, f.path, r.start_line
LIMIT {{ value "limit" }};
