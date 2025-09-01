-- migrate:up
CREATE TABLE IF NOT EXISTS special_date (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    date TEXT NOT NULL
);