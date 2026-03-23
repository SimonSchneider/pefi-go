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
