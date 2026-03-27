-- migrate:up
DROP INDEX IF EXISTS idx_transfer_template_parent_template_id;
ALTER TABLE transfer_template DROP COLUMN parent_template_id;
