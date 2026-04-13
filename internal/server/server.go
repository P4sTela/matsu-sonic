package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Server holds the HTTP router and dependencies.
type Server struct {
	Router *chi.Mux
	Hub    *Hub
}

// New creates a Server with middleware and the WebSocket endpoint.
// API route registration is done externally via Router.
func New(hub *Hub) *Server {
	r := chi.NewRouter()
	r.Use(Recovery)
	r.Use(Logger)
	r.Use(CORS)

	r.HandleFunc("/ws", hub.HandleWS)

	return &Server{Router: r, Hub: hub}
}

// MountSPA serves an embedded SPA filesystem with index.html fallback.
func (s *Server) MountSPA(frontendFS fs.FS) {
	spaHandler := SPAHandler(frontendFS)
	s.Router.Handle("/*", spaHandler)
}

// SPAHandler returns a handler that serves static files from the given FS,
// falling back to index.html for unmatched routes (SPA client-side routing).
func SPAHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file
		f, err := fsys.Open(path)
		if err != nil {
			// File not found — serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	})
}
