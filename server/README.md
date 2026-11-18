# Kanban server (Go)

The backend for the Vue Kanban app in the parent folder: a JSON API, live updates, and (once the frontend is built) the app itself, all from one executable. It uses only the Go standard library plus the pure-Go SQLite driver `modernc.org/sqlite`.

New to Go? Read [`LEARNING.md`](LEARNING.md) first (its "Start here" list says in what order), then try [`QUIZ.md`](QUIZ.md). The HTTP contract is in [`../docs/api.md`](../docs/api.md); diagrams are in [`../docs/architecture.md`](../docs/architecture.md).

## Requirements

Go 1.22 or newer (developed with 1.27). Nothing else: no C compiler, no database server.

## Run

From this folder:

```bash
go run ./cmd/kanban                       # API on http://localhost:8080, data in ./kanban.db
go run ./cmd/kanban -addr :9000           # another port (or 127.0.0.1:9000 to stay local)
go run ./cmd/kanban -db /tmp/boards.db    # another database file
```

On first start the database is created and filled with the demo board. Stop with Ctrl+C; the server finishes the requests in progress and closes the database.

With the Vue app in development, run this server and `npm run dev` (in the parent folder) side by side. Vite forwards `/api` to `http://localhost:8080`; to use another address set `KANBAN_API` (environment variable or a git-ignored `.env.local` in the parent folder).

## Build one executable that also serves the app

```bash
cd ..                                      # the repository root
npm run build:embed                        # builds the Vue app into server/web/dist
cd server
go build -o kanban.exe ./cmd/kanban        # on Linux or macOS: -o kanban
./kanban.exe                               # open http://localhost:8080
```

The built app is embedded in the executable (`web/embed.go`), so that one file is all you need to deploy. Without `npm run build:embed` the server still builds and runs; `/` then explains how to build the frontend. To build for another system, set `GOOS` and `GOARCH`, for example `GOOS=linux GOARCH=arm64 go build -o kanban ./cmd/kanban`.

## Test and check

```bash
go test ./...        # all tests
go vet ./...         # the compiler's extra checks
gofmt -l .           # lists badly formatted files; prints nothing when clean
go test -race ./...  # also looks for data races (needs a C compiler; see below)
```

Run a part of the suite with `-run`, for example `go test ./internal/kanban -run TestSQLite -v`.

`-race` needs `gcc`. It was not available where this code was written, so the concurrent code (the in-memory store's mutex and the event hub) was checked by tests that run it from many goroutines, but not with the race detector. Run it once on a machine that has a C compiler.

## Layout

```
cmd/kanban/main.go          flags, wiring, graceful shutdown
internal/kanban/            the model and the Store
  types.go, validate.go, seed.go, id.go, errors.go
  store.go                  the Store interface, CardPatch, Snapshot
  memory.go                 Store in memory (mutex)
  sqlite.go, migrate.go     Store in SQLite; migrations/*.sql are embedded
internal/api/               HTTP: routes, handlers, middleware
  server.go, json.go, middleware.go
  state.go, resources.go    GET/PUT /api/state and the per-action endpoints
  hub.go, events.go         live updates (server-sent events)
  static.go                 serves the embedded Vue app
web/                        embed.go and the built frontend (dist/, not committed)
LEARNING.md, QUIZ.md        for the reader
```

`internal/` is enforced by the compiler: nothing outside `server/` can import those packages.

## Behaviour worth knowing

- **One version number.** Every write raises it by one. `PUT /api/state` must send the version it was based on, and gets `409 Conflict` if another client has written since. The per-action endpoints carry no version, because each describes a small change that rarely conflicts.
- **IDs.** The client may propose an `id` when it creates something. The server keeps it if it is a valid, unused UUID and invents one otherwise.
- **Validation is stricter than the frontend's was** in one way: every column must belong to a board and every card to a column, because the database cannot store orphans.
- **No accounts, no authentication.** Anyone who can reach the port can read and change everything. Do not expose it to the internet as it is.
