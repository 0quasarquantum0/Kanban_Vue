package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

// liveServer runs the whole API on a real local port, which a streaming response needs: a
// ResponseRecorder cannot be read while the handler is still running.
func liveServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	srv := New(kanban.NewMemory(), slog.New(slog.DiscardHandler))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})
	return ts, srv
}

// stream is an open /api/events connection.
type stream struct {
	lines  chan string
	cancel context.CancelFunc
	resp   *http.Response
}

func openStream(t *testing.T, baseURL string) *stream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	s := &stream{lines: make(chan string, 100), cancel: cancel, resp: resp}
	t.Cleanup(s.close)

	go func() {
		defer close(s.lines)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			s.lines <- scanner.Text()
		}
	}()
	return s
}

func (s *stream) close() {
	s.cancel()
	_ = s.resp.Body.Close() // the test is over; a failing close tells us nothing
}

// next returns the next line that is not blank.
func (s *stream) next(t *testing.T) string {
	t.Helper()
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				t.Fatal("the stream ended")
			}
			if line != "" {
				return line
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for the next line of the stream")
		}
	}
}

// nextVersion reads one "version" event and returns its version.
func (s *stream) nextVersion(t *testing.T) int64 {
	t.Helper()
	if line := s.next(t); line != "event: version" {
		t.Fatalf("line = %q, want %q", line, "event: version")
	}
	data := s.next(t)
	var event struct {
		Version int64 `json:"version"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(data, "data: ")), &event); err != nil {
		t.Fatalf("data line %q is not an event: %v", data, err)
	}
	return event.Version
}

func post(t *testing.T, url, body string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s = %d: %s", url, resp.StatusCode, data)
	}
}

func TestEventsIsAnEventStream(t *testing.T) {
	ts, _ := liveServer(t)

	s := openStream(t, ts.URL)

	if got := s.resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if got := s.resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
}

func TestEventsStartsWithTheCurrentVersion(t *testing.T) {
	ts, _ := liveServer(t)
	post(t, ts.URL+"/api/boards", `{"title":"One"}`)
	post(t, ts.URL+"/api/boards", `{"title":"Two"}`)

	s := openStream(t, ts.URL)

	if got := s.nextVersion(t); got != 2 {
		t.Errorf("first event version = %d, want the current version 2", got)
	}
}

func TestEventsAnnouncesEveryWrite(t *testing.T) {
	ts, _ := liveServer(t)
	s := openStream(t, ts.URL)
	if got := s.nextVersion(t); got != 0 {
		t.Fatalf("first event = %d, want 0", got)
	}

	post(t, ts.URL+"/api/boards", `{"title":"One"}`)
	if got := s.nextVersion(t); got != 1 {
		t.Errorf("after the first write: version %d, want 1", got)
	}

	post(t, ts.URL+"/api/boards", `{"title":"Two"}`)
	if got := s.nextVersion(t); got != 2 {
		t.Errorf("after the second write: version %d, want 2", got)
	}
}

func TestEventsReachEveryConnectedClient(t *testing.T) {
	ts, _ := liveServer(t)
	a, b := openStream(t, ts.URL), openStream(t, ts.URL)
	a.nextVersion(t)
	b.nextVersion(t)

	post(t, ts.URL+"/api/boards", `{"title":"Shared"}`)

	if got := a.nextVersion(t); got != 1 {
		t.Errorf("client a: version %d, want 1", got)
	}
	if got := b.nextVersion(t); got != 1 {
		t.Errorf("client b: version %d, want 1", got)
	}
}

func TestFailedWritesAreNotAnnounced(t *testing.T) {
	ts, _ := liveServer(t)
	s := openStream(t, ts.URL)
	s.nextVersion(t)

	resp, err := http.Post(ts.URL+"/api/boards", "application/json", strings.NewReader(`{"title":""}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	post(t, ts.URL+"/api/boards", `{"title":"Real"}`)

	if got := s.nextVersion(t); got != 1 {
		t.Errorf("version %d, want 1: the rejected write must not have been announced", got)
	}
}

func TestEventsSendsHeartbeatsWhileIdle(t *testing.T) {
	srv := New(kanban.NewMemory(), slog.New(slog.DiscardHandler))
	srv.heartbeat = 10 * time.Millisecond
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})
	s := openStream(t, ts.URL)
	s.nextVersion(t)

	if line := s.next(t); !strings.HasPrefix(line, ":") {
		t.Errorf("line = %q, want a comment line starting with a colon", line)
	}
}

func TestADisconnectedClientIsRemoved(t *testing.T) {
	ts, srv := liveServer(t)
	s := openStream(t, ts.URL)
	s.nextVersion(t)
	if got := srv.hub.subscriberCount(); got != 1 {
		t.Fatalf("%d subscribers while connected, want 1", got)
	}

	s.close() // the browser tab was closed

	deadline := time.Now().Add(2 * time.Second)
	for srv.hub.subscriberCount() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the subscriber is still registered after the client disconnected")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
