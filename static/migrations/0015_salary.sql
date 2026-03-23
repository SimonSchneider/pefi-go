-- migrate:up
CREATE TABLE IF NOT EXISTS salary (
    id                 TEXT    NOT NULL PRIMARY KEY,
    name               TEXT    NOT NULL,
    to_account_id      TEXT,
    priority           INTEGER NOT NULL DEFAULT 0,
    recurrence         TEXT    NOT NULL DEFAULT '*-*-25',
    budget_category_id TEXT,
    enabled            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         INTEGER NOT NULL,
    updated_at         INTEGER NOT NULL,
    FOREIGN KEY (to_account_id)      REFERENCES account(id) ON DELETE SET NULL,
    FOREIGN KEY (budget_category_id) REFERENCES transfer_template_category(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS salary_amount (
    id         TEXT    NOT NULL PRIMARY KEY,
    salary_id  TEXT    NOT NULL,
    amount     TEXT    NOT NULL,
    start_date INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (salary_id) REFERENCES salary(id) ON DELETE CASCADE
);
