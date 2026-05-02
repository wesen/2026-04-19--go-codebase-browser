package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cbsqlite "github.com/wesen/codebase-browser/internal/sqlite"
)

type options struct {
	dbPath string
	file   string
	format string
}

// Register adds the top-level `query` command.
func Register(root *cobra.Command, conceptRepositoryFlags []string) error {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "query [sql]",
		Short: "Run SQL against a codebase-browser SQLite index",
		Long: `Run ad-hoc SQL against codebase.db.

Examples:
  codebase-browser query "SELECT COUNT(*) AS symbols FROM symbols"
  codebase-browser query -f queries/symbols/exported-functions.sql
  codebase-browser query --format json "SELECT name, kind FROM symbols LIMIT 5"
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.file == "" && len(args) == 0 {
				return fmt.Errorf("provide SQL as an argument or pass --file")
			}
			if opts.file != "" && len(args) > 0 {
				return fmt.Errorf("provide SQL either as an argument or via --file, not both")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sqlText, err := readSQL(opts, args)
			if err != nil {
				return err
			}
			store, err := cbsqlite.Open(opts.dbPath)
			if err != nil {
				return err
			}
			defer func() { _ = store.Close() }()
			return runSQL(ctx, store.DB(), cmd.OutOrStdout(), sqlText, opts.format)
		},
	}
	cmd.PersistentFlags().StringVar(&opts.dbPath, "db", "internal/sqlite/embed/codebase.db", "Path to codebase.db")
	cmd.PersistentFlags().StringVar(&opts.format, "format", "table", "Output format: table or json")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read SQL from a file")
	if err := addConceptCommands(cmd, opts, conceptRepositoryFlags); err != nil {
		return err
	}
	root.AddCommand(cmd)
	return nil
}

func readSQL(opts *options, args []string) (string, error) {
	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return "", fmt.Errorf("read sql file: %w", err)
		}
		return string(data), nil
	}
	return strings.Join(args, " "), nil
}

func runSQL(ctx context.Context, db *sql.DB, out io.Writer, sqlText, format string) error {
	trimmed := strings.TrimSpace(sqlText)
	if trimmed == "" {
		return fmt.Errorf("empty SQL")
	}

	rows, err := db.QueryContext(ctx, trimmed)
	if err != nil {
		if _, execErr := db.ExecContext(ctx, trimmed); execErr != nil {
			return fmt.Errorf("query sql: %w", err)
		}
		_, _ = fmt.Fprintln(out, "ok")
		return nil
	}
	defer func() { _ = rows.Close() }()

	switch format {
	case "", "table":
		return writeTable(out, rows)
	case "json":
		return writeJSON(out, rows)
	default:
		return fmt.Errorf("unsupported format %q (expected table or json)", format)
	}
}

func writeTable(out io.Writer, rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("read columns: %w", err)
	}
	_, _ = fmt.Fprintln(out, strings.Join(cols, "\t"))
	for rows.Next() {
		values, err := scanRow(rows, cols)
		if err != nil {
			return err
		}
		parts := make([]string, len(cols))
		for i, col := range cols {
			parts[i] = fmt.Sprint(values[col])
		}
		_, _ = fmt.Fprintln(out, strings.Join(parts, "\t"))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %w", err)
	}
	return nil
}

func writeJSON(out io.Writer, rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("read columns: %w", err)
	}
	var all []map[string]any
	for rows.Next() {
		row, err := scanRow(rows, cols)
		if err != nil {
			return err
		}
		all = append(all, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %w", err)
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(all)
}

func scanRow(rows *sql.Rows, cols []string) (map[string]any, error) {
	raw := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}
	out := make(map[string]any, len(cols))
	for i, col := range cols {
		switch v := raw[i].(type) {
		case []byte:
			out[col] = string(v)
		case nil:
			out[col] = ""
		default:
			out[col] = v
		}
	}
	return out, nil
}
