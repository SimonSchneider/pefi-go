-- migrate:up
ALTER TABLE investment_round ADD COLUMN pre_money_shares REAL NOT NULL DEFAULT 0;
ALTER TABLE investment_round ADD COLUMN investment REAL NOT NULL DEFAULT 0;

UPDATE investment_round
SET pre_money_shares = (SELECT total_shares FROM startup_share_account WHERE startup_share_account.account_id = investment_round.account_id),
    investment = 0
WHERE (SELECT 1 FROM startup_share_account WHERE startup_share_account.account_id = investment_round.account_id);

CREATE TABLE IF NOT EXISTS share_change (
    id TEXT NOT NULL PRIMARY KEY,
    account_id TEXT NOT NULL,
    date INTEGER NOT NULL,
    delta_shares REAL NOT NULL,
    total_price REAL NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_share_change_account_date ON share_change (account_id, date);
CREATE INDEX IF NOT EXISTS idx_share_change_account ON share_change (account_id);

INSERT INTO share_change (id, account_id, date, delta_shares, total_price, created_at, updated_at)
SELECT
    'mig-' || ssa.account_id,
    ssa.account_id,
    COALESCE((SELECT MIN(date) FROM investment_round WHERE account_id = ssa.account_id), 0),
    ssa.shares_owned,
    ssa.shares_owned * ssa.purchase_price_per_share,
    (unixepoch('now') * 1000),
    (unixepoch('now') * 1000)
FROM startup_share_account ssa
WHERE ssa.shares_owned != 0 OR ssa.purchase_price_per_share != 0;

CREATE TABLE startup_share_account_new (
    account_id TEXT NOT NULL PRIMARY KEY,
    tax_rate REAL NOT NULL,
    valuation_discount_factor REAL NOT NULL DEFAULT 0.5,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE
);
INSERT INTO startup_share_account_new (account_id, tax_rate, valuation_discount_factor)
SELECT account_id, tax_rate, valuation_discount_factor FROM startup_share_account;
DROP TABLE startup_share_account;
ALTER TABLE startup_share_account_new RENAME TO startup_share_account;
