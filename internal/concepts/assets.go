package concepts

import builtinconcepts "github.com/go-go-golems/codebase-browser/concepts"

func EmbeddedSourceRoot() SourceRoot {
	return SourceRoot{
		Name:    "embedded",
		FS:      builtinconcepts.Files,
		RootDir: ".",
	}
}

func LoadEmbeddedCatalog() (*Catalog, error) {
	return LoadCatalog([]SourceRoot{EmbeddedSourceRoot()})
}
