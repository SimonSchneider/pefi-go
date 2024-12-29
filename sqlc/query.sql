-- name: GetAccount :one
SELECT *
FROM account
WHERE id = ?;

-- name: UpdateAccount :one
UPDATE account
SET name       = ?,
    type       = ?,
    owner_id   = ?,
    updated_at = ?
WHERE id = ? RETURNING *;

-- name: DeleteAccount :one
DELETE
FROM account
WHERE id = ? RETURNING *;

-- name: ListAccounts :many
SELECT *
FROM account
ORDER BY name, id;

-- name: CreateAccount :one
INSERT INTO account
    (id, name, type, owner_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetUser :one
SELECT *
FROM user
WHERE id = ?;

-- name: UpdateUser :one
UPDATE user
SET name       = ?,
    updated_at = ?
WHERE id = ? RETURNING *;

-- name: DeleteUser :one
DELETE
FROM user
WHERE id = ? RETURNING *;

-- name: ListUsers :many
SELECT *
FROM user
ORDER BY name, id;

-- name: CreateUser :one
INSERT INTO user
    (id, name, created_at, updated_at)
VALUES (?, ?, ?, ?) RETURNING *;
