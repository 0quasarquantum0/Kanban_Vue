# API contract

The HTTP API the Vue app talks to. It is written in a language-neutral way: any backend that implements this document (the Go server in `server/` today; a Ruby one could follow) can be used by pointing the app's `KANBAN_API` at it.

_Last updated: Go step 3 (whole-state sync). Endpoints are added here as they are built._

## Conventions

- JSON in and out, `Content-Type: application/json`. Request bodies must be a single JSON value; unknown fields, trailing data, and bodies over the size limit are rejected.
- Errors are `{"error": "message"}` with one of these statuses:

  | Status | Meaning |
  |---|---|
  | 400 | the request is malformed or the data is invalid (the message says why) |
  | 404 | the path or a named board, column or card does not exist |
  | 405 | the path exists but not for this method (an `Allow` header lists the methods) |
  | 409 | the write was based on an old version of the state |
  | 413 | the request body is too large |
  | 500 | an unexpected server error (the message is generic; details are in the server log) |

- The server keeps one **version** number for the whole state. Every successful write increases it by one. A client that writes the whole state sends the version it last saw.
- IDs are UUIDs (lowercase, 8-4-4-4-12 hex digits).

## The state

`State` is exactly the frontend's `KanbanState` from `src/types.ts`, and the format of its Export JSON button:

```json
{
  "boards":  { "<id>": { "id": "<id>", "title": "…", "columnIds": ["<id>", …] } },
  "columns": { "<id>": { "id": "<id>", "title": "…", "cardIds": ["<id>", …] } },
  "cards":   { "<id>": { "id": "<id>", "title": "…", "description": "…",
                         "labels": ["…"], "dueDate": "2026-11-20" , "createdAt": "2026-11-18T01:02:03.000Z" } }
}
```

- `dueDate` is a string or `null`. `labels`, `columnIds` and `cardIds` are always arrays, never `null`.
- The order of the keys in `boards` is the order boards are listed in. Servers must preserve it.
- A state is valid when every entity is stored under its own `id`, every column a board lists and every card a column lists exists and is listed only once, **and every column is listed by a board and every card by a column** (a relational store cannot hold orphans; `src/validation.ts` is kept in line with this rule).

## Endpoints

### `GET /api/health`

`200` `{"status": "ok"}`.

### `GET /api/state`

`200` `{"version": 7, "state": <State>}`.

### `PUT /api/state`

Replaces the whole state. Used for Import JSON and Reset to demo data.

Request: `{"version": 7, "state": <State>}`. Both fields are required.

| Status | When |
|---|---|
| 200 `{"version": 8}` | saved |
| 400 | `version` or `state` missing, the state is invalid, or the JSON is malformed |
| 409 | `version` is not the server's current version; the client should reload |
| 413 | body over 16 MiB |
