package api

import (
	"fmt"
	"net/http"
	"time"
)

// defaultHeartbeat is how often an idle stream sends a comment line, so proxies and browsers
// do not decide the connection is dead.
const defaultHeartbeat = 25 * time.Second

// wrote announces to every connected client that the state is now at version, and returns the
// version so a handler can use it in its reply.
func (s *Server) wrote(version int64) int64 {
	s.hub.Publish(version)
	return version
}

// events answers GET /api/events with a server-sent event stream: one "version" event with the
// current version when the client connects, and another each time any client writes.
//
// The event only says *that* something changed. The client fetches the new state itself.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)

	// Subscribe before reading the version: a write between the two would otherwise be missed.
	events, cancel, err := s.hub.Subscribe(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer cancel()

	snap, err := s.store.Snapshot(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // ask reverse proxies not to hold the stream back
	if !send(w, rc, "event: version\ndata: {\"version\":%d}\n\n", snap.Version) {
		return
	}

	heartbeat := time.NewTicker(s.heartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return // the client went away
		case version, open := <-events:
			if !open {
				return // we were cut off for being slow, or the server is stopping
			}
			if !send(w, rc, "event: version\ndata: {\"version\":%d}\n\n", version) {
				return
			}
		case <-heartbeat.C:
			if !send(w, rc, ": keep-alive\n\n") {
				return
			}
		}
	}
}

// send writes one chunk of the stream and flushes it to the client. It reports false when the
// connection is no longer writable.
func send(w http.ResponseWriter, rc *http.ResponseController, format string, args ...any) bool {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return false
	}
	return rc.Flush() == nil
}
