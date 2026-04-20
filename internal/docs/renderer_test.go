package docs

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/indexer"
)

func fixtureLoaded(t *testing.T) (*browser.Loaded, fstest.MapFS) {
	t.Helper()
	src := "package foo\n\n// Hello greets.\nfunc Hello(name string) string {\n\treturn \"hi \" + name\n}\n"
	// Offsets below were computed for the source above.
	idx := &indexer.Index{
		Version: "1",
		Module:  "example.com/foo",
		Packages: []indexer.Package{{
			ID: indexer.PackageID("example.com/foo"), ImportPath: "example.com/foo", Name: "foo",
		}},
		Files: []indexer.File{{
			ID: indexer.FileID("foo.go"), Path: "foo.go",
			PackageID: indexer.PackageID("example.com/foo"),
			Size:      len(src),
		}},
		Symbols: []indexer.Symbol{{
			ID:        indexer.SymbolID("example.com/foo", "func", "Hello", ""),
			Kind:      "func",
			Name:      "Hello",
			PackageID: indexer.PackageID("example.com/foo"),
			FileID:    indexer.FileID("foo.go"),
			Signature: "func Hello(name string) string",
			Doc:       "Hello greets.",
			Range:     indexer.Range{StartOffset: 29, EndOffset: int(len(src) - 1), StartLine: 4, EndLine: 6},
		}},
	}
	data, _ := json.Marshal(idx)
	l, err := browser.LoadFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	return l, fstest.MapFS{"foo.go": &fstest.MapFile{Data: []byte(src)}}
}

func TestRender_Signature(t *testing.T) {
	l, srcFS := fixtureLoaded(t)
	md := "# title\n\n```codebase-signature sym=example.com/foo.Hello\n```\n"
	page, err := Render("p", []byte(md), l, srcFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Errors) > 0 {
		t.Fatalf("errors: %v", page.Errors)
	}
	if !strings.Contains(page.HTML, "func Hello(name string) string") {
		t.Errorf("html missing signature: %s", page.HTML)
	}
	if len(page.Snippets) != 1 || page.Snippets[0].Kind != "signature" {
		t.Errorf("snippets=%+v", page.Snippets)
	}
}

func TestRender_DocDirective(t *testing.T) {
	l, srcFS := fixtureLoaded(t)
	md := "```codebase-doc sym=example.com/foo.Hello\n```\n"
	page, err := Render("p", []byte(md), l, srcFS)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.HTML, "Hello greets.") {
		t.Errorf("html missing doc: %s", page.HTML)
	}
}

func TestRender_MissingSymbol_ReportsError(t *testing.T) {
	l, srcFS := fixtureLoaded(t)
	md := "```codebase-snippet sym=example.com/foo.Nope\n```\n"
	page, err := Render("p", []byte(md), l, srcFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Errors) == 0 {
		t.Fatal("expected error for missing symbol")
	}
	if !strings.Contains(page.HTML, "doc error") {
		t.Errorf("expected inline marker, got: %s", page.HTML)
	}
}

func TestFirstH1(t *testing.T) {
	if got := firstH1([]byte("# Hello\n\nblah")); got != "Hello" {
		t.Errorf("firstH1=%q", got)
	}
}
