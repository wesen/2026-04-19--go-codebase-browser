package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/indexer"
)

func buildFixtureLoaded(t *testing.T) (*browser.Loaded, fstest.MapFS) {
	t.Helper()
	srcContent := []byte("package foo\n\nfunc Bar() {}\n")
	idx := &indexer.Index{
		Version: "1",
		Module:  "example.com/foo",
		Packages: []indexer.Package{{
			ID:         indexer.PackageID("example.com/foo"),
			ImportPath: "example.com/foo",
			Name:       "foo",
			FileIDs:    []string{indexer.FileID("foo.go")},
			SymbolIDs:  []string{indexer.SymbolID("example.com/foo", "func", "Bar", "")},
		}},
		Files: []indexer.File{{
			ID:        indexer.FileID("foo.go"),
			Path:      "foo.go",
			PackageID: indexer.PackageID("example.com/foo"),
			Size:      len(srcContent),
			LineCount: 3,
		}},
		Symbols: []indexer.Symbol{{
			ID:        indexer.SymbolID("example.com/foo", "func", "Bar", ""),
			Kind:      "func",
			Name:      "Bar",
			PackageID: indexer.PackageID("example.com/foo"),
			FileID:    indexer.FileID("foo.go"),
			Signature: "func Bar()",
			Range:     indexer.Range{StartOffset: 13, EndOffset: 26, StartLine: 3, EndLine: 3},
		}},
	}
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := browser.LoadFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	srcFS := fstest.MapFS{"foo.go": &fstest.MapFile{Data: srcContent}}
	return loaded, srcFS
}

func TestHandleIndex(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	h := New(loaded, srcFS, spa, nil, nil).Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/index", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestHandleSource_Whitelist(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	h := New(loaded, srcFS, spa, nil, nil).Handler()

	t.Run("happy path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/source?path=foo.go", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("not in index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/source?path=bar.go", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Fatalf("code=%d", w.Code)
		}
	})
	t.Run("traversal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/source?path=../etc/passwd", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("code=%d", w.Code)
		}
	})
	t.Run("absolute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/source?path=/etc/passwd", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("code=%d", w.Code)
		}
	})
}

func TestHandleSnippet(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	h := New(loaded, srcFS, spa, nil, nil).Handler()
	symID := indexer.SymbolID("example.com/foo", "func", "Bar", "")
	req := httptest.NewRequest(http.MethodGet, "/api/snippet?sym="+symID, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	body, _ := io.ReadAll(w.Body)
	if string(body) != "func Bar() {}" {
		t.Errorf("body=%q", body)
	}
	if w.Header().Get("X-Codebase-Symbol-Id") != symID {
		t.Errorf("missing symbol header")
	}
}

func TestSPAFallback(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html>hi</html>")}}
	h := New(loaded, srcFS, spa, nil, nil).Handler()
	// Arbitrary client-side route -> serves index.html.
	req := httptest.NewRequest(http.MethodGet, "/symbols/pkg.func.X", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("content-type=%q", ct)
	}
}

func TestAPIPathNotShadowedBySPA(t *testing.T) {
	loaded, srcFS := buildFixtureLoaded(t)
	spa := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	h := New(loaded, srcFS, spa, nil, nil).Handler()
	// Non-existent /api route should 404, not fall through to index.html.
	req := httptest.NewRequest(http.MethodGet, "/api/nope", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("code=%d", w.Code)
	}
}
