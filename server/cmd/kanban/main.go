// Command kanban runs the Kanban API server.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/bqua0007/kanban-server/internal/api"
	"github.com/bqua0007/kanban-server/internal/kanban"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dbPath := flag.String("db", "kanban.db", "path of the SQLite database file (created if missing)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log, *addr, *dbPath); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run wires the pieces together and serves until the server fails. Returning an error to main
// (instead of calling os.Exit in here) keeps deferred cleanups working.
func run(log *slog.Logger, addr, dbPath string) error {
	ctx := context.Background()

	store, err := kanban.OpenSQLite(ctx, dbPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Error("closing the database", "error", err)
		}
	}()
	if err := kanban.EnsureSeeded(ctx, store); err != nil {
		return err
	}
	log.Info("database ready", "path", dbPath)

	app := api.New(store, log)
	defer app.Close()
	srv := &http.Server{
		Addr:              addr,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
