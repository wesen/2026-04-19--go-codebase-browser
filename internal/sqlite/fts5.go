//go:build sqlite_fts5

package sqlite

import (
	"context"
	"fmt"
)

const createFTS5SQL = `
CREATE VIRTUAL TABLE IF NOT EXISTS symbol_fts USING fts5(
    name,
    kind,
    package_id,
    signature,
    doc,
    content='symbols',
    content_rowid='rowid'
);

INSERT INTO symbol_fts(rowid, name, kind, package_id, signature, doc)
SELECT rowid, name, kind, package_id, signature, doc FROM symbols;
`

// EnableFTS5 creates and populates the optional FTS5 search table.
func (s *Store) EnableFTS5(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, createFTS5SQL); err != nil {
		return fmt.Errorf("enable sqlite fts5: %w", err)
	}
	return nil
}
