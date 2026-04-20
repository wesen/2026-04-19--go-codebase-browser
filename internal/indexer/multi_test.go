package indexer

import (
	"testing"
)

func mkIndex(lang, module string, syms []string) *Index {
	idx := &Index{Version: "1", Module: module}
	pkgID := "pkg:" + module
	idx.Packages = []Package{{ID: pkgID, ImportPath: module, Name: module, Language: lang}}
	for _, name := range syms {
		id := "sym:" + module + ".func." + name
		idx.Symbols = append(idx.Symbols, Symbol{
			ID: id, Name: name, Kind: "func",
			PackageID: pkgID, FileID: "file:" + module + ".go",
			Language: lang,
		})
	}
	idx.Files = []File{{
		ID: "file:" + module + ".go", Path: module + ".go",
		PackageID: pkgID, Language: lang,
	}}
	return idx
}

func TestMerge_TwoLanguages(t *testing.T) {
	goIdx := mkIndex("go", "example.com/foo", []string{"Alpha", "Beta"})
	tsIdx := mkIndex("ts", "fixture-ts/src", []string{"gamma", "delta"})
	merged, err := Merge([]*Index{goIdx, tsIdx})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Packages) != 2 {
		t.Errorf("packages=%d want 2", len(merged.Packages))
	}
	if len(merged.Symbols) != 4 {
		t.Errorf("symbols=%d want 4", len(merged.Symbols))
	}
	if merged.Module != "example.com/foo+fixture-ts/src" {
		t.Errorf("module=%q", merged.Module)
	}
	// Deterministic sort: packages ordered by importPath.
	if merged.Packages[0].ImportPath != "example.com/foo" {
		t.Errorf("pkg[0]=%q, sort broken", merged.Packages[0].ImportPath)
	}
}

func TestMerge_DuplicateSymbolIDIsError(t *testing.T) {
	a := mkIndex("go", "example.com/foo", []string{"Alpha"})
	b := mkIndex("ts", "example.com/foo", []string{"Alpha"}) // same pkgID + symID
	_, err := Merge([]*Index{a, b})
	if err == nil {
		t.Fatal("expected duplicate-id error")
	}
}

func TestMerge_NilParts(t *testing.T) {
	a := mkIndex("go", "example.com/foo", []string{"Alpha"})
	merged, err := Merge([]*Index{nil, a, nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Symbols) != 1 {
		t.Errorf("symbols=%d", len(merged.Symbols))
	}
}

func TestGoExtractorStampsLanguage(t *testing.T) {
	e := NewGoExtractor()
	if e.Language() != "go" {
		t.Fatalf("Language=%q", e.Language())
	}
	// Re-run the existing fixture test via the interface so stamping is
	// observable end-to-end without re-deriving the full extract.
	idx, err := Extract(ExtractOptions{ModuleRoot: "testdata/fixture", Patterns: []string{"./..."}})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range idx.Packages {
		if p.Language != "go" {
			t.Errorf("package %s: lang=%q", p.ID, p.Language)
		}
	}
	for _, s := range idx.Symbols {
		if s.Language != "go" {
			t.Errorf("symbol %s: lang=%q", s.ID, s.Language)
		}
	}
}
