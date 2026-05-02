package index

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"

	"github.com/wesen/codebase-browser/internal/indexer"
)

type BuildCommand struct {
	*cmds.CommandDescription
}

type BuildSettings struct {
	ModuleRoot   string   `glazed:"module-root"`
	Patterns     []string `glazed:"patterns"`
	IndexPath    string   `glazed:"index-path"`
	Pretty       bool     `glazed:"pretty"`
	IncludeTests bool     `glazed:"include-tests"`
	Lang         string   `glazed:"lang"`
	TSModuleRoot string   `glazed:"ts-module-root"`
	TSIndexPath  string   `glazed:"ts-index-path"`
}

func NewBuildCommand() (*BuildCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	cmdSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"build",
		cmds.WithShort("Build index.json from Go (and optionally TypeScript) source"),
		cmds.WithLong(`Walk the Go module at --module-root, and/or the TypeScript module at
--ts-module-root, and emit a merged JSON index to --index-path.

--lang selects the extractors to run:
  go    Go only (default, backward-compatible).
  ts    TypeScript only. Runs build-ts-index to produce index-ts.json.
  auto  Both extractors, merged into a single index.

Examples:
  codebase-browser index build
  codebase-browser index build --lang ts --ts-module-root ui
  codebase-browser index build --lang auto --ts-module-root ui
`),
		cmds.WithFlags(
			fields.New("module-root", fields.TypeString,
				fields.WithDefault("."),
				fields.WithHelp("Path to the Go module root (contains go.mod)")),
			fields.New("patterns", fields.TypeStringList,
				fields.WithDefault([]string{
					"./cmd/...",
					"./internal/browser",
					"./internal/docs",
					"./internal/indexer",
					"./internal/indexfs",
					"./internal/sourcefs",
				}),
				fields.WithHelp("Package patterns passed to go/packages.Load")),
			fields.New("index-path", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index.json"),
				fields.WithHelp("Output path for the merged index.json")),
			fields.New("pretty", fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Indent JSON output")),
			fields.New("include-tests", fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Include *_test.go files")),
			fields.New("lang", fields.TypeString,
				fields.WithDefault("go"),
				fields.WithChoices("go", "ts", "auto"),
				fields.WithHelp("Which extractors to run")),
			fields.New("ts-module-root", fields.TypeString,
				fields.WithDefault("ui"),
				fields.WithHelp("TypeScript module root (relative to --module-root)")),
			fields.New("ts-index-path", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index-ts.json"),
				fields.WithHelp("Where build-ts-index writes its intermediate JSON")),
		),
		cmds.WithSections(glazedSection, cmdSettingsSection),
	)
	return &BuildCommand{CommandDescription: desc}, nil
}

func (c *BuildCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	s := &BuildSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	var parts []*indexer.Index

	if s.Lang == "go" || s.Lang == "auto" {
		goIdx, err := indexer.NewGoExtractor().Extract(ctx, indexer.ExtractOptions{
			ModuleRoot:   s.ModuleRoot,
			Patterns:     s.Patterns,
			IncludeTests: s.IncludeTests,
		})
		if err != nil {
			return fmt.Errorf("go extractor: %w", err)
		}
		parts = append(parts, goIdx)
	}

	if s.Lang == "ts" || s.Lang == "auto" {
		tsIdx, err := runTSExtractor(ctx, s)
		if err != nil {
			return fmt.Errorf("ts extractor: %w", err)
		}
		parts = append(parts, tsIdx)
	}

	var merged *indexer.Index
	if len(parts) == 1 {
		merged = parts[0]
	} else {
		m, err := indexer.Merge(parts)
		if err != nil {
			return fmt.Errorf("merge: %w", err)
		}
		merged = m
	}

	if err := indexer.Write(merged, s.IndexPath, s.Pretty); err != nil {
		return err
	}

	goCount, tsCount := countByLanguage(merged)
	row := types.NewRow(
		types.MRP("output", s.IndexPath),
		types.MRP("module", merged.Module),
		types.MRP("packages", len(merged.Packages)),
		types.MRP("files", len(merged.Files)),
		types.MRP("symbols", len(merged.Symbols)),
		types.MRP("go", goCount),
		types.MRP("ts", tsCount),
	)
	return gp.AddRow(ctx, row)
}

func countByLanguage(idx *indexer.Index) (int, int) {
	var g, t int
	for _, s := range idx.Symbols {
		switch s.Language {
		case "ts":
			t++
		default:
			g++
		}
	}
	return g, t
}

// runTSExtractor shells out to the build-ts-index program so the TS pipeline
// (Dagger or local-pnpm fallback) stays in one place. The intermediate JSON
// is written to s.TSIndexPath, read back, and decoded into an Index.
func runTSExtractor(ctx context.Context, s *BuildSettings) (*indexer.Index, error) {
	absTS := s.TSModuleRoot
	if !filepath.IsAbs(absTS) {
		absTS = filepath.Join(s.ModuleRoot, s.TSModuleRoot)
	}
	absTS, _ = filepath.Abs(absTS)
	absOut, _ := filepath.Abs(s.TSIndexPath)

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/build-ts-index")
	cmd.Dir = s.ModuleRoot
	cmd.Env = append(os.Environ(),
		"TS_MODULE_ROOT="+absTS,
		"TS_INDEX_OUT="+absOut,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run build-ts-index: %w", err)
	}

	data, err := os.ReadFile(absOut)
	if err != nil {
		return nil, fmt.Errorf("read ts index: %w", err)
	}
	var idx indexer.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("decode ts index: %w", err)
	}
	return &idx, nil
}
