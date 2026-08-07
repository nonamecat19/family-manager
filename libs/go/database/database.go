// Package database owns the Postgres connection pool and the migration runner shared by
// every Go service. Services never build their own pgxpool config.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config describes a pool. Only URL is required.
type Config struct {
	// URL is a libpq connection string, e.g. postgres://user:pass@host:5432/db?sslmode=disable.
	URL string
	// MaxConns caps the pool. Zero means pgx's default (max(4, NumCPU)).
	MaxConns int32
	// MaxConnLifetime recycles connections. Zero means one hour.
	MaxConnLifetime time.Duration
	// ConnectTimeout bounds the initial ping. Zero means five seconds.
	ConnectTimeout time.Duration
}

// Connect opens the pool and verifies it with a ping, so a bad DSN fails at boot rather
// than on the first request.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database: empty URL")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("database: parse url: %w", err)
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	if poolCfg.MaxConnLifetime == 0 {
		poolCfg.MaxConnLifetime = time.Hour
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: new pool: %w", err)
	}

	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	return pool, nil
}
