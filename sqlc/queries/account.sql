-- name: GetAccount :one
SELECT *
FROM account
WHERE id = ?;

-- name: UpdateAccount :one
UPDATE account
SET name                     = ?,
    updated_at               = ?,
    balance_upper_limit      = ?,
    cash_flow_frequency      = ?,
    cash_flow_destination_id = ?,
    type_id                  = ?
WHERE id = ?
RETURNING *;

-- name: DeleteAccount :one
DELETE
FROM account
WHERE id = ?
RETURNING *;

-- name: ListAccounts :many
SELECT *
FROM account
ORDER BY name, id;

-- name: CreateAccount :one
INSERT INTO account
(id, name, balance_upper_limit, cash_flow_frequency, cash_flow_destination_id, type_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSnapshotsByAccount :many
SELECT *
FROM account_snapshot
WHERE account_id = ?;

-- name: GetSnapshotsByAccounts :many
SELECT *
FROM account_snapshot
WHERE account_id IN (sqlc.slice('ids'))
ORDER BY date, account_id;

-- name: GetSnapshot :one
SELECT *
FROM account_snapshot
WHERE account_id = ?
  AND date = ?;

-- name: UpsertSnapshot :one
INSERT OR
REPLACE
INTO account_snapshot
    (account_id, date, balance)
VALUES (?, ?, ?)
RETURNING *;

-- name: DeleteSnapshot :exec
DELETE
FROM account_snapshot
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
INSERT OR
REPLACE
INTO growth_model
(id, account_id, model_type, annual_growth_rate, annual_volatility, start_date, end_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteGrowthModel :exec
DELETE
FROM growth_model
WHERE id = ?;

-- name: GetTransferTemplates :many
SELECT *
FROM transfer_template
ORDER BY recurrence, priority, name, start_date, end_date, created_at;

-- name: GetTransferTemplate :one
SELECT *
FROM transfer_template
WHERE id = ?;

-- name: UpsertTransferTemplate :one
INSERT OR
REPLACE
INTO transfer_template
(id, name, from_account_id, to_account_id, amount_type, amount_fixed, amount_percent, priority, recurrence, start_date,
 end_date, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteTransferTemplate :exec
DELETE
FROM transfer_template
WHERE id = ?;

-- name: ListLatestSnapshotPerAccount :many
SELECT s.*
FROM account_snapshot s
INNER JOIN (
    SELECT account_id, MAX(date) AS max_date
    FROM account_snapshot
    GROUP BY account_id
) latest
ON s.account_id = latest.account_id AND s.date = latest.max_date;

-- name: ListActiveGrowthModels :many
SELECT *
FROM growth_model
WHERE end_date IS NULL OR end_date > @param1 AND start_date <= @param1;

-- name: UpdateSnapshotDate :many
UPDATE account_snapshot
SET date = ?
WHERE date = ?
RETURNING *;

-- name: ListAccountTypes :many
SELECT *
FROM account_type
ORDER BY name, id;

-- name: GetAccountType :one
SELECT *
FROM account_type
WHERE id = ?;

-- name: UpsertAccountType :one
INSERT OR
REPLACE
INTO account_type
(id, name)
VALUES (?, ?)
RETURNING *;

-- name: DeleteAccountType :exec
DELETE
FROM account_type
WHERE id = ?;
