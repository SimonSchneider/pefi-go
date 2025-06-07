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
