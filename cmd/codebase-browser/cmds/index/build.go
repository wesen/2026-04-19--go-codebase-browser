package index

import (
	"context"

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
		cmds.WithShort("Build index.json from Go source"),
		cmds.WithLong(`Walk the Go module at --module-root, load packages matching --patterns,
and emit a JSON index to --output.

Examples:
  codebase-browser index build --module-root . --patterns ./...
  codebase-browser index build --output internal/indexfs/embed/index.json
`),
		cmds.WithFlags(
			fields.New("module-root", fields.TypeString,
				fields.WithDefault("."),
				fields.WithHelp("Path to the module root (contains go.mod)")),
			fields.New("patterns", fields.TypeStringList,
				fields.WithDefault([]string{"./..."}),
				fields.WithHelp("Package patterns passed to go/packages.Load")),
			fields.New("index-path", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index.json"),
				fields.WithHelp("Output path for index.json")),
			fields.New("pretty", fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Indent JSON output")),
			fields.New("include-tests", fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Include *_test.go files")),
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

	idx, err := indexer.Extract(indexer.ExtractOptions{
		ModuleRoot:   s.ModuleRoot,
		Patterns:     s.Patterns,
		IncludeTests: s.IncludeTests,
	})
	if err != nil {
		return err
	}
	if err := indexer.Write(idx, s.IndexPath, s.Pretty); err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("output", s.IndexPath),
		types.MRP("module", idx.Module),
		types.MRP("packages", len(idx.Packages)),
		types.MRP("files", len(idx.Files)),
		types.MRP("symbols", len(idx.Symbols)),
	)
	return gp.AddRow(ctx, row)
}
