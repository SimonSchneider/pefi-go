-- migrate:up
CREATE TABLE IF NOT EXISTS account_type (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL
);

ALTER TABLE account
ADD COLUMN type_id TEXT REFERENCES account_type (id) ON DELETE
SET NULL;
