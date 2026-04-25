package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wesen/codebase-browser/internal/concepts"
)

type conceptParamView struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Help      string   `json:"help,omitempty"`
	Required  bool     `json:"required,omitempty"`
	Default   any      `json:"default,omitempty"`
	Choices   []string `json:"choices,omitempty"`
	ShortFlag string   `json:"shortFlag,omitempty"`
}

type conceptView struct {
	Path       string             `json:"path"`
	Name       string             `json:"name"`
	Folder     string             `json:"folder,omitempty"`
	Short      string             `json:"short"`
	Long       string             `json:"long,omitempty"`
	Tags       []string           `json:"tags,omitempty"`
	Params     []conceptParamView `json:"params,omitempty"`
	SourceRoot string             `json:"sourceRoot,omitempty"`
	SourcePath string             `json:"sourcePath,omitempty"`
	Query      string             `json:"query,omitempty"`
}

type executeConceptRequest struct {
	Params     map[string]any `json:"params"`
	RenderOnly bool           `json:"renderOnly,omitempty"`
}

type executeConceptResponse struct {
	ConceptPath string           `json:"conceptPath"`
	RenderedSQL string           `json:"renderedSql"`
	Columns     []string         `json:"columns,omitempty"`
	Rows        []map[string]any `json:"rows,omitempty"`
	RowCount    int              `json:"rowCount"`
	Rendered    bool             `json:"rendered"`
}

func (s *Server) handleConceptList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.ConceptCatalog == nil {
		writeJSON(w, []conceptView{})
		return
	}
	out := make([]conceptView, 0, len(s.ConceptCatalog.Concepts))
	for _, concept := range s.ConceptCatalog.Concepts {
		out = append(out, conceptToView(concept))
	}
	writeJSON(w, out)
}

func (s *Server) handleConceptSubtree(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/api/query-concepts/")
	raw = strings.Trim(raw, "/")
	if raw == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(raw, "/execute") {
		conceptPath, err := decodeConceptPath(strings.TrimSuffix(raw, "/execute"))
		if err != nil {
			http.Error(w, "invalid concept path", http.StatusBadRequest)
			return
		}
		s.handleConceptExecute(w, r, conceptPath)
		return
	}
	conceptPath, err := decodeConceptPath(raw)
	if err != nil {
		http.Error(w, "invalid concept path", http.StatusBadRequest)
		return
	}
	s.handleConceptDetail(w, r, conceptPath)
}

func (s *Server) handleConceptDetail(w http.ResponseWriter, r *http.Request, conceptPath string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	concept, ok := s.lookupConcept(conceptPath)
	if !ok {
		http.Error(w, "concept not found", http.StatusNotFound)
		return
	}
	writeJSON(w, conceptToView(concept))
}

func (s *Server) handleConceptExecute(w http.ResponseWriter, r *http.Request, conceptPath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	concept, ok := s.lookupConcept(conceptPath)
	if !ok {
		http.Error(w, "concept not found", http.StatusNotFound)
		return
	}

	// Pick the right database: history/ concepts use the history DB.
	var db *sql.DB
	if strings.HasPrefix(conceptPath, "history/") && s.History != nil {
		db = s.History.DB()
	} else if s.SQLite != nil {
		db = s.SQLite.DB()
	} else {
		http.Error(w, "query backend unavailable", http.StatusServiceUnavailable)
		return
	}
	var req executeConceptRequest
	if r.Body != nil {
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	}
	rendered, err := concepts.RenderConcept(concept, req.Params)
	if err != nil {
		http.Error(w, fmt.Sprintf("render concept: %v", err), http.StatusBadRequest)
		return
	}
	resp := executeConceptResponse{
		ConceptPath: concept.Path,
		RenderedSQL: rendered,
		Rendered:    true,
	}
	if req.RenderOnly {
		writeJSON(w, resp)
		return
	}
	columns, rows, err := runQueryRows(r.Context(), db, rendered)
	if err != nil {
		http.Error(w, fmt.Sprintf("execute query: %v", err), http.StatusBadRequest)
		return
	}
	resp.Columns = columns
	resp.Rows = rows
	resp.RowCount = len(rows)
	writeJSON(w, resp)
}

func (s *Server) lookupConcept(path string) (*concepts.Concept, bool) {
	if s.ConceptCatalog == nil {
		return nil, false
	}
	concept, ok := s.ConceptCatalog.ByPath[path]
	return concept, ok
}

func conceptToView(concept *concepts.Concept) conceptView {
	params := make([]conceptParamView, 0, len(concept.Params))
	for _, param := range concept.Params {
		params = append(params, conceptParamView{
			Name:      param.Name,
			Type:      string(param.Type),
			Help:      param.Help,
			Required:  param.Required,
			Default:   param.Default,
			Choices:   append([]string(nil), param.Choices...),
			ShortFlag: param.ShortFlag,
		})
	}
	return conceptView{
		Path:       concept.Path,
		Name:       concept.Name,
		Folder:     concept.Folder,
		Short:      concept.Short,
		Long:       concept.Long,
		Tags:       append([]string(nil), concept.Tags...),
		Params:     params,
		SourceRoot: concept.SourceRoot,
		SourcePath: concept.SourcePath,
		Query:      concept.Query,
	}
}

func decodeConceptPath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	parts := strings.Split(raw, "/")
	decoded := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segment, err := url.PathUnescape(part)
		if err != nil {
			return "", err
		}
		decoded = append(decoded, segment)
	}
	return strings.Join(decoded, "/"), nil
}

func runQueryRows(ctx context.Context, db *sql.DB, sqlText string) ([]string, []map[string]any, error) {
	rows, err := db.QueryContext(ctx, strings.TrimSpace(sqlText))
	if err != nil {
		if _, execErr := db.ExecContext(ctx, strings.TrimSpace(sqlText)); execErr != nil {
			return nil, nil, err
		}
		return nil, []map[string]any{}, nil
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	all := []map[string]any{}
	for rows.Next() {
		row, err := scanSQLRow(rows, columns)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return columns, all, nil
}

func scanSQLRow(rows *sql.Rows, cols []string) (map[string]any, error) {
	raw := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(cols))
	for i, col := range cols {
		switch v := raw[i].(type) {
		case []byte:
			out[col] = string(v)
		case nil:
			out[col] = ""
		default:
			out[col] = v
		}
	}
	return out, nil
}
