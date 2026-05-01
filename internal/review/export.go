package review

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/docs"
	"github.com/wesen/codebase-browser/internal/history"
)

// PrecomputedReview holds all data needed for a standalone WASM export.
type PrecomputedReview struct {
	Version     string              `json:"version"`
	GeneratedAt string              `json:"generatedAt"`
	CommitRange string              `json:"commitRange"`
	Commits     []CommitLite        `json:"commits"`
	Diffs       map[string]*DiffLite `json:"diffs"`
	Histories   map[string][]HistoryEntryLite `json:"histories"`
	Impacts     map[string]*ImpactLite `json:"impacts"`
	Docs        []ReviewDocLite     `json:"docs"`
}

type CommitLite struct {
	Hash       string `json:"hash"`
	ShortHash  string `json:"shortHash"`
	Message    string `json:"message"`
	AuthorName string `json:"authorName"`
	AuthorTime int64  `json:"authorTime"`
}

type DiffLite struct {
	OldHash string              `json:"oldHash"`
	NewHash string              `json:"newHash"`
	Stats   history.DiffStats   `json:"stats"`
	Symbols []history.SymbolDiff `json:"symbols"`
	Files   []history.FileDiff   `json:"files"`
}

type HistoryEntryLite struct {
	CommitHash string `json:"commitHash"`
	ShortHash  string `json:"shortHash"`
	AuthorTime int64  `json:"authorTime"`
	BodyHash   string `json:"bodyHash"`
	Signature  string `json:"signature"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
}

type ImpactLite struct {
	RootSymbol string       `json:"rootSymbol"`
	Direction  string       `json:"direction"`
	Depth      int          `json:"depth"`
	Nodes      []ImpactNode `json:"nodes"`
}

type ImpactNode struct {
	SymbolID      string `json:"symbolId"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Depth         int    `json:"depth"`
	Compatibility string `json:"compatibility"`
}

type ReviewDocLite struct {
	Slug     string       `json:"slug"`
	Title    string       `json:"title"`
	HTML     string       `json:"html"`
	Snippets []docs.SnippetRef `json:"snippets"`
}

// LoadForExport reads a review database and pre-computes all data needed for a static export.
func LoadForExport(dbPath string) (*PrecomputedReview, error) {
	store, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open review db: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Load commits.
	commits, err := store.History.ListCommits(ctx)
	if err != nil {
		return nil, fmt.Errorf("list commits: %w", err)
	}

	out := &PrecomputedReview{
		Version:     "1",
		GeneratedAt: "",
		Commits:     make([]CommitLite, 0, len(commits)),
		Diffs:       make(map[string]*DiffLite),
		Histories:   make(map[string][]HistoryEntryLite),
		Impacts:     make(map[string]*ImpactLite),
		Docs:        make([]ReviewDocLite, 0),
	}

	for _, c := range commits {
		out.Commits = append(out.Commits, CommitLite{
			Hash:       c.Hash,
			ShortHash:  c.ShortHash,
			Message:    c.Message,
			AuthorName: c.AuthorName,
			AuthorTime: c.AuthorTime,
		})
	}

	// Sort commits by author_time ascending for diff computation.
	sort.Slice(out.Commits, func(i, j int) bool {
		return out.Commits[i].AuthorTime < out.Commits[j].AuthorTime
	})

	// Pre-compute diffs for adjacent commit pairs.
	for i := 1; i < len(out.Commits); i++ {
		oldHash := out.Commits[i-1].Hash
		newHash := out.Commits[i].Hash
		key := oldHash + ".." + newHash

		diff, err := store.History.DiffCommits(ctx, oldHash, newHash)
		if err != nil {
			return nil, fmt.Errorf("diff %s: %w", key, err)
		}

		out.Diffs[key] = &DiffLite{
			OldHash: diff.OldHash,
			NewHash: diff.NewHash,
			Stats:   diff.Stats,
			Symbols: diff.Symbols,
			Files:   diff.Files,
		}
	}

	// Load latest snapshot for doc rendering.
	loaded, err := LoadLatestSnapshot(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	// Pre-compute histories for all symbols that appear in >1 commit.
	if err := computeHistories(ctx, store, out); err != nil {
		return nil, fmt.Errorf("compute histories: %w", err)
	}

	// Pre-compute impact for symbols referenced in review docs.
	if err := computeImpacts(ctx, store, loaded, out); err != nil {
		return nil, fmt.Errorf("compute impacts: %w", err)
	}

	// Render review docs.
	if err := renderReviewDocs(ctx, store, loaded, out); err != nil {
		return nil, fmt.Errorf("render docs: %w", err)
	}

	return out, nil
}

func computeHistories(ctx context.Context, store *Store, out *PrecomputedReview) error {
	rows, err := store.DB().QueryContext(ctx, `
		SELECT id, commit_hash, body_hash, signature, start_line, end_line
		FROM snapshot_symbols
		ORDER BY id, commit_hash
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Build map of symbol ID -> list of entries.
	histMap := make(map[string][]HistoryEntryLite)
	for rows.Next() {
		var symID, commitHash, bodyHash, signature string
		var startLine, endLine int
		if err := rows.Scan(&symID, &commitHash, &bodyHash, &signature, &startLine, &endLine); err != nil {
			return err
		}
		histMap[symID] = append(histMap[symID], HistoryEntryLite{
			CommitHash: commitHash,
			BodyHash:   bodyHash,
			Signature:  signature,
			StartLine:  startLine,
			EndLine:    endLine,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Only keep symbols with >1 commit (i.e. potential history).
	for symID, entries := range histMap {
		if len(entries) > 1 {
			// Add short_hash and author_time from commits.
			for i := range entries {
				for _, c := range out.Commits {
					if c.Hash == entries[i].CommitHash {
						entries[i].ShortHash = c.ShortHash
						entries[i].AuthorTime = c.AuthorTime
						break
					}
				}
			}
			out.Histories[symID] = entries
		}
	}

	return nil
}

func computeImpacts(ctx context.Context, store *Store, loaded *browser.Loaded, out *PrecomputedReview) error {
	// Find symbols referenced in review doc snippets.
	rows, err := store.DB().QueryContext(ctx, `
		SELECT DISTINCT symbol_id FROM review_doc_snippets WHERE symbol_id != ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var symbolIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		symbolIDs = append(symbolIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// For each symbol, compute usedBy impact up to depth 2.
	for _, rootID := range symbolIDs {
		impact, err := computeImpact(ctx, store, rootID, "usedby", 2)
		if err != nil {
			return err
		}
		out.Impacts[rootID+"|usedby|2"] = impact
	}

	return nil
}

func computeImpact(ctx context.Context, store *Store, rootID, direction string, maxDepth int) (*ImpactLite, error) {
	// BFS over snapshot_refs at the latest commit.
	latestHash := ""
	for _, c := range []CommitLite{} {
		_ = c
	}
	// Get latest commit hash.
	row := store.DB().QueryRowContext(ctx, `SELECT hash FROM commits ORDER BY author_time DESC LIMIT 1`)
	_ = row.Scan(&latestHash)

	type queueItem struct {
		symID string
		depth int
	}

	visited := map[string]bool{rootID: true}
	queue := []queueItem{{symID: rootID, depth: 0}}
	var nodes []ImpactNode

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.depth > 0 {
			// Lookup symbol name/kind.
			var name, kind string
			store.DB().QueryRowContext(ctx,
				`SELECT name, kind FROM snapshot_symbols WHERE id = ? AND commit_hash = ? LIMIT 1`,
				item.symID, latestHash).Scan(&name, &kind)

			nodes = append(nodes, ImpactNode{
				SymbolID:      item.symID,
				Name:          name,
				Kind:          kind,
				Depth:         item.depth,
				Compatibility: "unknown",
			})
		}

		if item.depth >= maxDepth {
			continue
		}

		var nextIDs []string
		if direction == "usedby" {
			// Find callers.
			rows, _ := store.DB().QueryContext(ctx,
				`SELECT DISTINCT from_symbol_id FROM snapshot_refs WHERE to_symbol_id = ? AND commit_hash = ?`,
				item.symID, latestHash)
			if rows != nil {
				for rows.Next() {
					var id string
					rows.Scan(&id)
					nextIDs = append(nextIDs, id)
				}
				rows.Close()
			}
		} else {
			// Find callees.
			rows, _ := store.DB().QueryContext(ctx,
				`SELECT DISTINCT to_symbol_id FROM snapshot_refs WHERE from_symbol_id = ? AND commit_hash = ?`,
				item.symID, latestHash)
			if rows != nil {
				for rows.Next() {
					var id string
					rows.Scan(&id)
					nextIDs = append(nextIDs, id)
				}
				rows.Close()
			}
		}

		for _, nextID := range nextIDs {
			if !visited[nextID] {
				visited[nextID] = true
				queue = append(queue, queueItem{symID: nextID, depth: item.depth + 1})
			}
		}
	}

	return &ImpactLite{
		RootSymbol: rootID,
		Direction:  direction,
		Depth:      maxDepth,
		Nodes:      nodes,
	}, nil
}

func renderReviewDocs(ctx context.Context, store *Store, loaded *browser.Loaded, out *PrecomputedReview) error {
	// Read all review docs from DB.
	rows, err := store.DB().QueryContext(ctx, `
		SELECT slug, content FROM review_docs ORDER BY slug
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Use repo root as source FS.
	sourceFS := os.DirFS(".")

	for rows.Next() {
		var slug, content string
		if err := rows.Scan(&slug, &content); err != nil {
			return err
		}

		page, err := docs.Render(slug, []byte(content), loaded, sourceFS)
		if err != nil {
			// Skip docs that fail to render.
			continue
		}

		out.Docs = append(out.Docs, ReviewDocLite{
			Slug:     page.Slug,
			Title:    page.Title,
			HTML:     page.HTML,
			Snippets: page.Snippets,
		})
	}
	return rows.Err()
}

// WritePrecomputed marshals the precomputed data to JSON and writes it to a file.
func WritePrecomputed(out *PrecomputedReview, path string) error {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal precomputed: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write precomputed: %w", err)
	}
	return nil
}

// Ensure types are used.
var _ = fs.FS(nil)
var _ = browser.Loaded{}
