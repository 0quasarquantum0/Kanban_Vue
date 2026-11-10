# AGENTS.md — Vue Kanban Board

## Purpose

This is a **learning project**. The developer is experienced with React and Next.js and is learning Vue 3. The goal is a working Kanban board, but the more important goal is that the developer understands *how Vue does things* along the way.

Build in the phases below, in order. Each phase teaches specific Vue features.

## Working agreement (read first)

- **One phase at a time.** Finish a phase, make sure `npm run dev` runs without errors and `npm run type-check` passes, then stop. Summarise what was built and wait for the developer to say "next" before starting the next phase.
- **Explain Vue concepts against React.** When a phase introduces a Vue feature, add a short explanation (3–6 sentences) in your summary comparing it to the React equivalent. Example: "`computed` is like `useMemo`, but it tracks its dependencies automatically, so there's no dependency array."
- **Skip JavaScript/TypeScript basics.** The developer knows them. Only explain what's Vue-specific.
- **Write idiomatic Vue, not React-in-Vue.** Mutate reactive state directly. Use `v-model`, `computed`, directives, and slots rather than reproducing React patterns like lifting every piece of state or passing callbacks for everything.
- **Prefer the platform and the Vue ecosystem over extra dependencies.** Ask before adding any npm package not listed in the Stack section.
- Commit at the end of each phase with a message like `phase 3: drag and drop between columns`.

## Stack

| Concern | Choice |
|---|---|
| Framework | Vue 3, Composition API, `<script setup lang="ts">` everywhere |
| Build tool | Vite (via `create-vue`) |
| Language | TypeScript (strict) |
| State | Pinia, using **setup stores** (`defineStore('id', () => { ... })`) |
| Routing | Vue Router |
| Utilities | VueUse (`@vueuse/core`) |
| Styling | Plain CSS: `<style scoped>` in components, design tokens as CSS custom properties in `src/assets/main.css`. No Tailwind, no UI kit. |
| Drag and drop | Native HTML5 drag-and-drop events, wrapped in our own composable (no library) |
| Unit tests | Vitest + `@vue/test-utils` |
| Linting | ESLint + Prettier (as configured by `create-vue`) |

Do **not** use the Options API, Vuex, class components, or JSX.

## Commands

```bash
npm install
npm run dev          # dev server (Vite), usually http://localhost:5173
npm run build        # type-check + production build to dist/
npm run preview      # serve the production build
npm run type-check   # vue-tsc
npm run test:unit    # Vitest
npm run lint
```

## Data model

```ts
// src/types.ts
export type ID = string // crypto.randomUUID()

export interface Card {
  id: ID
  title: string
  description: string
  labels: string[]       // e.g. ['bug', 'ui']
  dueDate: string | null // ISO date
  createdAt: string
}

export interface Column {
  id: ID
  title: string
  cardIds: ID[]          // order matters
}

export interface Board {
  id: ID
  title: string
  columnIds: ID[]        // order matters
}

export interface KanbanState {
  boards: Record<ID, Board>
  columns: Record<ID, Column>
  cards: Record<ID, Card>
}
```

State is normalised (entities stored by ID, order stored as ID arrays). This keeps moves cheap: moving a card is just splicing an ID between two `cardIds` arrays.

## Target structure

```
src/
  main.ts
  App.vue
  router/index.ts
  stores/kanban.ts            # Pinia setup store
  composables/
    useDragAndDrop.ts
    useKeyboardShortcuts.ts
  components/
    BoardView.vue
    KanbanColumn.vue
    KanbanCard.vue
    CardEditor.vue            # modal content
    BaseModal.vue             # uses <Teleport> + <Transition>
    InlineEdit.vue            # uses defineModel
    FilterBar.vue
  views/
    BoardsListView.vue
    BoardDetailView.vue
  assets/main.css
  types.ts
  seed.ts                     # demo data for first run
```

## Phases

### Phase 0 — Scaffold

- Run `npm create vue@latest` with TypeScript, Vue Router, Pinia, Vitest, ESLint, and Prettier enabled. Skip JSX and end-to-end tests.
- Install `@vueuse/core`.
- Delete the starter components and views. Leave a minimal `App.vue` with `<RouterView />`.
- **Explain:** the anatomy of a `.vue` single-file component, what `<script setup>` compiles to, and how `main.ts` mounts the app (vs `createRoot` in React).
- **Done when:** the dev server shows a blank "Kanban" heading with no console errors.

### Phase 1 — Static board with components and props

- Add `types.ts` and `seed.ts` (one board, three columns: To do / Doing / Done, about 6 cards).
- Build `BoardView`, `KanbanColumn`, and `KanbanCard`. For now, render the seed data from a plain `ref` in `BoardView`.
- Use `defineProps` with TypeScript types, `v-for` with `:key`, and `v-if`/`v-else` for empty columns.
- Give `KanbanColumn` a **named slot** for its header actions and a default slot for its cards.
- **Explain:** template syntax (`:prop`, `@event`, `{{ }}`), `v-for`/`v-if` vs `.map()`/`&&`, and slots vs `children`.
- **Done when:** the seed board renders in three styled columns.

### Phase 2 — Pinia store and editing

- Move state into `stores/kanban.ts` as a setup store. Use `ref` for state, `computed` for getters (e.g. `cardsForColumn(columnId)`), and plain functions for actions: `addCard`, `updateCard`, `deleteCard`, `addColumn`, `renameColumn`, `deleteColumn`, `moveCard(cardId, toColumnId, toIndex)`.
- Use `storeToRefs` in components where destructuring is needed.
- Add "+ Add card" at the bottom of each column. Bind the input with `v-model` and submit with `@keyup.enter`.
- Build `InlineEdit.vue` using `defineModel()` so that `<InlineEdit v-model="column.title" />` works for renaming columns.
- Child-to-parent communication uses `defineEmits` with typed events.
- **Explain:** `ref` vs `reactive` and why `.value` is needed in script but not in templates; how Pinia compares to Zustand or Redux; how `v-model` and `defineModel` replace controlled-input boilerplate.
- **Tests:** unit-test the store's `moveCard`, `addCard`, and `deleteColumn` (deleting a column also deletes its cards).
- **Done when:** cards and columns can be added, renamed, and deleted, and the tests pass.

### Phase 3 — Drag and drop

- Write `composables/useDragAndDrop.ts` around native `dragstart`, `dragover`, `dragleave`, and `drop` events. It should expose reactive state (`draggingCardId`, `overColumnId`, `overIndex`) and handler functions.
- Support reordering within a column and moving between columns, with a visible drop indicator.
- Use event modifiers where they fit (`@dragover.prevent`).
- **Explain:** composables vs custom hooks. In particular, they run once in setup rather than on every render, so React's rules of hooks don't apply.
- **Done when:** cards can be dragged anywhere, and the order is correct after drop.

### Phase 4 — Transitions

- Wrap each column's card list in `<TransitionGroup>` with a move class so cards animate smoothly when reordered or moved.
- Add enter and leave animations for adding and deleting cards.
- **Explain:** Vue's built-in `<Transition>` and `<TransitionGroup>`, and how they compare to Framer Motion or React Transition Group.
- **Done when:** card moves, adds, and deletes animate. Respect `prefers-reduced-motion`.

### Phase 5 — Persistence

- Persist the store's state to `localStorage` using VueUse's `useStorage` inside the setup store. Load the seed data only on first run.
- Add "Export JSON" and "Import JSON" buttons. Validate imported data before replacing state.
- Add a "Reset to demo data" action, with a confirmation step.
- **Explain:** how a deep reactive object syncs to storage without any manual subscription code.
- **Done when:** a page refresh keeps all changes.

### Phase 6 — Multiple boards and routing

- Routes:
  - `/` → `BoardsListView` (create, rename, and delete boards)
  - `/boards/:boardId` → `BoardDetailView`
  - `/boards/:boardId/cards/:cardId` → nested route that opens `CardEditor` in a modal over the board
- Add a `beforeEnter` guard that redirects to `/` if the board doesn't exist.
- Build `BaseModal.vue` with `<Teleport to="body">` and `<Transition>`. It should close on Escape and on backdrop click, and closing it navigates back to the board route.
- `CardEditor` edits the title, description, labels, and due date.
- **Explain:** Vue Router vs the Next.js App Router (no file-based routing here; mention that Nuxt adds it), `useRoute`/`useRouter`, nested `<RouterView>`, and `<Teleport>` vs `createPortal`.
- **Done when:** card deep links work on refresh, and back/forward navigation behaves correctly.

### Phase 7 — Filtering and search

- Build `FilterBar` with text search and label filters. Filtering is done with `computed` only. Never copy filtered results into state.
- Sync filters to the URL query (`?q=...&label=bug`) so filtered views are shareable. Use a debounced `watch` (VueUse `watchDebounced`) for the search text.
- Show a "no matches" state for each column.
- Disable drag and drop while filters are active, and explain why in the UI.
- **Explain:** `watch` vs `watchEffect` vs `useEffect`, and when you actually need a watcher rather than a `computed`.
- **Done when:** filters survive a refresh through the URL.

### Phase 8 — Polish

- **Theme:** add light and dark themes with CSS variables. Pass the current theme to deep components with `provide`/`inject` using a typed `InjectionKey`, then compare the result to React Context.
- **Keyboard shortcuts:** build `useKeyboardShortcuts` with VueUse `onKeyStroke`. `n` creates a new card in the first column, `/` focuses search, and `Esc` closes the modal.
- **Accessibility:** use focus management in the modal, ARIA labels on icon buttons, and add a keyboard alternative to drag (a "Move to…" menu on each card).
- **Lifecycle:** use lifecycle hooks where appropriate (e.g. `onMounted` to focus inputs).
- **Done when:** the board is usable with the keyboard alone.

### Phase 9 — Tests and build

- Add component tests with `@vue/test-utils`:
  - `KanbanCard` renders its title and labels.
  - `InlineEdit` emits `update:modelValue`.
  - `FilterBar` updates the route query.
- Make sure `npm run build` succeeds and `npm run preview` serves a working app.
- Write a short `README.md` covering how to run the app and a summary of the Vue concepts each phase covered.
- **Done when:** lint, type-check, tests, and build are all clean.

## Optional stretch goals (only if asked)

- Port the app to **Nuxt** to compare with Next.js: file-based routing, SSR, and server routes for persistence.
- Add touch support for drag and drop on mobile.
- Add undo/redo with Pinia `$subscribe` or VueUse `useRefHistory`.

## Conventions

- Component files are PascalCase and use multi-word names (`KanbanCard.vue`, not `Card.vue`).
- Composables are named `useX` and live in `src/composables/`.
- Props are typed with `defineProps<{...}>()`, with `withDefaults` where needed. Events are typed with `defineEmits<{...}>()`.
- Never mutate props. Emit an event or use `defineModel` instead.
- Keep components under about 150 lines. Extract a child component or a composable when one grows past that.
- No `any`.
