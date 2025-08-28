-- migrate:up
CREATE TABLE IF NOT EXISTS transfer_template
(
    id              TEXT    NOT NULL PRIMARY KEY,
    name            TEXT    NOT NULL,

    from_account_id TEXT,
    to_account_id   TEXT,

    amount_type     TEXT    NOT NULL,
    amount_fixed    TEXT    NOT NULL,
    amount_percent  FLOAT   NOT NULL,
    priority        INTEGER NOT NULL,
    recurrence      TEXT    NOT NULL,

    start_date      INTEGER NOT NULL,
    end_date        INTEGER,

    enabled         BOOLEAN NOT NULL DEFAULT TRUE,

    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,

    FOREIGN KEY (from_account_id) REFERENCES account (id) ON DELETE CASCADE,
    FOREIGN KEY (to_account_id) REFERENCES account (id) ON DELETE CASCADE
);
