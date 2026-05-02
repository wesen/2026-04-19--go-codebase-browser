package indexer

import (
	"context"
	"fmt"
)

// Extractor produces an Index for a given language. See design GCB-002 §8.
type Extractor interface {
	// Language returns the short language tag ("go", "ts", ...).
	Language() string
	// Extract loads a project and emits an Index. The returned Index has
	// Language stamped on every Package/File/Symbol — callers don't need to
	// post-process.
	Extract(ctx context.Context, opts ExtractOptions) (*Index, error)
}

// GoExtractor is the in-process Go/packages-based extractor. It wraps the
// existing Extract() function so callers that want the single-language
// shortcut keep working unchanged.
type GoExtractor struct{}

func NewGoExtractor() *GoExtractor { return &GoExtractor{} }

func (GoExtractor) Language() string { return "go" }

func (GoExtractor) Extract(_ context.Context, opts ExtractOptions) (*Index, error) {
	return Extract(opts)
}

// Merge concatenates multiple Index outputs into one, re-sorting the combined
// slices so downstream consumers see deterministic ordering. Returns an
// error if any ID appears in more than one input (we refuse to silently drop
// records). Nil inputs are skipped.
func Merge(parts []*Index) (*Index, error) {
	out := &Index{Version: "1"}
	seenPkg := map[string]string{}  // id → language (for error messages)
	seenFile := map[string]string{}
	seenSym := map[string]string{}
	langs := map[string]bool{}
	modules := map[string]bool{}

	for _, p := range parts {
		if p == nil {
			continue
		}
		if p.Module != "" {
			modules[p.Module] = true
		}
		if out.GeneratedAt == "" || p.GeneratedAt > out.GeneratedAt {
			out.GeneratedAt = p.GeneratedAt
		}
		if p.GoVersion != "" && out.GoVersion == "" {
			out.GoVersion = p.GoVersion
		}
		for _, pkg := range p.Packages {
			if prev, dup := seenPkg[pkg.ID]; dup {
				return nil, fmt.Errorf("duplicate package id %q (languages: %s and %s)", pkg.ID, prev, pkg.Language)
			}
			seenPkg[pkg.ID] = pkg.Language
			langs[pkg.Language] = true
			out.Packages = append(out.Packages, pkg)
		}
		for _, f := range p.Files {
			if prev, dup := seenFile[f.ID]; dup {
				return nil, fmt.Errorf("duplicate file id %q (languages: %s and %s)", f.ID, prev, f.Language)
			}
			seenFile[f.ID] = f.Language
			out.Files = append(out.Files, f)
		}
		for _, s := range p.Symbols {
			if prev, dup := seenSym[s.ID]; dup {
				return nil, fmt.Errorf("duplicate symbol id %q (languages: %s and %s)", s.ID, prev, s.Language)
			}
			seenSym[s.ID] = s.Language
			out.Symbols = append(out.Symbols, s)
		}
		// Refs don't have stable IDs of their own; concatenate without dedupe.
		out.Refs = append(out.Refs, p.Refs...)
	}

	out.Module = mergeModuleName(modules)
	sortIndex(out)
	return out, nil
}

func mergeModuleName(set map[string]bool) string {
	if len(set) == 0 {
		return ""
	}
	if len(set) == 1 {
		for k := range set {
			return k
		}
	}
	// Deterministic "+-joined" placeholder for multi-module indexes. The doc
	// design doesn't require a prettier choice yet; revisit if multi-module
	// becomes common.
	var names []string
	for k := range set {
		names = append(names, k)
	}
	// stable order
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return joinWith(names, "+")
}

func joinWith(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}
