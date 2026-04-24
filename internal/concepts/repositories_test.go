package concepts

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSourceRootsFromPaths_PutsEmbeddedLastAndSkipsMissingDirs(t *testing.T) {
	repo := t.TempDir()
	roots := SourceRootsFromPaths([]string{"/missing/repo", repo})
	if len(roots) != 2 {
		t.Fatalf("len(roots) = %d, want 2", len(roots))
	}
	if roots[0].Name != repo {
		t.Fatalf("roots[0].Name = %q, want %q", roots[0].Name, repo)
	}
	if roots[1].Name != "embedded" {
		t.Fatalf("roots[1].Name = %q, want embedded", roots[1].Name)
	}
}

func TestExtractRepositoryFlagValuesFromArgs_SupportsSplitAndEqualsForms(t *testing.T) {
	args := []string{
		"query",
		"commands",
		"--concept-repository", "./concepts/team",
		"--concept-repository=./concepts/shared",
		"symbols",
	}
	got := ExtractRepositoryFlagValuesFromArgs(args)
	want := []string{"./concepts/team", "./concepts/shared"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExtractRepositoryFlagValuesFromArgs_SplitsCommaSeparatedValues(t *testing.T) {
	args := []string{
		"query",
		"commands",
		"--concept-repository", "./concepts/team,./concepts/shared",
		"--concept-repository=./concepts/extra,./concepts/override",
	}
	got := ExtractRepositoryFlagValuesFromArgs(args)
	want := []string{"./concepts/team", "./concepts/shared", "./concepts/extra", "./concepts/override"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLoadConfiguredCatalog_ExternalRepositoryOverridesEmbedded(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "symbols"), 0o755); err != nil {
		t.Fatal(err)
	}
	const override = `/* codebase-browser concept
name: exported-functions
short: Override exported functions
tags: [symbols]
*/
SELECT 'override' AS source;
`
	if err := os.WriteFile(filepath.Join(repo, "symbols", "exported-functions.sql"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadConfiguredCatalog([]string{repo})
	if err != nil {
		t.Fatalf("LoadConfiguredCatalog() error = %v", err)
	}
	concept := catalog.ByPath["symbols/exported-functions"]
	if concept == nil {
		t.Fatalf("override concept not found")
	}
	if concept.Short != "Override exported functions" {
		t.Fatalf("concept.Short = %q, want override short description", concept.Short)
	}
	if concept.SourceRoot != repo {
		t.Fatalf("concept.SourceRoot = %q, want %q", concept.SourceRoot, repo)
	}
}

func TestLoadConfiguredCatalog_LoadsRepositoriesFromEnv(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "team"), 0o755); err != nil {
		t.Fatal(err)
	}
	const teamConcept = `/* codebase-browser concept
name: team-check
short: Team concept
*/
SELECT 'team' AS source;
`
	if err := os.WriteFile(filepath.Join(repo, "team", "team-check.sql"), []byte(teamConcept), 0o644); err != nil {
		t.Fatal(err)
	}
	oldValue, hadValue := os.LookupEnv(ConceptRepositoriesEnvVar)
	if err := os.Setenv(ConceptRepositoriesEnvVar, repo); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if hadValue {
			_ = os.Setenv(ConceptRepositoriesEnvVar, oldValue)
		} else {
			_ = os.Unsetenv(ConceptRepositoriesEnvVar)
		}
	}()

	catalog, err := LoadConfiguredCatalog(nil)
	if err != nil {
		t.Fatalf("LoadConfiguredCatalog() error = %v", err)
	}
	if catalog.ByPath["team/team-check"] == nil {
		t.Fatalf("env-loaded concept not found")
	}
}
