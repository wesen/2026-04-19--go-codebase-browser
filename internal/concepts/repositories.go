package concepts

import (
	"encoding/csv"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConceptRepositoriesEnvVar = "CODEBASE_BROWSER_CONCEPT_REPOSITORIES"
	ConceptRepositoryFlagName = "concept-repository"
)

func LoadConfiguredCatalog(flagPaths []string) (*Catalog, error) {
	return LoadCatalog(SourceRootsFromPaths(collectRepositoryPaths(flagPaths)))
}

func collectRepositoryPaths(flagPaths []string) []string {
	repositoryPaths := append([]string{}, normalizeRepositoryPaths(flagPaths)...)
	repositoryPaths = append(repositoryPaths, repositoriesFromEnv()...)
	return normalizeRepositoryPaths(repositoryPaths)
}

func repositoriesFromEnv() []string {
	value, ok := os.LookupEnv(ConceptRepositoriesEnvVar)
	if !ok || value == "" {
		return nil
	}
	return normalizeRepositoryPaths(filepath.SplitList(value))
}

func normalizeRepositoryPaths(paths []string) []string {
	ret := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		ret = append(ret, path)
	}
	return ret
}

func SourceRootsFromPaths(paths []string) []SourceRoot {
	roots := make([]SourceRoot, 0, len(paths)+1)
	for _, path := range normalizeRepositoryPaths(paths) {
		dir := filepath.Clean(os.ExpandEnv(path))
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			continue
		}
		roots = append(roots, SourceRoot{
			Name:    dir,
			FS:      os.DirFS(dir),
			RootDir: ".",
		})
	}
	roots = append(roots, EmbeddedSourceRoot())
	return roots
}

func ExtractRepositoryFlagValuesFromArgs(args []string) []string {
	values := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return normalizeRepositoryPaths(values)
		case arg == "--"+ConceptRepositoryFlagName:
			if i+1 >= len(args) {
				continue
			}
			values = append(values, splitRepositoryFlagValue(args[i+1])...)
			i++
		case strings.HasPrefix(arg, "--"+ConceptRepositoryFlagName+"="):
			values = append(values, splitRepositoryFlagValue(strings.TrimPrefix(arg, "--"+ConceptRepositoryFlagName+"="))...)
		}
	}
	return normalizeRepositoryPaths(values)
}

func splitRepositoryFlagValue(value string) []string {
	if value == "" {
		return nil
	}
	reader := csv.NewReader(strings.NewReader(value))
	values, err := reader.Read()
	if err != nil {
		return []string{value}
	}
	return values
}

func dirSourceRoot(name, dir string) SourceRoot {
	return SourceRoot{Name: name, FS: os.DirFS(dir), RootDir: "."}
}

func fsSourceRoot(name string, filesystem fs.FS, rootDir string) SourceRoot {
	return SourceRoot{Name: name, FS: filesystem, RootDir: rootDir}
}
