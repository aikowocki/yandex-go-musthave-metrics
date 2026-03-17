package metric

import (
	"context"
	"database/sql"
)

type MetricPostgresStorage struct {
	db *sql.DB
}

func NewMetricPostgresStorage(db *sql.DB) *MetricPostgresStorage {
	return &MetricPostgresStorage{db: db}
}

func (s *MetricPostgresStorage) GetGauge(ctx context.Context, name string) (float64, error) {
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

func (s *MetricPostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
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

func (s *MetricPostgresStorage) GetCounter(ctx context.Context, name string) (int64, error) {
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

func (s *MetricPostgresStorage) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
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

func (s *MetricPostgresStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	gauges := make(map[string]float64)

	q := `SELECT name, value FROM gauges`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64

		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}

		gauges[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return gauges, nil
}

func (s *MetricPostgresStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	counters := make(map[string]int64)

	q := `SELECT name, value FROM counters`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64

		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		counters[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return counters, nil
}
