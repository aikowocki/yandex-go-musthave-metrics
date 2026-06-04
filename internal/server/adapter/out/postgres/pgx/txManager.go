package postgres_pgx

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

type TxManager struct {
	pool *DB
}

func NewTxManager(pool *DB) *TxManager {
	return &TxManager{pool: pool}
}

func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// если уже в транзакции — не создаём новую
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ctxWithTx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctxWithTx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (m *TxManager) getQuerier(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return m.pool
}
