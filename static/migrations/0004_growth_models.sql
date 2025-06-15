-- migrate:up
CREATE TABLE IF NOT EXISTS growth_model
(
    id                 TEXT    NOT NULL PRIMARY KEY,
    account_id         TEXT    NOT NULL,

    model_type         TEXT    NOT NULL,
    annual_growth_rate TEXT    NOT NULL,
    annual_volatility  TEXT    NOT NULL,

    start_date         INTEGER NOT NULL,
    end_date           INTEGER,

    created_at         INTEGER NOT NULL,
    updated_at         INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE
);
