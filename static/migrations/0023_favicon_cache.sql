-- migrate:up
CREATE TABLE IF NOT EXISTS favicon_cache (
    domain       TEXT NOT NULL PRIMARY KEY,
    icon_data    BLOB NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'image/png',
    created_at   INTEGER NOT NULL
);
