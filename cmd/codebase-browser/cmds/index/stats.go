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

	"github.com/go-go-golems/codebase-browser/internal/browser"
)

type StatsCommand struct {
	*cmds.CommandDescription
}

type StatsSettings struct {
	Input string `glazed:"input"`
}

func NewStatsCommand() (*StatsCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	cmdSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"stats",
		cmds.WithShort("Print counts by kind and package from index.json"),
		cmds.WithFlags(
			fields.New("input", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index.json"),
				fields.WithHelp("Path to index.json")),
		),
		cmds.WithSections(glazedSection, cmdSettingsSection),
	)
	return &StatsCommand{CommandDescription: desc}, nil
}

func (c *StatsCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	s := &StatsSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	loaded, err := browser.LoadFromFile(s.Input)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, sym := range loaded.Index.Symbols {
		counts[sym.Kind]++
	}
	for kind, n := range counts {
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("kind", kind),
			types.MRP("count", n),
		)); err != nil {
			return err
		}
	}
	return nil
}
