package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/wesen/codebase-browser/internal/indexer"
)

// LoadFromIndex bulk-loads an in-memory index into the SQLite schema.
func (s *Store) LoadFromIndex(ctx context.Context, idx *indexer.Index) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin load tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertMeta(ctx, tx, idx); err != nil {
		return err
	}
	if err := insertPackages(ctx, tx, idx.Packages); err != nil {
		return err
	}
	if err := insertFiles(ctx, tx, idx.Files); err != nil {
		return err
	}
	if err := insertSymbols(ctx, tx, idx.Symbols); err != nil {
		return err
	}
	if err := insertRefs(ctx, tx, idx.Refs); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit load tx: %w", err)
	}
	return nil
}

func insertMeta(ctx context.Context, tx *sql.Tx, idx *indexer.Index) error {
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO meta(key, value) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare meta insert: %w", err)
	}
	defer stmt.Close()
	pairs := map[string]string{
		"version":      idx.Version,
		"generated_at": idx.GeneratedAt,
		"module":       idx.Module,
		"go_version":   idx.GoVersion,
	}
	for k, v := range pairs {
		if _, err := stmt.ExecContext(ctx, k, v); err != nil {
			return fmt.Errorf("insert meta %s: %w", k, err)
		}
	}
	return nil
}

func insertPackages(ctx context.Context, tx *sql.Tx, packages []indexer.Package) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO packages(id, import_path, name, doc, language)
VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare package insert: %w", err)
	}
	defer stmt.Close()
	for _, p := range packages {
		if _, err := stmt.ExecContext(ctx, p.ID, p.ImportPath, p.Name, p.Doc, languageOrGo(p.Language)); err != nil {
			return fmt.Errorf("insert package %s: %w", p.ID, err)
		}
	}
	return nil
}

func insertFiles(ctx context.Context, tx *sql.Tx, files []indexer.File) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO files(id, path, package_id, size, line_count, sha256, language, build_tags_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare file insert: %w", err)
	}
	defer stmt.Close()
	for _, f := range files {
		buildTags, err := jsonString(f.BuildTags)
		if err != nil {
			return fmt.Errorf("encode build tags for file %s: %w", f.ID, err)
		}
		if _, err := stmt.ExecContext(ctx, f.ID, f.Path, f.PackageID, f.Size, f.LineCount, f.SHA256, languageOrGo(f.Language), buildTags); err != nil {
			return fmt.Errorf("insert file %s: %w", f.ID, err)
		}
	}
	return nil
}

func insertSymbols(ctx context.Context, tx *sql.Tx, symbols []indexer.Symbol) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO symbols(
    id, kind, name, package_id, file_id,
    start_line, start_col, end_line, end_col, start_offset, end_offset,
    doc, signature, receiver_type, receiver_pointer, exported, language,
    type_params_json, tags_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare symbol insert: %w", err)
	}
	defer stmt.Close()
	for _, sym := range symbols {
		typeParams, err := jsonString(sym.TypeParams)
		if err != nil {
			return fmt.Errorf("encode type params for symbol %s: %w", sym.ID, err)
		}
		tags, err := jsonString(sym.Tags)
		if err != nil {
			return fmt.Errorf("encode tags for symbol %s: %w", sym.ID, err)
		}
		receiverType := ""
		receiverPointer := false
		if sym.Receiver != nil {
			receiverType = sym.Receiver.TypeName
			receiverPointer = sym.Receiver.Pointer
		}
		if _, err := stmt.ExecContext(ctx,
			sym.ID, sym.Kind, sym.Name, sym.PackageID, sym.FileID,
			sym.Range.StartLine, sym.Range.StartCol, sym.Range.EndLine, sym.Range.EndCol, sym.Range.StartOffset, sym.Range.EndOffset,
			sym.Doc, sym.Signature, receiverType, boolInt(receiverPointer), boolInt(sym.Exported), languageOrGo(sym.Language),
			typeParams, tags,
		); err != nil {
			return fmt.Errorf("insert symbol %s: %w", sym.ID, err)
		}
	}
	return nil
}

func insertRefs(ctx context.Context, tx *sql.Tx, refs []indexer.Ref) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO refs(
    from_symbol_id, to_symbol_id, kind, file_id,
    start_line, start_col, end_line, end_col, start_offset, end_offset
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare ref insert: %w", err)
	}
	defer stmt.Close()
	for _, ref := range refs {
		if _, err := stmt.ExecContext(ctx,
			ref.FromSymbolID, ref.ToSymbolID, ref.Kind, ref.FileID,
			ref.Range.StartLine, ref.Range.StartCol, ref.Range.EndLine, ref.Range.EndCol, ref.Range.StartOffset, ref.Range.EndOffset,
		); err != nil {
			return fmt.Errorf("insert ref %s -> %s: %w", ref.FromSymbolID, ref.ToSymbolID, err)
		}
	}
	return nil
}

func jsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func languageOrGo(language string) string {
	if language == "" {
		return "go"
	}
	return language
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
