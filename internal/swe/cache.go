package swe

import "context"

// Cache provides key-value storage for API response caching.
// Implementations should be safe for concurrent use.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, value string) error
}
