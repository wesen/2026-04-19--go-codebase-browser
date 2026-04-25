/* codebase-browser concept
name: pr-summary
short: Summarize symbol changes between two commits
params:
  - name: base
    type: string
    help: Base commit hash (full or short)
  - name: head
    type: string
    help: Head commit hash (full or short)
tags: [history, diff, pr]
*/
WITH old AS (
    SELECT id, body_hash, signature, name, kind, package_id
    FROM snapshot_symbols WHERE commit_hash = {{sqlString .base}}
),
new AS (
    SELECT id, body_hash, signature, name, kind, package_id
    FROM snapshot_symbols WHERE commit_hash = {{sqlString .head}}
)
SELECT
    COALESCE(old.name, new.name) AS name,
    COALESCE(old.kind, new.kind) AS kind,
    COALESCE(old.package_id, new.package_id) AS package_id,
    CASE
        WHEN old.id IS NULL THEN 'added'
        WHEN new.id IS NULL THEN 'removed'
        WHEN old.body_hash != new.body_hash AND old.body_hash != '' AND new.body_hash != '' THEN 'modified'
        WHEN old.signature != new.signature THEN 'signature-changed'
        ELSE 'unchanged'
    END AS change_type
FROM old
LEFT JOIN new ON old.id = new.id
UNION ALL
SELECT
    new.name,
    new.kind,
    new.package_id,
    'added' AS change_type
FROM new
WHERE new.id NOT IN (SELECT id FROM old)
ORDER BY change_type, name;
