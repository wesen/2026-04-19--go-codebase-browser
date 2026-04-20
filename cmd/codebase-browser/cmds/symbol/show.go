package symbol

import (
	"context"
	"fmt"

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

type ShowCommand struct {
	*cmds.CommandDescription
}

type ShowSettings struct {
	Input string `glazed:"input"`
	ID    string `glazed:"id"`
}

func NewShowCommand() (*ShowCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	cmdSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"show",
		cmds.WithShort("Show a single symbol by ID"),
		cmds.WithFlags(
			fields.New("input", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index.json"),
				fields.WithHelp("Path to index.json")),
			fields.New("id", fields.TypeString,
				fields.WithRequired(true),
				fields.WithHelp("Symbol ID (e.g. sym:pkg/foo.func.Bar)")),
		),
		cmds.WithSections(glazedSection, cmdSettingsSection),
	)
	return &ShowCommand{CommandDescription: desc}, nil
}

func (c *ShowCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	s := &ShowSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	loaded, err := browser.LoadFromFile(s.Input)
	if err != nil {
		return err
	}
	sym, ok := loaded.Symbol(s.ID)
	if !ok {
		return fmt.Errorf("symbol not found: %s", s.ID)
	}
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("id", sym.ID),
		types.MRP("kind", sym.Kind),
		types.MRP("name", sym.Name),
		types.MRP("packageId", sym.PackageID),
		types.MRP("fileId", sym.FileID),
		types.MRP("signature", sym.Signature),
		types.MRP("doc", sym.Doc),
		types.MRP("startLine", sym.Range.StartLine),
		types.MRP("endLine", sym.Range.EndLine),
		types.MRP("exported", sym.Exported),
	))
}
