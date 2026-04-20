package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// SymbolID encodes a stable identity for a symbol that survives file moves
// and most refactors. The scheme is:
//
//	sym:<importPath>.<Kind>.<Name>[#<signatureHash>]
//
// The optional #hash suffix only appears when two symbols in the same package
// would otherwise collide (e.g. two test helpers named `helper` in different
// files). importPath is used instead of file path so relocating a file does
// not invalidate IDs.
func SymbolID(importPath, kind, name, signatureForHash string) string {
	base := fmt.Sprintf("sym:%s.%s.%s", importPath, kind, name)
	if signatureForHash == "" {
		return base
	}
	h := sha256.Sum256([]byte(signatureForHash))
	return base + "#" + hex.EncodeToString(h[:4])
}

// MethodID encodes a method's ID including its receiver type. Methods share a
// (package, name) with other methods on different types, so we include the
// receiver in the Kind segment as `method.<recv>`.
func MethodID(importPath, recvType, name string) string {
	recv := strings.TrimPrefix(recvType, "*")
	return fmt.Sprintf("sym:%s.method.%s.%s", importPath, recv, name)
}

// FileID encodes a file's identity by module-relative path.
func FileID(relPath string) string { return "file:" + relPath }

// PackageID encodes a package's identity by import path.
func PackageID(importPath string) string { return "pkg:" + importPath }
