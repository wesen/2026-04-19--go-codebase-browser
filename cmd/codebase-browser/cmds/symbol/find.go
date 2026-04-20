package symbol

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

	"github.com/wesen/codebase-browser/internal/browser"
)

type FindCommand struct {
	*cmds.CommandDescription
}

type FindSettings struct {
	Input string `glazed:"input"`
	Name  string `glazed:"name"`
	Kind  string `glazed:"kind"`
}

func NewFindCommand() (*FindCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	cmdSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"find",
		cmds.WithShort("Find symbols by name substring and optional kind"),
		cmds.WithFlags(
			fields.New("input", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index.json"),
				fields.WithHelp("Path to index.json")),
			fields.New("name", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Substring to match (case-insensitive)")),
			fields.New("kind", fields.TypeString,
				fields.WithDefault(""),
				fields.WithChoices("", "func", "method", "type", "iface", "struct", "alias", "const", "var"),
				fields.WithHelp("Filter by symbol kind")),
		),
		cmds.WithSections(glazedSection, cmdSettingsSection),
	)
	return &FindCommand{CommandDescription: desc}, nil
}

func (c *FindCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	s := &FindSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	loaded, err := browser.LoadFromFile(s.Input)
	if err != nil {
		return err
	}
	for _, sym := range loaded.FindSymbols(s.Name, s.Kind) {
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("id", sym.ID),
			types.MRP("kind", sym.Kind),
			types.MRP("name", sym.Name),
			types.MRP("packageId", sym.PackageID),
			types.MRP("fileId", sym.FileID),
			types.MRP("exported", sym.Exported),
			types.MRP("startLine", sym.Range.StartLine),
		)); err != nil {
			return err
		}
	}
	return nil
}
