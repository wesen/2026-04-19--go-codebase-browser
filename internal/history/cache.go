package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wesen/codebase-browser/internal/gitutil"
)

// CacheFileContents reads all files from an index's snapshot and stores their
// contents in the file_contents table, keyed by SHA-256 hash. Only files not
// already cached are stored.
func CacheFileContents(ctx context.Context, store *Store, commitHash string, worktreeDir string) error {
	files, err := store.DB().QueryContext(ctx, `
SELECT id, path FROM snapshot_files WHERE commit_hash = ?`, commitHash)
	if err != nil {
		return fmt.Errorf("query snapshot files: %w", err)
	}
	defer files.Close()

	type fileInfo struct {
		id   string
		path string
	}
	var toCache []fileInfo

	for files.Next() {
		var fi fileInfo
		if err := files.Scan(&fi.id, &fi.path); err != nil {
			return err
		}
		toCache = append(toCache, fi)
	}
	if err := files.Err(); err != nil {
		return err
	}

	// Batch insert contents, skipping already-cached entries.
	var mu sync.Mutex
	batchErr := make(chan error, len(toCache))
	for _, fi := range toCache {
		func() {
			// Extract relative path from file ID.
			relPath := fi.path
			absPath := filepath.Join(worktreeDir, relPath)
			data, err := os.ReadFile(absPath)
			if err != nil {
				return // skip unreadable files
			}
			hash := sha256.Sum256(data)
			hashStr := hex.EncodeToString(hash[:])

			mu.Lock()
			defer mu.Unlock()
			// Check if already cached.
			var count int
			_ = store.DB().QueryRowContext(ctx,
				"SELECT COUNT(1) FROM file_contents WHERE content_hash = ?",
				hashStr,
			).Scan(&count)
			if count > 0 {
				return
			}
			_, err = store.DB().ExecContext(ctx,
				"INSERT OR IGNORE INTO file_contents(content_hash, content) VALUES (?, ?)",
				hashStr, data,
			)
			if err != nil {
				select {
				case batchErr <- err:
				default:
				}
			}
		}()
	}

	select {
	case err := <-batchErr:
		return err
	default:
		return nil
	}
}

// GetFileContent retrieves cached file content by content hash, or falls back
// to reading from git if the content is not in cache.
func GetFileContent(ctx context.Context, store *Store, repoRoot, commitHash, filePath string) ([]byte, error) {
	// Try cache first: look up the file's sha256 in snapshot_files.
	var sha string
	err := store.DB().QueryRowContext(ctx, `
SELECT sha256 FROM snapshot_files
WHERE commit_hash = ? AND path = ?`, commitHash, filePath).Scan(&sha)
	if err == nil && sha != "" {
		var content []byte
		err = store.DB().QueryRowContext(ctx,
			"SELECT content FROM file_contents WHERE content_hash = ?", sha,
		).Scan(&content)
		if err == nil {
			return content, nil
		}
	}

	// Fall back to git show.
	return gitutil.ShowFile(ctx, repoRoot, commitHash, filePath)
}
