package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bqua0007/kanban-server/internal/api"
	"github.com/bqua0007/kanban-server/internal/kanban"
)

const indexHTML = `<!doctype html><title>Kanban</title><div id="app"></div>`

// frontendHandler is an API handler that also serves the given files, as the real binary serves
// the built Vue app.
func frontendHandler(t *testing.T, files fstest.MapFS) http.Handler {
	t.Helper()
	srv := api.New(kanban.NewMemory(), slog.New(slog.DiscardHandler))
	t.Cleanup(srv.Close)
	srv.ServeFrontend(files)
	return srv.Handler()
}

func builtApp() fstest.MapFS {
	return fstest.MapFS{
		"index.html":            {Data: []byte(indexHTML)},
		"assets/app-1a2b3c.js":  {Data: []byte(`console.log("app")`)},
		"assets/app-1a2b3c.css": {Data: []byte(`body{margin:0}`)},
		"favicon.ico":           {Data: []byte("icon")},
	}
}

func TestServesTheAppAtTheRoot(t *testing.T) {
	rec := do(t, frontendHandler(t, builtApp()), http.MethodGet, "/", "")

	if rec.Code != http.StatusOK || rec.Body.String() != indexHTML {
		t.Fatalf("GET / = %d %q, want the index page", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache so a new build is picked up", got)
	}
}

func TestClientSideRoutesFallBackToTheIndexPage(t *testing.T) {
	handler := frontendHandler(t, builtApp())

	for _, path := range []string{
		"/boards/some-board",
		"/boards/some-board/cards/some-card",
		"/index.html",
		"/nothing-here",
	} {
		t.Run(path, func(t *testing.T) {
			rec := do(t, handler, http.MethodGet, path, "")

			if path == "/index.html" {
				// net/http redirects /index.html to / by convention; either answer is fine.
				if rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusOK {
					t.Errorf("status = %d", rec.Code)
				}
				return
			}
			if rec.Code != http.StatusOK || rec.Body.String() != indexHTML {
				t.Errorf("GET %s = %d %q, want the index page so the Vue router can take over", path, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestServesAssetsWithLongCaching(t *testing.T) {
	handler := frontendHandler(t, builtApp())

	rec := do(t, handler, http.MethodGet, "/assets/app-1a2b3c.js", "")

	if rec.Code != http.StatusOK || rec.Body.String() != `console.log("app")` {
		t.Fatalf("GET asset = %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Errorf("Content-Type = %q, want JavaScript", got)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q, want a long immutable cache for hashed assets", got)
	}
	if css := do(t, handler, http.MethodGet, "/assets/app-1a2b3c.css", ""); !strings.HasPrefix(css.Header().Get("Content-Type"), "text/css") {
		t.Errorf("CSS Content-Type = %q", css.Header().Get("Content-Type"))
	}
}

func TestAMissingFileIsA404NotTheIndexPage(t *testing.T) {
	handler := frontendHandler(t, builtApp())

	for _, path := range []string{"/assets/missing.js", "/missing.png"} {
		rec := do(t, handler, http.MethodGet, path, "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404: serving HTML where a script was expected hides the real problem", path, rec.Code)
		}
	}
}

func TestStaticFilesAreReadOnly(t *testing.T) {
	handler := frontendHandler(t, builtApp())

	if rec := do(t, handler, http.MethodPost, "/", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / = %d, want 405", rec.Code)
	}
	if rec := do(t, handler, http.MethodHead, "/", ""); rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Errorf("HEAD / = %d with %d body bytes, want 200 and none", rec.Code, rec.Body.Len())
	}
}

func TestTheAPIKeepsItsJSONErrorsAlongsideTheFrontend(t *testing.T) {
	handler := frontendHandler(t, builtApp())

	if rec := do(t, handler, http.MethodGet, "/api/health", ""); rec.Code != http.StatusOK {
		t.Errorf("GET /api/health = %d, want 200", rec.Code)
	}
	wantError(t, do(t, handler, http.MethodGet, "/api/nope", ""), http.StatusNotFound)
	wantError(t, do(t, handler, http.MethodPost, "/api/health", ""), http.StatusMethodNotAllowed)
	wantError(t, do(t, handler, http.MethodGet, "/api/boards/x/anything", ""), http.StatusNotFound)
}

func TestWithoutABuiltFrontendThePageSaysHowToBuildIt(t *testing.T) {
	handler := frontendHandler(t, fstest.MapFS{".gitkeep": {Data: nil}})

	rec := do(t, handler, http.MethodGet, "/", "")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET / = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "npm run build:embed") {
		t.Errorf("body %q does not say how to build the frontend", rec.Body.String())
	}
	if rec := do(t, handler, http.MethodGet, "/api/health", ""); rec.Code != http.StatusOK {
		t.Errorf("the API should work without a frontend; GET /api/health = %d", rec.Code)
	}
}

func TestTheServerWorksWithoutServeFrontend(t *testing.T) {
	// The default (all earlier tests): no frontend, so non-API paths are plain 404s.
	wantError(t, do(t, newHandler(t), http.MethodGet, "/", ""), http.StatusNotFound)
}
