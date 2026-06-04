package postgres_pgx

import (
	"context"
	"errors"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type baseRepo struct {
	txManager *TxManager
}

func (r *baseRepo) db(ctx context.Context) querier {
	return r.txManager.getQuerier(ctx)
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func handleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ErrMetricNotFound
	}
	return err
}
