-- migrate:up
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS user
(
    id         TEXT    NOT NUll PRIMARY KEY,
    name       TEXT    NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS account
(
    id         TEXT    NOT NULL PRIMARY KEY,
    name       TEXT    NOT NULL,
    type       TEXT    NOT NULL,
    owner_id   TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES user (id) ON DELETE CASCADE
);
