package server

import "net/http"

// handleDocList and handleDocPage are stubs until Phase 5 wires the doc
// renderer. They return empty but well-formed responses so frontend probing
// doesn't crash in earlier phases.
func (s *Server) handleDocList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []any{})
}

func (s *Server) handleDocPage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "doc renderer not yet wired (Phase 5)", http.StatusNotImplemented)
}
