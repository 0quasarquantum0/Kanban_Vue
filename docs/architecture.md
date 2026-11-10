# Architecture

_Last updated: phase 3 (drag and drop)._

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

  dnd[["composables/useDragAndDrop.ts"]]
  board -. "calls once in setup;<br/>attaches its handlers to column and card roots" .-> dnd
  dnd -. "onDrop → moveCard" .-> store

  column -- "v-model (title)" --> inlineCol["components/InlineEdit.vue<br/>defineModel"]
  card -- ":model-value + @update:model-value<br/>(title)" --> inlineCard["components/InlineEdit.vue"]
```

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
- Visual state is global CSS in `src/assets/main.css`: `.is-dragging`, `.is-drop-target`, and the `[data-drop]` indicator line.

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
- `moveCard` places the card at `toIndex` in the target column (clamped). It is called by the drag-and-drop composable.
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

- Transitions: phase 4
- Persistence: phase 5
