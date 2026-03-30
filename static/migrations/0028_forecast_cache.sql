-- migrate:up
CREATE TABLE forecast_cache (
    date INTEGER NOT NULL,
    account_type_id TEXT NOT NULL,
    median REAL NOT NULL,
    lower_bound REAL NOT NULL,
    upper_bound REAL NOT NULL,
    PRIMARY KEY (date, account_type_id)
);
