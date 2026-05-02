// Package doc wires the `codebase-browser doc render` command.
package doc

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/codebase-browser/internal/browser"
	"github.com/go-go-golems/codebase-browser/internal/docs"
)

type RenderCommand struct {
	*cmds.CommandDescription
}

type RenderSettings struct {
	Input  string `glazed:"input"`
	Pages  string `glazed:"pages"`
	Source string `glazed:"source"`
	Check  bool   `glazed:"check"`
}

func NewRenderCommand() (*RenderCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	cmdSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"render",
		cmds.WithShort("Render doc pages against the index (resolve codebase-* snippets)"),
		cmds.WithLong(`Walk the --pages directory, render each *.md page through the live-snippet
renderer, and emit one row per page with title / snippet count / error count.
With --check non-zero errors make the command exit non-zero (CI gate).`),
		cmds.WithFlags(
			fields.New("input", fields.TypeString,
				fields.WithDefault("internal/indexfs/embed/index.json"),
				fields.WithHelp("Path to index.json")),
			fields.New("pages", fields.TypeString,
				fields.WithDefault("internal/docs/embed/pages"),
				fields.WithHelp("Directory with *.md doc pages")),
			fields.New("source", fields.TypeString,
				fields.WithDefault("."),
				fields.WithHelp("Module root used as the source FS")),
			fields.New("check", fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Fail (non-zero exit) if any doc page has resolution errors")),
		),
		cmds.WithSections(glazedSection, cmdSettingsSection),
	)
	return &RenderCommand{CommandDescription: desc}, nil
}

func (c *RenderCommand) RunIntoGlazeProcessor(
	ctx context.Context, vals *values.Values, gp middlewares.Processor,
) error {
	s := &RenderSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	loaded, err := browser.LoadFromFile(s.Input)
	if err != nil {
		return err
	}
	pagesFS := os.DirFS(s.Pages)
	sourceFS := os.DirFS(s.Source)

	metas, err := docs.ListPages(pagesFS)
	if err != nil {
		return err
	}
	totalErrs := 0
	for _, m := range metas {
		data, err := fs.ReadFile(pagesFS, m.Path)
		if err != nil {
			return err
		}
		page, err := docs.Render(m.Slug, data, loaded, sourceFS)
		if err != nil {
			return err
		}
		totalErrs += len(page.Errors)
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("slug", page.Slug),
			types.MRP("title", page.Title),
			types.MRP("snippets", len(page.Snippets)),
			types.MRP("errors", len(page.Errors)),
		)); err != nil {
			return err
		}
	}
	if s.Check && totalErrs > 0 {
		return fmt.Errorf("doc render: %d error(s) across %d page(s)", totalErrs, len(metas))
	}
	return nil
}

func Register(root *cobra.Command) error {
	group := &cobra.Command{Use: "doc", Short: "Documentation rendering"}
	r, err := NewRenderCommand()
	if err != nil {
		return err
	}
	cobraCmd, err := cli.BuildCobraCommandFromCommand(r)
	if err != nil {
		return err
	}
	group.AddCommand(cobraCmd)
	root.AddCommand(group)
	return nil
}
