package server

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

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
	mux.HandleFunc("GET /api/history/impact", s.handleHistoryImpact)
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
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		PackageID string `json:"packageId"`
		FileID    string `json:"fileId"`
		StartLine int    `json:"startLine"`
		EndLine   int    `json:"endLine"`
		Signature string `json:"signature"`
		Exported  bool   `json:"exported"`
		BodyHash  string `json:"bodyHash"`
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

type impactEdge struct {
	FromSymbolID string `json:"fromSymbolId"`
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	FileID       string `json:"fileId"`
}

type impactNode struct {
	SymbolID      string       `json:"symbolId"`
	Name          string       `json:"name"`
	Kind          string       `json:"kind"`
	Depth         int          `json:"depth"`
	Edges         []impactEdge `json:"edges"`
	Compatibility string       `json:"compatibility"`
	Local         bool         `json:"local"`
}

type impactResponse struct {
	Root      string       `json:"root"`
	Direction string       `json:"direction"`
	Depth     int          `json:"depth"`
	Commit    string       `json:"commit"`
	Nodes     []impactNode `json:"nodes"`
}

func (s *Server) handleHistoryImpact(w http.ResponseWriter, r *http.Request) {
	symbolID := r.URL.Query().Get("sym")
	direction := r.URL.Query().Get("dir")
	if direction == "" {
		direction = "usedby"
	}
	if symbolID == "" {
		writeJSONError(w, http.StatusBadRequest, "sym query param required")
		return
	}
	if direction != "usedby" && direction != "uses" {
		writeJSONError(w, http.StatusBadRequest, "dir must be usedby or uses")
		return
	}
	depth := 2
	if raw := r.URL.Query().Get("depth"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeJSONError(w, http.StatusBadRequest, "depth must be a positive integer")
			return
		}
		depth = parsed
	}
	if depth > 5 {
		depth = 5
	}

	commitHash := r.URL.Query().Get("commit")
	if commitHash == "" {
		latest, err := latestHistoryCommit(r.Context(), s.History.DB())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		commitHash = latest
	}

	result, err := impactBFS(r.Context(), s.History.DB(), commitHash, symbolID, direction, depth)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, result)
}

func latestHistoryCommit(ctx context.Context, db *sql.DB) (string, error) {
	var hash string
	err := db.QueryRowContext(ctx, `SELECT hash FROM commits WHERE error = '' ORDER BY author_time DESC LIMIT 1`).Scan(&hash)
	return hash, err
}

func impactBFS(ctx context.Context, db *sql.DB, commitHash, root, direction string, maxDepth int) (*impactResponse, error) {
	response := &impactResponse{Root: root, Direction: direction, Depth: maxDepth, Commit: commitHash}
	visited := map[string]bool{root: true}
	type queueItem struct {
		symbolID string
		depth    int
	}
	queue := []queueItem{{symbolID: root, depth: 0}}
	nodeByID := map[string]*impactNode{}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.depth >= maxDepth {
			continue
		}

		edges, err := impactOneHop(ctx, db, commitHash, item.symbolID, direction)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			nextID := edge.FromSymbolID
			if direction == "uses" {
				nextID = edge.ToSymbolID
			}
			nextDepth := item.depth + 1
			node := nodeByID[nextID]
			if node == nil {
				name, kind, local, err := impactSymbolMeta(ctx, db, commitHash, nextID)
				if err != nil {
					name, kind, local = impactFallbackName(nextID), "external", false
				}
				node = &impactNode{SymbolID: nextID, Name: name, Kind: kind, Depth: nextDepth, Compatibility: "unknown", Local: local}
				nodeByID[nextID] = node
				response.Nodes = append(response.Nodes, *node)
			}
			node.Edges = append(node.Edges, edge)
			// Keep response slice in sync after appending edge to node pointer.
			for i := range response.Nodes {
				if response.Nodes[i].SymbolID == nextID {
					response.Nodes[i] = *node
					break
				}
			}
			if !visited[nextID] {
				visited[nextID] = true
				queue = append(queue, queueItem{symbolID: nextID, depth: nextDepth})
			}
		}
	}
	if response.Nodes == nil {
		response.Nodes = []impactNode{}
	}
	return response, nil
}

func impactOneHop(ctx context.Context, db *sql.DB, commitHash, symbolID, direction string) ([]impactEdge, error) {
	query := `
SELECT from_symbol_id, to_symbol_id, kind, file_id
FROM   snapshot_refs
WHERE  commit_hash = ? AND to_symbol_id = ?
ORDER BY from_symbol_id, kind`
	if direction == "uses" {
		query = `
SELECT from_symbol_id, to_symbol_id, kind, file_id
FROM   snapshot_refs
WHERE  commit_hash = ? AND from_symbol_id = ?
ORDER BY to_symbol_id, kind`
	}
	rows, err := db.QueryContext(ctx, query, commitHash, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []impactEdge
	for rows.Next() {
		var edge impactEdge
		if err := rows.Scan(&edge.FromSymbolID, &edge.ToSymbolID, &edge.Kind, &edge.FileID); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

func impactSymbolMeta(ctx context.Context, db *sql.DB, commitHash, symbolID string) (string, string, bool, error) {
	var name, kind string
	err := db.QueryRowContext(ctx, `
SELECT name, kind FROM snapshot_symbols WHERE commit_hash = ? AND id = ?`, commitHash, symbolID).Scan(&name, &kind)
	return name, kind, err == nil, err
}

func impactFallbackName(symbolID string) string {
	trimmed := symbolID
	if len(trimmed) > 4 && trimmed[:4] == "sym:" {
		trimmed = trimmed[4:]
	}
	lastDot := -1
	for i := len(trimmed) - 1; i >= 0; i-- {
		if trimmed[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot >= 0 && lastDot+1 < len(trimmed) {
		return trimmed[lastDot+1:]
	}
	return trimmed
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
