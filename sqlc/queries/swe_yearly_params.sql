-- name: ListSweYearlyParams :many
SELECT *
FROM swe_yearly_params
ORDER BY valid_from, id;

-- name: GetSweYearlyParams :one
SELECT *
FROM swe_yearly_params
WHERE id = ?;

-- name: UpsertSweYearlyParams :one
INSERT INTO swe_yearly_params (
    id,
    amount,
    prisbasbelopp,
    schablon_ranta,
    isk_fribelopp,
    valid_from,
    created_at,
    updated_at
  )
VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (id) DO
UPDATE
SET amount = EXCLUDED.amount,
  prisbasbelopp = EXCLUDED.prisbasbelopp,
  schablon_ranta = EXCLUDED.schablon_ranta,
  isk_fribelopp = EXCLUDED.isk_fribelopp,
  valid_from = EXCLUDED.valid_from,
  updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteSweYearlyParams :exec
DELETE FROM swe_yearly_params
WHERE id = ?;
