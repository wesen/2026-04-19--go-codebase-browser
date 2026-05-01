package wasm

// ReviewData holds pre-computed review-specific data loaded into WASM.
type ReviewData struct {
	Version     string                        `json:"version"`
	GeneratedAt string                        `json:"generatedAt"`
	CommitRange string                        `json:"commitRange"`
	Commits     []CommitLite                  `json:"commits"`
	Diffs       map[string]*DiffLite          `json:"diffs"`
	Histories   map[string][]HistoryEntryLite `json:"histories"`
	Impacts     map[string]*ImpactLite        `json:"impacts"`
	BodyDiffs   map[string]*BodyDiffResult    `json:"bodyDiffs"`
	Docs        []ReviewDocLite               `json:"docs"`
}

type CommitLite struct {
	Hash       string `json:"hash"`
	ShortHash  string `json:"shortHash"`
	Message    string `json:"message"`
	AuthorName string `json:"authorName"`
	AuthorTime int64  `json:"authorTime"`
}

type DiffLite struct {
	OldHash string       `json:"oldHash"`
	NewHash string       `json:"newHash"`
	Stats   DiffStats    `json:"stats"`
	Symbols []SymbolDiff `json:"symbols"`
	Files   []FileDiff   `json:"files"`
}

type DiffStats struct {
	FilesAdded       int `json:"filesAdded"`
	FilesRemoved     int `json:"filesRemoved"`
	FilesModified    int `json:"filesModified"`
	SymbolsAdded     int `json:"symbolsAdded"`
	SymbolsRemoved   int `json:"symbolsRemoved"`
	SymbolsModified  int `json:"symbolsModified"`
	SymbolsMoved     int `json:"symbolsMoved"`
	SymbolsUnchanged int `json:"symbolsUnchanged"`
}

type SymbolDiff struct {
	SymbolID     string `json:"symbolId"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	PackageID    string `json:"packageId"`
	ChangeType   string `json:"changeType"`
	OldStartLine int    `json:"oldStartLine"`
	OldEndLine   int    `json:"oldEndLine"`
	NewStartLine int    `json:"newStartLine"`
	NewEndLine   int    `json:"newEndLine"`
	OldSignature string `json:"oldSignature"`
	NewSignature string `json:"newSignature"`
	OldBodyHash  string `json:"oldBodyHash"`
	NewBodyHash  string `json:"newBodyHash"`
}

type FileDiff struct {
	FileID     string `json:"fileId"`
	Path       string `json:"path"`
	ChangeType string `json:"changeType"`
	OldSHA256  string `json:"oldSha256"`
	NewSHA256  string `json:"newSha256"`
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

type BodyDiffResult struct {
	SymbolID    string `json:"symbolId"`
	Name        string `json:"name"`
	OldCommit   string `json:"oldCommit"`
	NewCommit   string `json:"newCommit"`
	OldBody     string `json:"oldBody"`
	NewBody     string `json:"newBody"`
	UnifiedDiff string `json:"unifiedDiff"`
	OldRange    string `json:"oldRange"`
	NewRange    string `json:"newRange"`
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
	Slug     string       `json:"slug"`
	Title    string       `json:"title"`
	HTML     string       `json:"html"`
	Snippets []SnippetRef `json:"snippets"`
}

type SnippetRef struct {
	SymbolID    string `json:"symbolId"`
	Kind        string `json:"kind"`
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
	LineStart   int    `json:"lineStart"`
	LineEnd     int    `json:"lineEnd"`
	HTML        string `json:"html,omitempty"`
}
