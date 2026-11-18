package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var testFrontend = fstest.MapFS{"index.html": {Data: []byte("<title>Kanban</title>")}}

// start runs the server on a free local port and returns its URL and a function that stops it
// and returns whatever run returned.
func start(t *testing.T, dbPath string) (baseURL string, stop func() error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- run(ctx, slog.New(slog.DiscardHandler), ln, dbPath, testFrontend) }()

	stopped := false
	stop = func() error {
		stopped = true
		cancel()
		select {
		case err := <-result:
			return err
		case <-time.After(5 * time.Second):
			t.Fatal("the server did not shut down within 5 seconds")
			return nil
		}
	}
	t.Cleanup(func() {
		if !stopped {
			_ = stop() // best effort: the test already failed or finished
		}
	})
	return "http://" + ln.Addr().String(), stop
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

func TestServerServesTheAPIAndTheFrontendFromOnePort(t *testing.T) {
	url, _ := start(t, filepath.Join(t.TempDir(), "kanban.db"))

	status, body := get(t, url+"/api/health")
	if status != http.StatusOK || !strings.Contains(body, `"ok"`) {
		t.Errorf("GET /api/health = %d %q", status, body)
	}
	if status, body := get(t, url+"/boards/some-board/cards/some-card"); status != http.StatusOK || !strings.Contains(body, "<title>Kanban") {
		t.Errorf("a deep link = %d %q, want the index page", status, body)
	}
	status, body = get(t, url+"/api/state")
	if status != http.StatusOK || !strings.Contains(body, "Vue Kanban") {
		t.Errorf("GET /api/state = %d %q, want the seeded demo board", status, body)
	}
}

func TestServerKeepsItsDataBetweenRuns(t *testing.T) {
	db := filepath.Join(t.TempDir(), "kanban.db")
	url, stop := start(t, db)
	resp, err := http.Post(url+"/api/boards", "application/json", strings.NewReader(`{"title":"Survivor"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if err := stop(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	url, _ = start(t, db)

	if _, body := get(t, url+"/api/state"); !strings.Contains(body, "Survivor") {
		t.Errorf("the board written before the restart is gone: %s", body)
	}
}

func TestShutdownIsGracefulEvenWithAnOpenEventStream(t *testing.T) {
	url, stop := start(t, filepath.Join(t.TempDir(), "kanban.db"))
	resp, err := http.Get(url + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if line, _ := bufio.NewReader(resp.Body).ReadString('\n'); !strings.HasPrefix(line, "event: version") {
		t.Fatalf("first line of the stream = %q", line)
	}

	started := time.Now()
	if err := stop(); err != nil {
		t.Fatalf("shutdown returned %v, want nil", err)
	}

	if took := time.Since(started); took > 3*time.Second {
		t.Errorf("shutdown took %v: an idle browser tab must not hold the server up", took)
	}
	if _, err := http.Get(url + "/api/health"); err == nil {
		t.Error("the server still answers after shutdown")
	}
}

func TestRunFailsClearlyWhenTheDatabaseCannotBeOpened(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	notADirectory := filepath.Join(t.TempDir(), "missing-dir", "kanban.db")

	err = run(context.Background(), slog.New(slog.DiscardHandler), ln, notADirectory, testFrontend)

	if err == nil {
		t.Fatal("run succeeded with an unusable database path")
	}
	if !strings.Contains(err.Error(), "opening the database") {
		t.Errorf("error %q does not say which step failed", err)
	}
}
