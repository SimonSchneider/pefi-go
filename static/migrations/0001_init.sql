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
    owner_id   TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES user (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS account_snapshot
(
    account_id TEXT    NOT NULL,
    date       INTEGER NOT NULL,
    balance    TEXT    NOT NULL,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, date)
);
