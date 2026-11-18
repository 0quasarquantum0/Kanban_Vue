package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder wraps a ResponseWriter to remember the status code that was sent.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status, r.wroteHeader = status, true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.status, r.wroteHeader = http.StatusOK, true
	}
	return r.ResponseWriter.Write(p)
}

// Unwrap lets http.ResponseController reach the real writer underneath, so features such as
// Flush (needed for server-sent events) keep working through this wrapper.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// logRequests logs one line per request, after it has been handled.
//
// Middleware in Go is just a function that takes a handler and returns a handler that does
// something before and after calling it.
func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).Round(time.Microsecond),
		)
	})
}

// recoverPanics turns a panic in a handler into a JSON 500 instead of a dropped connection.
// Expected failures are returned as errors; a panic is a bug, so it is logged with a stack.
func recoverPanics(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			value := recover()
			if value == nil {
				return
			}
			if value == http.ErrAbortHandler {
				panic(value) // net/http uses this panic on purpose to abort a response
			}
			log.Error("panic in handler",
				"method", r.Method, "path", r.URL.Path,
				"panic", fmt.Sprint(value), "stack", string(debug.Stack()))
			writeError(w, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}

// jsonRoutingErrors makes the 404 and 405 answers of the ServeMux speak JSON like everything
// else. The mux decides between them (and sets the Allow header); we only replace the body.
func jsonRoutingErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := mux.Handler(r)
		if pattern != "" {
			// Go through mux.ServeHTTP, not handler.ServeHTTP: only the mux fills in the values
			// of {wildcards} that r.PathValue reads.
			mux.ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(&routingErrorWriter{ResponseWriter: w}, r)
	})
}

// routingErrorWriter swaps the plain-text body of the mux's 404 and 405 for a JSON one.
type routingErrorWriter struct {
	http.ResponseWriter
	replaced bool
}

func (w *routingErrorWriter) WriteHeader(status int) {
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		w.replaced = true
		w.Header().Set("Content-Type", "application/json")
		w.Header().Del("X-Content-Type-Options")
		w.ResponseWriter.WriteHeader(status)
		message := "not found"
		if status == http.StatusMethodNotAllowed {
			message = "method not allowed"
		}
		_, _ = w.ResponseWriter.Write([]byte(`{"error":"` + message + `"}` + "\n"))
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *routingErrorWriter) Write(p []byte) (int, error) {
	if w.replaced {
		return len(p), nil // drop the mux's own text body
	}
	return w.ResponseWriter.Write(p)
}

func (w *routingErrorWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
