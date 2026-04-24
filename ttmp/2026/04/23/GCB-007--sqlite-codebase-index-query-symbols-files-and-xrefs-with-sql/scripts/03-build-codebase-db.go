//go:build ignore

// 03-build-codebase-db.go reads the existing index.json and produces
// a SQLite database (codebase.db) with the GCB-007 schema.
//
// Run:  go run scripts/03-build-codebase-db.go
//       (produces scripts/codebase.db)
//
// Then serve the scripts/ directory and open 04-sqlite-browser-demo.html.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Load index.json
	idxPath := filepath.Join(root, "internal", "indexfs", "embed", "index.json")
	data, err := os.ReadFile(idxPath)
	if err != nil {
		log.Fatal("read index.json: ", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		log.Fatal("parse index.json: ", err)
	}
	fmt.Printf("Loaded index: %d packages, %d files, %d symbols, %d refs\n",
		len(idx.Packages), len(idx.Files), len(idx.Symbols), len(idx.Refs))

	// Create database
	outPath := filepath.Join(root, "scripts", "codebase.db")
	os.Remove(outPath)

	db, err := sql.Open("sqlite3", outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create schema
	if err := createSchema(db); err != nil {
		log.Fatal("schema: ", err)
	}

	// Insert data
	if err := loadData(db, &idx); err != nil {
		log.Fatal("load: ", err)
	}

	// Report
	var count int
	db.QueryRow("SELECT COUNT(*) FROM symbols").Scan(&count)
	fmt.Printf("Wrote %s (%d symbols)\n", outPath, count)
}

func createSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS packages (
			id          TEXT PRIMARY KEY,
			import_path TEXT NOT NULL,
			name        TEXT NOT NULL,
			doc         TEXT DEFAULT '',
			language    TEXT DEFAULT 'go'
		)`,
		`CREATE TABLE IF NOT EXISTS files (
			id          TEXT PRIMARY KEY,
			path        TEXT NOT NULL,
			package_id  TEXT NOT NULL REFERENCES packages(id),
			language    TEXT DEFAULT 'go',
			size        INTEGER NOT NULL DEFAULT 0,
			line_count  INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS symbols (
			id          TEXT PRIMARY KEY,
			kind        TEXT NOT NULL,
			name        TEXT NOT NULL,
			package_id  TEXT NOT NULL REFERENCES packages(id),
			file_id     TEXT NOT NULL REFERENCES files(id),
			start_line  INTEGER NOT NULL,
			end_line    INTEGER NOT NULL DEFAULT 0,
			signature   TEXT DEFAULT '',
			doc         TEXT DEFAULT '',
			receiver    TEXT DEFAULT NULL,
			exported    BOOLEAN NOT NULL DEFAULT FALSE,
			language    TEXT DEFAULT 'go'
		)`,
		`CREATE TABLE IF NOT EXISTS refs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			from_id     TEXT NOT NULL REFERENCES symbols(id),
			to_id       TEXT NOT NULL REFERENCES symbols(id),
			kind        TEXT NOT NULL DEFAULT '',
			file_id     TEXT NOT NULL REFERENCES files(id),
			start_line  INTEGER NOT NULL,
			end_line    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS index_meta (
			id           INTEGER PRIMARY KEY CHECK (id = 1),
			version      TEXT NOT NULL DEFAULT '1',
			generated_at TEXT NOT NULL,
			module       TEXT NOT NULL DEFAULT '',
			go_version   TEXT NOT NULL DEFAULT ''
		)`,
		// FTS5 virtual table
		`CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
			name, signature, doc, kind, import_path,
			content='symbols', content_rowid='rowid'
		)`,
		// Triggers to keep FTS in sync
		`CREATE TRIGGER IF NOT EXISTS symbols_fts_insert AFTER INSERT ON symbols BEGIN
			INSERT INTO symbols_fts(rowid, name, signature, doc, kind, import_path)
			SELECT new.rowid, new.name, new.signature, new.doc, new.kind, p.import_path
			FROM packages p WHERE p.id = new.package_id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS symbols_fts_delete AFTER DELETE ON symbols BEGIN
			INSERT INTO symbols_fts(symbols_fts, rowid) VALUES ('delete', old.rowid);
		END`,
		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_files_package_id ON files(package_id)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_package_id ON symbols(package_id)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_exported ON symbols(exported)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_from_id ON refs(from_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_to_id ON refs(to_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("%s: %w", s[:60], err)
		}
	}
	return nil
}

func loadData(db *sql.DB, idx *Index) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Meta
	_, err = tx.Exec(`INSERT INTO index_meta (id, version, generated_at, module, go_version)
		VALUES (1, ?, ?, ?, ?)`, idx.Version, idx.GeneratedAt, idx.Module, idx.GoVersion)
	if err != nil {
		return err
	}

	// Packages
	pkgStmt, err := tx.Prepare(`INSERT INTO packages (id, import_path, name, doc, language) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, p := range idx.Packages {
		lang := p.Language
		if lang == "" {
			lang = "go"
		}
		pkgStmt.Exec(p.ID, p.ImportPath, p.Name, p.Doc, lang)
	}
	pkgStmt.Close()

	// Files
	fileStmt, err := tx.Prepare(`INSERT INTO files (id, path, package_id, language, size, line_count) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, f := range idx.Files {
		lang := f.Language
		if lang == "" {
			lang = "go"
		}
		fileStmt.Exec(f.ID, f.Path, f.PackageID, lang, f.Size, f.LineCount)
	}
	fileStmt.Close()

	// Symbols
	symStmt, err := tx.Prepare(`INSERT INTO symbols (id, kind, name, package_id, file_id,
		start_line, end_line, signature, doc, receiver, exported, language) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	var insertSymbols func(syms []Symbol) error
	insertSymbols = func(syms []Symbol) error {
		for _, s := range syms {
			recv := ""
			if s.Receiver != nil {
				recv = s.Receiver.TypeName
			}
			lang := s.Language
			if lang == "" {
				lang = "go"
			}
			symStmt.Exec(s.ID, s.Kind, s.Name, s.PackageID, s.FileID,
				s.Range.StartLine, s.Range.EndLine, s.Signature, s.Doc, recv, s.Exported, lang)
			if len(s.Children) > 0 {
				if err := insertSymbols(s.Children); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := insertSymbols(idx.Symbols); err != nil {
		return err
	}
	symStmt.Close()

	// Refs
	refStmt, err := tx.Prepare(`INSERT INTO refs (from_id, to_id, kind, file_id, start_line, end_line) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, r := range idx.Refs {
		refStmt.Exec(r.FromSymbolID, r.ToSymbolID, r.Kind, r.FileID, r.Range.StartLine, r.Range.EndLine)
	}
	refStmt.Close()

	return tx.Commit()
}

// --- Types matching internal/indexer/types.go ---

type Index struct {
	Version     string   `json:"version"`
	GeneratedAt string   `json:"generatedAt"`
	Module      string   `json:"module"`
	GoVersion   string   `json:"goVersion"`
	Packages    []Package `json:"packages"`
	Files       []File    `json:"files"`
	Symbols     []Symbol  `json:"symbols"`
	Refs        []Ref     `json:"refs,omitempty"`
}

type Package struct {
	ID         string   `json:"id"`
	ImportPath string   `json:"importPath"`
	Name       string   `json:"name"`
	Doc        string   `json:"doc,omitempty"`
	Language   string   `json:"language,omitempty"`
}

type File struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	PackageID string `json:"packageId"`
	Size      int    `json:"size"`
	LineCount int    `json:"lineCount"`
	Language  string `json:"language,omitempty"`
}

type Range struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

type Symbol struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	PackageID string    `json:"packageId"`
	FileID    string    `json:"fileId"`
	Range     Range     `json:"range"`
	Doc       string    `json:"doc,omitempty"`
	Signature string    `json:"signature,omitempty"`
	Receiver  *Receiver `json:"receiver,omitempty"`
	Exported  bool      `json:"exported"`
	Children  []Symbol  `json:"children,omitempty"`
	Language  string    `json:"language,omitempty"`
}

type Receiver struct {
	TypeName string `json:"typeName"`
}

type Ref struct {
	FromSymbolID string `json:"fromSymbolId"`
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	FileID       string `json:"fileId"`
	Range        Range  `json:"range"`
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found from %s", dir)
}
