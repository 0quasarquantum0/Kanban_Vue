# Architecture

_Last updated: phase 1 (static board with components and props)._

This file shows the project as it currently stands. It is updated at the end of every phase and only diagrams what exists.

## App bootstrap

```mermaid
flowchart TD
  html["index.html<br/>div#app"] --> main["src/main.ts<br/>createApp(App)"]
  main -- "app.use()" --> pinia["Pinia<br/>(no stores yet)"]
  main -- "app.use()" --> router["Vue Router<br/>src/router/index.ts"]
  main -- "mount('#app')" --> app["App.vue<br/>RouterView"]
  router -. "resolves current route" .-> app
```

## Component tree

Solid arrows are parent → child, labelled with the props passed down. Nothing emits events yet; the board is read-only.

```mermaid
flowchart TD
  app["App.vue<br/>RouterView"] --> home["views/HomeView.vue"]
  home --> board["components/BoardView.vue<br/>state: ref&lt;KanbanState&gt; from createSeedState()"]
  board -- "props: title, cardCount<br/>(v-for over columns)" --> column["components/KanbanColumn.vue<br/>slots: #actions, default<br/>empty state when cardCount is 0"]
  board -- "props: card<br/>(v-for, rendered into the column's default slot)" --> card["components/KanbanCard.vue"]
  column -. "default slot renders" .-> card
```

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

`src/seed.ts` exports `createSeedState()`, which builds one board with three columns (To do / Doing / Done) and six cards, with fresh IDs on each call.

## Routes

```mermaid
flowchart LR
  root["/"] --> home["HomeView → BoardView"]
```

## Not built yet

- Pinia store, editing, and events from child components: phase 2
- Drag and drop: phase 3
