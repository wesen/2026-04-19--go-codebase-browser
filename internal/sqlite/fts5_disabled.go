//go:build !sqlite_fts5

package sqlite

import "context"

// EnableFTS5 is a no-op unless the sqlite_fts5 build tag is enabled.
func (s *Store) EnableFTS5(_ context.Context) error { return nil }
