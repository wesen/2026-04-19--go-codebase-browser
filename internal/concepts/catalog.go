package concepts

import (
	"fmt"
	"io/fs"
	"os"
	"path"
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
		clean := filepath.Clean(dir)
		roots = append(roots, SourceRoot{
			Name:    filepath.Base(clean),
			FS:      os.DirFS(clean),
			RootDir: ".",
		})
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
		rootFS := root.FS
		if rootFS == nil {
			continue
		}
		rootDir := strings.TrimSpace(root.RootDir)
		if rootDir == "" {
			rootDir = "."
		}
		if _, err := fs.Stat(rootFS, rootDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat concept root %s:%s: %w", root.Name, rootDir, err)
		}
		if err := fs.WalkDir(rootFS, rootDir, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(filePath), ".sql") {
				return nil
			}
			data, err := fs.ReadFile(rootFS, filePath)
			if err != nil {
				return err
			}
			if !LooksLikeConceptSQL(data) {
				return nil
			}
			spec, err := ParseSQLConcept(filePath, data)
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(path.Clean(filePath), "./")
			if rootDir != "." {
				rel = strings.TrimPrefix(strings.TrimPrefix(rel, strings.TrimPrefix(path.Clean(rootDir), "./")), "/")
			}
			concept := Compile(spec, rel, sourcePath(root.Name, rel), root.Name)
			if _, exists := catalog.ByPath[concept.Path]; exists {
				return nil
			}
			catalog.ByPath[concept.Path] = concept
			if _, exists := catalog.ByName[concept.Name]; !exists {
				catalog.ByName[concept.Name] = concept
			}
			catalog.Concepts = append(catalog.Concepts, concept)
			return nil
		}); err != nil {
			return nil, fmt.Errorf("load concept root %s: %w", root.Name, err)
		}
	}
	sort.Slice(catalog.Concepts, func(i, j int) bool {
		return catalog.Concepts[i].Path < catalog.Concepts[j].Path
	})
	return catalog, nil
}

func Compile(spec *ConceptSpec, relPath, sourcePath, sourceRoot string) *Concept {
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
		SourceRoot: strings.TrimSpace(sourceRoot),
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

func sourcePath(rootName, rel string) string {
	rootName = strings.TrimSpace(rootName)
	rel = strings.TrimPrefix(path.Clean(rel), "./")
	if rootName == "" {
		return rel
	}
	return rootName + ":" + rel
}
