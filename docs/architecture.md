# Architecture

_Last updated: Go step 8 (one binary). The Vue phases (0–9) are complete; the Go backend in `server/` is being built step by step._

This file shows the project as it currently stands. It is updated at the end of every phase and only diagrams what exists. The frontend sections come first; the **Go backend** section is near the end.

## App bootstrap

```mermaid
flowchart TD
  html["index.html<br/>div#app"] --> main["src/main.ts<br/>createApp(App)"]
  main -- "app.use()" --> pinia["Pinia<br/>stores/kanban.ts"]
  main -- "app.use()" --> router["Vue Router<br/>src/router/index.ts"]
  main -- "mount('#app')" --> app["App.vue<br/>loading / error screen, else RouterView"]
  router -. "resolves current route" .-> app
```

## Component tree

Solid arrows are parent → child, labelled with what is passed down (props, `v-model`) and what comes back up (events). Dotted arrows are store or router access. The views, `BoardView`, `BoardToolbar`, `CardEditor` and `CardMoveMenu` talk to the store; the rest are driven by props, models and events.

```mermaid
flowchart TD
  app["App.vue<br/>RouterView (once the store is ready)<br/>SyncNotice<br/>provideTheme()"]
  store[("stores/kanban.ts<br/>useKanbanStore")]

  app -- "route /" --> list["views/BoardsListView.vue<br/>create, rename, delete boards"]
  app -- "route /boards/:boardId<br/>prop boardId" --> detail["views/BoardDetailView.vue<br/>nested RouterView (keyed by path)"]

  list -. "boardList, addBoard, deleteBoard,<br/>v-model writes board.title" .-> store
  list --> toggleList["ThemeToggle.vue"]
  list --> toolbar["components/BoardToolbar.vue<br/>Export JSON, Import JSON, Reset"]
  list -- "v-model (board title)" --> inlineList["InlineEdit.vue"]
  list -- "emits add(title)" --> addBoard["AddItemForm.vue"]
  toolbar -. "reads state, importState, resetToDemo" .-> store

  detail -- "prop board" --> board["components/BoardView.vue"]
  detail -- "child route cards/:cardId<br/>props boardId, cardId" --> editor["components/CardEditor.vue<br/>draft form: title, description, labels, due date"]
  detail -. "reads state.boards[boardId]" .-> store

  board -. "getters, actions,<br/>v-model writes column.title" .-> store
  board -- ":title + @update:title<br/>(→ renameBoard)" --> header["components/BoardHeader.vue<br/>back link, title via InlineEdit"]
  header --> toggle["components/ThemeToggle.vue<br/>useTheme() via inject"]
  app -. "provide(themeKey)" .-> toggle
  board -- "prop labels" --> filterBar["components/FilterBar.vue<br/>search input, label toggles, clear"]
  filters[["composables/useBoardFilters.ts<br/>derived from route.query"]]
  board -. "isFiltering, matches" .-> filters
  filterBar -. "reads filters;<br/>setSearchText (debounced), toggleLabel, clearFilters" .-> filters
  board -- "v-model:title,<br/>props cardCount, totalCount, emptyText" --> column["components/KanbanColumn.vue<br/>slots: #actions, default, #footer<br/>TransitionGroup around the default slot"]
  board -- "prop card, emits delete<br/>(into column's default slot)" --> card["components/KanbanCard.vue<br/>title is a RouterLink to the card route<br/>default slot for extra controls"]
  board -- "props card, column, boardId, canReorder<br/>(into the card's default slot)" --> move["components/CardMoveMenu.vue<br/>details/summary: columns, Up, Down"]
  move -. "columnsForBoard, moveCard" .-> store
  board -- ":open + @update:open (card forms only)<br/>emits add(title)<br/>(column #footer, and 'Add column')" --> add["AddItemForm.vue<br/>v-model:open, v-focus directive"]
  column -- "v-model (title)" --> inlineCol["InlineEdit.vue<br/>defineModel"]

  dnd[["composables/useDragAndDrop.ts"]]
  board -. "calls once in setup" .-> dnd
  dnd -. "onDrop → moveCard" .-> store

  editor -- "v-model:open, prop title<br/>emits closed" --> modal["components/BaseModal.vue<br/>Teleport to body + Transition<br/>closes on Esc, backdrop, ×<br/>useModalFocus()"]
  editor -. "updateCard, deleteCard" .-> store
  editor -. "on closed: router.push(board)" .-> detail
```

## Filtering

The URL query is the only place filter state lives: `?q=text` and `?label=bug&label=ui`. Filtered lists are never copied into state.

```mermaid
flowchart LR
  url[("route.query<br/>q, label")]
  filters["useBoardFilters()<br/>computed: searchText, activeLabels,<br/>isFiltering, matches"]
  bar["FilterBar.vue<br/>draft ref for the input"]
  board["BoardView.vue<br/>computed columns:<br/>{ column, cards.filter(matches), total }"]
  store[("kanban store<br/>full card lists")]

  url -- "computed" --> filters
  filters --> bar
  filters --> board
  store --> board
  bar -- "watchDebounced(draft, 300ms)<br/>→ router.replace" --> url
  bar -- "label click, Clear<br/>→ router.replace" --> url
  url -. "watch(searchText): back/forward or Clear<br/>resets the input" .-> bar
```

- Text matches title or description (case-insensitive). With several labels active, a card needs any one of them.
- While `isFiltering`, cards render with `draggable="false"` and `FilterBar` explains why.
- A column shows "N of M" and, if nothing matches, "No matching cards".
- `KanbanCard`'s link and `CardEditor`'s close navigation both carry `route.query`, so filters survive opening a card.
- `store.labelsForBoard(boardId)` supplies the label list.

## Theme

`composables/useTheme.ts` holds a typed `InjectionKey<ThemeContext>`. `App.vue` calls `provideTheme()` once; any descendant calls `useTheme()`.

```mermaid
flowchart LR
  os["OS preference<br/>usePreferredDark()"]
  ls[("localStorage<br/>kanban-theme: auto | light | dark")]
  ctx["provideTheme() in App.vue<br/>theme: computed, toggleTheme()"]
  html["html data-theme attribute"]
  css["main.css<br/>:root[data-theme='dark'] overrides the colour tokens"]
  toggle["ThemeToggle.vue<br/>inject via useTheme()"]

  os --> ctx
  ls --> ctx
  ctx -- "watchEffect" --> html --> css
  ctx -- "provide(themeKey)" --> toggle
  toggle -- "toggleTheme()" --> ls
```

- Components never read the theme to pick colours; they use the CSS variables, which change with the attribute.
- `--color-accent-fill`, `--color-danger-fill` and `--color-on-fill` are for solid buttons, so text on them keeps its contrast in both themes.

## Keyboard and accessibility

`composables/useKeyboardShortcuts.ts` wraps VueUse's `onKeyStroke`. Each component registers the shortcut it owns, and the listener is removed when that component unmounts.

| Key | Registered in | Effect |
|---|---|---|
| `n` | `BoardView` | Opens the add-card form in the first column (sets `addingToColumnId`) |
| `/` | `FilterBar` | Focuses the search input |
| `Esc` | `BaseModal` | Closes the modal |

- Except for `Esc`, shortcuts are ignored while typing in a field, with Ctrl/Cmd/Alt held, or while a modal is open.
- `composables/useModalFocus.ts` moves focus into the dialog on mount (to the `data-autofocus` field), wraps Tab at both ends, and returns focus to the opener on unmount.
- `CardMoveMenu` is the keyboard alternative to drag and drop. After a move it focuses the card's menu in its new place. Up and Down are disabled while filters are active.
- `InlineEdit` and `AddItemForm` return focus to their button after Enter or Esc.
- `main.css` sets one `:focus-visible` outline for every focusable element.

## Drag and drop

`composables/useDragAndDrop.ts` wraps the native HTML5 drag events. `BoardView` calls it once, passing `store.moveCard` as `onDrop`, and binds the handlers in its template. The listeners and attributes are not declared as props or emits, so they fall through to the root elements of `KanbanCard` and `KanbanColumn`.

```mermaid
sequenceDiagram
  participant Card as KanbanCard root (draggable, data-card-id)
  participant Col as KanbanColumn root (drop zone)
  participant DnD as useDragAndDrop
  participant Store as kanban store

  Card->>DnD: dragstart → onDragStart(cardId)
  Note over DnD: draggingCardId = cardId
  Col->>DnD: dragover.prevent → onDragOver(columnId)
  Note over DnD: overColumnId, overIndex from pointer Y vs card midpoints
  Col->>DnD: dragleave → onDragLeave (ignored when moving onto a child)
  Col->>DnD: drop.prevent → onDrop(columnId)
  DnD->>Store: moveCard(cardId, columnId, index)
  Card->>DnD: dragend → reset state
```

- Reactive state: `draggingCardId`, `overColumnId`, `overIndex` (counted among the column's cards without the dragged one, which is the final index `moveCard` expects).
- `dropPosition(columnId, cardIds, cardId)` returns `'before' | 'after' | null`; `BoardView` binds it to `data-drop` on each card.
- Visual state is global CSS in `src/assets/main.css`: `.is-dragging`, `.is-drop-target`, and the `[data-drop]` indicator line. The shared `.button` and `.icon-button` classes live there too.

## Transitions

`KanbanColumn` wraps its default slot in `<TransitionGroup tag="div" name="card-list">`, so every keyed card passed in by `BoardView` animates without either the card or the board knowing about it.

```mermaid
stateDiagram-v2
  [*] --> Entering: card added, or moved in from another column
  Entering --> Idle: card-list-enter-from → enter-active (fade + scale in, 0.2s)
  Idle --> Moving: a sibling was added, removed or reordered
  Moving --> Idle: card-list-move (FLIP transform, 0.25s)
  Idle --> Leaving: card deleted, or moved to another column
  Leaving --> [*]: card-list-leave-active → leave-to (fade + scale out, 0.2s, position absolute)
```

- The "No cards yet" message is a keyed child of the same group, so it fades in and out too.
- A move between columns is a leave in one group and an enter in the other, not a single slide.
- The styles live in `KanbanColumn.vue` and use `:deep()` because slotted cards don't carry the column's scope id.
- `prefers-reduced-motion: reduce` sets all three transitions to `none`.
- Columns grow to their content and clip it (`overflow: hidden`); `.board__columns` in `BoardView` scrolls in both directions.

## Store

`stores/kanban.ts` is a Pinia setup store. State is one deep ref of `KanbanState`, empty until the Go server's copy has been loaded. `composables/useServerSync.ts` (called from inside the store, its refs and functions spread into the store's return value) loads it and sends every action to the server (see Server sync below).

```mermaid
classDiagram
  class useKanbanStore {
    +state: Ref~KanbanState~
    +status: idle | loading | ready | error
    +loadError: string | null
    +notice: string | null
    +version: number
    +ensureLoaded() Promise
    +load() Promise
    +settled() Promise
    +refreshIfBehind(version) Promise
    +startListening()
    +stopListening()
    +dismissNotice()
    +boardList: Board[]
    +columnsForBoard(boardId) Column[]
    +cardsForColumn(columnId) Card[]
    +labelsForBoard(boardId) string[]
    +addBoard(title) Board
    +renameBoard(boardId, title)
    +deleteBoard(boardId)
    +addCard(columnId, title) Card
    +updateCard(cardId, patch)
    +deleteCard(cardId)
    +addColumn(boardId, title) Column
    +renameColumn(columnId, title)
    +deleteColumn(columnId)
    +moveCard(cardId, toColumnId, toIndex)
    +importState(data) boolean
    +resetToDemo()
  }
```

- `boardList`, `columnsForBoard`, `cardsForColumn` and `labelsForBoard` are `computed` getters; the last three return lookup functions.
- `deleteColumn` also deletes the column's cards; `deleteBoard` deletes its columns the same way.
- `moveCard` places the card at `toIndex` in the target column (clamped). It is called by the drag-and-drop composable.
- `importState` replaces the whole state only if `isKanbanState(data)` passes, and returns whether it did.
- Unit tests: `src/stores/__tests__/kanban.spec.ts` (`addCard`, `moveCard`, `deleteColumn`, `deleteBoard`, `importState`, `resetToDemo`) and `src/composables/__tests__/useServerSync.spec.ts` (loading, saving, conflicts). Both run against `src/__tests__/fakeServer.ts`, a stand-in for the Go server's `fetch` endpoints; `src/__tests__/renames.spec.ts` checks that renaming in the UI reaches the server.

## Server sync

The server owns the data (`docs/api.md`). The Vite dev server forwards `/api` to it; `KANBAN_API` (environment or a git-ignored `.env.local`) picks the target, default `http://localhost:8080`.

Reading: the store loads the whole state once (`GET /api/state`) and reloads it when it learns it is behind. Writing: every store action updates `state` first (optimistic) and queues one request.

```mermaid
sequenceDiagram
  participant UI as component
  participant Store as kanban store action
  participant Q as useServerSync queue
  participant Go as Go server

  UI->>Store: addCard(columnId, title)
  Store->>Store: state changes at once (client-made UUID)
  Store->>Q: enqueue(() => api.addCard(...), expectedId)
  Note over Q: requests run one at a time, in order
  Q->>Go: POST /api/columns/{id}/cards { id, title }
  alt success
    Go-->>Q: 201 { version, card }
    Note over Q: version = reply.version<br/>gap or different id → reload
  else any failure (404, 409, 500, offline)
    Go-->>Q: error
    Note over Q: generation++ (queued requests are skipped)<br/>GET /api/state replaces the optimistic state<br/>notice shown to the user
  end
```

```mermaid
flowchart LR
  state["store.state"]
  seed["seed.ts<br/>createSeedState()"]
  valid{{"validation.ts<br/>isKanbanState()"}}
  toolbar["BoardToolbar.vue"]
  file[/"kanban-board.json"/]
  api["api.ts<br/>getState, per-action calls,<br/>putState, ApiError"]

  api -- "load / reload: validated response" --> state
  state -- "each action: one queued request" --> api
  seed -- "Reset to demo data → PUT /api/state" --> state
  state -- "Export JSON" --> toolbar --> file
  file -- "Import JSON" --> toolbar -- "importState(data)" --> valid
  valid -- "valid: replace state, PUT /api/state" --> state
  valid -. "invalid: error shown, nothing changes" .-> toolbar
```

- `composables/useServerSync.ts` holds the queue (`enqueue`, `settled`), the version, `status`, `notice`, and `load`/`ensureLoaded`/`refreshIfBehind`. It is called inside the store and its members are spread into the store's return value.
- `load` waits for the queue to empty and discards a response if a change was queued while it was in flight, so a reload can never wipe out a change the server has not seen.
- Whole-state writes (`importState`, `resetToDemo`) send the version they are based on; a stale one gets 409, which shows the conflict notice and reloads.
- `App.vue` shows "Loading…" until the first load finishes and an error screen (with a reload button) if it failed; `SyncNotice.vue` shows failures.
- Components never write into store entities. `BoardView` and `BoardsListView` rename through `renameColumn` / `renameBoard`; a direct `v-model` on store state would change the screen but never reach the server.
- `src/validation.ts` checks the shape of every entity, that each is stored under its own id, that every referenced column and card exists and is placed only once, and that no column or card is left unplaced (the server's database cannot hold orphans). Tests: `src/__tests__/validation.spec.ts`.
- The theme preference is the only thing still in `localStorage`. `vitest.config.ts` passes `--no-experimental-webstorage` so Node's own `localStorage` global doesn't hide jsdom's in tests.

## Data model

Defined in `src/types.ts`. State is normalised: entities live in records keyed by ID, and order lives in the ID arrays.

```mermaid
erDiagram
  KanbanState ||--o{ Board : "boards (Record by id)"
  KanbanState ||--o{ Column : "columns (Record by id)"
  KanbanState ||--o{ Card : "cards (Record by id)"
  Board ||--o{ Column : "columnIds (ordered)"
  Column ||--o{ Card : "cardIds (ordered)"

  Board {
    ID id
    string title
    ID[] columnIds
  }
  Column {
    ID id
    string title
    ID[] cardIds
  }
  Card {
    ID id
    string title
    string description
    string[] labels
    string dueDate "ISO date or null"
    string createdAt
  }
```

`CardPatch` is the editable subset of a card (`title`, `description`, `labels`, `dueDate`), used by `updateCard`, which `CardEditor` calls on Save.

`src/seed.ts` exports `createSeedState()`, which builds one board with three columns (To do / Doing / Done) and six cards, with fresh IDs on each call.

## Routes

Defined in `src/router/index.ts` (history mode, no file-based routing).

```mermaid
flowchart TD
  any["any other path"] -- "redirect" --> boards
  boards["/ (name: boards)<br/>BoardsListView"]
  board["/boards/:boardId (name: board)<br/>BoardDetailView"]
  card["/boards/:boardId/cards/:cardId (name: card)<br/>CardEditor, rendered in BoardDetailView's RouterView"]

  boards -- "Open board →" --> board
  board -- "← All boards" --> boards
  board -- "click a card title" --> card
  card -- "Save, Cancel, Delete, Esc, backdrop<br/>(after the modal's leave animation)" --> board

  board -. "beforeEnter requireBoard:<br/>unknown board → /" .-> boards
  card -. "beforeEnter requireCard:<br/>card not on this board → board" .-> board
```

- Params reach components as props (`props` on the route), so `BoardDetailView` and `CardEditor` don't read `route.params`.
- The guards are `async`: they `await store.ensureLoaded()` first, because the data now arrives from the server. If that load failed they let the navigation through, and `App.vue` shows the error instead of the route. Tests: `src/router/__tests__/guards.spec.ts`.
- `beforeEnter` only runs when the route is entered, so `BoardDetailView` also renders a "no longer exists" message if its board disappears later.
- `KanbanCard` links to `{ name: 'card', params: { cardId } }`; the router reuses `boardId` from the current location.

## Tests

```mermaid
flowchart LR
  subgraph unit["Unit (no components)"]
    storeSpec["stores/__tests__/kanban.spec.ts"] --> store["stores/kanban.ts"]
    valSpec["__tests__/validation.spec.ts"] --> val["validation.ts"]
    syncSpec["composables/__tests__/useServerSync.spec.ts"] --> store
    guardSpec["router/__tests__/guards.spec.ts"] --> store
    appSpec["__tests__/App.spec.ts"] --> store
    fake["__tests__/fakeServer.ts<br/>stubs fetch like the Go server"]
    storeSpec -. uses .-> fake
    syncSpec -. uses .-> fake
  end
  subgraph comp["Component (@vue/test-utils)"]
    cardSpec["KanbanCard.spec.ts"] --> card["KanbanCard.vue"]
    inlineSpec["InlineEdit.spec.ts"] --> inline["InlineEdit.vue"]
    filterSpec["FilterBar.spec.ts"] --> filter["FilterBar.vue"]
  end
  router["components/__tests__/testRouter.ts<br/>real router, memory history, empty route components"]
  cardSpec -. uses .-> router
  filterSpec -. uses .-> router
```

- Component tests mount one component with `global.plugins: [router]`, so `RouterLink`, `useRoute` and `useRouter` behave as in the app.
- `FilterBar.spec.ts` uses Vitest fake timers to skip the 300ms debounce.
- Run once with `npx vitest run`; `npm run test:unit` watches.

## Go backend

Lives in `server/` (its own Go module). See `server/CLAUDE.md` for the plan and `server/LEARNING.md` for the Go concepts, step by step.

### Package layout

```mermaid
flowchart TD
  main["cmd/kanban/main.go<br/>flags, wiring, graceful shutdown"] --> web["web/embed.go<br/>//go:embed all:dist"]
  main --> api["internal/api<br/>routes, handlers, middleware"]
  api --> kanban["internal/kanban<br/>model, validation, Store"]
  kanban --- types["types.go, store.go (Snapshot, CardPatch)<br/>validate.go, seed.go, id.go, errors.go"]
  kanban --- mem["memory.go<br/>Memory: Store behind a sync.Mutex<br/>(used by tests)"]
  api --- static["internal/api/static.go<br/>serves the embedded Vue app + index.html fallback"]
  api --- hub["internal/api/hub.go, events.go<br/>Hub goroutine + GET /api/events"]
  kanban --- sql["sqlite.go, migrate.go<br/>SQLite: Store on database/sql<br/>migrations/*.sql (embedded)"]
```

`internal/` means nothing outside `server/` can import these packages.

### Store

`Store` is the interface the HTTP layer will use. Every write returns the new `version`; unknown IDs return `ErrNotFound`, bad input `ErrInvalid`, a stale `ReplaceState` `ErrConflict`.

```mermaid
classDiagram
  class Store {
    <<interface>>
    +Snapshot(ctx) Snapshot
    +ReplaceState(ctx, baseVersion, State) int64
    +AddBoard(ctx, id, title) Board
    +RenameBoard(ctx, boardId, title)
    +DeleteBoard(ctx, boardId)
    +AddColumn(ctx, boardId, id, title) Column
    +RenameColumn(ctx, columnId, title)
    +DeleteColumn(ctx, columnId)
    +AddCard(ctx, columnId, id, title) Card
    +UpdateCard(ctx, cardId, CardPatch) Card
    +DeleteCard(ctx, cardId)
    +MoveCard(ctx, cardId, toColumnId, toIndex)
  }
  class Memory {
    -mu sync.Mutex
    -state State
    -version int64
  }
  class SQLite {
    -db *sql.DB
  }
  Store <|.. Memory
  Store <|.. SQLite
```

- The Go types (`Board`, `Column`, `Card`, `State`) mirror `src/types.ts`; `State` also carries `BoardOrder` so the order of boards survives JSON (a Go map has no order).
- `Validate` ports `src/validation.ts` and adds: every column must be listed by a board and every card by a column.
- `testStore` (`internal/kanban/store_suite_test.go`) is the one behaviour test suite every `Store` implementation runs: `Memory` and `SQLite` both pass it.

### Database schema

SQLite, created by `internal/kanban/migrations/001_init.sql`, which `migrate` applies at startup (applied files are recorded in `schema_migrations`). Order is an integer `position` within the parent; foreign keys cascade, so deleting a board removes its columns and their cards.

```mermaid
erDiagram
  meta {
    int id "always 1"
    int version "raises on every write"
  }
  boards ||--o{ columns : "board_id, ON DELETE CASCADE"
  columns ||--o{ cards : "column_id, ON DELETE CASCADE"
  boards {
    text id PK
    text title
    int position
  }
  columns {
    text id PK
    text board_id FK
    text title
    int position
  }
  cards {
    text id PK
    text column_id FK
    text title
    text description
    text labels "JSON array"
    text due_date "nullable"
    text created_at
    int position
  }
```

- Every write runs in one transaction (`SQLite.transact`) that also bumps `meta.version`. Moving a card is one transaction of three updates.
- The connection pool is limited to one connection; `foreign_keys`, `busy_timeout` and WAL are set in the DSN.

### Request flow

The contract (methods, paths, bodies, status codes) is in [`api.md`](api.md).

```mermaid
flowchart LR
  req["HTTP request"] --> log["logRequests<br/>one slog line per request"]
  log --> rec["recoverPanics<br/>panic → JSON 500"]
  rec --> route["jsonRoutingErrors<br/>mux 404/405 → JSON"]
  route --> mux{"ServeMux<br/>'GET /api/state' …"}
  mux --> h["handler (state.go)<br/>decodeBody → Store → writeJSON"]
  h --> store[("kanban.Store")]
  h -. "error" .-> fail["fail(): ErrInvalid→400,<br/>ErrNotFound→404,<br/>ErrConflict→409, else 500"]
```

- `GET /api/state` returns `{ version, state }`; `PUT /api/state` takes the same shape, validates it, checks the version (stale → 409) and saves it.
- `resources.go` registers the ten per-action routes (`POST /api/boards`, `PATCH /api/columns/{columnId}`, `PUT /api/cards/{cardId}/position`, …). `rename` and `remove` are functions that return handlers, so boards, columns and cards share the code. Each write returns the new version.
- `decodeBody` caps the body size, rejects unknown fields and trailing data.
- `main.go` opens `kanban.OpenSQLite(ctx, *dbPath)` (flag `-db`, default `kanban.db`), seeds it on first run with `EnsureSeeded`, and passes it to `api.New(store, log).Handler()`.

### Live updates

`Hub` (`internal/api/hub.go`) is one goroutine that owns the set of subscribers; other goroutines talk to it only through channels. Every successful write handler calls `s.wrote(version)`, which publishes the version to the hub.

```mermaid
flowchart LR
  w["write handlers<br/>(state.go, resources.go)"] -- "Publish(version)" --> pub(["publish chan"])
  sub(["subscribe / unsubscribe chans"]) --> hub
  pub --> hub{{"Hub.Run goroutine<br/>select loop, owns the subscriber set"}}
  hub -- "buffered chan per client<br/>(full buffer → client dropped)" --> ev["events handler<br/>one goroutine per open GET /api/events"]
  ev -- "event: version<br/>data: {version}" --> es["browser EventSource<br/>(useServerSync.startListening)"]
  es -- "version > local version<br/>→ GET /api/state" --> store["kanban store"]
```

- A new connection first receives the current version, so a reconnecting tab catches up.
- A request's context being cancelled (client closed the tab) ends the handler and unsubscribes it.
- `App.vue` starts listening on mount and stops on unmount. A tab's own writes echo back as events but are ignored, because the reply to the write already moved its version forward.

### One binary

`npm run build:embed` builds the Vue app into `server/web/dist`; `//go:embed` puts those files inside the Go executable, which then serves both the API and the app.

```mermaid
flowchart LR
  vue["src/ (Vue app)"] -- "npm run build:embed<br/>vite build --outDir server/web/dist" --> dist[("server/web/dist<br/>index.html, assets/*<br/>(only .gitkeep is committed)")]
  dist -- "//go:embed all:dist" --> bin{{"kanban.exe<br/>one file"}}
  req["request"] --> bin
  bin -- "/api/..." --> apih["API handlers + middleware"]
  bin -- "a file that exists" --> file["file (assets: immutable cache)"]
  bin -- "no extension: /boards/x/cards/y" --> idx["index.html (Vue router takes over)"]
  bin -- "missing file with an extension" --> nf["404"]
```

- `main.go` opens the listener, then `run(ctx, log, ln, dbPath, frontend)` serves until the context (cancelled by Ctrl+C or SIGTERM) ends, then calls `http.Server.Shutdown`. `RegisterOnShutdown(app.Close)` ends open event streams so shutdown is not held up.
- Flags: `-addr` (default `:8080`) and `-db` (default `kanban.db`).
- In development nothing changes: Vite serves the app on 5173 and proxies `/api` to the Go server.

## Not built yet

Nothing from the Vue plan. Optional stretch goals in `CLAUDE.md` (only if asked): a Nuxt port, touch drag and drop, undo/redo.

Go backend: all planned steps are built (step 9 finishes the docs).
