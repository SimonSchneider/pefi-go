-- migrate:up
ALTER TABLE account
    ADD COLUMN balance_upper_limit FLOAT;
