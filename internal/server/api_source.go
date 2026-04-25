package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/wesen/codebase-browser/internal/history"
)

// handleSource returns raw file bytes for a module-relative path that must
// appear in the index files table (whitelist). Rejects traversal / absolute
// paths. See design §9.2.
func (s *Server) handleSource(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	if !safePath(p) {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	if _, ok := s.Loaded.File("file:" + p); !ok {
		http.Error(w, "not in index", http.StatusNotFound)
		return
	}
	data, err := fs.ReadFile(s.SourceFS, p)
	if err != nil {
		http.Error(w, fmt.Sprintf("read: %v", err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

// handleSnippet returns an exact byte range from a symbol's source. kind
// accepts declaration | body | signature. See design §9.3.
//
// When the "commit" query parameter is present and the server has a history
// DB, the snippet is resolved from the per-commit snapshot instead of the
// static embedded index. This is the Slice 0 extension (GCB-010).
func (s *Server) handleSnippet(w http.ResponseWriter, r *http.Request) {
	symID := r.URL.Query().Get("sym")
	kind := r.URL.Query().Get("kind")
	commitHash := r.URL.Query().Get("commit")
	if kind == "" {
		kind = "declaration"
	}

	// If a commit hash is specified and history DB is available, resolve
	// from the per-commit snapshot (GCB-010 Slice 0).
	if commitHash != "" && s.History != nil {
		s.handleSnippetFromHistory(w, r, symID, kind, commitHash)
		return
	}

	// Existing path: resolve from static embedded index.
	sym, ok := s.Loaded.Symbol(symID)
	if !ok {
		http.Error(w, "symbol not found", http.StatusNotFound)
		return
	}
	file, ok := s.Loaded.File(sym.FileID)
	if !ok {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if !safePath(file.Path) {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	data, err := fs.ReadFile(s.SourceFS, file.Path)
	if err != nil {
		http.Error(w, "read", http.StatusNotFound)
		return
	}
	start, end := sym.Range.StartOffset, sym.Range.EndOffset
	if start < 0 || end > len(data) || start > end {
		http.Error(w, "range out of file", http.StatusInternalServerError)
		return
	}
	snippet := string(data[start:end])
	snippet = applyKind(snippet, kind)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Codebase-Symbol-Id", sym.ID)
	w.Header().Set("X-Codebase-File-Sha256", file.SHA256)
	w.Header().Set("X-Codebase-Range", fmt.Sprintf("%d-%d", start, end))
	_, _ = w.Write([]byte(snippet))
}

// handleSnippetFromHistory resolves a snippet from a per-commit snapshot
// in the history database. It reads file content from the cache or falls
// back to `git show`.
func (s *Server) handleSnippetFromHistory(w http.ResponseWriter, r *http.Request, symID, kind, commitHash string) {
	var startOffset, endOffset int
	var filePath string

	err := s.History.DB().QueryRowContext(r.Context(), `
SELECT f.path, s.start_offset, s.end_offset
FROM   snapshot_symbols s
JOIN   snapshot_files f ON f.commit_hash = s.commit_hash AND f.id = s.file_id
WHERE  s.commit_hash = ? AND s.id = ?`, commitHash, symID).Scan(&filePath, &startOffset, &endOffset)
	if err != nil {
		http.Error(w, fmt.Sprintf("symbol %s not found at commit %s", symID, commitHash[:7]), http.StatusNotFound)
		return
	}

	var content []byte
	if s.RepoRoot != "" {
		var cerr error
		content, cerr = history.GetFileContent(r.Context(), s.History, s.RepoRoot, commitHash, filePath)
		if cerr != nil {
			http.Error(w, fmt.Sprintf("read %s at %s: %v", filePath, commitHash[:7], cerr), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "repo root not configured; pass --repo-root to serve", http.StatusNotFound)
		return
	}

	if startOffset < 0 || endOffset > len(content) || startOffset > endOffset {
		http.Error(w, "range out of file", http.StatusInternalServerError)
		return
	}
	snippet := string(content[startOffset:endOffset])
	snippet = applyKind(snippet, kind)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Codebase-Symbol-Id", symID)
	w.Header().Set("X-Codebase-Commit", commitHash)
	w.Header().Set("X-Codebase-Range", fmt.Sprintf("%d-%d", startOffset, endOffset))
	_, _ = w.Write([]byte(snippet))
}

// applyKind trims a snippet according to the requested kind (signature, body, declaration).
func applyKind(snippet, kind string) string {
	switch kind {
	case "signature":
		if nl := strings.IndexByte(snippet, '\n'); nl > 0 {
			snippet = snippet[:nl]
		}
	case "body":
		if open := strings.IndexByte(snippet, '{'); open >= 0 {
			snippet = strings.TrimSpace(snippet[open:])
		}
	}
	return snippet
}

func safePath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") {
		return false
	}
	clean := path.Clean(p)
	if clean != p {
		return false
	}
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "/../") {
		return false
	}
	return true
}
