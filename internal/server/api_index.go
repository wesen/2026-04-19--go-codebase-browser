package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(s.Loaded.Raw)
}

func (s *Server) handlePackages(w http.ResponseWriter, r *http.Request) {
	type lite struct {
		ID         string `json:"id"`
		ImportPath string `json:"importPath"`
		Name       string `json:"name"`
		Files      int    `json:"files"`
		Symbols    int    `json:"symbols"`
	}
	out := make([]lite, 0, len(s.Loaded.Index.Packages))
	for _, p := range s.Loaded.Index.Packages {
		out = append(out, lite{
			ID:         p.ID,
			ImportPath: p.ImportPath,
			Name:       p.Name,
			Files:      len(p.FileIDs),
			Symbols:    len(p.SymbolIDs),
		})
	}
	writeJSON(w, out)
}

func (s *Server) handleSymbol(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/symbol/")
	sym, ok := s.Loaded.Symbol(id)
	if !ok {
		http.Error(w, "symbol not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sym)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	kind := r.URL.Query().Get("kind")
	hits := s.Loaded.FindSymbols(q, kind)
	if len(hits) > 200 {
		hits = hits[:200]
	}
	writeJSON(w, hits)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
