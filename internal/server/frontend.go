package server

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/web"
)

// mountDevProxy sets up a reverse proxy to the Vite dev server for non-API requests.
func mountDevProxy(r *chi.Mux) {
	viteURL, _ := url.Parse("http://localhost:5173")
	proxy := httputil.NewSingleHostReverseProxy(viteURL)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(w, req)
	})
}

// mountEmbeddedFrontend serves the embedded frontend build files.
func mountEmbeddedFrontend(r *chi.Mux) {
	distFS, err := fs.Sub(web.DistFS, "frontend/build")
	if err != nil {
		slog.Error("failed to create sub filesystem for frontend", "error", err)
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		// Try to serve the file directly
		f, err := distFS.Open(req.URL.Path[1:]) // strip leading /
		if err != nil {
			// Fall back to index.html for SPA routing
			req.URL.Path = "/"
		} else {
			_ = f.Close()
		}
		fileServer.ServeHTTP(w, req)
	})
}
