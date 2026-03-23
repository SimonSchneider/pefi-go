-- migrate:up
CREATE TABLE IF NOT EXISTS app_setting (
    key   TEXT NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS api_cache (
    cache_key  TEXT    NOT NULL PRIMARY KEY,
    value      TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);
