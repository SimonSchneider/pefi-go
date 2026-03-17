-- migrate:up
-- Allow multiple share changes per (account_id, date). Uniqueness is on id only.
DROP INDEX IF EXISTS idx_share_change_account_date;
CREATE INDEX IF NOT EXISTS idx_share_change_account_date ON share_change (account_id, date);
