# Architecture

_Last updated: phase 0 (scaffold)._

This file shows the project as it currently stands. It is updated at the end of every phase and only diagrams what exists.

## App bootstrap and component tree

```mermaid
flowchart TD
  html["index.html<br/>div#app"] --> main["src/main.ts<br/>createApp(App)"]
  main -- "app.use()" --> pinia["Pinia<br/>(no stores yet)"]
  main -- "app.use()" --> router["Vue Router<br/>src/router/index.ts"]
  main -- "mount('#app')" --> app["App.vue<br/>RouterView"]
  router -. "resolves current route" .-> app
  app --> home["views/HomeView.vue<br/>placeholder: h1 Kanban"]
```

## Routes

```mermaid
flowchart LR
  root["/"] --> home["HomeView"]
```

## Not built yet

- Data model (`src/types.ts`) and seed data: phase 1
- Board components: phase 1
- Pinia store: phase 2
