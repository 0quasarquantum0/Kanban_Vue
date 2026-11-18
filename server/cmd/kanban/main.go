// Command kanban runs the Kanban API server.
package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/bqua0007/kanban-server/internal/api"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.NewHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
