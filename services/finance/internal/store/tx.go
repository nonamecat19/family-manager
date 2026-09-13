package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nnc/family-manager/services/finance/db"
)

type Pool struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Pool { return &Pool{pool: pool} }

func (p *Pool) Queries() db.Querier { return db.New(p.pool) }

func (p *Pool) InTx(ctx context.Context, fn func(q db.Querier) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(db.New(p.pool).WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
