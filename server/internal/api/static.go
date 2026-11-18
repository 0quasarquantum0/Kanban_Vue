package api

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// ServeFrontend makes the server also serve the built Vue app from files (whose root holds
// index.html). Call it before Handler. Without it the server is API-only.
func (s *Server) ServeFrontend(files fs.FS) { s.frontend = files }

// isAPIPath reports whether the request belongs to the JSON API rather than the frontend.
func isAPIPath(p string) bool { return p == "/api" || strings.HasPrefix(p, "/api/") }

// frontendHandler serves the files of a built single-page app:
//
//   - a file that exists is served as it is (assets have hashed names, so they are cached for a
//     year; anything else must be re-checked);
//   - a path with no file extension is a client-side route such as /boards/x/cards/y, which only
//     the Vue router understands, so it gets index.html and the router takes over;
//   - a path with an extension that does not exist (a missing script, say) is a real 404, since
//     answering it with HTML would only produce a confusing error in the browser.
func frontendHandler(files fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if info, err := fs.Stat(files, name); err == nil && !info.IsDir() && name != "index.html" {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, files, name)
			return
		}
		if path.Ext(name) != "" && name != "index.html" {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r, files)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, files fs.FS) {
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		http.Error(w, "The frontend has not been built into this binary.\n"+
			"From the repository root run: npm run build:embed, then rebuild the server.\n",
			http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	// ServeContent sets Content-Type from the name and handles HEAD; a zero time means "no
	// modification date", so no conditional-request headers are made up.
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
}
