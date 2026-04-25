/* codebase-browser concept
name: hotspots
short: Most frequently changed symbols (by body hash)
params:
  - name: limit
    type: int
    default: 20
    help: Max results
  - name: min_versions
    type: int
    default: 2
    help: Minimum distinct body versions to be considered a hotspot
tags: [history, analysis]
*/
SELECT
    s.id AS symbol_id,
    s.name,
    s.kind,
    s.package_id,
    COUNT(DISTINCT s.body_hash) AS distinct_versions,
    COUNT(DISTINCT c.hash) AS commit_count,
    MIN(c.author_time) AS first_seen,
    MAX(c.author_time) AS last_changed
FROM snapshot_symbols s
JOIN commits c ON c.hash = s.commit_hash
WHERE s.body_hash != ''
GROUP BY s.id
HAVING COUNT(DISTINCT s.body_hash) >= {{.min_versions}}
ORDER BY distinct_versions DESC, commit_count DESC
LIMIT {{.limit}};
