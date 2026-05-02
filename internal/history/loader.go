package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wesen/codebase-browser/internal/gitutil"
	"github.com/wesen/codebase-browser/internal/indexer"
)

// LoadSnapshot bulk-loads a single commit's index into the history database.
// The worktreeDir parameter is the git worktree directory for the commit (used
// to compute body hashes). If worktreeDir is empty, body hashes are skipped.
func (s *Store) LoadSnapshot(ctx context.Context, commit gitutil.Commit, idx *indexer.Index, worktreeDir string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin load tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Insert commit metadata.
	now := time.Now().Unix()
	parentJSON, _ := json.Marshal(commit.ParentHashes)
	_, err = tx.ExecContext(ctx, `
INSERT INTO commits(hash, short_hash, message, author_name, author_email,
                    author_time, parent_hashes, tree_hash, indexed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		commit.Hash, commit.ShortHash, commit.Message,
		commit.AuthorName, commit.AuthorEmail,
		commit.AuthorTime.Unix(), string(parentJSON),
		commit.TreeHash, now,
	)
	if err != nil {
		return fmt.Errorf("insert commit %s: %w", commit.ShortHash, err)
	}

	if err := insertSnapshotPackages(ctx, tx, commit.Hash, idx.Packages); err != nil {
		return err
	}
	if err := insertSnapshotFiles(ctx, tx, commit.Hash, idx.Files); err != nil {
		return err
	}
	if err := insertSnapshotSymbols(ctx, tx, commit.Hash, idx.Symbols, worktreeDir); err != nil {
		return err
	}
	if err := insertSnapshotRefs(ctx, tx, commit.Hash, idx.Refs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit load tx: %w", err)
	}
	return nil
}

func insertSnapshotPackages(ctx context.Context, tx *sql.Tx, commitHash string, packages []indexer.Package) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO snapshot_packages(commit_hash, id, import_path, name, doc, language)
VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare snapshot package insert: %w", err)
	}
	defer stmt.Close()
	for _, p := range packages {
		lang := p.Language
		if lang == "" {
			lang = "go"
		}
		if _, err := stmt.ExecContext(ctx, commitHash, p.ID, p.ImportPath, p.Name, p.Doc, lang); err != nil {
			return fmt.Errorf("insert snapshot package %s: %w", p.ID, err)
		}
	}
	return nil
}

func insertSnapshotFiles(ctx context.Context, tx *sql.Tx, commitHash string, files []indexer.File) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO snapshot_files(commit_hash, id, path, package_id, size, line_count,
                           sha256, language, build_tags_json, content_hash)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '')`)
	if err != nil {
		return fmt.Errorf("prepare snapshot file insert: %w", err)
	}
	defer stmt.Close()
	for _, f := range files {
		lang := f.Language
		if lang == "" {
			lang = "go"
		}
		buildTags, _ := json.Marshal(f.BuildTags)
		if _, err := stmt.ExecContext(ctx,
			commitHash, f.ID, f.Path, f.PackageID,
			f.Size, f.LineCount, f.SHA256, lang, string(buildTags),
		); err != nil {
			return fmt.Errorf("insert snapshot file %s: %w", f.ID, err)
		}
	}
	return nil
}

func insertSnapshotSymbols(ctx context.Context, tx *sql.Tx, commitHash string, symbols []indexer.Symbol, worktreeDir string) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO snapshot_symbols(
    commit_hash, id, kind, name, package_id, file_id,
    start_line, start_col, end_line, end_col, start_offset, end_offset,
    doc, signature, receiver_type, receiver_pointer, exported, language,
    type_params_json, tags_json, body_hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare snapshot symbol insert: %w", err)
	}
	defer stmt.Close()

	seen := make(map[string]bool, len(symbols))
	for _, sym := range symbols {
		if seen[sym.ID] {
			continue
		}
		seen[sym.ID] = true

		lang := sym.Language
		if lang == "" {
			lang = "go"
		}
		receiverType := ""
		receiverPointer := 0
		if sym.Receiver != nil {
			receiverType = sym.Receiver.TypeName
			if sym.Receiver.Pointer {
				receiverPointer = 1
			}
		}
		typeParams, _ := json.Marshal(sym.TypeParams)
		tags, _ := json.Marshal(sym.Tags)

		// Compute body hash from file content if worktree is available.
		bodyHash := ""
		if worktreeDir != "" {
			bodyHash = computeBodyHash(worktreeDir, sym)
		}

		if _, err := stmt.ExecContext(ctx,
			commitHash, sym.ID, sym.Kind, sym.Name, sym.PackageID, sym.FileID,
			sym.Range.StartLine, sym.Range.StartCol, sym.Range.EndLine, sym.Range.EndCol,
			sym.Range.StartOffset, sym.Range.EndOffset,
			sym.Doc, sym.Signature, receiverType, receiverPointer,
			boolInt(sym.Exported), lang, string(typeParams), string(tags), bodyHash,
		); err != nil {
			return fmt.Errorf("insert snapshot symbol %s: %w", sym.ID, err)
		}
	}
	return nil
}

func insertSnapshotRefs(ctx context.Context, tx *sql.Tx, commitHash string, refs []indexer.Ref) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO snapshot_refs(
    commit_hash, id, from_symbol_id, to_symbol_id, kind, file_id,
    start_line, start_col, end_line, end_col, start_offset, end_offset
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare snapshot ref insert: %w", err)
	}
	defer stmt.Close()
	for i, ref := range refs {
		if _, err := stmt.ExecContext(ctx,
			commitHash, i,
			ref.FromSymbolID, ref.ToSymbolID, ref.Kind, ref.FileID,
			ref.Range.StartLine, ref.Range.StartCol, ref.Range.EndLine, ref.Range.EndCol,
			ref.Range.StartOffset, ref.Range.EndOffset,
		); err != nil {
			return fmt.Errorf("insert snapshot ref %d: %w", i, err)
		}
	}
	return nil
}

// computeBodyHash reads the file from the worktree and hashes the byte range
// for the symbol body. Returns empty string on any error (non-fatal).
func computeBodyHash(worktreeDir string, sym indexer.Symbol) string {
	// Extract relative path from file ID ("file:foo/bar.go" → "foo/bar.go").
	relPath := sym.FileID
	if len(relPath) > 5 && relPath[:5] == "file:" {
		relPath = relPath[5:]
	}
	absPath := filepath.Join(worktreeDir, relPath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	start := sym.Range.StartOffset
	end := sym.Range.EndOffset
	if start < 0 || end > len(data) || start > end {
		return ""
	}
	body := data[start:end]
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
