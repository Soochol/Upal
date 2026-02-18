package api

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// StaticHandler serves the React frontend from a directory.
func StaticHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// StaticHandlerFS serves the React frontend from an embedded filesystem.
func StaticHandlerFS(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(fsys, r.URL.Path[1:]); errors.Is(err, fs.ErrNotExist) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
