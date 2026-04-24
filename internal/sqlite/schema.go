package sqlite

import (
	"context"
	"fmt"
)

const dropSchemaSQL = `
DROP TABLE IF EXISTS symbol_fts;
DROP TABLE IF EXISTS refs;
DROP TABLE IF EXISTS symbols;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS meta;
`

const createSchemaSQL = `
CREATE TABLE meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE packages (
    id TEXT PRIMARY KEY,
    import_path TEXT NOT NULL,
    name TEXT NOT NULL,
    doc TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'go'
);

CREATE TABLE files (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    package_id TEXT NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    size INTEGER NOT NULL DEFAULT 0,
    line_count INTEGER NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'go',
    build_tags_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE symbols (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    package_id TEXT NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
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
    tags_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE refs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_symbol_id TEXT NOT NULL,
    to_symbol_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL DEFAULT 0,
    start_col INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    end_col INTEGER NOT NULL DEFAULT 0,
    start_offset INTEGER NOT NULL DEFAULT 0,
    end_offset INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_packages_import_path ON packages(import_path);
CREATE INDEX idx_files_package_id ON files(package_id);
CREATE INDEX idx_files_path ON files(path);
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_kind ON symbols(kind);
CREATE INDEX idx_symbols_package_id ON symbols(package_id);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);
CREATE INDEX idx_symbols_exported ON symbols(exported);
CREATE INDEX idx_refs_from_symbol_id ON refs(from_symbol_id);
CREATE INDEX idx_refs_to_symbol_id ON refs(to_symbol_id);
CREATE INDEX idx_refs_file_id ON refs(file_id);
`

// ResetSchema drops and recreates the canonical codebase-browser SQLite schema.
func (s *Store) ResetSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, dropSchemaSQL); err != nil {
		return fmt.Errorf("drop sqlite schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("create sqlite schema: %w", err)
	}
	return nil
}
