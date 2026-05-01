package review

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/wesen/codebase-browser/internal/browser"
	"github.com/wesen/codebase-browser/internal/docs"
	"github.com/wesen/codebase-browser/internal/history"
)

// PrecomputedReview holds all data needed for a standalone WASM export.
type PrecomputedReview struct {
	Version     string                             `json:"version"`
	GeneratedAt string                             `json:"generatedAt"`
	CommitRange string                             `json:"commitRange"`
	Commits     []CommitLite                       `json:"commits"`
	Diffs       map[string]*DiffLite               `json:"diffs"`
	Histories   map[string][]HistoryEntryLite      `json:"histories"`
	Impacts     map[string]*ImpactLite             `json:"impacts"`
	BodyDiffs   map[string]*history.BodyDiffResult `json:"bodyDiffs"`
	Docs        []ReviewDocLite                    `json:"docs"`
}

type CommitLite struct {
	Hash       string `json:"hash"`
	ShortHash  string `json:"shortHash"`
	Message    string `json:"message"`
	AuthorName string `json:"authorName"`
	AuthorTime int64  `json:"authorTime"`
}

type DiffLite struct {
	OldHash string               `json:"oldHash"`
	NewHash string               `json:"newHash"`
	Stats   history.DiffStats    `json:"stats"`
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
	Root       string       `json:"root"`
	RootSymbol string       `json:"rootSymbol,omitempty"`
	Direction  string       `json:"direction"`
	Depth      int          `json:"depth"`
	Commit     string       `json:"commit"`
	Nodes      []ImpactNode `json:"nodes"`
}

type ImpactEdge struct {
	FromSymbolID string `json:"fromSymbolId"`
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	FileID       string `json:"fileId"`
}

type ImpactNode struct {
	SymbolID      string       `json:"symbolId"`
	Name          string       `json:"name"`
	Kind          string       `json:"kind"`
	Depth         int          `json:"depth"`
	Edges         []ImpactEdge `json:"edges"`
	Compatibility string       `json:"compatibility"`
	Local         bool         `json:"local"`
}

type ReviewDocLite struct {
	Slug     string            `json:"slug"`
	Title    string            `json:"title"`
	HTML     string            `json:"html"`
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
		BodyDiffs:   make(map[string]*history.BodyDiffResult),
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

	// Pre-compute body diffs for changed symbols in adjacent commit pairs.
	if err := computeBodyDiffs(ctx, store, out, "."); err != nil {
		return nil, fmt.Errorf("compute body diffs: %w", err)
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

func computeBodyDiffs(ctx context.Context, store *Store, out *PrecomputedReview, repoRoot string) error {
	compute := func(oldHash, newHash, symbolID string) {
		if oldHash == "" || newHash == "" || symbolID == "" {
			return
		}
		key := oldHash + ".." + newHash + "|" + symbolID
		if _, ok := out.BodyDiffs[key]; ok {
			return
		}
		bodyDiff, err := history.DiffSymbolBodyWithContent(ctx, store.History, repoRoot, oldHash, newHash, symbolID)
		if err != nil {
			// Body diffs are useful but not required for the export. Keep the
			// commit-level diff and let the static UI report the missing body diff.
			return
		}
		out.BodyDiffs[key] = bodyDiff
	}

	for _, diff := range out.Diffs {
		for _, sym := range diff.Symbols {
			if sym.ChangeType == history.ChangeUnchanged || sym.SymbolID == "" {
				continue
			}
			compute(diff.OldHash, diff.NewHash, sym.SymbolID)
		}
	}

	rows, err := store.DB().QueryContext(ctx, `
		SELECT symbol_id, params_json
		FROM review_doc_snippets
		WHERE directive = 'codebase-diff' AND symbol_id != ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var symbolID, paramsJSON string
		if err := rows.Scan(&symbolID, &paramsJSON); err != nil {
			return err
		}
		params := map[string]string{}
		_ = json.Unmarshal([]byte(paramsJSON), &params)
		oldHash := resolveExportCommitRef(params["from"], out.Commits)
		newHash := resolveExportCommitRef(params["to"], out.Commits)
		compute(oldHash, newHash, symbolID)
	}
	return rows.Err()
}

func resolveExportCommitRef(ref string, commits []CommitLite) string {
	if ref == "" || len(commits) == 0 {
		return ""
	}
	if ref == "HEAD" {
		return commits[len(commits)-1].Hash
	}
	if strings.HasPrefix(ref, "HEAD~") {
		var offset int
		if _, err := fmt.Sscanf(ref, "HEAD~%d", &offset); err == nil {
			idx := len(commits) - 1 - offset
			if idx >= 0 && idx < len(commits) {
				return commits[idx].Hash
			}
		}
	}
	for _, c := range commits {
		if c.Hash == ref || c.ShortHash == ref || strings.HasPrefix(c.Hash, ref) {
			return c.Hash
		}
	}
	return ""
}

func computeImpacts(ctx context.Context, store *Store, loaded *browser.Loaded, out *PrecomputedReview) error {
	latestHash, err := latestExportCommit(ctx, store)
	if err != nil {
		return err
	}

	type impactQuery struct {
		root      string
		direction string
		depth     int
		commit    string
	}
	queries := map[string]impactQuery{}
	addQuery := func(root, direction string, depth int, commit string) {
		if root == "" {
			return
		}
		if direction != "uses" && direction != "usedby" {
			direction = "usedby"
		}
		if depth < 1 {
			depth = 2
		}
		if depth > 5 {
			depth = 5
		}
		if commit == "" {
			commit = latestHash
		}
		key := impactKey(root, direction, depth, commit)
		queries[key] = impactQuery{root: root, direction: direction, depth: depth, commit: commit}
	}

	// Keep the old default behavior: every snippet symbol gets usedby depth=2.
	rows, err := store.DB().QueryContext(ctx, `
		SELECT DISTINCT symbol_id FROM review_doc_snippets WHERE symbol_id != ''
	`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		addQuery(id, "usedby", 2, latestHash)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	// Also honor explicit codebase-impact parameters so static exports match
	// server widgets for direction, depth, and commit-specific impact views.
	rows, err = store.DB().QueryContext(ctx, `
		SELECT symbol_id, params_json
		FROM review_doc_snippets
		WHERE directive = 'codebase-impact' AND symbol_id != ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rootID, paramsJSON string
		if err := rows.Scan(&rootID, &paramsJSON); err != nil {
			return err
		}
		params := map[string]string{}
		_ = json.Unmarshal([]byte(paramsJSON), &params)
		depth := 2
		if raw := params["depth"]; raw != "" {
			_, _ = fmt.Sscanf(raw, "%d", &depth)
		}
		commit := latestHash
		if raw := params["commit"]; raw != "" {
			if resolved := resolveExportCommitRef(raw, out.Commits); resolved != "" {
				commit = resolved
			}
		}
		addQuery(rootID, params["dir"], depth, commit)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for key, query := range queries {
		impact, err := computeImpact(ctx, store, query.commit, query.root, query.direction, query.depth)
		if err != nil {
			return err
		}
		out.Impacts[key] = impact
		defaultKey := query.root + "|" + query.direction + "|" + fmt.Sprint(query.depth)
		if query.commit == latestHash {
			out.Impacts[defaultKey] = impact
		}
	}

	return nil
}

func latestExportCommit(ctx context.Context, store *Store) (string, error) {
	var hash string
	err := store.DB().QueryRowContext(ctx, `SELECT hash FROM commits WHERE error = '' ORDER BY author_time DESC LIMIT 1`).Scan(&hash)
	return hash, err
}

func impactKey(root, direction string, depth int, commit string) string {
	if commit == "" {
		return root + "|" + direction + "|" + fmt.Sprint(depth)
	}
	return root + "|" + direction + "|" + fmt.Sprint(depth) + "|" + commit
}

func computeImpact(ctx context.Context, store *Store, commitHash, rootID, direction string, maxDepth int) (*ImpactLite, error) {
	response := &ImpactLite{Root: rootID, RootSymbol: rootID, Direction: direction, Depth: maxDepth, Commit: commitHash, Nodes: []ImpactNode{}}
	visited := map[string]bool{rootID: true}
	type queueItem struct {
		symbolID string
		depth    int
	}
	queue := []queueItem{{symbolID: rootID, depth: 0}}
	nodeByID := map[string]*ImpactNode{}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.depth >= maxDepth {
			continue
		}

		edges, err := impactOneHop(ctx, store, commitHash, item.symbolID, direction)
		if err != nil {
			return nil, err
		}
		for _, edge := range edges {
			nextID := edge.FromSymbolID
			if direction == "uses" {
				nextID = edge.ToSymbolID
			}
			nextDepth := item.depth + 1
			node := nodeByID[nextID]
			if node == nil {
				name, kind, local, err := impactSymbolMeta(ctx, store, commitHash, nextID)
				if err != nil {
					name, kind, local = impactFallbackName(nextID), "external", false
				}
				node = &ImpactNode{SymbolID: nextID, Name: name, Kind: kind, Depth: nextDepth, Edges: []ImpactEdge{}, Compatibility: "unknown", Local: local}
				nodeByID[nextID] = node
				response.Nodes = append(response.Nodes, *node)
			}
			node.Edges = append(node.Edges, edge)
			for i := range response.Nodes {
				if response.Nodes[i].SymbolID == nextID {
					response.Nodes[i] = *node
					break
				}
			}
			if !visited[nextID] {
				visited[nextID] = true
				queue = append(queue, queueItem{symbolID: nextID, depth: nextDepth})
			}
		}
	}
	return response, nil
}

func impactOneHop(ctx context.Context, store *Store, commitHash, symbolID, direction string) ([]ImpactEdge, error) {
	query := `
SELECT from_symbol_id, to_symbol_id, kind, file_id
FROM   snapshot_refs
WHERE  commit_hash = ? AND to_symbol_id = ?
ORDER BY from_symbol_id, kind`
	if direction == "uses" {
		query = `
SELECT from_symbol_id, to_symbol_id, kind, file_id
FROM   snapshot_refs
WHERE  commit_hash = ? AND from_symbol_id = ?
ORDER BY to_symbol_id, kind`
	}
	rows, err := store.DB().QueryContext(ctx, query, commitHash, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []ImpactEdge
	for rows.Next() {
		var edge ImpactEdge
		if err := rows.Scan(&edge.FromSymbolID, &edge.ToSymbolID, &edge.Kind, &edge.FileID); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

func impactSymbolMeta(ctx context.Context, store *Store, commitHash, symbolID string) (string, string, bool, error) {
	var name, kind string
	err := store.DB().QueryRowContext(ctx, `
SELECT name, kind FROM snapshot_symbols WHERE commit_hash = ? AND id = ?`, commitHash, symbolID).Scan(&name, &kind)
	return name, kind, err == nil, err
}

func impactFallbackName(symbolID string) string {
	trimmed := strings.TrimPrefix(symbolID, "sym:")
	lastDot := strings.LastIndex(trimmed, ".")
	if lastDot >= 0 && lastDot+1 < len(trimmed) {
		return trimmed[lastDot+1:]
	}
	return trimmed
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
