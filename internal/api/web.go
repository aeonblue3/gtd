package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func newWebHandler(webRoot string) http.Handler {
	fileServer := http.FileServer(http.Dir(webRoot))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			serveIndex(w, r, webRoot)
			return
		}

		clean := filepath.Clean(strings.TrimPrefix(path, "/"))
		full := filepath.Join(webRoot, clean)
		if info, err := os.Stat(full); err == nil {
			if info.IsDir() {
				index := filepath.Join(full, "index.html")
				if _, err := os.Stat(index); err == nil {
					http.ServeFile(w, r, index)
					return
				}
			} else {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: unknown client-side route -> app entrypoint.
		serveIndex(w, r, webRoot)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, webRoot string) {
	indexPath := filepath.Join(webRoot, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "web assets not found; expected ./web/index.html", http.StatusServiceUnavailable)
		return
	}
	http.ServeFile(w, r, indexPath)
}

