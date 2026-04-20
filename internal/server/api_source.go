package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
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
func (s *Server) handleSnippet(w http.ResponseWriter, r *http.Request) {
	symID := r.URL.Query().Get("sym")
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "declaration"
	}
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
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Codebase-Symbol-Id", sym.ID)
	w.Header().Set("X-Codebase-File-Sha256", file.SHA256)
	w.Header().Set("X-Codebase-Range", fmt.Sprintf("%d-%d", start, end))
	_, _ = w.Write([]byte(snippet))
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
