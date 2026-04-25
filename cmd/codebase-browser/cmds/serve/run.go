// Package serve wires the `codebase-browser serve` command.
package serve

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/concepts"
	"github.com/wesen/codebase-browser/internal/indexfs"
	"github.com/wesen/codebase-browser/internal/server"
	"github.com/wesen/codebase-browser/internal/sourcefs"
	cbsqlite "github.com/wesen/codebase-browser/internal/sqlite"
	"github.com/wesen/codebase-browser/internal/web"

	"github.com/wesen/codebase-browser/internal/history"
)

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	Addr          string `glazed:"addr"`
	DBPath        string `glazed:"db"`
	HistoryDBPath string `glazed:"history-db"`
	RepoRoot      string `glazed:"repo-root"`
}

func NewServeCommand() (*ServeCommand, error) {
	cmdSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Run the embedded codebase browser web server"),
		cmds.WithLong(`Serve the React SPA + /api/* routes backed by the embedded index.json
and embedded source tree.

Examples:
  codebase-browser serve --addr :3001
`),
		cmds.WithFlags(
			fields.New("addr", fields.TypeString,
				fields.WithDefault(":3001"),
				fields.WithHelp("Bind address")),
			fields.New("db", fields.TypeString,
				fields.WithDefault("internal/sqlite/embed/codebase.db"),
				fields.WithHelp("Path to codebase.db for structured query concepts")),
			fields.New("history-db", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Path to history.db for git-aware codebase history")),
			fields.New("repo-root", fields.TypeString,
				fields.WithDefault("."),
				fields.WithHelp("Path to git repo root for reading file contents at specific commits")),
		),
		cmds.WithSections(cmdSettingsSection),
	)
	return &ServeCommand{CommandDescription: desc}, nil
}

// Run implements cmds.BareCommand.
func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
	s := &ServeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	data := indexfs.Bytes()
	if len(data) == 0 {
		return errors.New("index.json not found; run `codebase-browser index build` or `make build` first")
	}
	loaded, err := browser.LoadFromBytes(data)
	if err != nil {
		return fmt.Errorf("load index: %w", err)
	}

	catalog, err := concepts.LoadConfiguredCatalog(nil)
	if err != nil {
		log.Warn().Err(err).Msg("could not load configured concept repositories; falling back to embedded concepts")
		catalog, err = concepts.LoadEmbeddedCatalog()
		if err != nil {
			return fmt.Errorf("load embedded concepts: %w", err)
		}
	}

	sqliteStore, err := cbsqlite.Open(s.DBPath)
	if err != nil {
		log.Warn().Err(err).Str("db", s.DBPath).Msg("structured query SQLite DB unavailable; concept execution API will be disabled")
	}
	defer func() {
		if sqliteStore != nil {
			_ = sqliteStore.Close()
		}
	}()

	srv := server.New(loaded, sourcefs.FS(), web.FS(), sqliteStore, catalog)

	// Optionally open history DB.
	if s.HistoryDBPath != "" {
		histStore, err := history.Open(s.HistoryDBPath)
		if err != nil {
			log.Warn().Err(err).Str("history-db", s.HistoryDBPath).Msg("history DB unavailable; history API will be disabled")
		} else {
			srv.History = histStore
			srv.RepoRoot = s.RepoRoot
			defer func() { _ = histStore.Close() }()
		}
	}
	h := srv.Handler()

	log.Info().Str("addr", s.Addr).
		Int("packages", len(loaded.Index.Packages)).
		Int("files", len(loaded.Index.Files)).
		Int("symbols", len(loaded.Index.Symbols)).
		Msg("codebase-browser serving")

	httpSrv := &http.Server{Addr: s.Addr, Handler: h}
	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		_ = httpSrv.Close()
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// Register attaches `serve` to the root command.
func Register(root *cobra.Command) error {
	srv, err := NewServeCommand()
	if err != nil {
		return err
	}
	cobraCmd, err := cli.BuildCobraCommandFromCommand(srv)
	if err != nil {
		return err
	}
	root.AddCommand(cobraCmd)
	return nil
}
