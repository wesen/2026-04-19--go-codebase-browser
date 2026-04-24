package static

import (
	"encoding/json"
	"testing"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/indexer"
)

func TestBuildSearchIndexFast(t *testing.T) {
	idx := &indexer.Index{
		Symbols: []indexer.Symbol{
			{ID: "sym:pkg1.func.Merge", Kind: "func", Name: "Merge"},
			{ID: "sym:pkg1.func.Extract", Kind: "func", Name: "Extract"},
			{ID: "sym:pkg1.type.Loaded", Kind: "type", Name: "Loaded"},
		},
	}
	data, _ := json.Marshal(idx)
	loaded, _ := browser.LoadFromBytes(data)

	searchIdx := BuildSearchIndexFast(loaded)

	// "merge" should map to Merge
	if ids, ok := searchIdx["merge"]; !ok || len(ids) != 1 || ids[0] != "sym:pkg1.func.Merge" {
		t.Fatalf("expected merge → [sym:pkg1.func.Merge], got %v", ids)
	}

	// "m" prefix should map to Merge
	if ids, ok := searchIdx["m"]; !ok || len(ids) != 1 {
		t.Fatalf("expected m prefix to match Merge, got %v", ids)
	}
}

func TestBuildXrefIndex(t *testing.T) {
	idx := &indexer.Index{
		Symbols: []indexer.Symbol{
			{ID: "sym:a.func.Foo", Kind: "func", Name: "Foo"},
			{ID: "sym:a.func.Bar", Kind: "func", Name: "Bar"},
		},
		Refs: []indexer.Ref{
			{FromSymbolID: "sym:a.func.Foo", ToSymbolID: "sym:a.func.Bar", Kind: "call", FileID: "file:a.go", Range: indexer.Range{StartLine: 10, EndLine: 10}},
			{FromSymbolID: "sym:a.func.Bar", ToSymbolID: "sym:a.func.Foo", Kind: "call", FileID: "file:a.go", Range: indexer.Range{StartLine: 20, EndLine: 20}},
		},
	}
	data, _ := json.Marshal(idx)
	loaded, _ := browser.LoadFromBytes(data)

	xrefIdx := BuildXrefIndex(loaded)

	// Foo is usedBy Bar
	fooXref := xrefIdx["sym:a.func.Foo"]
	if fooXref == nil {
		t.Fatal("expected xref for Foo")
	}
	if len(fooXref.UsedBy) != 1 || fooXref.UsedBy[0].FromSymbolID != "sym:a.func.Bar" {
		t.Fatalf("expected Foo usedBy Bar, got %v", fooXref.UsedBy)
	}
	if len(fooXref.Uses) != 1 || fooXref.Uses[0].ToSymbolID != "sym:a.func.Bar" {
		t.Fatalf("expected Foo uses Bar, got %v", fooXref.Uses)
	}

	// Bar is usedBy Foo
	barXref := xrefIdx["sym:a.func.Bar"]
	if barXref == nil {
		t.Fatal("expected xref for Bar")
	}
	if len(barXref.UsedBy) != 1 || barXref.UsedBy[0].FromSymbolID != "sym:a.func.Foo" {
		t.Fatalf("expected Bar usedBy Foo, got %v", barXref.UsedBy)
	}
}
