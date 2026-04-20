package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/wesen/codebase-browser/internal/docs"
)

func (s *Server) handleDocList(w http.ResponseWriter, r *http.Request) {
	pages, err := docs.ListPages(docs.PagesFS())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, pages)
}

func (s *Server) handleDocPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/api/doc/")
	if slug == "" {
		http.Error(w, "missing slug", http.StatusBadRequest)
		return
	}
	// Resolve slug → pages-FS path (append .md if missing).
	path := slug
	if !strings.HasSuffix(path, ".md") {
		path = path + ".md"
	}
	data, err := fs.ReadFile(docs.PagesFS(), path)
	if err != nil {
		http.Error(w, "doc not found", http.StatusNotFound)
		return
	}
	page, err := docs.Render(slug, data, s.Loaded, s.SourceFS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, page)
}
