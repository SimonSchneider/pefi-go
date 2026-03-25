-- migrate:up
CREATE TABLE IF NOT EXISTS bill_account (
    id              TEXT    NOT NULL PRIMARY KEY,
    name            TEXT    NOT NULL,
    from_account_id TEXT,
    recurrence      TEXT    NOT NULL DEFAULT '*-*-01',
    priority        INTEGER NOT NULL DEFAULT 0,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    FOREIGN KEY (from_account_id) REFERENCES account(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS bill (
    id                 TEXT    NOT NULL PRIMARY KEY,
    bill_account_id    TEXT    NOT NULL,
    name               TEXT    NOT NULL,
    budget_category_id TEXT,
    enabled            BOOLEAN NOT NULL DEFAULT TRUE,
    notes              TEXT    NOT NULL DEFAULT '',
    url                TEXT    NOT NULL DEFAULT '',
    created_at         INTEGER NOT NULL,
    updated_at         INTEGER NOT NULL,
    FOREIGN KEY (bill_account_id)    REFERENCES bill_account(id) ON DELETE CASCADE,
    FOREIGN KEY (budget_category_id) REFERENCES transfer_template_category(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS bill_amount (
    id         TEXT    NOT NULL PRIMARY KEY,
    bill_id    TEXT    NOT NULL,
    amount     TEXT    NOT NULL,
    start_date INTEGER NOT NULL,
    end_date   INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (bill_id) REFERENCES bill(id) ON DELETE CASCADE
);
