-- migrate:up
CREATE TABLE IF NOT EXISTS transfer_template_category (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

ALTER TABLE transfer_template
ADD COLUMN parent_template_id TEXT REFERENCES transfer_template (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transfer_template_parent_template_id ON transfer_template (parent_template_id);

