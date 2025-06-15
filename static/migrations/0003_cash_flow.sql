-- migrate:up
ALTER TABLE account
    ADD COLUMN cash_flow_frequency TEXT;
ALTER TABLE account
    ADD COLUMN cash_flow_destination_id TEXT REFERENCES account (id) ON DELETE SET NULL;
