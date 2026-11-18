package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverPanicsTurnsAPanicIntoAJSON500(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	handler := recoverPanics(log, panicking)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
	}
	if body["error"] != "internal server error" {
		t.Errorf("error = %q, want a generic message", body["error"])
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("the panic value leaked to the client")
	}
	if !strings.Contains(logs.String(), "boom") {
		t.Errorf("the panic was not logged: %q", logs.String())
	}
}

func TestRecoverPanicsLeavesNormalResponsesAlone(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	rec := httptest.NewRecorder()

	recoverPanics(slog.New(slog.DiscardHandler), ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", rec.Code)
	}
}

func TestStatusRecorderKeepsFlushAvailable(t *testing.T) {
	// Server-sent events (step 7) need http.ResponseController to reach Flush through our
	// wrapper, which only works if the wrapper has an Unwrap method.
	rec := httptest.NewRecorder()
	wrapped := &statusRecorder{ResponseWriter: rec}

	if err := http.NewResponseController(wrapped).Flush(); err != nil {
		t.Errorf("Flush through the wrapper: %v", err)
	}
	if !rec.Flushed {
		t.Error("the underlying writer was not flushed")
	}
}
