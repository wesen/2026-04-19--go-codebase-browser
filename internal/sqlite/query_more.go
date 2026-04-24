package sqlite

import (
	"context"
	"fmt"
	"strings"
)

// PackageRow is a CLI-friendly row from packages.
type PackageRow struct {
	ID         string
	ImportPath string
	Name       string
	Doc        string
	Language   string
}

type packageQuery struct {
	where []string
	args  []any
	limit int
}

// PackagePredicate mutates a package query.
type PackagePredicate func(*packageQuery)

func PackageImportPathLike(value string) PackagePredicate {
	return func(q *packageQuery) {
		if value == "" {
			return
		}
		q.where = append(q.where, "lower(import_path) LIKE ?")
		q.args = append(q.args, "%"+strings.ToLower(value)+"%")
	}
}

func PackageNameLike(value string) PackagePredicate {
	return func(q *packageQuery) {
		if value == "" {
			return
		}
		q.where = append(q.where, "lower(name) LIKE ?")
		q.args = append(q.args, "%"+strings.ToLower(value)+"%")
	}
}

func PackageLanguage(language string) PackagePredicate {
	return func(q *packageQuery) {
		if language == "" {
			return
		}
		q.where = append(q.where, "language = ?")
		q.args = append(q.args, language)
	}
}

func PackageLimit(n int) PackagePredicate {
	return func(q *packageQuery) {
		if n > 0 {
			q.limit = n
		}
	}
}

func (s *Store) FindPackages(ctx context.Context, predicates ...PackagePredicate) ([]PackageRow, error) {
	q := &packageQuery{limit: 200}
	for _, predicate := range predicates {
		predicate(q)
	}
	query := `SELECT id, import_path, name, doc, language FROM packages`
	if len(q.where) > 0 {
		query += "\nWHERE " + strings.Join(q.where, " AND ")
	}
	query += "\nORDER BY import_path COLLATE NOCASE, id"
	if q.limit > 0 {
		query += "\nLIMIT ?"
		q.args = append(q.args, q.limit)
	}
	rows, err := s.db.QueryContext(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("query packages: %w", err)
	}
	defer rows.Close()
	var out []PackageRow
	for rows.Next() {
		var row PackageRow
		if err := rows.Scan(&row.ID, &row.ImportPath, &row.Name, &row.Doc, &row.Language); err != nil {
			return nil, fmt.Errorf("scan package: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate packages: %w", err)
	}
	return out, nil
}

// FileRow is a CLI-friendly row from files.
type FileRow struct {
	ID            string
	Path          string
	PackageID     string
	Size          int
	LineCount     int
	SHA256        string
	Language      string
	BuildTagsJSON string
}

type fileQuery struct {
	where []string
	args  []any
	limit int
}

// FilePredicate mutates a file query.
type FilePredicate func(*fileQuery)

func FilePathLike(value string) FilePredicate {
	return func(q *fileQuery) {
		if value == "" {
			return
		}
		q.where = append(q.where, "lower(path) LIKE ?")
		q.args = append(q.args, "%"+strings.ToLower(value)+"%")
	}
}

func FilePackage(packageID string) FilePredicate {
	return func(q *fileQuery) {
		if packageID == "" {
			return
		}
		q.where = append(q.where, "package_id = ?")
		q.args = append(q.args, packageID)
	}
}

func FileLanguage(language string) FilePredicate {
	return func(q *fileQuery) {
		if language == "" {
			return
		}
		q.where = append(q.where, "language = ?")
		q.args = append(q.args, language)
	}
}

func FileLimit(n int) FilePredicate {
	return func(q *fileQuery) {
		if n > 0 {
			q.limit = n
		}
	}
}

func (s *Store) FindFiles(ctx context.Context, predicates ...FilePredicate) ([]FileRow, error) {
	q := &fileQuery{limit: 200}
	for _, predicate := range predicates {
		predicate(q)
	}
	query := `SELECT id, path, package_id, size, line_count, sha256, language, build_tags_json FROM files`
	if len(q.where) > 0 {
		query += "\nWHERE " + strings.Join(q.where, " AND ")
	}
	query += "\nORDER BY path COLLATE NOCASE, id"
	if q.limit > 0 {
		query += "\nLIMIT ?"
		q.args = append(q.args, q.limit)
	}
	rows, err := s.db.QueryContext(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()
	var out []FileRow
	for rows.Next() {
		var row FileRow
		if err := rows.Scan(&row.ID, &row.Path, &row.PackageID, &row.Size, &row.LineCount, &row.SHA256, &row.Language, &row.BuildTagsJSON); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}
	return out, nil
}

// RefRow is a CLI-friendly row from refs.
type RefRow struct {
	ID           int64
	FromSymbolID string
	ToSymbolID   string
	Kind         string
	FileID       string
	StartLine    int
	EndLine      int
}

type refQuery struct {
	where []string
	args  []any
	limit int
}

// RefPredicate mutates a ref query.
type RefPredicate func(*refQuery)

func RefFrom(symbolID string) RefPredicate {
	return func(q *refQuery) {
		if symbolID == "" {
			return
		}
		q.where = append(q.where, "from_symbol_id = ?")
		q.args = append(q.args, symbolID)
	}
}

func RefTo(symbolID string) RefPredicate {
	return func(q *refQuery) {
		if symbolID == "" {
			return
		}
		q.where = append(q.where, "to_symbol_id = ?")
		q.args = append(q.args, symbolID)
	}
}

func RefKind(kind string) RefPredicate {
	return func(q *refQuery) {
		if kind == "" {
			return
		}
		q.where = append(q.where, "kind = ?")
		q.args = append(q.args, kind)
	}
}

func RefFile(fileID string) RefPredicate {
	return func(q *refQuery) {
		if fileID == "" {
			return
		}
		q.where = append(q.where, "file_id = ?")
		q.args = append(q.args, fileID)
	}
}

func RefLimit(n int) RefPredicate {
	return func(q *refQuery) {
		if n > 0 {
			q.limit = n
		}
	}
}

func (s *Store) FindRefs(ctx context.Context, predicates ...RefPredicate) ([]RefRow, error) {
	q := &refQuery{limit: 200}
	for _, predicate := range predicates {
		predicate(q)
	}
	query := `SELECT id, from_symbol_id, to_symbol_id, kind, file_id, start_line, end_line FROM refs`
	if len(q.where) > 0 {
		query += "\nWHERE " + strings.Join(q.where, " AND ")
	}
	query += "\nORDER BY file_id, start_line, id"
	if q.limit > 0 {
		query += "\nLIMIT ?"
		q.args = append(q.args, q.limit)
	}
	rows, err := s.db.QueryContext(ctx, query, q.args...)
	if err != nil {
		return nil, fmt.Errorf("query refs: %w", err)
	}
	defer rows.Close()
	var out []RefRow
	for rows.Next() {
		var row RefRow
		if err := rows.Scan(&row.ID, &row.FromSymbolID, &row.ToSymbolID, &row.Kind, &row.FileID, &row.StartLine, &row.EndLine); err != nil {
			return nil, fmt.Errorf("scan ref: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate refs: %w", err)
	}
	return out, nil
}
