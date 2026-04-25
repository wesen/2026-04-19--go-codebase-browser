package server

import (
	"net/http"

	"github.com/wesen/codebase-browser/internal/history"
)

// historyHandlers registers all /api/history/* routes.
func (s *Server) registerHistoryRoutes() {
	if s.History == nil {
		return
	}
	mux := s.mux
	mux.HandleFunc("GET /api/history/commits", s.handleHistoryCommits)
	mux.HandleFunc("GET /api/history/commits/{hash}", s.handleHistoryCommitDetail)
	mux.HandleFunc("GET /api/history/commits/{hash}/symbols", s.handleHistoryCommitSymbols)
	mux.HandleFunc("GET /api/history/diff", s.handleHistoryDiff)
	mux.HandleFunc("GET /api/history/symbol-body-diff", s.handleSymbolBodyDiff)
	mux.HandleFunc("GET /api/history/symbols/{symbolID}/history", s.handleSymbolHistory)
}

func (s *Server) handleHistoryCommits(w http.ResponseWriter, r *http.Request) {
	commits, err := s.History.ListCommits(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, commits)
}

func (s *Server) handleHistoryCommitDetail(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	commit, err := s.History.GetCommit(r.Context(), hash)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if commit == nil {
		writeJSONError(w, http.StatusNotFound, "commit not found")
		return
	}
	writeJSON(w, commit)
}

func (s *Server) handleHistoryCommitSymbols(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	rows, err := s.History.DB().QueryContext(r.Context(), `
SELECT id, kind, name, package_id, file_id,
       start_line, end_line, signature, exported, body_hash
FROM   snapshot_symbols
WHERE  commit_hash = ?
ORDER BY name`, hash)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type sym struct {
		ID         string `json:"id"`
		Kind       string `json:"kind"`
		Name       string `json:"name"`
		PackageID  string `json:"packageId"`
		FileID     string `json:"fileId"`
		StartLine  int    `json:"startLine"`
		EndLine    int    `json:"endLine"`
		Signature  string `json:"signature"`
		Exported   bool   `json:"exported"`
		BodyHash   string `json:"bodyHash"`
	}
	var result []sym
	for rows.Next() {
		var s sym
		var exported int
		if err := rows.Scan(&s.ID, &s.Kind, &s.Name, &s.PackageID, &s.FileID,
			&s.StartLine, &s.EndLine, &s.Signature, &exported, &s.BodyHash); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.Exported = exported == 1
		result = append(result, s)
	}
	if result == nil {
		result = []sym{}
	}
	writeJSON(w, result)
}

func (s *Server) handleHistoryDiff(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeJSONError(w, http.StatusBadRequest, "both 'from' and 'to' query params required")
		return
	}
	diff, err := s.History.DiffCommits(r.Context(), from, to)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, diff)
}

func (s *Server) handleSymbolHistory(w http.ResponseWriter, r *http.Request) {
	symbolID := r.PathValue("symbolID")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}

	rows, err := s.History.DB().QueryContext(r.Context(), `
SELECT c.hash, c.short_hash, c.message, c.author_time,
       s.body_hash, s.start_line, s.end_line, s.signature, s.kind
FROM   snapshot_symbols s
JOIN   commits c ON c.hash = s.commit_hash
WHERE  s.id = ?
ORDER BY c.author_time DESC
LIMIT  ?`, symbolID, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type entry struct {
		CommitHash string `json:"commitHash"`
		ShortHash  string `json:"shortHash"`
		Message    string `json:"message"`
		AuthorTime int64  `json:"authorTime"`
		BodyHash   string `json:"bodyHash"`
		StartLine  int    `json:"startLine"`
		EndLine    int    `json:"endLine"`
		Signature  string `json:"signature"`
		Kind       string `json:"kind"`
	}
	var result []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.CommitHash, &e.ShortHash, &e.Message, &e.AuthorTime,
			&e.BodyHash, &e.StartLine, &e.EndLine, &e.Signature, &e.Kind); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		result = append(result, e)
	}
	if result == nil {
		result = []entry{}
	}
	writeJSON(w, result)
}

// HistoryStore is a dependency the Server can optionally carry.
var _ = (*history.Store)(nil)

func (s *Server) handleSymbolBodyDiff(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	symbolID := r.URL.Query().Get("symbol")
	if from == "" || to == "" || symbolID == "" {
		writeJSONError(w, http.StatusBadRequest, "'from', 'to', and 'symbol' query params required")
		return
	}

	repoRoot := s.RepoRoot
	if repoRoot == "" {
		writeJSONError(w, http.StatusNotFound, "repo root not configured; pass --repo-root to serve")
		return
	}

	result, err := history.DiffSymbolBodyWithContent(r.Context(), s.History, repoRoot, from, to, symbolID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, result)
}
