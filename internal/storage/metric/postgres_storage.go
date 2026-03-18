package metric

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64

	q := `SELECT value FROM gauges WHERE name = $1`

	err := s.db.QueryRowContext(ctx, q, name).Scan(&value)

	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}

	if err != nil {
		return 0, err
	}

	return value, nil
}

func (s *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	q := `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = $2
	`
	_, err := s.db.ExecContext(ctx, q, name, value)

	if err != nil {
		return 0, err
	}

	return value, nil

}

func (s *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64

	q := "SELECT value FROM counters WHERE name = $1"

	err := s.db.QueryRowContext(ctx, q, name).Scan(&value)

	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}

	if err != nil {
		return 0, err
	}

	return value, nil
}

func (s *PostgresStorage) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	q := `
		INSERT INTO counters(name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counters.value + $2
		RETURNING value
	`

	var newValue int64
	err := s.db.QueryRowContext(ctx, q, name, value).Scan(&newValue)
	return newValue, err

}

func getAll[T float64 | int64](ctx context.Context, db *sql.DB, query string) (map[string]T, error) {
	result := make(map[string]T)
	rows, err := db.QueryContext(ctx, query)
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

func (s *PostgresStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return getAll[float64](ctx, s.db, `SELECT name, value FROM gauges`)
}

func (s *PostgresStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return getAll[int64](ctx, s.db, `SELECT name, value FROM counters`)
}

func (s *PostgresStorage) UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			zap.S().Warnw("failed to rollback transaction", "error", err)
		}
	}()

	if err := insertGaugesBatch(ctx, tx, gauges); err != nil {
		return err
	}

	if err := insertCountersBatch(ctx, tx, counters); err != nil {
		return err
	}

	return tx.Commit()
}

func insertBatch[T float64 | int64](ctx context.Context, tx *sql.Tx, data map[string]T, query string) error {
	if len(data) == 0 {
		return nil
	}

	names := make([]string, 0, len(data))
	values := make([]T, 0, len(data))

	for name, value := range data {
		names = append(names, name)
		values = append(values, value)
	}

	_, err := tx.ExecContext(ctx, query, pgtype.FlatArray[string](names), pgtype.FlatArray[T](values))
	if err != nil {
		return err
	}
	return nil
}

func insertCountersBatch(ctx context.Context, tx *sql.Tx, counters map[string]int64) error {
	query := `
		INSERT INTO counters (name, value)
		SELECT * FROM UNNEST($1::text[], $2::int8[])
		ON CONFLICT (name) DO UPDATE SET value = counters.value +  EXCLUDED.value
	`
	return insertBatch(ctx, tx, counters, query)
}

func insertGaugesBatch(ctx context.Context, tx *sql.Tx, gauges map[string]float64) error {
	query := `
		INSERT INTO gauges (name, value)
		SELECT * FROM UNNEST($1::text[], $2::float8[])
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`
	return insertBatch(ctx, tx, gauges, query)
}
