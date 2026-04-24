package wasm

import (
	"encoding/json"
	"testing"
)

func TestFindSymbols(t *testing.T) {
	idx := &Index{
		Packages: []Package{{ID: "pkg1", ImportPath: "a/b", Name: "b"}},
		Files:    []File{{ID: "file:a/b.go", Path: "a/b.go", PackageID: "pkg1"}},
		Symbols: []Symbol{
			{ID: "sym:pkg1.func.Merge", Kind: "func", Name: "Merge", PackageID: "pkg1"},
			{ID: "sym:pkg1.func.Extract", Kind: "func", Name: "Extract", PackageID: "pkg1"},
			{ID: "sym:pkg1.type.Loaded", Kind: "type", Name: "Loaded", PackageID: "pkg1"},
		},
	}

	ctx := &SearchCtx{Index: idx}
	ctx.bySymbolID = make(map[string]*Symbol)
	for i := range idx.Symbols {
		ctx.bySymbolID[idx.Symbols[i].ID] = &idx.Symbols[i]
	}

	// Empty query matches everything
	data := ctx.FindSymbols("", "")
	var all []*Symbol
	json.Unmarshal(data, &all)
	if len(all) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(all))
	}

	// Substring match
	data = ctx.FindSymbols("Merge", "")
	var merge []*Symbol
	json.Unmarshal(data, &merge)
	if len(merge) != 1 || merge[0].Name != "Merge" {
		t.Fatalf("expected 1 Merge, got %v", merge)
	}

	// Case-insensitive match
	data = ctx.FindSymbols("merge", "")
	json.Unmarshal(data, &merge)
	if len(merge) != 1 {
		t.Fatalf("expected 1 merge (case-insensitive), got %d", len(merge))
	}

	// Kind filter
	data = ctx.FindSymbols("", "func")
	var funcs []*Symbol
	json.Unmarshal(data, &funcs)
	if len(funcs) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(funcs))
	}
}

func TestGetSymbol(t *testing.T) {
	idx := &Index{
		Symbols: []Symbol{
			{ID: "sym:pkg1.func.Merge", Kind: "func", Name: "Merge"},
		},
	}
	ctx := &SearchCtx{Index: idx, bySymbolID: map[string]*Symbol{"sym:pkg1.func.Merge": &idx.Symbols[0]}}

	data := ctx.GetSymbol("sym:pkg1.func.Merge")
	var sym Symbol
	json.Unmarshal(data, &sym)
	if sym.Name != "Merge" {
		t.Fatalf("expected Merge, got %s", sym.Name)
	}

	data = ctx.GetSymbol("missing")
	if string(data) != "null" {
		t.Fatalf("expected null, got %s", string(data))
	}
}
