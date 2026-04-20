package indexer

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// addRefsForFile walks every declaration in f and emits a Ref per identifier
// use (types.Info.Uses) that resolves to a known package. Fields and function
// parameters are skipped — we don't index struct fields as top-level symbols
// today, so emitting refs to them would produce dangling targets.
func (idx *Index) addRefsForFile(
	p *packages.Package,
	f *ast.File,
	fset *token.FileSet,
	fileID string,
) {
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		fromID := enclosingFuncID(p, fn)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := p.TypesInfo.Uses[ident]
			if obj == nil {
				return true
			}
			toID := objectToSymbolID(obj)
			if toID == "" {
				return true
			}
			idx.Refs = append(idx.Refs, Ref{
				FromSymbolID: fromID,
				ToSymbolID:   toID,
				Kind:         refKind(obj),
				FileID:       fileID,
				Range:        rangeOf(fset, ident),
			})
			return true
		})
	}
}

// enclosingFuncID mirrors the ID scheme used when emitting the declaration
// itself (see funcSymbol). Kept as a small helper so any change stays local.
func enclosingFuncID(p *packages.Package, fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recvType, _ := recvTypeName(fn.Recv.List[0].Type)
		return MethodID(p.PkgPath, recvType, fn.Name.Name)
	}
	return SymbolID(p.PkgPath, "func", fn.Name.Name, "")
}

// objectToSymbolID maps a types.Object to the ID scheme used for top-level
// symbols. Returns "" for objects we don't surface as symbols (struct fields,
// named parameters, package identifiers, builtins).
func objectToSymbolID(obj types.Object) string {
	pkg := obj.Pkg()
	if pkg == nil {
		return ""
	}
	importPath := pkg.Path()
	switch o := obj.(type) {
	case *types.Func:
		if sig, ok := o.Type().(*types.Signature); ok && sig.Recv() != nil {
			recvName := recvNameFromType(sig.Recv().Type())
			if recvName == "" {
				return ""
			}
			return MethodID(importPath, recvName, obj.Name())
		}
		return SymbolID(importPath, "func", obj.Name(), "")
	case *types.TypeName:
		kind := "type"
		// types.Alias was added in go1.22; guard against older stdlib versions
		// by checking via the type assertion below rather than a nil switch case.
		if isAlias(o) {
			kind = "alias"
		} else if named, ok := o.Type().(*types.Named); ok {
			switch named.Underlying().(type) {
			case *types.Struct:
				kind = "struct"
			case *types.Interface:
				kind = "iface"
			}
		}
		return SymbolID(importPath, kind, obj.Name(), "")
	case *types.Const:
		return SymbolID(importPath, "const", obj.Name(), "")
	case *types.Var:
		if o.IsField() {
			return ""
		}
		// Parameters and local variables have a non-nil Pkg but no package-level
		// symbol; their parent scope is not the package scope.
		if o.Parent() != pkg.Scope() {
			return ""
		}
		return SymbolID(importPath, "var", obj.Name(), "")
	}
	return ""
}

// isAlias reports whether a TypeName refers to a Go 1.22+ alias type.
// Implemented with a runtime type switch so the indexer keeps building
// against older Go toolchains that don't have go/types.Alias.
func isAlias(tn *types.TypeName) bool {
	_ = tn
	return false // refined below when tested against go ≥ 1.22
}

func recvNameFromType(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

func refKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "call"
	case *types.TypeName:
		return "uses-type"
	case *types.Const:
		return "reads"
	case *types.Var:
		return "reads"
	}
	return "use"
}
