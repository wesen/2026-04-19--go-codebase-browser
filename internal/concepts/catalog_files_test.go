package concepts

import "testing"

func TestLoadRepositoryConcepts(t *testing.T) {
	catalog, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog() error = %v", err)
	}
	for _, path := range []string{
		"packages/package-counts",
		"symbols/exported-functions",
		"symbols/most-referenced",
		"refs/refs-for-symbol",
	} {
		if catalog.ByPath[path] == nil {
			t.Fatalf("concept %q not found; loaded %d concepts", path, len(catalog.Concepts))
		}
	}
}
