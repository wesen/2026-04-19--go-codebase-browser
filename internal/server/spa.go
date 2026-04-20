package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler serves files from SPAFS if they exist, otherwise falls back to
// index.html. Rejects paths under /api so the API is never shadowed.
func (s *Server) spaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			reqPath = "index.html"
		}
		reqPath = path.Clean(reqPath)
		if strings.HasPrefix(reqPath, "..") {
			http.NotFound(w, r)
			return
		}

		if data, err := fs.ReadFile(s.SPAFS, reqPath); err == nil {
			writeSPA(w, reqPath, data)
			return
		}
		// SPA fallback: serve index.html for client-side-routed paths.
		data, err := fs.ReadFile(s.SPAFS, "index.html")
		if err != nil {
			http.Error(w, "SPA not built. Run 'go generate ./internal/web'.", http.StatusNotFound)
			return
		}
		writeSPA(w, "index.html", data)
	})
}

func writeSPA(w http.ResponseWriter, name string, data []byte) {
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(name, ".json"):
		w.Header().Set("Content-Type", "application/json")
	case strings.HasSuffix(name, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	}
	_, _ = w.Write(data)
}
