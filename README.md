# Vue Kanban

A Kanban board built with Vue 3 as a learning project, by a developer coming from React and Next.js. It was built in phases, each one introducing a group of Vue features. The app works; the comments and this README are aimed at understanding *how Vue does things*.

The next project is a **Go backend** for this app, planned in [Part 2](#part-2-go-backend-planned) below.

Features: multiple boards, columns and cards; drag and drop (with a keyboard alternative); animated moves; a card editor in a modal with its own URL; search and label filters in the URL; `localStorage` persistence with JSON import/export; light and dark themes; keyboard shortcuts.

## Running it

Needs Node `^22.18.0` or `>=24.12.0`.

```bash
npm install
npm run dev          # http://localhost:5173
```

| Command | What it does |
|---|---|
| `npm run dev` | Dev server with hot reload |
| `npm run build` | Type-check and production build to `dist/` |
| `npm run preview` | Serve the production build at http://localhost:4173 |
| `npm run test:unit` | Vitest in watch mode (`npx vitest run` for a single run) |
| `npm run type-check` | `vue-tsc` |
| `npm run lint` | oxlint, then ESLint (both with `--fix`) |

### Keyboard shortcuts

| Key | Where | Effect |
|---|---|---|
| `n` | Board | New card in the first column |
| `/` | Board | Focus the search box |
| `Esc` | Card editor | Close it |

Each card also has a **Move to…** menu, so everything drag and drop does can be done from the keyboard.

## Where things are

```
src/
  main.ts                 creates the app, installs Pinia and the router
  App.vue                 <RouterView /> and the theme provider
  router/index.ts         routes and guards
  stores/kanban.ts        Pinia setup store, persisted with useStorage
  composables/            useDragAndDrop, useBoardFilters, useKeyboardShortcuts, useModalFocus, useTheme
  components/             BoardView, KanbanColumn, KanbanCard, CardEditor, BaseModal, InlineEdit, FilterBar, ...
  views/                  BoardsListView, BoardDetailView
  types.ts, seed.ts, validation.ts
docs/architecture.md      Mermaid diagrams of the current structure (start here)
```

State is normalised: boards, columns and cards are stored by ID, and order lives in ID arrays, so moving a card is two `splice` calls.

## Vue concepts by phase

Each phase is one commit (`git log --oneline`), so you can check out any phase to see the code at that point.

| Phase | Built | Vue concepts | Closest React equivalent |
|---|---|---|---|
| 0. Scaffold | Project from `create-vue` | Single-file components; `<script setup>` runs **once** per instance; `createApp().use().mount()` | `createRoot().render()`, providers as wrappers |
| 1. Static board | Board, column and card components | `{{ }}`, `:prop`, `@event`; `v-for`/`v-if` directives; default and **named slots** | JSX, `.map()`, `&&`, `children` and render props |
| 2. Store and editing | Add, rename, delete | `ref` vs `reactive` and `.value`; Pinia setup stores and `storeToRefs`; `v-model`, `defineModel`, typed `defineEmits` | Zustand without selectors; controlled inputs; callback props |
| 3. Drag and drop | `useDragAndDrop` composable | **Composables** run once, so no rules of hooks or stale closures; event modifiers (`.prevent`); fallthrough attributes | Custom hooks, `useCallback`, `...rest` spreading |
| 4. Transitions | Card enter, leave and move animations | `<TransitionGroup>` and its CSS classes; FLIP move animations; `:deep()` for slotted content | React Transition Group, Framer Motion `layout` |
| 5. Persistence | `localStorage`, import/export, reset | A deep reactive ref synced by a deep watcher (VueUse `useStorage`) with no subscription code | `useState` initialiser plus a `useEffect` writer |
| 6. Routing | Boards list, board page, card modal | Vue Router route table, `beforeEnter` guards, nested `<RouterView>`, route params as props; `useRoute`/`useRouter`; `<Teleport>` | Next.js App Router (Nuxt adds file-based routing), `createPortal` |
| 7. Filtering | Search and labels in the URL | `computed` for derived data; `watch` vs `watchEffect`; `watchDebounced` | `useMemo`; `useEffect` with dependencies and a timeout |
| 8. Polish | Themes, shortcuts, accessibility | `provide`/`inject` with a typed `InjectionKey`; `onMounted`/`onUnmounted`; custom directives (`v-focus`); `defineModel` with a default | Context, `useEffect` cleanup, ref callbacks, (un)controlled components |
| 9. Tests and build | Component tests, this README | `@vue/test-utils` `mount`, `setValue`, `trigger`, `emitted()`; mounting with a real in-memory router | React Testing Library |

## Tests

- `src/stores/__tests__/kanban.spec.ts`: store actions, persistence, import and reset.
- `src/__tests__/validation.spec.ts`: the import validator.
- `src/components/__tests__/`: `KanbanCard` renders its title and labels, `InlineEdit` emits `update:modelValue`, and `FilterBar` updates the route query.

`vitest.config.ts` passes `--no-experimental-webstorage` because newer Node versions define a `localStorage` global that hides jsdom's.

## Known limitations

- Drag and drop is mouse-only (no touch) and was not tested in Firefox. Use **Move to…** on touch devices.
- Deleting a card or column from the keyboard leaves focus at the top of the page.
- The browser's Back button closes the card editor without its leave animation.
- Data lives in one browser's `localStorage`; use Export JSON to move it elsewhere. Part 2 replaces this with a server.

## Part 2: Go backend (planned)

The second learning project replaces `localStorage` with a Go API that this app talks to. Go was picked first because it builds directly on this app and is the gentlest of the languages considered (Go, Rust, Ruby).

The full instructions, which a Claude Code session loads on its own, are in [`server/CLAUDE.md`](server/CLAUDE.md). It runs differently from the Vue project:

- **All steps in one go.** No pause between steps, but a commit at the end of each one (`go step 3: ...`), with every commit passing its checks.
- **Tests first**, so the tests explain what the code is meant to do.
- **Go concepts are explained in `server/LEARNING.md`** as they first appear, compared with TypeScript, instead of in chat.
- **Standard library only**, plus one SQLite driver.

### Before you start

1. Install Go from https://go.dev/dl/ (1.22 or newer), and check with `go version`.
2. Start a new Claude Code session in this folder and ask it to build the Go backend following `server/CLAUDE.md`.

### Steps

| Step | Built | Go concepts | Closest TypeScript / Vue equivalent |
|---|---|---|---|
| 0. Module and first server | `server/` module, `GET /api/health` | Modules and packages, `func main`, exported names by capitalisation, the `go` toolchain | `package.json`, `export`, ESLint + Prettier + Vitest in one tool |
| 1. Types and validation | Structs for boards, columns and cards; seed data; validation | Structs and JSON tags, maps and slices, pointers for `null`, errors as values, table-driven tests | `types.ts`, `validation.ts`, `it.each` |
| 2. In-memory store | Every board operation, with the store tests ported | Methods, implicit interfaces, `sync.Mutex`, `defer` | The Pinia store, TS structural typing |
| 3. Whole-state API | `GET`/`PUT /api/state` with version checks | `net/http` handlers, `ServeMux` patterns, middleware, JSON decoding | Vue Router's route table, Express-style middleware |
| 4. Connect the Vue app | Vite proxy; the store loads and saves through the API | (Vue side) async route guards, `fetch` instead of `useStorage` | Phase 5's deep watch, with a server behind it |
| 5. SQLite | Tables with cascading deletes, embedded migrations | `database/sql`, transactions, `embed`, one interface with two implementations | An ORM, without the ORM |
| 6. Per-action endpoints | REST endpoints for each store action, optimistic updates | Resource design, `PATCH` vs `PUT`, status codes | The store actions from phase 2 |
| 7. Live updates | Changes appear in other tabs without a refresh | Goroutines, channels, `select`, `context` cancellation | The event loop and `async`/`await`; `EventSource` |
| 8. One binary | The Go executable serves the API and the built Vue app | `embed.FS`, flags, graceful shutdown, cross-compiling | `vite preview` and a Node server, in one file |
| 9. Finish | READMEs, `LEARNING.md` exercises, `QUIZ.md` | (Review) | Phase 9 |

The API is written down language-neutrally in `docs/api.md` as it is built, so another backend can implement the same contract later.

### Afterwards: the other languages

These are options for later sessions, not part of the Go plan.

- **Ruby (Sinatra or Rails):** the same API in `server-ruby/`, implementing `docs/api.md`. The Vue app switches to it by changing one environment variable (`KANBAN_API`), which makes the trade-off visible: Rails gets there faster, while Go gives more speed and explicit control.
- **Rust (a small `grep` clone):** a standalone command-line tool with regex search, recursive directory walking, coloured output and flags like `--ignore-case`. It doesn't connect to this app, so it gets its own folder. It's the hardest of the three, which is why it comes after Go.
- **Comparison exercise:** the same small program (a word-frequency counter or a CSV summariser) in Go, Rust and Ruby, to compare syntax, error handling and speed side by side.
