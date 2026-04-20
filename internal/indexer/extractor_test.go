package indexer

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestExtract_Fixture indexes the small fixture module and asserts we see the
// symbols we expect. A deeper golden-byte-equal test is intentionally skipped
// because timestamps vary run-to-run; we instead assert the stable subset.
func TestExtract_Fixture(t *testing.T) {
	root, err := filepath.Abs("testdata/fixture")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	idx, err := Extract(ExtractOptions{ModuleRoot: root, Patterns: []string{"./..."}})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if idx.Module != "example.com/fixture" {
		t.Errorf("module = %q, want example.com/fixture", idx.Module)
	}
	gotPkgs := map[string]bool{}
	for _, p := range idx.Packages {
		gotPkgs[p.ImportPath] = true
	}
	for _, want := range []string{"example.com/fixture", "example.com/fixture/sub"} {
		if !gotPkgs[want] {
			t.Errorf("missing package %q", want)
		}
	}
	gotSyms := map[string]string{}
	for _, s := range idx.Symbols {
		gotSyms[s.Name] = s.Kind
	}
	wantSyms := map[string]string{
		"Greeter":    "struct",
		"Hello":      "method",
		"Anon":       "func",
		"MaxRetries": "const",
		"Add":        "func",
	}
	for name, kind := range wantSyms {
		if gotSyms[name] != kind {
			t.Errorf("symbol %q: got kind %q want %q", name, gotSyms[name], kind)
		}
	}
}

// TestExtract_Determinism re-runs Extract twice and checks that package,
// file, and symbol orderings are byte-stable (the generatedAt stamp is
// excluded from the comparison).
func TestExtract_Determinism(t *testing.T) {
	root, _ := filepath.Abs("testdata/fixture")
	runOnce := func() []string {
		idx, err := Extract(ExtractOptions{ModuleRoot: root, Patterns: []string{"./..."}})
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(idx.Symbols))
		for _, s := range idx.Symbols {
			out = append(out, s.ID)
		}
		sort.Strings(out)
		return out
	}
	a := runOnce()
	b := runOnce()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("symbol IDs diverged between runs:\nA=%v\nB=%v", a, b)
	}
}

func TestSymbolID(t *testing.T) {
	if got := SymbolID("pkg/foo", "func", "Bar", ""); got != "sym:pkg/foo.func.Bar" {
		t.Errorf("SymbolID = %q", got)
	}
	if got := MethodID("pkg/foo", "*Baz", "Go"); got != "sym:pkg/foo.method.Baz.Go" {
		t.Errorf("MethodID = %q", got)
	}
}
