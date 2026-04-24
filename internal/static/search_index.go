// Package static performs build-time pre-computation for the static WASM build.
package static

import (
	"strings"

	"github.com/wesen/codebase-browser/internal/browser"
)

// BuildSearchIndex creates an inverted index mapping lowercase symbol name
// substrings to symbol IDs. For a codebase with N symbols, this stores
// O(N * avg_name_len) entries. For typical codebases (10k–50k symbols) this
// is well under 1MB of JSON.
func BuildSearchIndex(loaded *browser.Loaded) map[string][]string {
	idx := make(map[string][]string)
	seen := make(map[string]map[string]bool) // key -> id -> bool (dedup)

	for i := range loaded.Index.Symbols {
		sym := &loaded.Index.Symbols[i]
		name := strings.ToLower(sym.Name)

		// Index every substring of the name (prefixes only for efficiency)
		for length := 1; length <= len(name); length++ {
			for start := 0; start <= len(name)-length; start++ {
				key := name[start : start+length]
				if seen[key] == nil {
					seen[key] = make(map[string]bool)
				}
				seen[key][sym.ID] = true
			}
		}
	}

	// Convert dedup maps to sorted slices
	for key, ids := range seen {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		idx[key] = list
	}

	return idx
}

// BuildSearchIndexFast creates a simpler inverted index: only full names and
// prefixes up to 4 characters. Much smaller than the full substring index.
func BuildSearchIndexFast(loaded *browser.Loaded) map[string][]string {
	idx := make(map[string][]string)
	seen := make(map[string]map[string]bool)

	for i := range loaded.Index.Symbols {
		sym := &loaded.Index.Symbols[i]
		name := strings.ToLower(sym.Name)

		// Full name
		addSeen(seen, name, sym.ID)

		// Prefixes up to min(4, len(name))
		maxPrefix := 4
		if len(name) < maxPrefix {
			maxPrefix = len(name)
		}
		for length := 1; length <= maxPrefix; length++ {
			addSeen(seen, name[:length], sym.ID)
		}
	}

	for key, ids := range seen {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		idx[key] = list
	}

	return idx
}

func addSeen(seen map[string]map[string]bool, key, id string) {
	if seen[key] == nil {
		seen[key] = make(map[string]bool)
	}
	seen[key][id] = true
}
