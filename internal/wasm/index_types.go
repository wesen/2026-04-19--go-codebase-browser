// Package wasm contains browser-side search and lookup logic compiled to WASM.
// It defines its own lightweight index types to avoid importing heavy
// internal/indexer dependencies (go/ast, go/packages, etc.) into the TinyGo build.
package wasm

// Index is the top-level shape, matching internal/indexer.Index.
type Index struct {
	Version     string    `json:"version"`
	GeneratedAt string    `json:"generatedAt"`
	Module      string    `json:"module"`
	GoVersion   string    `json:"goVersion"`
	Packages    []Package `json:"packages"`
	Files       []File    `json:"files"`
	Symbols     []Symbol  `json:"symbols"`
	Refs        []Ref     `json:"refs,omitempty"`
}

// Package mirrors internal/indexer.Package.
type Package struct {
	ID         string   `json:"id"`
	ImportPath string   `json:"importPath"`
	Name       string   `json:"name"`
	Doc        string   `json:"doc,omitempty"`
	FileIDs    []string `json:"fileIds"`
	SymbolIDs  []string `json:"symbolIds"`
	Language   string   `json:"language,omitempty"`
}

// File mirrors internal/indexer.File.
type File struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	PackageID string   `json:"packageId"`
	Size      int      `json:"size"`
	LineCount int      `json:"lineCount"`
	BuildTags []string `json:"buildTags,omitempty"`
	SHA256    string   `json:"sha256"`
	Language  string   `json:"language,omitempty"`
}

// Range mirrors internal/indexer.Range.
type Range struct {
	StartLine   int `json:"startLine"`
	StartCol    int `json:"startCol"`
	EndLine     int `json:"endLine"`
	EndCol      int `json:"endCol"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

// Symbol mirrors internal/indexer.Symbol.
type Symbol struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	PackageID  string   `json:"packageId"`
	FileID     string   `json:"fileId"`
	Range      Range    `json:"range"`
	Doc        string   `json:"doc,omitempty"`
	Signature  string   `json:"signature,omitempty"`
	Receiver   *Receiver `json:"receiver,omitempty"`
	TypeParams []string  `json:"typeParams,omitempty"`
	Exported   bool      `json:"exported"`
	Children   []Symbol  `json:"children,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Language   string    `json:"language,omitempty"`
}

// Receiver mirrors internal/indexer.Receiver.
type Receiver struct {
	TypeName string `json:"typeName"`
	Pointer  bool   `json:"pointer"`
}

// Ref mirrors internal/indexer.Ref.
type Ref struct {
	FromSymbolID string `json:"fromSymbolId"`
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	FileID       string `json:"fileId"`
	Range        Range  `json:"range"`
}
