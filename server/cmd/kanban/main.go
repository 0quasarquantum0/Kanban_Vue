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
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log, *addr); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run wires the pieces together and serves until the server fails. Returning an error to main
// (instead of calling os.Exit in here) keeps deferred cleanups working.
func run(log *slog.Logger, addr string) error {
	store := kanban.NewMemory()
	if err := kanban.EnsureSeeded(context.Background(), store); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(store, log).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
