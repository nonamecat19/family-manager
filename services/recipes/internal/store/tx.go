// Package store binds the generated queries to a connection pool and provides the transaction
// boundary the handler uses.
//
// It exists because db.Queries is generated and must not be edited, and because the handler
// takes a db.Querier interface — which has no notion of a transaction — so that its tests can
// substitute an in-memory fake.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nnc/family-manager/services/recipes/db"
)

// Pool runs queries against a pgx pool, and transactions against connections from it.
type Pool struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Pool { return &Pool{pool: pool} }

// Queries returns the non-transactional querier for reads and single-statement writes.
func (p *Pool) Queries() db.Querier { return db.New(p.pool) }

// InTx runs fn inside a transaction, committing if it returns nil and rolling back otherwise.
//
// The Querier handed to fn is bound to the transaction: queries made through the handler's
// ordinary one would run on a different connection and outside it, which is the mistake this
// signature exists to make impossible.
func (p *Pool) InTx(ctx context.Context, fn func(q db.Querier) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	// Rollback after a successful Commit is a no-op that returns ErrTxClosed, so this is safe
	// unconditionally — and unconditional is the point: an early return inside fn must not be
	// able to leave a transaction open holding a pool connection.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(db.New(p.pool).WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
