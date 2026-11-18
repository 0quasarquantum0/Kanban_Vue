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

## Step 1 — Types, seed and validation

### Structs and JSON tags

A `struct` is Go's record type: a fixed list of named, typed fields. Where TypeScript has `interface Card { id: string; ... }` (a compile-time description of a plain object), a Go struct is a real type with a memory layout, and values are created with a literal: `Card{ID: "c", Title: "t"}`. Fields you leave out get their **zero value** (`""`, `0`, `false`, `nil`), so there is no `undefined`. The backtick strings after each field are **struct tags**: `json:"dueDate"` tells `encoding/json` which JSON key to use, because Go field names must be capitalised to be visible to the encoder (`DueDate`) while the frontend's keys are camelCase. See `internal/kanban/types.go:17`.

### Maps, slices and `nil`

`Record<ID, Card>` becomes `map[string]Card`; `string[]` becomes `[]string` (a *slice*: a growable view over an array). Reading a missing map key returns the zero value, and `card, ok := cards[id]` tells you whether it was there. Two traps matter here. A `nil` slice encodes to JSON `null`, not `[]`, so the seed always creates labels as `[]string{}` or a literal list. And a `nil` map is readable but panics on write, so `NewSeedState` makes its maps with `map[string]Card{}` (`internal/kanban/seed.go:64`). Validation uses the nil/empty difference on purpose: a field missing from the JSON stays `nil`, `[]` becomes an empty non-nil slice (`internal/kanban/validate.go:30`).

### Pointers, only for `null`

Go values are copied when passed around; a pointer (`*string`) is the address of a value. The only reason we need one here is that a plain `string` cannot be "absent": `DueDate *string` is `nil` for JSON `null` and points at a string otherwise (`internal/kanban/types.go:22`). Compare `string | null` in TypeScript. The `&seed.dueDate` in `internal/kanban/seed.go:80` takes the address of a variable; since Go 1.22 each loop iteration has its own copy of `seed`, so each card gets its own pointer.

### Errors as values

A Go function that can fail returns an extra `error` result, and the caller must look at it: `if err != nil { return err }`. There are no `try`/`catch` blocks for expected failures (`panic` exists but is for bugs, like `NewID`'s "the OS has no randomness" at `internal/kanban/id.go:15`). `Validate` returns `error` so it can say *what* is wrong, where `isKanbanState` could only answer `false` (`internal/kanban/validate.go:17`).

An error is any value with an `Error() string` method. `errors.New("...")` makes one; the exported ones in `internal/kanban/errors.go:11` are **sentinel errors** that callers compare with `errors.Is(err, kanban.ErrNotFound)`. `invalidError` (`internal/kanban/errors.go:20`) is a small struct that has both a message and an `Is` method, so one error can be shown to the user *and* matched by the HTTP layer to pick a status code.

### Custom JSON encoding

A type can take over its own JSON by implementing `MarshalJSON` / `UnmarshalJSON`. This is where `State` needs it: a JavaScript object keeps its keys in insertion order, and the frontend lists boards in that order, but a Go map does not, and `encoding/json` writes map keys sorted. So `State` carries a separate `BoardOrder` slice (tagged `json:"-"`, "not a JSON field") and writes the boards by hand in that order (`internal/kanban/types.go:73`); reading uses `json.Decoder.Token()` to walk the object key by key (`internal/kanban/types.go:142`). It is the same job as a custom `toJSON`/`JSON.parse` reviver in JS.

### Generics

`func checkReferences[V any](..., entities map[string]V, ...)` (`internal/kanban/validate.go:72`) has a type parameter in square brackets, like `<V>` in TypeScript. It lets one function check boards' columns and columns' cards without caring about the map's value type. Go generics are deliberately small: the constraint `any` means "anything", and `comparable` means "usable with `==`". `writeMap` in `internal/kanban/types.go:105` is another.

### The standard library's `maps` and `slices`

`slices.Sorted(maps.Keys(m))` returns a map's keys in sorted order (`internal/kanban/validate.go:29`). Go randomises map iteration order on purpose, so any code that needs a stable result (like a deterministic error message) must sort. `slices.Equal` is used in the tests instead of a hand-written loop.

### Table-driven tests and `t.Run`

When many cases share the same steps, Go tests put the cases in a slice of structs and loop over it, calling `t.Run(name, func(t *testing.T) {...})` so each case reports separately, like `it.each` in Vitest. See `internal/kanban/validate_test.go:44`, whose rows are the ported cases from `src/__tests__/validation.spec.ts`. Test names are sentences, so a failure reads `TestValidateRejectsBrokenReferences/rejects_a_card_placed_in_two_columns`.

Helpers call `t.Helper()` so a failure points at the calling test line, not the helper (`internal/kanban/validate_test.go:11`). The `for range 1000` in `internal/kanban/id_test.go:24` is Go 1.22's range-over-integer.

### `package kanban` tests vs `package kanban_test`

In step 0 the test was in `package api_test`, an external test that can only use exported names. Here the tests are in `package kanban` itself, so they can reach unexported helpers such as `invalidf`. Both styles are normal; use the external one when you want to test the public surface only.

### Constants and layouts for time

`const timestampLayout = "2006-01-02T15:04:05.000Z"` (`internal/kanban/seed.go:6`): Go formats dates by showing the reference time `Mon Jan 2 15:04:05 2006` in the shape you want, rather than with `YYYY-MM-DD` codes. This layout reproduces JavaScript's `toISOString()`, so the frontend can read the dates.

## Step 2 — In-memory store and the board operations

### Methods and receivers

A method is a function with a *receiver* written before its name: `func (m *Memory) AddBoard(...)`. It is how Go attaches behaviour to a type, in place of a `class` (Go has no classes, no inheritance and no `this`; the receiver is an ordinary named parameter). A **pointer receiver** (`*Memory`) gets the original value and can change it; a **value receiver** (`State.Clone`, `internal/kanban/store.go:90`) gets a copy. `Memory` holds a mutex, which must never be copied, so all its methods use pointer receivers (`internal/kanban/memory.go:41`).

### Interfaces are satisfied implicitly

`Store` (`internal/kanban/store.go:44`) is a list of method signatures. `Memory` never says "implements Store": having the methods is enough. This is TypeScript's structural typing, but checked for method sets, and the interface is usually declared *where it is used* rather than next to the implementation. The line `var _ Store = (*Memory)(nil)` (`internal/kanban/memory.go:22`) is a compile-time assertion: assign a nil `*Memory` to a `Store` variable and throw it away, so a missing method breaks the build right there instead of at a distant call site.

Interfaces are why `testStore` in `internal/kanban/store_suite_test.go:16` can take *any* `Store`. The same test list runs against `Memory` now and against SQLite in step 5.

### Why a mutex, and `defer`

The Vue app is single-threaded: two event handlers never run at the same instant. Go's `net/http` serves each request on its own goroutine, so two requests can touch the same map at once, and Go **crashes the program** on concurrent map writes. `sync.Mutex` fixes this: `m.mu.Lock()` makes other goroutines wait until `Unlock()`. `defer m.mu.Unlock()` schedules the unlock for when the function returns, by any path (early return, error), so you cannot forget it; compare `finally` in JavaScript, but attached to the function, not a block. See `internal/kanban/memory.go:43`.

The last test of the suite, "concurrent writes are all applied" (`internal/kanban/store_suite_test.go:474`), starts 25 goroutines that all call `AddCard`. Without the mutex it would crash.

### Goroutines and `sync.WaitGroup` (a first look)

`go func() { ... }()` runs a function concurrently on a goroutine, a very cheap thread managed by Go's runtime (`internal/kanban/store_suite_test.go:481`). The `WaitGroup` counts them: `Add(1)` before starting, `Done()` when each finishes, `Wait()` blocks until the count is zero. Step 7 builds on this; for now read it as "like `Promise.all` over 25 tasks".

### Closures capture variables

The anonymous function above uses `i` and `f` from its surroundings, just as a JavaScript closure does. Since Go 1.22 every iteration of a `for` loop has its *own* `i`, so this is safe. In older Go all closures shared one variable and you had to copy it first.

### Sentinel errors and `%w`

`fmt.Errorf("%s %q: %w", kind, id, ErrNotFound)` (`internal/kanban/memory.go:262`) builds a new error whose message adds context while *wrapping* the original. The wrapped chain is then searchable: `errors.Is(err, kanban.ErrNotFound)` is true. The HTTP layer in step 3 uses exactly this to turn an error into a status code without parsing text. It replaces `instanceof NotFoundError` in TypeScript.

### Generics with state: `Optional[T]`

`Optional[*string]` (`internal/kanban/store.go:17`) solves a JSON problem. For a PATCH, "`dueDate` is missing" (leave it) and "`dueDate` is null" (clear it) must differ. A `*string` cannot tell them apart, since both read as `nil`. Since `encoding/json` only calls `UnmarshalJSON` when the key is *present*, the method can set a flag. TypeScript gets this for free from `undefined` vs `null`; Go has one zero value, so you add the bit yourself. `internal/kanban/patch_test.go:9` shows the three cases.

### Zero values, `nil` slices and copying

Go slices and maps are *references*: assigning one copies a small header, not the data. So `Snapshot` must clone (`State.Clone`, `internal/kanban/store.go:90`), or callers could change the store's internals without holding the lock. `TestMemoryStore/Snapshot returns a copy...` proves it. The same reason explains the `slices.Clone(board.ColumnIDs)` in `DeleteBoard` (`internal/kanban/memory.go:106`): `removeColumn` edits that slice while we loop over it, the same trap the Pinia version avoids with `.slice()`.

### `min`, `max` and the `slices` package

`max(0, min(toIndex, len(...)))` (`internal/kanban/memory.go:254`) are built-in functions since Go 1.21, replacing `Math.max(0, Math.min(...))`. `slices.Insert`, `slices.Delete`, `slices.Index` replace `splice` and `indexOf`, but they return the new slice, so always assign the result.

### `context.Context` as the first argument

Every `Store` method takes `ctx context.Context` first. A context carries cancellation and deadlines: when an HTTP client disconnects, the request's context is cancelled, and a database call that received it can stop early. The in-memory store ignores it (`_ context.Context`), but the SQLite store will not, and the interface is shared. Compare `AbortSignal` passed to `fetch`.

### Test fixtures with closures

`run := func(name string, test func(...))` at `internal/kanban/store_suite_test.go:20` is a local function that makes a fresh store for each subtest, like `beforeEach`. Go has no built-in `beforeEach`; a helper function (or `t.Cleanup`) does the same job in plain code.
