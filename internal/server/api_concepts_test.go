package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/wesen/codebase-browser/internal/concepts"
	cbsqlite "github.com/wesen/codebase-browser/internal/sqlite"
)

func TestHandleConceptList(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	catalog := mustLoadTestCatalog(t, `/* codebase-browser concept
name: hello-world
short: Hello world query
*/
SELECT 'hello' AS value;
`)
	h := New(loaded, srcFS, spa, nil, catalog).Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/query-concepts", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var concepts []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &concepts); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(concepts) != 1 {
		t.Fatalf("len(concepts)=%d, want 1", len(concepts))
	}
	if concepts[0]["path"] != "test/hello-world" {
		t.Fatalf("path=%v, want test/hello-world", concepts[0]["path"])
	}
}

func TestHandleConceptExecute(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	catalog := mustLoadTestCatalog(t, `/* codebase-browser concept
name: greet
short: Greeting query
params:
  - name: who
    type: string
    default: world
*/
SELECT {{ sqlString (value "who") }} AS greeting;
`)
	store := mustOpenTempSQLite(t)
	defer store.Close()
	h := New(loaded, srcFS, spa, store, catalog).Handler()

	body := bytes.NewBufferString(`{"params":{"who":"team"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/query-concepts/test/greet/execute", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		ConceptPath string           `json:"conceptPath"`
		RenderedSQL string           `json:"renderedSql"`
		Columns     []string         `json:"columns"`
		Rows        []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ConceptPath != "test/greet" {
		t.Fatalf("ConceptPath=%q, want test/greet", resp.ConceptPath)
	}
	if resp.RenderedSQL == "" {
		t.Fatalf("expected rendered SQL")
	}
	if len(resp.Rows) != 1 || resp.Rows[0]["greeting"] != "team" {
		t.Fatalf("unexpected rows: %#v", resp.Rows)
	}
}

func TestHandleConceptExecute_RenderOnly(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	catalog := mustLoadTestCatalog(t, `/* codebase-browser concept
name: numbers
short: Number query
*/
SELECT 1 AS n;
`)
	store := mustOpenTempSQLite(t)
	defer store.Close()
	h := New(loaded, srcFS, spa, store, catalog).Handler()

	body := bytes.NewBufferString(`{"renderOnly":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/query-concepts/test/numbers/execute", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		RenderedSQL string           `json:"renderedSql"`
		Rows        []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.RenderedSQL == "" {
		t.Fatalf("expected rendered SQL")
	}
	if len(resp.Rows) != 0 {
		t.Fatalf("render-only rows = %#v, want empty", resp.Rows)
	}
}

func mustLoadTestCatalog(t *testing.T, sqlText string) *concepts.Catalog {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "concept.sql"), []byte(sqlText), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := concepts.LoadCatalogFromDirs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func mustOpenTempSQLite(t *testing.T) *cbsqlite.Store {
	t.Helper()
	store, err := cbsqlite.Open(filepath.Join(t.TempDir(), "codebase.db"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
