package concepts

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func LoadCatalogFromDirs(dirs ...string) (*Catalog, error) {
	roots := make([]SourceRoot, 0, len(dirs))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		roots = append(roots, SourceRoot{Name: filepath.Base(dir), RootDir: dir})
	}
	return LoadCatalog(roots)
}

func LoadCatalog(roots []SourceRoot) (*Catalog, error) {
	catalog := &Catalog{
		Concepts: []*Concept{},
		ByPath:   map[string]*Concept{},
		ByName:   map[string]*Concept{},
	}
	for _, root := range roots {
		rootDir := root.RootDir
		if rootDir == "" {
			rootDir = "."
		}
		if _, err := os.Stat(rootDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat concept root %s: %w", rootDir, err)
		}
		if err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".sql") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !LooksLikeConceptSQL(data) {
				return nil
			}
			spec, err := ParseSQLConcept(path, data)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			concept := Compile(spec, rel, path)
			if _, exists := catalog.ByPath[concept.Path]; exists {
				return fmt.Errorf("duplicate concept path %q", concept.Path)
			}
			catalog.ByPath[concept.Path] = concept
			if _, exists := catalog.ByName[concept.Name]; !exists {
				catalog.ByName[concept.Name] = concept
			}
			catalog.Concepts = append(catalog.Concepts, concept)
			return nil
		}); err != nil {
			return nil, fmt.Errorf("load concept root %s: %w", rootDir, err)
		}
	}
	sort.Slice(catalog.Concepts, func(i, j int) bool {
		return catalog.Concepts[i].Path < catalog.Concepts[j].Path
	})
	return catalog, nil
}

func Compile(spec *ConceptSpec, relPath, sourcePath string) *Concept {
	relPath = filepath.ToSlash(relPath)
	folder := filepath.ToSlash(filepath.Dir(relPath))
	if folder == "." {
		folder = ""
	}
	return &Concept{
		Name:       strings.TrimSpace(spec.Name),
		Folder:     folder,
		Path:       conceptPath(folder, spec.Name),
		Short:      strings.TrimSpace(spec.Short),
		Long:       strings.TrimSpace(spec.Long),
		Tags:       append([]string(nil), spec.Tags...),
		Params:     append([]Param(nil), spec.Params...),
		Query:      strings.TrimSpace(spec.Query),
		SourcePath: sourcePath,
	}
}

func conceptPath(folder, name string) string {
	folder = strings.Trim(strings.TrimSpace(filepath.ToSlash(folder)), "/")
	name = strings.TrimSpace(name)
	if folder == "" {
		return name
	}
	return folder + "/" + name
}
