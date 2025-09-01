-- migrate:up
ALTER TABLE account_type
ADD COLUMN color TEXT;
