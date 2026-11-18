# CLAUDE.md — Go backend for the Vue Kanban board

## Purpose

This is the second **learning project** in this repo. The developer knows TypeScript, React, Next.js and now Vue 3 (the app in `../src`, built in phases 0–9). They are learning **Go**. The goal is a Go API that the existing Vue app talks to instead of `localStorage`. The more important goal is that the developer understands *how Go does things*.

The developer already knows what the API needs to do, because they built the frontend. Use that: every Go concept should be tied back to something in the Vue app they recognise.

## Working agreement (read first)

This project runs differently from the Vue one.

- **Build all the steps in one go.** Do not stop between steps or wait for "next". When a step is done and committed, start the next one immediately. Only stop early for a real blocker: Go is not installed, a check fails and you cannot fix it, or a decision only the developer can make. In that case, say exactly what is blocking and what you need.
- **Commit at the end of every step**, with a message like `go step 3: http api for whole-state sync`. Each commit must leave everything green (see "Checks before every commit").
- **Tests first.** For each piece of behaviour, write the test before the code, in the same commit. The developer will read the tests to learn what the code is meant to do, so make test names read like sentences.
- **Explain Go concepts in `LEARNING.md`, not in chat.** Since there are no pauses, the explanations go in `server/LEARNING.md`. Every time a Go concept appears for the first time, add a short entry (3–6 sentences): what it is, how it compares to TypeScript (and to Vue or React where that helps), and a `file:line` pointer to where it is used. Group entries under the step that introduced them. Skip general programming basics; the developer knows them.
- **Write idiomatic Go, not TypeScript-in-Go.** Return errors as values (no panics for expected failures). Keep interfaces small and define them where they are used. Prefer plain structs and functions over layers of abstraction. Pass `context.Context` as the first argument to anything that does I/O. Let `gofmt` decide formatting.
- **Standard library first.** The only approved third-party module is `modernc.org/sqlite`. Do not add any other Go module or npm package; find a standard-library way instead. If something truly cannot be done without one, stop and ask.
- **Don't break the Vue app.** Frontend changes still follow `../CLAUDE.md`'s Vue conventions (`<script setup lang="ts">`, no `any`, components under about 150 lines, `defineModel`/`defineEmits`, and so on).
- **Keep the architecture diagram current.** Add backend diagrams to `../docs/architecture.md` as each part comes into existence: the package layout, the request flow, the database schema, and the live-update flow. Diagram only what exists. Update the "Last updated" line to the Go step.
- **Finish with a summary in chat**: the commits made, what to read first, and anything left undone or unverified.

### Checks before every commit

From `server/`:

```bash
gofmt -l .          # must print nothing
go vet ./...
go test ./...
```

If the step touched the frontend, also run from the repo root: `npm run type-check`, `npm run lint`, `npx vitest run` and `npm run build`. Check behaviour in the browser too, as the Vue phases did.

### Before step 0

Run `go version`. If Go is missing, stop and ask the developer to install the latest stable release from https://go.dev/dl/ (1.22 or newer is required for the routing patterns used here). Do not install it yourself.

## Stack

| Concern | Choice |
|---|---|
| Language | Go (latest stable, at least 1.22) |
| HTTP | `net/http` with the method-and-path `ServeMux` patterns (`"GET /api/boards/{boardId}"`) |
| JSON | `encoding/json` |
| Database | SQLite through `database/sql` and `modernc.org/sqlite` (pure Go, so no C compiler is needed on Windows) |
| Migrations | Plain `.sql` files embedded with `embed`, applied in order at startup |
| Live updates | Server-sent events, using goroutines and channels (no WebSocket library) |
| Tests | `testing` and `net/http/httptest`, table-driven where it fits |
| Frontend calls | `fetch` in the Pinia store (no new npm packages) |

## Commands

From `server/`:

```bash
go run ./cmd/kanban            # API on http://localhost:8080
go test ./...
go vet ./...
gofmt -l .
go build -o kanban.exe ./cmd/kanban
```

During development, run `npm run dev` in the repo root at the same time. From step 4, Vite proxies `/api` to the Go server.

## Target structure

```
server/
  go.mod                      module github.com/<you>/kanban-server (any module path is fine)
  cmd/kanban/main.go          flags, wiring, graceful shutdown
  internal/kanban/
    types.go                  Board, Column, Card, State: mirrors ../src/types.ts
    validate.go               ports ../src/validation.ts
    seed.go                   ports ../src/seed.ts
    store.go                  the Store interface
    memory.go                 in-memory Store (steps 2–4)
    sqlite.go                 SQLite Store (step 5 onwards)
    migrations/001_init.sql
  internal/api/
    server.go                 routes and middleware
    state.go                  GET/PUT /api/state
    resources.go              per-action endpoints (step 6)
    events.go                 SSE hub (step 7)
    static.go                 embedded frontend (step 8)
  web/dist/                   built Vue app for embedding (gitignored except a placeholder)
  LEARNING.md
  QUIZ.md
  CLAUDE.md, AGENTS.md        this file (keep both identical)
```

`internal/` is deliberate: Go stops code outside `server/` from importing those packages. Explain this in `LEARNING.md` when it first comes up.

## API contract

Write the contract in `../docs/api.md` as the endpoints are built, in a language-neutral way (method, path, request body, responses, status codes). Later backends in other languages will implement the same contract, so the Vue app can switch between them.

- JSON everywhere. Errors are `{"error": "message"}` with a fitting status code (400, 404, 409, 500).
- The JSON shape of `State` is exactly the frontend's `KanbanState` (and its Export JSON format).
- Every write increases a single `version` number. Clients send the version they last saw, and a stale write gets `409 Conflict`.
- IDs are UUID strings. The server accepts an `id` from the client when it is a valid, unused UUID (so the frontend can update optimistically), and generates one otherwise.

## Steps

### Step 0 — Module and first server

- `go mod init`, `cmd/kanban/main.go` with an HTTP server on `:8080` and `GET /api/health` returning `{"status":"ok"}`.
- A first test using `httptest`.
- Start `LEARNING.md` with: modules and packages, `package main` and `func main`, exported vs unexported names (capitalisation instead of `export`), `go run` vs `go build`, and the toolchain replacing ESLint/Prettier/Vitest.
- **Done when:** the server runs, the test passes, and the checks are clean.

### Step 1 — Types, seed and validation

- Port `../src/types.ts` to structs with JSON tags. `Record<ID, Card>` becomes `map[string]Card`; `string | null` becomes `*string`.
- Port `../src/seed.ts` and `../src/validation.ts`. Validation returns an `error` saying what is wrong, not just `false`.
- Port the cases from `../src/__tests__/validation.spec.ts` as a table-driven test.
- **Explain:** structs vs interfaces in TS, struct tags, maps and slices, pointers (only as far as needed for `null` and for mutation), errors as values, table-driven tests and `t.Run`.

### Step 2 — In-memory store and the board operations

- A `Store` interface with the operations the Pinia store has: get state, replace state, add/rename/delete board, add/rename/delete column, add/update/delete card, move card. Deleting a board or column deletes what it contains.
- `memory.go` implements it, guarded by a `sync.Mutex`.
- Port the cases from `../src/stores/__tests__/kanban.spec.ts`. Write them against the interface so step 5 can run the same tests on SQLite.
- **Explain:** methods and pointer receivers, implicit interface satisfaction (compare with TS structural typing), why a mutex is needed (HTTP handlers run concurrently; the browser's JS never did), `defer` for unlocking.

### Step 3 — HTTP API for whole-state sync

- `GET /api/state` returns `{ "version": n, "state": {...} }`.
- `PUT /api/state` takes the same shape. It validates, checks the version, saves, and returns the new version. A stale version gets 409; invalid data gets 400.
- Middleware for request logging and for turning unexpected errors into JSON 500s.
- Handler tests with `httptest`. Start `../docs/api.md`.
- **Explain:** `http.Handler` and `http.HandlerFunc`, `ServeMux` patterns and `r.PathValue` (compare with Vue Router's route table), middleware as a function wrapping a handler, decoding JSON safely (`DisallowUnknownFields`, size limits), status codes.

### Step 4 — Connect the Vue app

- Vite proxies `/api` to the Go server, with the target read from a `KANBAN_API` environment variable that defaults to `http://localhost:8080`. This makes it easy to point the app at another backend later.
- The Pinia store loads state from `GET /api/state` instead of `useStorage`. Changes are saved with a debounced deep watch that sends `PUT /api/state` (the same idea as phase 5, with a server behind it). On 409, reload from the server and tell the user their change conflicted.
- Route guards wait for the first load, since the data now arrives asynchronously.
- Show loading and error states (server unreachable).
- Update the frontend store tests to stub `fetch` instead of using `localStorage`. The theme preference stays in `localStorage`.
- **Explain (in `LEARNING.md`, briefly):** how the proxy works, and why the Vue side needed async guards.
- **Done when:** a refresh keeps changes because the Go server has them, and two tabs don't silently overwrite each other.

### Step 5 — SQLite persistence

- Tables for boards, columns and cards, with an integer `position` for order, foreign keys and `ON DELETE CASCADE`. Migrations are embedded `.sql` files applied at startup.
- `sqlite.go` implements `Store`. Moving a card is one transaction. Run the step 2 tests against both implementations.
- `main.go` takes a `-db` flag (default `kanban.db`, gitignored).
- **Explain:** `database/sql` (connection pool, `QueryContext`, `Scan`), transactions with `defer tx.Rollback()`, `embed`, the blank import that registers a driver, and why `Store` is an interface (two implementations, one set of tests).

### Step 6 — Per-action endpoints

- Resource endpoints mirroring the store actions:

  | Method and path | Action |
  |---|---|
  | `POST /api/boards` | add board |
  | `PATCH /api/boards/{boardId}` | rename board |
  | `DELETE /api/boards/{boardId}` | delete board and its contents |
  | `POST /api/boards/{boardId}/columns` | add column |
  | `PATCH /api/columns/{columnId}` | rename column |
  | `DELETE /api/columns/{columnId}` | delete column and its cards |
  | `POST /api/columns/{columnId}/cards` | add card |
  | `PATCH /api/cards/{cardId}` | update title, description, labels, due date |
  | `DELETE /api/cards/{cardId}` | delete card |
  | `PUT /api/cards/{cardId}/position` | move: `{ "columnId": "...", "index": n }` |

- `PUT /api/state` stays for Import JSON and Reset to demo data.
- Frontend: each store action updates the state locally first, then calls its endpoint. On failure, reload the state and show an error. The whole-state watch from step 4 is removed.
- Two places currently write titles straight into store objects, bypassing actions: `v-model:title="column.title"` in `../src/components/BoardView.vue` and `v-model="board.title"` in `../src/views/BoardsListView.vue`. Route both through `renameColumn` and `renameBoard`, or the server never hears about the renames.
- **Explain:** designing REST resources, `PATCH` vs `PUT`, returning 404 for missing IDs, and optimistic updates (compare with the drag-and-drop and Pinia code they already know).

### Step 7 — Live updates with server-sent events

- `GET /api/events` streams an event whenever the version changes.
- A hub goroutine owns the set of subscribers. Subscribe, unsubscribe and broadcast go through channels, and `select` handles them. Each client gets a buffered channel; a slow client is dropped rather than blocking everyone. Disconnects are detected through `r.Context().Done()`.
- Frontend: an `EventSource` in the store. When an event's version is newer than the local one, refetch the state. Close it when the app unmounts.
- Test the hub without HTTP, and the endpoint with `httptest.NewServer`.
- **Explain:** goroutines (compare with the browser event loop and `async`/`await`), channels, `select`, the "share memory by communicating" idea, `context` cancellation, and why this is where Go shines.
- **Done when:** a change in one tab appears in another without a refresh.

### Step 8 — One binary

- Add an npm script `build:embed` that runs `vite build --outDir server/web/dist --emptyOutDir`. `//go:embed` cannot reach outside `server/`, which is why the build output has to land inside it. Keep a placeholder file so `go build` works before the frontend is built, and gitignore the rest.
- Serve the embedded app at `/`, with an `index.html` fallback for client-side routes such as `/boards/x/cards/y`.
- Flags for port and database path. Graceful shutdown with `signal.NotifyContext` and `http.Server.Shutdown`.
- **Explain:** `embed.FS`, `flag`, graceful shutdown, and cross-compiling (`GOOS`/`GOARCH`) to show the single-binary appeal.
- **Done when:** `go build` produces one executable that serves the whole app, and deep links work after a refresh.

### Step 9 — Finish

- `server/README.md`: how to run, build and test the backend.
- Root `../README.md`: update the "Go backend" section from a plan to a description of what exists.
- Complete `LEARNING.md`, and add an "Exercises" section with three small pieces for the developer to delete and rewrite themselves (for example the move-card SQL, the SSE hub, the validation function), with the tests that will tell them when they're right.
- `QUIZ.md`: about 10 questions on the Go code, with answers inside `<details>` blocks.
- Refresh `../docs/architecture.md`. Refresh the Graphify graph with `graphify update .` from the repo root and `GRAPHIFY_BACKEND=none` set, which uses no model. Do not run `graphify install` or any `graphify <platform> install` command. `graphify-out/` is gitignored.
- **Done when:** all checks, Go and frontend, are clean and the binary serves a working app.

## Optional follow-ups (only if asked)

These came out of the same "which language next" discussion. Each is a separate session.

- **Ruby:** the same API in Sinatra or Rails, in `../server-ruby/`, implementing `../docs/api.md`. Point the Vue app at it with `KANBAN_API`. It shows the productivity-vs-control trade-off against the Go version directly.
- **Rust:** a small `grep` clone (regex, recursive directory walk, coloured output, `--ignore-case`, later parallel). It is standalone and doesn't connect to the Kanban app, so it belongs in its own folder outside this repo.
- **Comparison exercise:** the same small program (a word-frequency counter or a CSV summariser) in Go, Rust and Ruby side by side, in a separate folder.

## Conventions

- `gofmt` formatting, `go vet` clean, no ignored errors (`_ =` only with a comment saying why).
- Package names are short, lowercase, single words. No `utils` or `helpers` packages.
- Exported identifiers get a doc comment starting with their name.
- No `interface{}` or `any`, except at JSON boundaries where it truly can't be avoided.
- Keep files focused. Split a file once it does two unrelated things, as with the Vue components.
