package history

import (
	"context"
	"fmt"
)

// BodyDiffResult holds the result of a per-symbol body diff.
type BodyDiffResult struct {
	SymbolID    string `json:"symbolId"`
	Name        string `json:"name"`
	OldCommit   string `json:"oldCommit"`
	NewCommit   string `json:"newCommit"`
	OldBody     string `json:"oldBody"`
	NewBody     string `json:"newBody"`
	UnifiedDiff string `json:"unifiedDiff"`
	OldRange    string `json:"oldRange"`
	NewRange    string `json:"newRange"`
}

// DiffSymbolBody returns the body diff of a symbol between two commits.
// It reads the file content at both commits, extracts the symbol body using
// byte offsets, and computes a simple line-by-line diff.
func (s *Store) DiffSymbolBody(ctx context.Context, oldHash, newHash, symbolID string) (*BodyDiffResult, error) {
	// Look up symbol in both commits.
	var oldSym, newSym struct {
		fileID      string
		startOffset int
		endOffset   int
		startLine   int
		endLine     int
		name        string
	}

	err := s.db.QueryRowContext(ctx, `
SELECT file_id, start_offset, end_offset, start_line, end_line, name
FROM   snapshot_symbols
WHERE  commit_hash = ? AND id = ?`, oldHash, symbolID).Scan(
		&oldSym.fileID, &oldSym.startOffset, &oldSym.endOffset,
		&oldSym.startLine, &oldSym.endLine, &oldSym.name,
	)
	if err != nil {
		return nil, fmt.Errorf("symbol %s not found at %s: %w", symbolID, shortHashForError(oldHash), err)
	}

	err = s.db.QueryRowContext(ctx, `
SELECT file_id, start_offset, end_offset, start_line, end_line, name
FROM   snapshot_symbols
WHERE  commit_hash = ? AND id = ?`, newHash, symbolID).Scan(
		&newSym.fileID, &newSym.startOffset, &newSym.endOffset,
		&newSym.startLine, &newSym.endLine, &newSym.name,
	)
	if err != nil {
		return nil, fmt.Errorf("symbol %s not found at %s: %w", symbolID, shortHashForError(newHash), err)
	}

	// We can't read from cache without a repoRoot, so return what we can.
	// The caller (CLI or server) should use GetFileContent to read bodies.
	return &BodyDiffResult{
		SymbolID:  symbolID,
		Name:      newSym.name,
		OldCommit: oldHash,
		NewCommit: newHash,
		OldRange:  fmt.Sprintf("lines %d-%d", oldSym.startLine, oldSym.endLine),
		NewRange:  fmt.Sprintf("lines %d-%d", newSym.startLine, newSym.endLine),
	}, nil
}

// DiffSymbolBodyWithContent computes the full body diff when file contents are provided.
// This is used by the CLI and server which have access to the repo root.
func DiffSymbolBodyWithContent(ctx context.Context, store *Store, repoRoot, oldHash, newHash, symbolID string) (*BodyDiffResult, error) {
	result, err := store.DiffSymbolBody(ctx, oldHash, newHash, symbolID)
	if err != nil {
		return nil, err
	}

	// Extract old body.
	oldBody, err := extractBody(ctx, store, repoRoot, oldHash, symbolID)
	if err != nil {
		result.OldBody = fmt.Sprintf("(error: %v)", err)
	} else {
		result.OldBody = oldBody
	}

	// Extract new body.
	newBody, err := extractBody(ctx, store, repoRoot, newHash, symbolID)
	if err != nil {
		result.NewBody = fmt.Sprintf("(error: %v)", err)
	} else {
		result.NewBody = newBody
	}

	// Compute unified diff.
	result.UnifiedDiff = simpleUnifiedDiff(result.OldBody, result.NewBody)

	return result, nil
}

func extractBody(ctx context.Context, store *Store, repoRoot, commitHash, symbolID string) (string, error) {
	var filePath string
	var startOffset, endOffset int

	err := store.db.QueryRowContext(ctx, `
SELECT f.path, s.start_offset, s.end_offset
FROM   snapshot_symbols s
JOIN   snapshot_files f ON f.commit_hash = s.commit_hash AND f.id = s.file_id
WHERE  s.commit_hash = ? AND s.id = ?`, commitHash, symbolID).Scan(&filePath, &startOffset, &endOffset)
	if err != nil {
		return "", fmt.Errorf("lookup %s at %s: %w", symbolID, shortHashForError(commitHash), err)
	}

	content, err := GetFileContent(ctx, store, repoRoot, commitHash, filePath)
	if err != nil {
		return "", fmt.Errorf("read %s at %s: %w", filePath, shortHashForError(commitHash), err)
	}

	if startOffset < 0 || endOffset > len(content) || startOffset > endOffset {
		return "", fmt.Errorf("invalid range %d-%d for file %s (len=%d)", startOffset, endOffset, filePath, len(content))
	}

	return string(content[startOffset:endOffset]), nil
}

// simpleUnifiedDiff produces a unified diff between two strings.
// It shows every line: unchanged lines prefixed with "  ", removed with "- ",
// added with "+ ". This gives the user the full function context, not just
// the changed region.
func simpleUnifiedDiff(old, new_ string) string {
	oldLines := splitLines(old)
	newLines := splitLines(new_)

	// Compute the edit script using a simple LCS-based approach.
	// For small functions (<500 lines) this is fast enough.
	var out string

	// Find common prefix
	prefix := 0
	for prefix < len(oldLines) && prefix < len(newLines) && oldLines[prefix] == newLines[prefix] {
		prefix++
	}

	// Find common suffix (but not overlapping with prefix)
	suffix := 0
	for suffix < len(oldLines)-prefix && suffix < len(newLines)-prefix &&
		oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}

	// Emit common prefix
	for i := 0; i < prefix; i++ {
		out += fmt.Sprintf("  %s\n", oldLines[i])
	}

	// Emit removed lines from old
	oldEnd := len(oldLines) - suffix
	for i := prefix; i < oldEnd; i++ {
		out += fmt.Sprintf("- %s\n", oldLines[i])
	}

	// Emit added lines from new
	newEnd := len(newLines) - suffix
	for i := prefix; i < newEnd; i++ {
		out += fmt.Sprintf("+ %s\n", newLines[i])
	}

	// Emit common suffix
	for i := len(oldLines) - suffix; i < len(oldLines); i++ {
		out += fmt.Sprintf("  %s\n", oldLines[i])
	}

	return out
}

func shortHashForError(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
