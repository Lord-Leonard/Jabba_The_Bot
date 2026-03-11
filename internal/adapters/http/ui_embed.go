package http

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed ui/* ui/assets/*
var embeddedUI embed.FS

func NewEmbeddedUIHandler() http.Handler {
	sub, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		return http.NotFoundHandler()
	}

	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestPath == "." || requestPath == "" {
			http.ServeFileFS(w, r, sub, "index.html")
			return
		}

		if _, statErr := fs.Stat(sub, requestPath); statErr == nil {
			files.ServeHTTP(w, r)
			return
		}

		http.ServeFileFS(w, r, sub, "index.html")
	})
}
