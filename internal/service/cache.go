package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/SimonSchneider/pefigo/internal/pdb"
)

// SQLiteCache implements swe.Cache backed by the api_cache SQLite table.
type SQLiteCache struct {
	db *sql.DB
}

func NewSQLiteCache(db *sql.DB) *SQLiteCache {
	return &SQLiteCache{db: db}
}

func (c *SQLiteCache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := pdb.New(c.db).GetCacheEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (c *SQLiteCache) Set(ctx context.Context, key string, value string) error {
	return pdb.New(c.db).UpsertCacheEntry(ctx, pdb.UpsertCacheEntryParams{
		CacheKey:  key,
		Value:     value,
		CreatedAt: time.Now().Unix(),
	})
}

// TTLSQLiteCache implements currency.Cache with TTL-aware reads.
type TTLSQLiteCache struct {
	db *sql.DB
}

func NewTTLSQLiteCache(db *sql.DB) *TTLSQLiteCache {
	return &TTLSQLiteCache{db: db}
}

func (c *TTLSQLiteCache) Get(ctx context.Context, key string, maxAge time.Duration) (string, bool, error) {
	minCreatedAt := time.Now().Add(-maxAge).Unix()
	val, err := pdb.New(c.db).GetCacheEntryIfFresh(ctx, pdb.GetCacheEntryIfFreshParams{
		CacheKey:  key,
		CreatedAt: minCreatedAt,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (c *TTLSQLiteCache) Set(ctx context.Context, key string, value string) error {
	return pdb.New(c.db).UpsertCacheEntry(ctx, pdb.UpsertCacheEntryParams{
		CacheKey:  key,
		Value:     value,
		CreatedAt: time.Now().Unix(),
	})
}
