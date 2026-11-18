package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bqua0007/kanban-server/internal/api"
	"github.com/bqua0007/kanban-server/internal/kanban"
)

// newHandler returns the API handler on top of a fresh in-memory store, with logging off.
func newHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, _ := newHandlerWithStore(t)
	return handler
}

func newHandlerWithStore(t *testing.T) (http.Handler, kanban.Store) {
	t.Helper()
	store := kanban.NewMemory()
	return newHandlerOn(t, store), store
}

// newHandlerOn returns the API handler on top of the given store, with logging off.
func newHandlerOn(t *testing.T, store kanban.Store) http.Handler {
	t.Helper()
	srv := api.New(store, slog.New(slog.DiscardHandler))
	t.Cleanup(srv.Close)
	return srv.Handler()
}

// failingStore is a Store whose Snapshot always fails. Embedding the Store interface gives it
// every other method for free; only Snapshot is replaced.
type failingStore struct{ kanban.Store }

func (failingStore) Snapshot(context.Context) (kanban.Snapshot, error) {
	return kanban.Snapshot{}, errors.New("disk on fire")
}

// do sends one request straight into the handler, without a network.
func do(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decode reads a JSON response body into dst.
func decode(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("body %q is not the expected JSON: %v", rec.Body.String(), err)
	}
}

// wantError checks the status code and that the body is {"error": "..."}, and returns the
// message so the caller can check what it says.
func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int) string {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	decode(t, rec, &body)
	if body["error"] == "" || len(body) != 1 {
		t.Fatalf(`body = %s, want {"error": "<message>"}`, rec.Body.String())
	}
	return body["error"]
}

func TestHealthReportsOK(t *testing.T) {
	rec := do(t, newHandler(t), http.MethodGet, "/api/health", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	decode(t, rec, &body)
	if body["status"] != "ok" {
		t.Errorf(`body["status"] = %q, want "ok"`, body["status"])
	}
}

func TestRoutingErrorsAreJSON(t *testing.T) {
	handler := newHandler(t)

	t.Run("an unknown path is a 404", func(t *testing.T) {
		wantError(t, do(t, handler, http.MethodGet, "/api/nope", ""), http.StatusNotFound)
	})
	t.Run("a wrong method is a 405 that names the allowed methods", func(t *testing.T) {
		rec := do(t, handler, http.MethodPost, "/api/health", "")

		wantError(t, rec, http.StatusMethodNotAllowed)
		if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
			t.Errorf("Allow = %q, want it to list GET", allow)
		}
	})
}

func TestRequestsAreLogged(t *testing.T) {
	var logs bytes.Buffer
	srv := api.New(kanban.NewMemory(), slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(srv.Close)
	handler := srv.Handler()

	do(t, handler, http.MethodGet, "/api/health", "")

	line := logs.String()
	for _, want := range []string{"method=GET", "path=/api/health", "status=200"} {
		if !strings.Contains(line, want) {
			t.Errorf("log %q does not contain %q", line, want)
		}
	}
}
