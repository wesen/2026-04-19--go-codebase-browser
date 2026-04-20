package indexer

// Index is the top-level shape serialised as index.json. See design §6.1.
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

type Package struct {
	ID         string   `json:"id"`
	ImportPath string   `json:"importPath"`
	Name       string   `json:"name"`
	Doc        string   `json:"doc,omitempty"`
	FileIDs    []string `json:"fileIds"`
	SymbolIDs  []string `json:"symbolIds"`
}

type File struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	PackageID string   `json:"packageId"`
	Size      int      `json:"size"`
	LineCount int      `json:"lineCount"`
	BuildTags []string `json:"buildTags,omitempty"`
	SHA256    string   `json:"sha256"`
}

// Range holds both line/col (for display) and byte offsets (authoritative
// for slicing).
type Range struct {
	StartLine   int `json:"startLine"`
	StartCol    int `json:"startCol"`
	EndLine     int `json:"endLine"`
	EndCol      int `json:"endCol"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

// Symbol represents a Go identifier (func, method, type, const, var, field, ...).
type Symbol struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	PackageID  string    `json:"packageId"`
	FileID     string    `json:"fileId"`
	Range      Range     `json:"range"`
	Doc        string    `json:"doc,omitempty"`
	Signature  string    `json:"signature,omitempty"`
	Receiver   *Receiver `json:"receiver,omitempty"`
	TypeParams []string  `json:"typeParams,omitempty"`
	Exported   bool      `json:"exported"`
	Children   []Symbol  `json:"children,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
}

type Receiver struct {
	TypeName string `json:"typeName"`
	Pointer  bool   `json:"pointer"`
}

// Ref is a cross-reference (phase 2+). Left as a declared type so consumers
// can depend on the shape.
type Ref struct {
	FromSymbolID string `json:"fromSymbolId"`
	ToSymbolID   string `json:"toSymbolId"`
	Kind         string `json:"kind"`
	FileID       string `json:"fileId"`
	Range        Range  `json:"range"`
}
