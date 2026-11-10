# Architecture

_Last updated: phase 2 (Pinia store and editing)._

This file shows the project as it currently stands. It is updated at the end of every phase and only diagrams what exists.

## App bootstrap

```mermaid
flowchart TD
  html["index.html<br/>div#app"] --> main["src/main.ts<br/>createApp(App)"]
  main -- "app.use()" --> pinia["Pinia<br/>stores/kanban.ts"]
  main -- "app.use()" --> router["Vue Router<br/>src/router/index.ts"]
  main -- "mount('#app')" --> app["App.vue<br/>RouterView"]
  router -. "resolves current route" .-> app
```

## Component tree

Solid arrows are parent → child, labelled with what is passed down (props, `v-model`) and what comes back up (events). Dotted arrows are store access. Only `BoardView` talks to the store; every other component is driven by props, models and events.

```mermaid
flowchart TD
  app["App.vue<br/>RouterView"] --> home["views/HomeView.vue"]
  home --> board["components/BoardView.vue"]
  store[("stores/kanban.ts<br/>useKanbanStore")]
  board -. "reads getters, calls actions,<br/>v-model writes column.title" .-> store

  board -- "v-model:title, prop cardCount<br/>(v-for over columns)" --> column["components/KanbanColumn.vue<br/>slots: #actions, default, #footer<br/>empty state when cardCount is 0"]
  board -- "prop card<br/>emits update(patch), delete<br/>(into column's default slot)" --> card["components/KanbanCard.vue"]
  board -- "props label, placeholder<br/>emits add(title)<br/>(column #footer, and once for 'Add column')" --> add["components/AddItemForm.vue"]

  column -- "v-model (title)" --> inlineCol["components/InlineEdit.vue<br/>defineModel"]
  card -- ":model-value + @update:model-value<br/>(title)" --> inlineCard["components/InlineEdit.vue"]
```

## Store

`stores/kanban.ts` is a Pinia setup store. State is one deep `ref<KanbanState>`, initialised from `createSeedState()` (not persisted yet, so a refresh resets it).

```mermaid
classDiagram
  class useKanbanStore {
    +state: Ref~KanbanState~
    +firstBoard: Board | undefined
    +columnsForBoard(boardId) Column[]
    +cardsForColumn(columnId) Card[]
    +addCard(columnId, title) Card
    +updateCard(cardId, patch)
    +deleteCard(cardId)
    +addColumn(boardId, title) Column
    +renameColumn(columnId, title)
    +deleteColumn(columnId)
    +moveCard(cardId, toColumnId, toIndex)
  }
```

- `firstBoard`, `columnsForBoard` and `cardsForColumn` are `computed` getters; the last two return lookup functions.
- `deleteColumn` also deletes the column's cards.
- `moveCard` places the card at `toIndex` in the target column (clamped). No UI calls it yet; drag and drop arrives in phase 3.
- Unit tests: `src/stores/__tests__/kanban.spec.ts` (`addCard`, `moveCard`, `deleteColumn`).

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

`CardPatch` is the editable subset of a card (`title`, `description`, `labels`, `dueDate`), used by `updateCard` and `KanbanCard`'s `update` event.

`src/seed.ts` exports `createSeedState()`, which builds one board with three columns (To do / Doing / Done) and six cards, with fresh IDs on each call.

## Routes

```mermaid
flowchart LR
  root["/"] --> home["HomeView → BoardView"]
```

## Not built yet

- Drag and drop (the UI for `moveCard`): phase 3
- Persistence: phase 5
