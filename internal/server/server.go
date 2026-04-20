// Package server wires the HTTP API and the SPA handler. It takes a loaded
// index, a source filesystem, and an SPA filesystem as dependencies so it
// can be driven by either the embedded FS (production) or on-disk FS (dev).
package server

import (
	"io/fs"
	"net/http"

	"github.com/wesen/codebase-browser/internal/browser"
)

// Server holds shared state for HTTP handlers.
type Server struct {
	Loaded   *browser.Loaded
	SourceFS fs.FS
	SPAFS    fs.FS
}

// New constructs a Server.
func New(l *browser.Loaded, srcFS, spaFS fs.FS) *Server {
	return &Server{Loaded: l, SourceFS: srcFS, SPAFS: spaFS}
}

// Handler returns an http.Handler with all routes mounted. API routes are
// registered before the SPA fallback so unknown /api/* paths return 404
// rather than index.html.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/index", s.handleIndex)
	mux.HandleFunc("/api/packages", s.handlePackages)
	mux.HandleFunc("/api/symbol/", s.handleSymbol)
	mux.HandleFunc("/api/source", s.handleSource)
	mux.HandleFunc("/api/snippet", s.handleSnippet)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/doc", s.handleDocList)
	mux.HandleFunc("/api/doc/", s.handleDocPage)
	mux.HandleFunc("/api/xref/", s.handleXref)
	mux.HandleFunc("/api/snippet-refs", s.handleSnippetRefs)
	mux.HandleFunc("/api/source-refs", s.handleSourceRefs)
	mux.HandleFunc("/api/file-xref", s.handleFileXref)

	mux.Handle("/", s.spaHandler())
	return withCommonHeaders(mux)
}

// withCommonHeaders stamps every response with cache + content-type hints.
// Because the binary is immutable per-build, aggressive caching is safe.
func withCommonHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Codebase-Browser", "1")
		h.ServeHTTP(w, r)
	})
}
