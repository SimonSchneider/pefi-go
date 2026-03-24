-- name: ListInkomstbasbelopp :many
SELECT *
FROM inkomstbasbelopp
ORDER BY valid_from, id;

-- name: GetInkomstbasbelopp :one
SELECT *
FROM inkomstbasbelopp
WHERE id = ?;

-- name: UpsertInkomstbasbelopp :one
INSERT INTO inkomstbasbelopp (
    id,
    amount,
    prisbasbelopp,
    valid_from,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET amount = EXCLUDED.amount,
  prisbasbelopp = EXCLUDED.prisbasbelopp,
  valid_from = EXCLUDED.valid_from,
  updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteInkomstbasbelopp :exec
DELETE FROM inkomstbasbelopp
WHERE id = ?;
