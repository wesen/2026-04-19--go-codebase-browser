package history

import (
	"context"
	"database/sql"
	"fmt"
)

// ChangeType describes how an entity changed between two commits.
type ChangeType string

const (
	ChangeAdded            ChangeType = "added"
	ChangeRemoved          ChangeType = "removed"
	ChangeModified         ChangeType = "modified"
	ChangeSignatureChanged ChangeType = "signature-changed"
	ChangeMoved            ChangeType = "moved"
	ChangeUnchanged        ChangeType = "unchanged"
)

// FileDiff describes how a single file changed between two commits.
type FileDiff struct {
	FileID     string
	Path       string
	ChangeType ChangeType
	OldSHA256  string
	NewSHA256  string
}

// SymbolDiff describes how a single symbol changed between two commits.
type SymbolDiff struct {
	SymbolID     string
	Name         string
	Kind         string
	PackageID    string
	ChangeType   ChangeType
	OldStartLine int
	OldEndLine   int
	NewStartLine int
	NewEndLine   int
	OldSignature string
	NewSignature string
	OldBodyHash  string
	NewBodyHash  string
}

// DiffStats counts the different types of changes.
type DiffStats struct {
	FilesAdded      int
	FilesRemoved    int
	FilesModified   int
	SymbolsAdded    int
	SymbolsRemoved  int
	SymbolsModified int
	SymbolsMoved    int
	SymbolsUnchanged int
}

// CommitDiff is the complete diff between two commits.
type CommitDiff struct {
	OldHash string
	NewHash string
	Files   []FileDiff
	Symbols []SymbolDiff
	Stats   DiffStats
}

// DiffCommits compares two commits and returns the complete diff.
func (s *Store) DiffCommits(ctx context.Context, oldHash, newHash string) (*CommitDiff, error) {
	diff := &CommitDiff{
		OldHash: oldHash,
		NewHash: newHash,
	}

	if err := diffFiles(ctx, s.db, diff); err != nil {
		return nil, fmt.Errorf("diff files: %w", err)
	}
	if err := diffSymbols(ctx, s.db, diff); err != nil {
		return nil, fmt.Errorf("diff symbols: %w", err)
	}

	return diff, nil
}

func diffFiles(ctx context.Context, db *sql.DB, diff *CommitDiff) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    COALESCE(a.id, b.id) AS file_id,
    COALESCE(a.path, b.path) AS path,
    CASE
        WHEN a.id IS NULL THEN 'added'
        WHEN b.id IS NULL THEN 'removed'
        WHEN a.sha256 != b.sha256 THEN 'modified'
        ELSE 'unchanged'
    END AS change_type,
    COALESCE(a.sha256, '') AS old_sha,
    COALESCE(b.sha256, '') AS new_sha
FROM snapshot_files a
FULL OUTER JOIN snapshot_files b ON a.id = b.id
WHERE a.commit_hash = ? AND b.commit_hash = ?
  AND (a.id IS NULL OR b.id IS NULL OR a.sha256 != b.sha256)`, diff.OldHash, diff.NewHash)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var fd FileDiff
		var ct string
		if err := rows.Scan(&fd.FileID, &fd.Path, &ct, &fd.OldSHA256, &fd.NewSHA256); err != nil {
			return err
		}
		fd.ChangeType = ChangeType(ct)
		switch fd.ChangeType {
		case ChangeAdded:
			diff.Stats.FilesAdded++
		case ChangeRemoved:
			diff.Stats.FilesRemoved++
		case ChangeModified:
			diff.Stats.FilesModified++
		}
		diff.Files = append(diff.Files, fd)
	}
	return rows.Err()
}

func diffSymbols(ctx context.Context, db *sql.DB, diff *CommitDiff) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    COALESCE(a.id, b.id) AS symbol_id,
    COALESCE(a.name, b.name) AS name,
    COALESCE(a.kind, b.kind) AS kind,
    COALESCE(a.package_id, b.package_id) AS package_id,
    CASE
        WHEN a.id IS NULL THEN 'added'
        WHEN b.id IS NULL THEN 'removed'
        WHEN a.body_hash != b.body_hash AND a.body_hash != '' AND b.body_hash != '' THEN 'modified'
        WHEN a.signature != b.signature THEN 'signature-changed'
        WHEN a.start_line != b.start_line OR a.end_line != b.end_line THEN 'moved'
        ELSE 'unchanged'
    END AS change_type,
    COALESCE(a.start_line, 0),
    COALESCE(a.end_line, 0),
    COALESCE(b.start_line, 0),
    COALESCE(b.end_line, 0),
    COALESCE(a.signature, ''),
    COALESCE(b.signature, ''),
    COALESCE(a.body_hash, ''),
    COALESCE(b.body_hash, '')
FROM snapshot_symbols a
FULL OUTER JOIN snapshot_symbols b ON a.id = b.id
WHERE a.commit_hash = ? AND b.commit_hash = ?
  AND (a.id IS NULL OR b.id IS NULL OR a.body_hash != b.body_hash
       OR a.signature != b.signature
       OR a.start_line != b.start_line OR a.end_line != b.end_line)`, diff.OldHash, diff.NewHash)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sd SymbolDiff
		var ct string
		if err := rows.Scan(
			&sd.SymbolID, &sd.Name, &sd.Kind, &sd.PackageID, &ct,
			&sd.OldStartLine, &sd.OldEndLine, &sd.NewStartLine, &sd.NewEndLine,
			&sd.OldSignature, &sd.NewSignature, &sd.OldBodyHash, &sd.NewBodyHash,
		); err != nil {
			return err
		}
		sd.ChangeType = ChangeType(ct)
		switch sd.ChangeType {
		case ChangeAdded:
			diff.Stats.SymbolsAdded++
		case ChangeRemoved:
			diff.Stats.SymbolsRemoved++
		case ChangeModified:
			diff.Stats.SymbolsModified++
		case ChangeMoved:
			diff.Stats.SymbolsMoved++
		case ChangeUnchanged:
			diff.Stats.SymbolsUnchanged++
		}
		// Skip unchanged symbols from the output by default.
		if sd.ChangeType != ChangeUnchanged {
			diff.Symbols = append(diff.Symbols, sd)
		}
	}
	return rows.Err()
}
