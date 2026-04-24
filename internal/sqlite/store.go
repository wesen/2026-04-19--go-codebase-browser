package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Store owns a SQLite connection for the codebase index.
//
// The SQLite database is the canonical Go-side index representation for GCB-007.
// Callers should create or open a Store and then query its DB directly or through
// the helpers in query.go.
type Store struct {
	db *sql.DB
}

// Open opens an existing SQLite database file.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := configure(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Create opens path, drops any existing codebase-browser tables, and recreates
// the schema. It is intended for deterministic build-time generation.
func Create(path string) (*Store, error) {
	store, err := Open(path)
	if err != nil {
		return nil, err
	}
	if err := store.ResetSchema(context.Background()); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// DB exposes the underlying database for the CLI query command.
func (s *Store) DB() *sql.DB { return s.db }

// Close closes the underlying database connection.
func (s *Store) Close() error { return s.db.Close() }

func configure(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	return nil
}
