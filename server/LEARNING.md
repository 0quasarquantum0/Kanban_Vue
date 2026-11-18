# Learning Go through the Kanban backend

Each entry says what a Go concept is, how it compares to TypeScript (or Vue/React), and where to see it in this code. Entries are grouped by the step that first used them. Read them in order; later steps assume the earlier ones.

## Step 0 — Module and first server

### Modules and packages

A **module** is a project with a `go.mod` file (like `package.json`, but with no scripts or dev-dependencies: just the module path, the Go version and required modules). A **package** is one directory of `.go` files that share a `package x` line (like an ES module, except the unit is the whole folder, not one file). Every file in a folder is in the same package and sees all the others' names without importing them. You import a package by its full path: `github.com/bqua0007/kanban-server/internal/api`. The path need not exist online; it is only a unique name. See `go.mod:1` and the import block at `cmd/kanban/main.go:11`.

### `package main` and `func main`

A package called `main` builds an executable; any other package is a library. The program starts at `func main()` in package `main`, with no framework, no `createApp(...).mount()`: you build the server and start it yourself. `cmd/kanban/main.go:14` is the whole entry point. The `cmd/<name>/` layout is a convention that lets one module hold several programs.

### Exported vs unexported names

There is no `export` keyword. A name that starts with a capital letter (`NewHandler`) is visible to other packages; lowercase (`health`) is private to its own package. This applies to functions, types, struct fields and methods. Compare `api.NewHandler` at `internal/api/server.go:10` with the private `health` just below it.

### `internal/`

A directory named `internal` is enforced by the compiler: only code inside the parent of `internal/` (here `server/`) can import what is inside it. That is stronger than a TypeScript "don't import this" comment: nobody outside `server/` can depend on `internal/api`, so we are free to change it. All the real code lives under `server/internal/`; only `cmd/` is the public face.

### `http.Handler`, `ServeMux` and method patterns

An `http.Handler` is anything with a `ServeHTTP(w, r)` method. `http.ServeMux` is the router (compare Vue Router's route table). Since Go 1.22 a pattern can include the method and path wildcards: `"GET /api/health"`. A request to that path with another method gets `405 Method Not Allowed` for free, which the second test checks. See `internal/api/server.go:12`. (Step 3 goes deeper.)

### `go run` vs `go build`, and the toolchain

`go run ./cmd/kanban` compiles to a temporary executable and runs it (like `tsx` or `vite-node`). `go build -o kanban.exe ./cmd/kanban` writes a real, self-contained executable that needs no Go installation to run (nothing like `node_modules`). The `go` command also replaces most of the tooling you add to a JS project by hand:

| JS/TS project | Go equivalent (already installed) |
|---|---|
| Prettier | `gofmt` (one style, no config; `gofmt -l .` lists unformatted files) |
| ESLint / `tsc` checks | `go vet ./...` and the compiler itself (unused imports and variables are *errors*) |
| Vitest | `go test ./...` and the `testing` package |
| `npm install` | `go mod tidy` / `go get` |

### Tests live next to the code

A test is a function `func TestXxx(t *testing.T)` in a file ending in `_test.go`. There is no `describe`/`it`/`expect`: you write plain Go and call `t.Errorf` (report and continue) or `t.Fatalf` (report and stop). `package api_test` (with the `_test` suffix) makes this an *external* test: it can only use what `api` exports, just like a real client. See `internal/api/server_test.go:12`.

`httptest.NewRecorder()` stands in for the network: `handler.ServeHTTP(rec, req)` runs the handler in memory and `rec` captures the response, so the test needs no open port.

### Errors you may not ignore

`json.NewEncoder(w).Encode(...)` returns an `error`. Go has no exceptions for ordinary failures: functions return errors as values and the caller decides. Writing `_ =` says "I am deliberately ignoring this", and the project rule is to add a comment explaining why (`internal/api/server.go:19`). More in step 1.

### The `-addr` flag

`flag` is the standard library's command-line parser: `flag.String("addr", ":8080", "usage")` returns a `*string` (a pointer, filled in by `flag.Parse()`), so you read it as `*addr`. See `cmd/kanban/main.go:15`. A bare `:8080` means "all interfaces, port 8080".
