package review

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/docs"
	"github.com/wesen/codebase-browser/internal/server"
)

// ReviewServer extends the base server with review-specific routes.
type ReviewServer struct {
	*server.Server
	Store *Store
}

// NewReviewServer constructs a ReviewServer.
func NewReviewServer(base *server.Server, store *Store) *ReviewServer {
	return &ReviewServer{Server: base, Store: store}
}

// Handler returns an http.Handler with review routes mounted.
// Review-specific routes take precedence over base routes.
func (rs *ReviewServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// Review-specific routes.
	mux.HandleFunc("/api/review/docs", rs.handleReviewDocList)
	mux.HandleFunc("/api/review/docs/", rs.handleReviewDocPage)
	mux.HandleFunc("/api/review/commits", rs.handleReviewCommits)
	mux.HandleFunc("/api/review/stats", rs.handleReviewStats)

	// Everything else goes to the base server.
	base := rs.Server.Handler()
	mux.Handle("/", base)

	return mux
}

// handleReviewDocList lists all review documents.
func (rs *ReviewServer) handleReviewDocList(w http.ResponseWriter, r *http.Request) {
	rows, err := rs.Store.DB().QueryContext(r.Context(), `
		SELECT slug, title, path, indexed_at FROM review_docs ORDER BY slug
	`)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var docs []DocMeta
	for rows.Next() {
		var d DocMeta
		if err := rows.Scan(&d.Slug, &d.Title, &d.Path, &d.IndexedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, docs)
}

// handleReviewDocPage renders a single review document.
func (rs *ReviewServer) handleReviewDocPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/api/review/docs/")
	if slug == "" {
		writeJSONError(w, http.StatusBadRequest, "missing slug")
		return
	}

	var content string
	err := rs.Store.DB().QueryRowContext(r.Context(), `
		SELECT content FROM review_docs WHERE slug = ?
	`, slug).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "doc not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	page, err := docs.Render(slug, []byte(content), rs.Server.Loaded, rs.Server.SourceFS)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, page)
}

// handleReviewCommits lists all commits in the review database.
func (rs *ReviewServer) handleReviewCommits(w http.ResponseWriter, r *http.Request) {
	commits, err := rs.Store.History.ListCommits(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, commits)
}

// handleReviewStats returns counts of entities in the review database.
func (rs *ReviewServer) handleReviewStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := rs.Store.DB()

	stats := map[string]int{}

	var counts = []struct {
		key   string
		query string
	}{
		{"commits", "SELECT COUNT(*) FROM commits"},
		{"docs", "SELECT COUNT(*) FROM review_docs"},
		{"snippets", "SELECT COUNT(*) FROM review_doc_snippets"},
		{"symbols", "SELECT COUNT(*) FROM snapshot_symbols"},
		{"files", "SELECT COUNT(*) FROM snapshot_files"},
		{"refs", "SELECT COUNT(*) FROM snapshot_refs"},
	}

	for _, c := range counts {
		var n int
		if err := db.QueryRowContext(ctx, c.query).Scan(&n); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		stats[c.key] = n
	}

	writeJSON(w, stats)
}

// DocMeta is a lightweight review document summary.
type DocMeta struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	IndexedAt int64  `json:"indexedAt"`
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

// Ensure ReviewServer implements the same patterns as the base server.
var _ = browser.Loaded{}
var _ = docs.Page{}
