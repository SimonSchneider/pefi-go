-- migrate:up
ALTER TABLE transfer_template
ADD COLUMN budget_category_id TEXT REFERENCES transfer_template_category(id) ON DELETE SET NULL;

ALTER TABLE account
ADD COLUMN budget_category_id TEXT REFERENCES transfer_template_category(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transfer_template_budget_category ON transfer_template (budget_category_id);
CREATE INDEX IF NOT EXISTS idx_account_budget_category ON account (budget_category_id);
