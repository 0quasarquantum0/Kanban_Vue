# Quiz: the Go code in this project

Ten questions about the code in `server/`. Try to answer before opening the answer. Each one points at the file to look in.

---

**1. `internal/kanban/validate.go` defines `Validate` and `internal/kanban/errors.go` defines `invalidf`. Which of the two can code in `internal/api` call, and why?**

<details>
<summary>Answer</summary>

Only `Validate`. In Go, a name that starts with a capital letter is exported (visible to other packages); a lowercase name is private to its own package. `invalidf` and `checkTitle` are lowercase, so only files in package `kanban` can use them. There is no `export` keyword, and no `private` either.

</details>

---

**2. Why does `Memory` need a `sync.Mutex`, when the Pinia store in the Vue app did not need one?**

(`internal/kanban/memory.go`)

<details>
<summary>Answer</summary>

`net/http` handles every request on its own goroutine, so two requests can read and write the same maps at the same moment, and Go crashes the program on concurrent map writes. Browser JavaScript runs on one thread, so two event handlers never run at the same instant. Each `Memory` method locks `m.mu` and `defer`s the unlock, so only one runs at a time. (The step 2 test with 25 goroutines would crash without it.)

</details>

---

**3. `UpdateCard` takes a `CardPatch` whose `DueDate` field has the type `Optional[*string]`, not just `*string`. What problem does `Optional` solve?**

(`internal/kanban/store.go`)

<details>
<summary>Answer</summary>

In a `PATCH` body, a missing `dueDate` means "leave it alone", while `"dueDate": null` means "clear it". A plain `*string` is `nil` in both cases, so they cannot be told apart. `Optional` has a `Set` flag that its `UnmarshalJSON` method switches on; `encoding/json` only calls that method when the key is present (including when its value is `null`). TypeScript gets the distinction from `undefined` vs `null`; Go has a single zero value.

</details>

---

**4. What does `defer func() { _ = tx.Rollback() }()` do in `SQLite.transact`, and why is it safe to run even after a successful `Commit`?**

(`internal/kanban/sqlite.go`)

<details>
<summary>Answer</summary>

`defer` schedules the call for when the surrounding function returns, whichever way it returns. If `fn` fails and the function returns early, `Rollback` undoes everything done in the transaction. If `Commit` already succeeded, `Rollback` does nothing and returns `sql.ErrTxDone`, which is ignored (with a comment saying why). So no `return` path has to remember to roll back.

</details>

---

**5. `TestMemoryStore` and `TestSQLiteStore` both call `testStore`. What does that buy you, and what in the code makes it possible?**

(`internal/kanban/store_suite_test.go`, `memory_test.go`, `sqlite_test.go`)

<details>
<summary>Answer</summary>

Both implementations are held to exactly the same behaviour tests, so they cannot drift apart. It works because `testStore` takes a function that returns a `Store`, an interface, and Go interfaces are satisfied implicitly: `Memory` and `SQLite` never say "implements Store", they simply have the methods. (`var _ Store = (*Memory)(nil)` is a compile-time check of that.)

</details>

---

**6. Why does `rename` in `internal/api/resources.go` return an `http.HandlerFunc` instead of being a handler itself?**

<details>
<summary>Answer</summary>

Renaming a board and renaming a column are the same code with two differences: the name of the path parameter (`"boardId"` or `"columnId"`) and which `Store` method does the work. `rename(param, change)` takes those as arguments and returns a handler that uses them (a closure), so one function serves both routes: `s.rename("boardId", s.store.RenameBoard)`. `s.store.RenameBoard` is a method value: a method already bound to its receiver, passed like any function.

</details>

---

**7. In `Hub.Run`, what happens when a client's buffered channel is full on a publish, and why is it done that way?**

(`internal/api/hub.go`)

<details>
<summary>Answer</summary>

The `select` with a `default:` branch tries to send without waiting; if the buffer is full it falls into `default`, which removes the subscriber and closes its channel. If the hub waited for a slow client, every other client (and every write handler calling `Publish`) would be blocked behind it. The dropped client's `EventSource` reconnects by itself, and the first event on the new connection brings it up to date, so nothing is lost.

</details>

---

**8. The hub has no mutex, yet many goroutines subscribe and publish at once. Why is that safe?**

<details>
<summary>Answer</summary>

Only one goroutine, `Run`, ever touches the set of subscribers. Everyone else sends a request to it on a channel (`subscribe`, `unsubscribe`, `publish`), and `Run`'s `select` handles one request at a time. This is "share memory by communicating": ownership by one goroutine replaces locking. (A mutex, as in `Memory`, is the other valid approach.)

</details>

---

**9. The `/api/events` handler calls `Subscribe` before it reads the current version. What bug would the other order allow?**

(`internal/api/events.go`)

<details>
<summary>Answer</summary>

A write could land between reading the version and subscribing. The client would be sent a version that is already out of date and never receive an event for the write, so its screen would stay stale until the next change. Subscribing first means every write after that moment is queued for this client, and the version read afterwards is at least as new as anything it will be sent.

</details>

---

**10. How can you build a Linux ARM executable on Windows, and why does it work for this project without installing anything?**

(`LEARNING.md`, step 8)

<details>
<summary>Answer</summary>

`GOOS=linux GOARCH=arm64 go build ./cmd/kanban` (with `CGO_ENABLED=0`). Go's compiler can target any supported system from any system. It works here because every dependency is pure Go: `modernc.org/sqlite` is SQLite translated into Go, so no C compiler or C libraries are involved. The frontend and the SQL migrations are inside the binary too (`//go:embed`), so the result is a single file with nothing to install on the target.

</details>
