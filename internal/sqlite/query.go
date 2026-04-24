package sqlite

import (
	"context"
	"fmt"
	"strings"
)

// SymbolRow is the CLI-friendly relational representation of a symbol.
type SymbolRow struct {
	ID        string
	Kind      string
	Name      string
	PackageID string
	FileID    string
	StartLine int
	EndLine   int
	Signature string
	Doc       string
	Exported  bool
	Language  string
}

type symbolQuery struct {
	where []string
	args  []any
	limit int
}

// SymbolPredicate mutates a symbol query. Predicates compose in the order they
// are passed to FindSymbols.
type SymbolPredicate func(*symbolQuery)

// ByKind restricts symbols to one kind, for example "func" or "type".
func ByKind(kind string) SymbolPredicate {
	return func(q *symbolQuery) {
		if kind == "" {
			return
		}
		q.where = append(q.where, "kind = ?")
		q.args = append(q.args, kind)
	}
}

// ByPackage restricts symbols to one package ID or import path.
func ByPackage(packageIDOrImportPath string) SymbolPredicate {
	return func(q *symbolQuery) {
		if packageIDOrImportPath == "" {
			return
		}
		q.where = append(q.where, `package_id IN (
    SELECT id FROM packages WHERE id = ? OR import_path = ?
)`)
		q.args = append(q.args, packageIDOrImportPath, packageIDOrImportPath)
	}
}

// NameLike performs case-insensitive substring matching over symbol names.
func NameLike(name string) SymbolPredicate {
	return func(q *symbolQuery) {
		if name == "" {
			return
		}
		q.where = append(q.where, "lower(name) LIKE ?")
		q.args = append(q.args, "%"+strings.ToLower(name)+"%")
	}
}

// IsExported restricts results to exported symbols.
func IsExported() SymbolPredicate {
	return func(q *symbolQuery) {
		q.where = append(q.where, "exported = 1")
	}
}

// Limit caps the number of returned symbols.
func Limit(n int) SymbolPredicate {
	return func(q *symbolQuery) {
		if n > 0 {
			q.limit = n
		}
	}
}

// FindSymbols runs a composable symbol query against SQLite.
func (s *Store) FindSymbols(ctx context.Context, predicates ...SymbolPredicate) ([]SymbolRow, error) {
	q := &symbolQuery{limit: 200}
	for _, predicate := range predicates {
		predicate(q)
	}

	query := `
SELECT id, kind, name, package_id, file_id, start_line, end_line, signature, doc, exported, language
FROM symbols`
	if len(q.where) > 0 {
		query += "\nWHERE " + strings.Join(q.where, " AND ")
	}
	query += "\nORDER BY name COLLATE NOCASE, kind, id"
	if q.limit > 0 {
		query += "\nLIMIT ?"
		q.args = append(q.args, q.limit)
	}

	rows, err := s.db.QueryContext(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("query symbols: %w", err)
	}
	defer rows.Close()

	var out []SymbolRow
	for rows.Next() {
		var row SymbolRow
		var exported int
		if err := rows.Scan(
			&row.ID, &row.Kind, &row.Name, &row.PackageID, &row.FileID,
			&row.StartLine, &row.EndLine, &row.Signature, &row.Doc, &exported, &row.Language,
		); err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		row.Exported = exported != 0
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbols: %w", err)
	}
	return out, nil
}
