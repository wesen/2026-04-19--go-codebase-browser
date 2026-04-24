package concepts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleConcept = `/* codebase-browser concept
name: exported-functions
short: List exported functions
long: |
  Finds exported functions by package.
tags: [symbols]
params:
  - name: package
    type: string
    help: Package substring
    default: ""
  - name: limit
    type: int
    help: Maximum rows
    default: 50
*/
SELECT name
FROM symbols
WHERE ({{ sqlString (value "package") }} = '' OR package_id LIKE {{ sqlLike (value "package") }})
LIMIT {{ value "limit" }};
`

func TestParseSQLConcept(t *testing.T) {
	if !LooksLikeConceptSQL([]byte(sampleConcept)) {
		t.Fatalf("LooksLikeConceptSQL returned false")
	}
	spec, err := ParseSQLConcept("symbols/exported-functions.sql", []byte(sampleConcept))
	if err != nil {
		t.Fatalf("ParseSQLConcept() error = %v", err)
	}
	if spec.Name != "exported-functions" || spec.Short == "" || len(spec.Params) != 2 {
		t.Fatalf("unexpected spec: %#v", spec)
	}
}

func TestLoadCatalog(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "symbols"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "symbols", "exported-functions.sql"), []byte(sampleConcept), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadCatalogFromDirs(dir)
	if err != nil {
		t.Fatalf("LoadCatalogFromDirs() error = %v", err)
	}
	if len(catalog.Concepts) != 1 {
		t.Fatalf("concept count = %d, want 1", len(catalog.Concepts))
	}
	concept := catalog.ByPath["symbols/exported-functions"]
	if concept == nil {
		t.Fatalf("concept by path not found")
	}
	if concept.Folder != "symbols" || concept.Name != "exported-functions" {
		t.Fatalf("unexpected concept: %#v", concept)
	}
}

func TestRenderConcept(t *testing.T) {
	spec, err := ParseSQLConcept("symbols/exported-functions.sql", []byte(sampleConcept))
	if err != nil {
		t.Fatal(err)
	}
	concept := Compile(spec, "symbols/exported-functions.sql", "symbols/exported-functions.sql", "test")
	sql, err := RenderConcept(concept, map[string]any{"package": "internal/server", "limit": "10"})
	if err != nil {
		t.Fatalf("RenderConcept() error = %v", err)
	}
	for _, want := range []string{"'internal/server'", "'%internal/server%'", "LIMIT 10"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("rendered SQL missing %q:\n%s", want, sql)
		}
	}
}

func TestHydrateRequiredValue(t *testing.T) {
	concept := &Concept{
		Name:  "refs-for-symbol",
		Short: "Refs for symbol",
		Query: "SELECT 1",
		Params: []Param{{
			Name:     "symbol-id",
			Type:     ParamString,
			Required: true,
		}},
	}
	_, err := HydrateValues(concept, nil)
	if err == nil {
		t.Fatalf("HydrateValues() expected missing required error")
	}
}
