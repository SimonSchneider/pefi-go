-- name: GetFavicon :one
SELECT icon_data, content_type FROM favicon_cache WHERE domain = ?;

-- name: UpsertFavicon :exec
INSERT INTO favicon_cache (domain, icon_data, content_type, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (domain) DO UPDATE SET icon_data = EXCLUDED.icon_data, content_type = EXCLUDED.content_type, created_at = EXCLUDED.created_at;
