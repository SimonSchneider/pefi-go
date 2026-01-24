-- migrate:up
CREATE TABLE IF NOT EXISTS startup_share_account (
    account_id TEXT NOT NULL PRIMARY KEY,
    shares_owned REAL NOT NULL,
    total_shares REAL NOT NULL,
    purchase_price_per_share REAL NOT NULL,
    tax_rate REAL NOT NULL,
    valuation_discount_factor REAL NOT NULL DEFAULT 0.5,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS investment_round (
    id TEXT NOT NULL PRIMARY KEY,
    account_id TEXT NOT NULL,
    date INTEGER NOT NULL,
    valuation REAL NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE,
    UNIQUE (account_id, date)
);

CREATE TABLE IF NOT EXISTS startup_share_option (
    id TEXT NOT NULL PRIMARY KEY,
    account_id TEXT NOT NULL,
    source_account_id TEXT NOT NULL,
    shares REAL NOT NULL,
    strike_price_per_share REAL NOT NULL,
    grant_date INTEGER NOT NULL,
    end_date INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE,
    FOREIGN KEY (source_account_id) REFERENCES account (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_investment_round_account_date ON investment_round (account_id, date);
CREATE INDEX IF NOT EXISTS idx_startup_share_option_account ON startup_share_option (account_id);
CREATE INDEX IF NOT EXISTS idx_startup_share_option_source_account ON startup_share_option (source_account_id);

