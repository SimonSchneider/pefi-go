-- name: GetCacheEntry :one
SELECT value
FROM api_cache
WHERE cache_key = ?;

-- name: GetCacheEntryIfFresh :one
SELECT value
FROM api_cache
WHERE cache_key = ? AND created_at >= ?;

-- name: UpsertCacheEntry :exec
INSERT INTO api_cache (cache_key, value, created_at)
VALUES (?, ?, ?)
ON CONFLICT (cache_key) DO UPDATE SET value = EXCLUDED.value, created_at = EXCLUDED.created_at;

-- name: DeleteCacheEntry :exec
DELETE FROM api_cache
WHERE cache_key = ?;
