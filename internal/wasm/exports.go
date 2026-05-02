//go:build wasm

// Package wasm provides JS-interop exports for the browser.
// Uses TinyGo's syscall/js for direct JS object access.
package wasm

import "syscall/js"

// globalCtx is the loaded search context. Set by initWasm() from JS.
var globalCtx *SearchCtx

// keepAlive holds exported functions so they aren't GC'd.
var keepAlive []js.Func

// RegisterExports registers all WASM functions on the JS global object.
// Called from main() after module loads.
func RegisterExports() {
	exports := js.ValueOf(map[string]interface{}{})

	register := func(name string, fn func(this js.Value, args []js.Value) interface{}) {
		f := js.FuncOf(fn)
		keepAlive = append(keepAlive, f)
		exports.Set(name, f)
	}

	// initWasm(jsonIndex, jsonSearchIdx, jsonXrefIdx, jsonSnippets, jsonDocManifest, jsonDocHTML)
	register("initWasm", func(this js.Value, args []js.Value) interface{} {
		if len(args) < 6 {
			return js.ValueOf("error: expected at least 6 JSON string arguments")
		}
		jsonIndex := []byte(args[0].String())
		jsonSearchIdx := []byte(args[1].String())
		jsonXrefIdx := []byte(args[2].String())
		jsonSnippets := []byte(args[3].String())
		jsonDocManifest := []byte(args[4].String())
		jsonDocHTML := []byte(args[5].String())
		var err error
		globalCtx, err = Init(jsonIndex, jsonSearchIdx, jsonXrefIdx, jsonSnippets, jsonDocManifest, jsonDocHTML)
		if err != nil {
			return js.ValueOf("error: " + err.Error())
		}
		return js.ValueOf("ok")
	})

	// findSymbols(query, kind) → JSON string
	register("findSymbols", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		q := args[0].String()
		kind := ""
		if len(args) > 1 {
			kind = args[1].String()
		}
		return js.ValueOf(string(globalCtx.FindSymbols(q, kind)))
	})

	// getSymbol(id) → JSON string
	register("getSymbol", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		return js.ValueOf(string(globalCtx.GetSymbol(args[0].String())))
	})

	// getXref(id) → JSON string
	register("getXref", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		return js.ValueOf(string(globalCtx.GetXref(args[0].String())))
	})

	// getSnippet(id, kind) → JSON string
	register("getSnippet", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		id := args[0].String()
		kind := "declaration"
		if len(args) > 1 {
			kind = args[1].String()
		}
		return js.ValueOf(string(globalCtx.GetSnippet(id, kind)))
	})

	// getPackages() → JSON string
	register("getPackages", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		return js.ValueOf(string(globalCtx.GetPackages()))
	})

	// getIndexSummary() → JSON string
	register("getIndexSummary", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		return js.ValueOf(string(globalCtx.GetIndexSummary()))
	})

	// getDocPages() → JSON string
	register("getDocPages", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		return js.ValueOf(string(globalCtx.GetDocPages()))
	})

	// getDocPage(slug) → JSON string
	register("getDocPage", func(this js.Value, args []js.Value) interface{} {
		if globalCtx == nil {
			return js.ValueOf("null")
		}
		return js.ValueOf(string(globalCtx.GetDocPage(args[0].String())))
	})

	js.Global().Set("codebaseBrowser", exports)
}
