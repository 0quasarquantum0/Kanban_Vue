// Command kanban runs the Kanban server: the JSON API and, from the same port, the Vue app.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bqua0007/kanban-server/internal/api"
	"github.com/bqua0007/kanban-server/internal/kanban"
	"github.com/bqua0007/kanban-server/web"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on, for example :8080 or 127.0.0.1:9000")
	dbPath := flag.String("db", "kanban.db", "path of the SQLite database file (created if missing)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Ctrl+C (or a termination signal from the system) cancels ctx, which starts a graceful
	// shutdown instead of killing the process in the middle of a request.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Error("cannot listen", "addr", *addr, "error", err)
		os.Exit(1)
	}
	if err := run(ctx, log, ln, *dbPath, web.Files()); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
	log.Info("stopped")
}

// run opens the database, serves on ln until ctx is cancelled, then shuts down gracefully:
// requests in progress get time to finish, and the database is closed last. It returns nil
// after a clean shutdown.
func run(ctx context.Context, log *slog.Logger, ln net.Listener, dbPath string, frontend fs.FS) error {
	store, err := kanban.OpenSQLite(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("opening the database: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Error("closing the database", "error", err)
		}
	}()
	if err := kanban.EnsureSeeded(ctx, store); err != nil {
		return err
	}

	app := api.New(store, log)
	app.ServeFrontend(frontend)
	srv := &http.Server{
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	// Open event streams never finish by themselves, so Shutdown would wait for them forever.
	// Closing the app ends them.
	srv.RegisterOnShutdown(app.Close)
	defer app.Close()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	log.Info("listening", "addr", ln.Addr().String(), "database", dbPath)

	select {
	case err := <-serveErr:
		return err // the server failed on its own
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
