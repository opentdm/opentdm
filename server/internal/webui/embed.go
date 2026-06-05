// Package webui embeds the built React/Primer SPA into the server binary and
// serves it with history-API fallback. In production the UI ships inside the
// single binary; OPENTDM_WEB_DIR can serve it from disk during development.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler serves the embedded SPA.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return spaHandler(
		http.FS(sub),
		func(name string) bool { _, err := fs.Stat(sub, name); return err == nil },
		func() ([]byte, error) { return fs.ReadFile(sub, "index.html") },
	), nil
}

// DirHandler serves the SPA from a directory on disk.
func DirHandler(dir string) http.Handler {
	fsys := os.DirFS(dir)
	return spaHandler(
		http.Dir(dir),
		func(name string) bool { _, err := fs.Stat(fsys, name); return err == nil },
		func() ([]byte, error) { return os.ReadFile(filepath.Join(dir, "index.html")) },
	)
}

// spaHandler serves static files when they exist, otherwise falls back to
// index.html so client-side routes resolve.
func spaHandler(fsys http.FileSystem, exists func(string) bool, index func() ([]byte, error)) http.Handler {
	fileServer := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && exists(name) {
			fileServer.ServeHTTP(w, r)
			return
		}
		data, err := index()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
}
