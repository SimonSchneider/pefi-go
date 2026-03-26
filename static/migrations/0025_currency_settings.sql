-- migrate:up
CREATE TABLE IF NOT EXISTS app_setting (
    key   TEXT NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT OR IGNORE INTO app_setting (key, value) VALUES ('default_currency', 'SEK');

ALTER TABLE bill_amount ADD COLUMN currency TEXT NOT NULL DEFAULT '';
