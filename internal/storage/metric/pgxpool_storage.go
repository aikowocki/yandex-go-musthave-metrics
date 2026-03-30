package metric

import (
	"context"
	"errors"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/retry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PgxPoolStorage struct {
	pool *pgxpool.Pool
}

func NewPgxPoolStorage(pool *pgxpool.Pool) *PgxPoolStorage {
	return &PgxPoolStorage{pool: pool}
}

func (s *PgxPoolStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64

	q := `SELECT value FROM gauges WHERE name = $1`

	err := s.pool.QueryRow(ctx, q, name).Scan(&value)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}

	if err != nil {
		return 0, err
	}

	return value, nil
}

func (s *PgxPoolStorage) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	q := `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = $2, updated_at = NOW()
	`
	_, err := s.pool.Exec(ctx, q, name, value)

	if err != nil {
		return 0, err
	}

	return value, nil

}

func (s *PgxPoolStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64

	q := "SELECT value FROM counters WHERE name = $1"

	err := s.pool.QueryRow(ctx, q, name).Scan(&value)

	if err == pgx.ErrNoRows {
		return 0, ErrNotFound
	}

	if err != nil {
		return 0, err
	}

	return value, nil
}

func (s *PgxPoolStorage) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	q := `
		INSERT INTO counters(name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counters.value + $2, updated_at = NOW()
		RETURNING value
	`

	var newValue int64
	err := s.pool.QueryRow(ctx, q, name, value).Scan(&newValue)
	return newValue, err

}

func pgxGetAll[T float64 | int64](ctx context.Context, pool *pgxpool.Pool, query string) (map[string]T, error) {
	result := make(map[string]T)
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value T

		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}

		result[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *PgxPoolStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return pgxGetAll[float64](ctx, s.pool, `SELECT name, value FROM gauges`)
}

func (s *PgxPoolStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return pgxGetAll[int64](ctx, s.pool, `SELECT name, value FROM counters`)
}

func (s *PgxPoolStorage) UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	operation := func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}

		defer func() {
			if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				zap.S().Warnw("failed to rollback transaction", "error", err)
			}
		}()

		if err := pgxInsertGaugesBatch(ctx, tx, gauges); err != nil {
			return err
		}

		if err := pgxInsertCountersBatch(ctx, tx, counters); err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	return retry.Do(ctx, operation, retry.WithRetryIf(isPgConnectionError))
}

func pgxInsertBatch[T float64 | int64](ctx context.Context, tx pgx.Tx, data map[string]T, query string) error {
	if len(data) == 0 {
		return nil
	}

	names := make([]string, 0, len(data))
	values := make([]T, 0, len(data))

	for name, value := range data {
		names = append(names, name)
		values = append(values, value)
	}

	_, err := tx.Exec(ctx, query, pgtype.FlatArray[string](names), pgtype.FlatArray[T](values))
	if err != nil {
		return err
	}
	return nil
}

func pgxInsertCountersBatch(ctx context.Context, tx pgx.Tx, counters map[string]int64) error {
	query := `
		INSERT INTO counters (name, value)
		SELECT * FROM UNNEST($1::text[], $2::int8[])
		ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value, updated_at = NOW()
	`
	return pgxInsertBatch(ctx, tx, counters, query)
}

func pgxInsertGaugesBatch(ctx context.Context, tx pgx.Tx, gauges map[string]float64) error {
	query := `
		INSERT INTO gauges (name, value)
		SELECT * FROM UNNEST($1::text[], $2::float8[])
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`
	return pgxInsertBatch(ctx, tx, gauges, query)
}
