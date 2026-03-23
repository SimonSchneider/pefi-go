-- migrate:up
CREATE TABLE IF NOT EXISTS inkomstbasbelopp (
    id         TEXT    NOT NULL PRIMARY KEY,
    amount     REAL    NOT NULL,
    valid_from INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
