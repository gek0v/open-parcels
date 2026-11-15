package web

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed index.html
var embeddedFS embed.FS

// FS returns the embedded filesystem containing web dashboard assets.
func FS() embed.FS {
	return embeddedFS
}

// IsAsset checks if a requested path corresponds to an embedded static dashboard asset.
func IsAsset(path string) bool {
	if path == "/" || path == "/index.html" {
		return true
	}
	clean := strings.TrimPrefix(path, "/")
	if clean == "" {
		return true
	}
	_, err := embeddedFS.Open(clean)
	return err == nil
}

// Handler returns an http.Handler that serves the embedded web dashboard.
// GET / and /index.html serve index.html with appropriate headers.
// Any other embedded static files are served via http.FileServer.
// Non-existent paths return 404 Not Found.
func Handler() http.Handler {
	fileServer := http.FileServer(http.FS(embeddedFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "/index.html" {
			data, err := embeddedFS.ReadFile("index.html")
			if err != nil {
				http.Error(w, "Dashboard asset not found", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}

		clean := strings.TrimPrefix(path, "/")
		if f, err := embeddedFS.Open(clean); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}
