package history

const DropSchemaSQL = `
DROP VIEW IF EXISTS symbol_history;
DROP TABLE IF EXISTS snapshot_refs;
DROP TABLE IF EXISTS snapshot_symbols;
DROP TABLE IF EXISTS snapshot_files;
DROP TABLE IF EXISTS snapshot_packages;
DROP TABLE IF EXISTS file_contents;
DROP TABLE IF EXISTS commits;
`

const CreateSchemaSQL = `
CREATE TABLE commits (
    hash TEXT PRIMARY KEY,
    short_hash TEXT NOT NULL,
    message TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_time INTEGER NOT NULL,
    parent_hashes TEXT NOT NULL DEFAULT '[]',
    tree_hash TEXT NOT NULL DEFAULT '',
    indexed_at INTEGER NOT NULL DEFAULT 0,
    branch TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_commits_author_time ON commits(author_time);
CREATE INDEX idx_commits_branch ON commits(branch);

CREATE TABLE snapshot_packages (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id TEXT NOT NULL,
    import_path TEXT NOT NULL,
    name TEXT NOT NULL,
    doc TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'go',
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE snapshot_files (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id TEXT NOT NULL,
    path TEXT NOT NULL,
    package_id TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    line_count INTEGER NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'go',
    build_tags_json TEXT NOT NULL DEFAULT '[]',
    content_hash TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE snapshot_symbols (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id TEXT NOT NULL,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    package_id TEXT NOT NULL,
    file_id TEXT NOT NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_col INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_col INTEGER NOT NULL DEFAULT 0,
    start_offset INTEGER NOT NULL DEFAULT 0,
    end_offset INTEGER NOT NULL DEFAULT 0,
    doc TEXT NOT NULL DEFAULT '',
    signature TEXT NOT NULL DEFAULT '',
    receiver_type TEXT NOT NULL DEFAULT '',
    receiver_pointer INTEGER NOT NULL DEFAULT 0,
    exported INTEGER NOT NULL DEFAULT 0,
    language TEXT NOT NULL DEFAULT 'go',
    type_params_json TEXT NOT NULL DEFAULT '[]',
    tags_json TEXT NOT NULL DEFAULT '[]',
    body_hash TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE snapshot_refs (
    commit_hash TEXT NOT NULL REFERENCES commits(hash),
    id INTEGER NOT NULL,
    from_symbol_id TEXT NOT NULL,
    to_symbol_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    file_id TEXT NOT NULL,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_col INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_col INTEGER NOT NULL DEFAULT 0,
    start_offset INTEGER NOT NULL DEFAULT 0,
    end_offset INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (commit_hash, id)
);

CREATE TABLE file_contents (
    content_hash TEXT PRIMARY KEY,
    content BLOB NOT NULL
);

-- Indexes for common queries
CREATE INDEX idx_snap_pkg_commit ON snapshot_packages(commit_hash);
CREATE INDEX idx_snap_file_commit ON snapshot_files(commit_hash);
CREATE INDEX idx_snap_file_sha ON snapshot_files(sha256);
CREATE INDEX idx_snap_sym_commit ON snapshot_symbols(commit_hash);
CREATE INDEX idx_snap_sym_name ON snapshot_symbols(name);
CREATE INDEX idx_snap_sym_kind ON snapshot_symbols(kind);
CREATE INDEX idx_snap_sym_pkg ON snapshot_symbols(package_id);
CREATE INDEX idx_snap_sym_body ON snapshot_symbols(body_hash);
CREATE INDEX idx_snap_ref_commit ON snapshot_refs(commit_hash);
CREATE INDEX idx_snap_ref_from ON snapshot_refs(from_symbol_id, commit_hash);
CREATE INDEX idx_snap_ref_to ON snapshot_refs(to_symbol_id, commit_hash);
`

const CreateViewsSQL = `
CREATE VIEW IF NOT EXISTS symbol_history AS
SELECT
    s.id AS symbol_id,
    s.name,
    s.kind,
    s.package_id,
    c.hash AS commit_hash,
    c.short_hash,
    c.message AS commit_message,
    c.author_time,
    s.body_hash,
    s.start_line,
    s.end_line,
    s.signature,
    s.file_id
FROM snapshot_symbols s
JOIN commits c ON c.hash = s.commit_hash;
`
