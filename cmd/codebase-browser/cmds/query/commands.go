package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/go-go-golems/codebase-browser/internal/concepts"
	cbsqlite "github.com/go-go-golems/codebase-browser/internal/sqlite"
)

func addConceptCommands(queryRoot *cobra.Command, opts *options, flagPaths []string) error {
	catalog, err := concepts.LoadConfiguredCatalog(flagPaths)
	if err != nil {
		return err
	}

	commandsRoot := &cobra.Command{
		Use:   "commands",
		Short: "Run repository-backed structured SQL concepts",
		Long: `Run repository-backed structured SQL concepts loaded from the embedded
catalog plus any configured external concept repositories.

Subdirectories in a repository are exposed as nested CLI groups. SQL concept
files map directly to leaf commands, so a file like symbols/exported-functions.sql
becomes:
  codebase-browser query commands symbols exported-functions

Additional repositories can be provided through:
  - environment: ` + concepts.ConceptRepositoriesEnvVar + `
  - repeated CLI flags: --` + concepts.ConceptRepositoryFlagName + ` ./my-concepts

Higher-precedence repositories are mounted first so they can override embedded
built-ins without changing loader behavior.

Use --render-only on a leaf command to preview the rendered SQL without
executing it.`,
	}
	commandsRoot.PersistentFlags().StringSlice(concepts.ConceptRepositoryFlagName, flagPaths, "Repeatable directory flag for additional structured SQL concept repository roots")
	groups := map[string]*cobra.Command{}
	for _, concept := range catalog.Concepts {
		parent := ensureGroup(commandsRoot, groups, concept.Folder)
		parent.AddCommand(newConceptCommand(concept, opts))
	}
	queryRoot.AddCommand(commandsRoot)
	return nil
}

func ensureGroup(root *cobra.Command, groups map[string]*cobra.Command, folder string) *cobra.Command {
	if strings.TrimSpace(folder) == "" {
		return root
	}
	parent := root
	current := ""
	for _, segment := range strings.Split(folder, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if current == "" {
			current = segment
		} else {
			current += "/" + segment
		}
		if group, ok := groups[current]; ok {
			parent = group
			continue
		}
		group := &cobra.Command{
			Use:   segment,
			Short: "Concept group: " + current,
		}
		parent.AddCommand(group)
		groups[current] = group
		parent = group
	}
	return parent
}

func newConceptCommand(concept *concepts.Concept, opts *options) *cobra.Command {
	values := map[string]any{}
	renderOnly := false
	cmd := &cobra.Command{
		Use:   concept.Name,
		Short: concept.Short,
		Long:  concept.Long,
		RunE: func(cmd *cobra.Command, args []string) error {
			rendered, err := concepts.RenderConcept(concept, values)
			if err != nil {
				return err
			}
			if renderOnly {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), rendered)
				return nil
			}
			store, err := cbsqlite.Open(opts.dbPath)
			if err != nil {
				return err
			}
			defer func() { _ = store.Close() }()
			return runSQL(cmd.Context(), store.DB(), cmd.OutOrStdout(), rendered, opts.format)
		},
	}
	cmd.Flags().BoolVar(&renderOnly, "render-only", false, "Print rendered SQL without executing it")
	for _, param := range concept.Params {
		addParamFlag(cmd, values, param)
	}
	return cmd
}

func addParamFlag(cmd *cobra.Command, values map[string]any, param concepts.Param) {
	name := param.Name
	help := param.Help
	if param.Required {
		help = strings.TrimSpace(help + " (required)")
	}
	short := strings.TrimSpace(param.ShortFlag)

	switch param.Type {
	case concepts.ParamInt:
		values[name] = defaultInt(param.Default)
		addValueFlag(cmd.Flags(), name, short, help, &intValue{values: values, name: name})
	case concepts.ParamBool:
		values[name] = defaultBool(param.Default)
		addValueFlag(cmd.Flags(), name, short, help, &boolValue{values: values, name: name})
	case concepts.ParamStringList, concepts.ParamIntList:
		values[name] = defaultStringSlice(param.Default)
		addValueFlag(cmd.Flags(), name, short, help, &stringSliceValue{values: values, name: name})
	case concepts.ParamChoice:
		values[name] = defaultString(param.Default)
		addValueFlag(cmd.Flags(), name, short, help, &stringValue{values: values, name: name})
		_ = cmd.RegisterFlagCompletionFunc(name, func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return param.Choices, cobra.ShellCompDirectiveNoFileComp
		})
	case concepts.ParamString:
		values[name] = defaultString(param.Default)
		addValueFlag(cmd.Flags(), name, short, help, &stringValue{values: values, name: name})
	}
}

func addValueFlag(flags *pflag.FlagSet, name, short, help string, value pflag.Value) {
	if short != "" {
		flags.VarP(value, name, short, help)
		return
	}
	flags.Var(value, name, help)
}

type stringValue struct {
	values map[string]any
	name   string
}

func (v *stringValue) Set(s string) error { v.values[v.name] = s; return nil }
func (v *stringValue) Type() string       { return "string" }
func (v *stringValue) String() string     { return defaultString(v.values[v.name]) }

type intValue struct {
	values map[string]any
	name   string
}

func (v *intValue) Set(s string) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	v.values[v.name] = i
	return nil
}
func (v *intValue) Type() string   { return "int" }
func (v *intValue) String() string { return strconv.Itoa(defaultInt(v.values[v.name])) }

type boolValue struct {
	values map[string]any
	name   string
}

func (v *boolValue) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	v.values[v.name] = b
	return nil
}
func (v *boolValue) Type() string   { return "bool" }
func (v *boolValue) String() string { return strconv.FormatBool(defaultBool(v.values[v.name])) }

type stringSliceValue struct {
	values map[string]any
	name   string
}

func (v *stringSliceValue) Set(s string) error { v.values[v.name] = splitCSV(s); return nil }
func (v *stringSliceValue) Type() string       { return "stringSlice" }
func (v *stringSliceValue) String() string {
	return strings.Join(defaultStringSlice(v.values[v.name]), ",")
}

func defaultString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func defaultInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, _ := strconv.Atoi(t)
		return i
	default:
		return 0
	}
}

func defaultBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconv.ParseBool(t)
		return b
	default:
		return false
	}
}

func defaultStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		ret := make([]string, 0, len(t))
		for _, item := range t {
			ret = append(ret, fmt.Sprint(item))
		}
		return ret
	case string:
		return splitCSV(t)
	default:
		return nil
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	ret := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			ret = append(ret, trimmed)
		}
	}
	return ret
}
