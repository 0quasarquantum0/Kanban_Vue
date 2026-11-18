# API contract

The HTTP API the Vue app talks to. It is written in a language-neutral way: any backend that implements this document (the Go server in `server/` today; a Ruby one could follow) can be used by pointing the app's `KANBAN_API` at it.

_Last updated: Go step 6 (per-action endpoints). Endpoints are added here as they are built._

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
- IDs are UUIDs (lowercase, 8-4-4-4-12 hex digits). A request that creates something may carry an `id`; the server keeps it when it is a valid UUID that is not in use and generates one otherwise, so the reply's `id` is the one to trust. (This lets a client show a new item before the server has answered.)
- Titles of boards, columns and cards must not be empty or only whitespace.

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

## Per-action endpoints

Each one is a single change, applied in one transaction, and raises the version by one. Unlike `PUT /api/state` they carry **no version**: they describe what to change rather than what the whole state should be, so they do not conflict with each other. An ID in the path that does not exist is a **404**.

Every success reply contains `"version"`, the version after the change.

| Method and path | Request body | Success | Action |
|---|---|---|---|
| `POST /api/boards` | `{"id"?, "title"}` | 201 `{"version", "board": Board}` | add a board (it goes last) |
| `PATCH /api/boards/{boardId}` | `{"title"}` | 200 `{"version"}` | rename a board |
| `DELETE /api/boards/{boardId}` | none | 200 `{"version"}` | delete a board with its columns and cards |
| `POST /api/boards/{boardId}/columns` | `{"id"?, "title"}` | 201 `{"version", "column": Column}` | add a column (it goes last) |
| `PATCH /api/columns/{columnId}` | `{"title"}` | 200 `{"version"}` | rename a column |
| `DELETE /api/columns/{columnId}` | none | 200 `{"version"}` | delete a column with its cards |
| `POST /api/columns/{columnId}/cards` | `{"id"?, "title"}` | 201 `{"version", "card": Card}` | add a card (it goes last) |
| `PATCH /api/cards/{cardId}` | any of `title`, `description`, `labels`, `dueDate` | 200 `{"version", "card": Card}` | change those fields only |
| `DELETE /api/cards/{cardId}` | none | 200 `{"version"}` | delete a card |
| `PUT /api/cards/{cardId}/position` | `{"columnId", "index"}` | 200 `{"version"}` | move a card |

Details:

- **PATCH on a card** distinguishes a missing field (leave it alone) from `null` (only `dueDate` can be cleared this way: `{"dueDate": null}`). `labels` replaces the whole list.
- **Moving** puts the card at `index` among the cards of `columnId` *without the moving card* (so within a column it is the index the card ends up at). An index outside the column is clamped: `-1` means first and a huge number means last. Both fields are required, and `index` must be a whole number.
- **400** for an empty title, a missing required field, an unknown field, or a field of the wrong type.
- `PATCH` is used where a request changes *some* fields of a resource (rename, edit a card). `PUT` is used for the position because the request states a whole value (the card's place), and repeating it changes nothing more.
