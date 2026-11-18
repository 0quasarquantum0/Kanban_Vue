-- The whole state is one version number plus three tables. Order is stored as an integer
-- "position" within the parent (the frontend keeps it as the order of an ID array).

-- A single row (id is always 1) that counts the writes made so far.
CREATE TABLE meta (
    id      INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL
);
INSERT INTO meta (id, version) VALUES (1, 0);

CREATE TABLE boards (
    id       TEXT PRIMARY KEY,
    title    TEXT NOT NULL,
    position INTEGER NOT NULL
);

-- Deleting a board deletes its columns, and deleting a column deletes its cards: the database
-- does it (ON DELETE CASCADE), so no code path can leave orphans behind.
CREATE TABLE columns (
    id       TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    title    TEXT NOT NULL,
    position INTEGER NOT NULL
);
CREATE INDEX columns_by_board ON columns (board_id, position);

CREATE TABLE cards (
    id          TEXT PRIMARY KEY,
    column_id   TEXT NOT NULL REFERENCES columns (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    labels      TEXT NOT NULL, -- a JSON array of strings
    due_date    TEXT,          -- NULL means no due date
    created_at  TEXT NOT NULL,
    position    INTEGER NOT NULL
);
CREATE INDEX cards_by_column ON cards (column_id, position);
