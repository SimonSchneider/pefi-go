-- name: GetAccount :one
SELECT *
FROM account
WHERE id = ?;

-- name: UpdateAccount :one
UPDATE account
SET name                = ?,
    updated_at          = ?,
    balance_upper_limit = ?,
    cash_flow_frequency = ?,
    cash_flow_destination_id = ?
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
(id, name, balance_upper_limit, cash_flow_frequency, cash_flow_destination_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
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
