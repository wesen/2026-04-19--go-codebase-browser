/* codebase-browser concept
name: symbol-changes
short: Symbols that changed between two commits
params:
  - name: base
    type: string
    help: Base commit hash
  - name: head
    type: string
    help: Head commit hash
  - name: change_type
    type: string
    default: ""
    help: "Filter: added, removed, modified, moved (empty = all)"
tags: [history, diff, symbols]
*/
SELECT
    COALESCE(old_sym.name, new_sym.name) AS name,
    COALESCE(old_sym.kind, new_sym.kind) AS kind,
    COALESCE(old_sym.package_id, new_sym.package_id) AS package_id,
    COALESCE(old_sym.id, new_sym.id) AS symbol_id,
    CASE
        WHEN old_sym.id IS NULL THEN 'added'
        WHEN new_sym.id IS NULL THEN 'removed'
        WHEN old_sym.body_hash != new_sym.body_hash AND old_sym.body_hash != '' AND new_sym.body_hash != '' THEN 'modified'
        WHEN old_sym.start_line != new_sym.start_line OR old_sym.end_line != new_sym.end_line THEN 'moved'
        ELSE 'unchanged'
    END AS change_type,
    COALESCE(old_sym.start_line, 0) AS old_start_line,
    COALESCE(new_sym.start_line, 0) AS new_start_line
FROM (SELECT * FROM snapshot_symbols WHERE commit_hash = {{sqlString .base}}) old_sym
FULL OUTER JOIN (SELECT * FROM snapshot_symbols WHERE commit_hash = {{sqlString .head}}) new_sym
     ON old_sym.id = new_sym.id
WHERE old_sym.id IS NULL
   OR new_sym.id IS NULL
   OR old_sym.body_hash != new_sym.body_hash
   OR old_sym.start_line != new_sym.start_line
ORDER BY change_type, name;
