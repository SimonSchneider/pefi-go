-- migrate:up
ALTER TABLE bill_amount ADD COLUMN period TEXT NOT NULL DEFAULT 'monthly';
