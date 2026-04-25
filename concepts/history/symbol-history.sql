/* codebase-browser concept
name: symbol-history
short: History of a symbol across all indexed commits
params:
  - name: symbol_id
    type: string
    help: Full symbol ID
  - name: limit
    type: int
    default: 30
    help: Max results
tags: [history, symbols]
*/
SELECT
    c.short_hash,
    datetime(c.author_time, 'unixepoch') AS author_date,
    c.message,
    s.body_hash,
    s.start_line,
    s.end_line,
    s.signature
FROM   snapshot_symbols s
JOIN   commits c ON c.hash = s.commit_hash
WHERE  s.id = {{sqlString .symbol_id}}
ORDER BY c.author_time DESC
LIMIT  {{.limit}};
