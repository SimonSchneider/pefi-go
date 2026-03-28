package model

import (
	"context"
	"database/sql"
	"flag"
	"io/fs"

	"github.com/SimonSchneider/goslu/config"
	"github.com/SimonSchneider/goslu/migrate"
	"github.com/SimonSchneider/pefigo/pkg/currency"
	"github.com/SimonSchneider/pefigo/pkg/swe"
)

type Service struct {
	db             *sql.DB
	sweClient      *swe.Client
	currencyClient *currency.Client
}

type ServiceOption func(*Service)

func WithCurrencyOptions(opts ...currency.ClientOption) ServiceOption {
	return func(s *Service) {
		ttlCache := NewTTLSQLiteCache(s.db)
		s.currencyClient = currency.NewClient(ttlCache, opts...)
	}
}

func New(db *sql.DB, opts ...ServiceOption) *Service {
	cache := NewSQLiteCache(db)
	ttlCache := NewTTLSQLiteCache(db)
	s := &Service{
		db:             db,
		sweClient:      swe.NewClient(cache),
		currencyClient: currency.NewClient(ttlCache),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Service) SweClient() *swe.Client {
	return s.sweClient
}

func (s *Service) DB() *sql.DB {
	return s.db
}

type Config struct {
	Addr  string
	Watch bool
	DbURL string
}

func ParseConfig(args []string, getEnv func(string) string) (cfg Config, err error) {
	err = config.ParseInto(&cfg, flag.NewFlagSet("", flag.ExitOnError), args, getEnv)
	return cfg, err
}

func GetMigratedDB(ctx context.Context, dir fs.FS, path string, conn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, err
	}
	migrations, err := fs.Sub(dir, path)
	if err != nil {
		return nil, err
	}
	if err := migrate.Migrate(ctx, migrations, db); err != nil {
		return nil, err
	}
	return db, nil
}

func KeyBy[T any](items []T, key func(T) string) map[string]T {
	m := make(map[string]T)
	for _, item := range items {
		m[key(item)] = item
	}
	return m
}

func ptr[T any](v T) *T {
	return &v
}

func initMap[K comparable, V any](m map[K]V, ks ...K) map[K]V {
	var v V
	for _, k := range ks {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m
}
