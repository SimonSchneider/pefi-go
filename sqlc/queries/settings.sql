-- name: GetSetting :one
SELECT value
FROM app_setting
WHERE key = ?;

-- name: UpsertSetting :exec
INSERT INTO app_setting (key, value)
VALUES (?, ?)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- name: ListSettings :many
SELECT key, value
FROM app_setting
ORDER BY key;
