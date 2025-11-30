-- migrate:up
CREATE TABLE IF NOT EXISTS transfer_template_category (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS transfer_template_category_assignment (
    transfer_template_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    PRIMARY KEY (transfer_template_id, category_id),
    FOREIGN KEY (transfer_template_id) REFERENCES transfer_template (id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES transfer_template_category (id) ON DELETE CASCADE
);

ALTER TABLE transfer_template
ADD COLUMN parent_template_id TEXT REFERENCES transfer_template (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transfer_template_parent_template_id ON transfer_template (parent_template_id);
CREATE INDEX IF NOT EXISTS idx_transfer_template_category_assignment_template ON transfer_template_category_assignment (transfer_template_id);
CREATE INDEX IF NOT EXISTS idx_transfer_template_category_assignment_category ON transfer_template_category_assignment (category_id);

