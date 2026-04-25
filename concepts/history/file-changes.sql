/* codebase-browser concept
name: file-changes
short: Files that changed between two commits
params:
  - name: base
    type: string
    help: Base commit hash
  - name: head
    type: string
    help: Head commit hash
tags: [history, diff, files]
*/
SELECT
    COALESCE(old_f.path, new_f.path) AS path,
    CASE
        WHEN old_f.id IS NULL THEN 'added'
        WHEN new_f.id IS NULL THEN 'removed'
        WHEN old_f.sha256 != new_f.sha256 THEN 'modified'
        ELSE 'unchanged'
    END AS change_type,
    COALESCE(old_f.line_count, 0) AS old_lines,
    COALESCE(new_f.line_count, 0) AS new_lines,
    COALESCE(new_f.line_count, 0) - COALESCE(old_f.line_count, 0) AS line_delta
FROM (SELECT * FROM snapshot_files WHERE commit_hash = {{sqlString .base}}) old_f
FULL OUTER JOIN (SELECT * FROM snapshot_files WHERE commit_hash = {{sqlString .head}}) new_f
     ON old_f.id = new_f.id
WHERE old_f.id IS NULL
   OR new_f.id IS NULL
   OR old_f.sha256 != new_f.sha256
ORDER BY change_type, path;
