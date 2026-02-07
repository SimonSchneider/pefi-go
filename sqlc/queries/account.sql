-- name: GetAccount :one
SELECT *
FROM account
WHERE id = ?;
-- name: UpdateAccount :one
UPDATE account
SET name = ?,
  updated_at = ?,
  balance_upper_limit = ?,
  cash_flow_frequency = ?,
  cash_flow_destination_id = ?,
  type_id = ?,
  budget_category_id = ?
WHERE id = ?
RETURNING *;
-- name: DeleteAccount :one
DELETE FROM account
WHERE id = ?
RETURNING *;
-- name: ListAccounts :many
SELECT *
FROM account
ORDER BY name,
  id;
-- name: CreateAccount :one
INSERT INTO account (
    id,
    name,
    balance_upper_limit,
    cash_flow_frequency,
    cash_flow_destination_id,
    type_id,
    budget_category_id,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;
-- name: GetSnapshotsByAccount :many
SELECT *
FROM account_snapshot
WHERE account_id = ?;
-- name: GetSnapshotsByAccounts :many
SELECT *
FROM account_snapshot
WHERE account_id IN (sqlc.slice('ids'))
ORDER BY date,
  account_id;
-- name: GetSnapshot :one
SELECT *
FROM account_snapshot
WHERE account_id = ?
  AND date = ?;
-- name: UpsertSnapshot :one
INSERT INTO account_snapshot (account_id, date, balance)
VALUES (?, ?, ?) ON CONFLICT (account_id, date) DO
UPDATE
SET balance = EXCLUDED.balance
RETURNING *;
-- name: DeleteSnapshot :exec
DELETE FROM account_snapshot
WHERE account_id = ?
  AND date = ?;
-- name: GetGrowthModelsByAccount :many
SELECT *
FROM growth_model
WHERE account_id = ?;
-- name: GetGrowthModel :one
SELECT *
FROM growth_model
WHERE id = ?;
-- name: UpsertGrowthModel :one
INSERT INTO growth_model (
    id,
    account_id,
    model_type,
    annual_growth_rate,
    annual_volatility,
    start_date,
    end_date,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET model_type = EXCLUDED.model_type,
  annual_growth_rate = EXCLUDED.annual_growth_rate,
  annual_volatility = EXCLUDED.annual_volatility,
  start_date = EXCLUDED.start_date,
  end_date = EXCLUDED.end_date,
  updated_at = EXCLUDED.updated_at
RETURNING *;
-- name: DeleteGrowthModel :exec
DELETE FROM growth_model
WHERE id = ?;
-- name: GetTransferTemplates :many
SELECT *
FROM transfer_template
ORDER BY recurrence,
  priority,
  name,
  start_date,
  end_date,
  created_at;
-- name: GetTransferTemplate :one
SELECT *
FROM transfer_template
WHERE id = ?;
-- name: UpsertTransferTemplate :one
INSERT INTO transfer_template (
    id,
    name,
    from_account_id,
    to_account_id,
    amount_type,
    amount_fixed,
    amount_percent,
    priority,
    recurrence,
    start_date,
    end_date,
    enabled,
    parent_template_id,
    budget_category_id,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET name = EXCLUDED.name,
  from_account_id = EXCLUDED.from_account_id,
  to_account_id = EXCLUDED.to_account_id,
  amount_type = EXCLUDED.amount_type,
  amount_fixed = EXCLUDED.amount_fixed,
  amount_percent = EXCLUDED.amount_percent,
  priority = EXCLUDED.priority,
  recurrence = EXCLUDED.recurrence,
  start_date = EXCLUDED.start_date,
  end_date = EXCLUDED.end_date,
  enabled = EXCLUDED.enabled,
  parent_template_id = EXCLUDED.parent_template_id,
  budget_category_id = EXCLUDED.budget_category_id,
  updated_at = EXCLUDED.updated_at
RETURNING *;
-- name: DeleteTransferTemplate :exec
DELETE FROM transfer_template
WHERE id = ?;
-- name: ListLatestSnapshotPerAccount :many
SELECT s.*
FROM account_snapshot s
  INNER JOIN (
    SELECT account_id,
      MAX(date) AS max_date
    FROM account_snapshot
    GROUP BY account_id
  ) latest ON s.account_id = latest.account_id
  AND s.date = latest.max_date;
-- name: ListActiveGrowthModels :many
SELECT *
FROM growth_model
WHERE end_date IS NULL
  OR end_date > @param1
  AND start_date <= @param1;
-- name: UpdateSnapshotDate :many
UPDATE account_snapshot
SET date = ?
WHERE date = ?
RETURNING *;
-- name: ListAccountTypes :many
SELECT *
FROM account_type
ORDER BY name,
  id;
-- name: GetAccountType :one
SELECT *
FROM account_type
WHERE id = ?;
-- name: UpsertAccountType :one
INSERT INTO account_type (id, name, color)
VALUES (?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET name = EXCLUDED.name,
  color = EXCLUDED.color
RETURNING *;
-- name: DeleteAccountType :exec
DELETE FROM account_type
WHERE id = ?;
-- name: GetSpecialDates :many
SELECT *
FROM special_date
ORDER BY date,
  name,
  id;
-- name: GetSpecialDate :one
SELECT *
FROM special_date
WHERE id = ?;
-- name: UpsertSpecialDate :one
INSERT INTO special_date (id, name, date, color)
VALUES (?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET name = EXCLUDED.name,
  date = EXCLUDED.date,
  color = EXCLUDED.color
RETURNING *;
-- name: DeleteSpecialDate :exec
DELETE FROM special_date
WHERE id = ?;
-- name: ListTransferTemplateCategories :many
SELECT *
FROM transfer_template_category
ORDER BY name,
  id;
-- name: GetTransferTemplateCategory :one
SELECT *
FROM transfer_template_category
WHERE id = ?;
-- name: UpsertTransferTemplateCategory :one
INSERT INTO transfer_template_category (id, name, color, created_at, updated_at)
VALUES (?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET name = EXCLUDED.name,
  color = EXCLUDED.color,
  updated_at = EXCLUDED.updated_at
RETURNING *;
-- name: DeleteTransferTemplateCategory :exec
DELETE FROM transfer_template_category
WHERE id = ?;
-- name: GetCategoriesForTransferTemplate :many
SELECT c.*
FROM transfer_template_category c
  INNER JOIN transfer_template_category_assignment a ON c.id = a.category_id
WHERE a.transfer_template_id = ?;
-- name: AssignCategoryToTransferTemplate :exec
INSERT INTO transfer_template_category_assignment (transfer_template_id, category_id)
VALUES (?, ?) ON CONFLICT DO NOTHING;
-- name: RemoveCategoryFromTransferTemplate :exec
DELETE FROM transfer_template_category_assignment
WHERE transfer_template_id = ?
  AND category_id = ?;
-- name: GetChildTemplates :many
SELECT *
FROM transfer_template
WHERE parent_template_id = ?
ORDER BY start_date,
  name,
  id;
-- name: UpsertStartupShareAccount :one
INSERT INTO startup_share_account (
    account_id,
    shares_owned,
    total_shares,
    purchase_price_per_share,
    tax_rate,
    valuation_discount_factor
  )
VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (account_id) DO
UPDATE
SET shares_owned = EXCLUDED.shares_owned,
  total_shares = EXCLUDED.total_shares,
  purchase_price_per_share = EXCLUDED.purchase_price_per_share,
  tax_rate = EXCLUDED.tax_rate,
  valuation_discount_factor = EXCLUDED.valuation_discount_factor
RETURNING *;
-- name: GetStartupShareAccount :one
SELECT *
FROM startup_share_account
WHERE account_id = ?;
-- name: DeleteStartupShareAccount :exec
DELETE FROM startup_share_account
WHERE account_id = ?;
-- name: UpsertInvestmentRound :one
INSERT INTO investment_round (
    id,
    account_id,
    date,
    valuation,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (account_id, date) DO
UPDATE
SET valuation = EXCLUDED.valuation,
  updated_at = EXCLUDED.updated_at
RETURNING *;
-- name: GetInvestmentRound :one
SELECT *
FROM investment_round
WHERE id = ?;
-- name: ListInvestmentRounds :many
SELECT *
FROM investment_round
WHERE account_id = ?
ORDER BY date DESC,
  id;
-- name: GetLatestInvestmentRound :one
SELECT *
FROM investment_round
WHERE account_id = ?
  AND date <= ?
ORDER BY date DESC
LIMIT 1;
-- name: DeleteInvestmentRound :exec
DELETE FROM investment_round
WHERE id = ?;
-- name: UpsertStartupShareOption :one
INSERT INTO startup_share_option (
    id,
    account_id,
    source_account_id,
    shares,
    strike_price_per_share,
    grant_date,
    end_date,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET account_id = EXCLUDED.account_id,
  source_account_id = EXCLUDED.source_account_id,
  shares = EXCLUDED.shares,
  strike_price_per_share = EXCLUDED.strike_price_per_share,
  grant_date = EXCLUDED.grant_date,
  end_date = EXCLUDED.end_date,
  updated_at = EXCLUDED.updated_at
RETURNING *;
-- name: GetStartupShareOption :one
SELECT *
FROM startup_share_option
WHERE id = ?;
-- name: ListStartupShareOptions :many
SELECT *
FROM startup_share_option
WHERE account_id = ?
ORDER BY grant_date,
  id;
-- name: DeleteStartupShareOption :exec
DELETE FROM startup_share_option
WHERE id = ?;
-- name: GetBudgetAccounts :many
SELECT *
FROM account
WHERE budget_category_id IS NOT NULL
ORDER BY name,
  id;