package postgres_pgx

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type MetricRepo struct {
	baseRepo
}

func NewMetricRepo(txManager *TxManager) *MetricRepo {
	return &MetricRepo{baseRepo: baseRepo{txManager: txManager}}
}

func (r *MetricRepo) GetGauge(ctx context.Context, name string) (*entity.GaugeMetric, error) {
	gauge := &entity.GaugeMetric{}
	err := r.db(ctx).QueryRow(ctx, `
		SELECT value, created_at, updated_at FROM gauges WHERE name = $1
	`, name).Scan(&gauge.Value, &gauge.CreatedAt, &gauge.UpdatedAt)

	if err != nil {
		return nil, handleError(err)
	}
	return gauge, nil
}

func (r *MetricRepo) GetCounter(ctx context.Context, name string) (*entity.CounterMetric, error) {
	counter := &entity.CounterMetric{}
	err := r.db(ctx).QueryRow(ctx, `
		SELECT value, created_at, updated_at FROM counters WHERE name = $1
	`, name).Scan(&counter.Value, &counter.CreatedAt, &counter.UpdatedAt)

	if err != nil {
		return nil, handleError(err)
	}
	return counter, nil
}

func (r *MetricRepo) CreateOrUpdateGauge(ctx context.Context, gauge *entity.GaugeMetric) error {
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = $2, updated_at = NOW()
		RETURNING created_at, updated_at
	`, gauge.Name, gauge.Value).Scan(&gauge.CreatedAt, &gauge.UpdatedAt)
	return handleError(err)
}

func (r *MetricRepo) CreateOrUpdateCounter(ctx context.Context, counter *entity.CounterMetric) error {
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counters.value + $2, updated_at = NOW()
		RETURNING value, created_at, updated_at
	`, counter.Name, counter.Value).Scan(&counter.Value, &counter.CreatedAt, &counter.UpdatedAt)
	return handleError(err)
}

func (r *MetricRepo) GetGauges(ctx context.Context) ([]*entity.GaugeMetric, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT name, value,created_at, updated_at FROM gauges`)
	if err != nil {
		return nil, handleError(err)
	}
	defer rows.Close()

	var result []*entity.GaugeMetric
	for rows.Next() {
		g := &entity.GaugeMetric{}
		if err := rows.Scan(&g.Name, &g.Value, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, handleError(err)
		}
		result = append(result, g)
	}
	if err := rows.Err(); err != nil {
		return nil, handleError(err)
	}
	return result, nil
}

func (r *MetricRepo) GetCounters(ctx context.Context) ([]*entity.CounterMetric, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT name, value,created_at, updated_at FROM counters`)
	if err != nil {
		return nil, handleError(err)
	}
	defer rows.Close()

	var result []*entity.CounterMetric
	for rows.Next() {
		c := &entity.CounterMetric{}
		if err := rows.Scan(&c.Name, &c.Value, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, handleError(err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, handleError(err)
	}
	return result, nil
}

func (r *MetricRepo) GetAll(ctx context.Context) ([]entity.Metric, error) {
	gauges, err := r.GetGauges(ctx)
	if err != nil {
		return nil, err
	}

	counters, err := r.GetCounters(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]entity.Metric, 0, len(gauges)+len(counters))
	for _, g := range gauges {
		result = append(result, g)
	}
	for _, c := range counters {
		result = append(result, c)
	}
	return result, nil
}

func (r *MetricRepo) SaveBatch(ctx context.Context, metrics []entity.Metric) error {
	return r.txManager.Do(ctx, func(ctx context.Context) error {
		gauges := make(map[string]float64)
		counters := make(map[string]int64)

		for _, m := range metrics {
			switch v := m.(type) {
			case *entity.GaugeMetric:
				gauges[v.Name] = v.Value
			case *entity.CounterMetric:
				counters[v.Name] += v.Value
			default:
				return fmt.Errorf("unknown metric type %T", m)
			}
		}

		if err := insertGaugesBatch(ctx, r.db(ctx), gauges); err != nil {
			return err
		}
		return insertCountersBatch(ctx, r.db(ctx), counters)
	})
}

func insertGaugesBatch(ctx context.Context, q querier, gauges map[string]float64) error {
	if len(gauges) == 0 {
		return nil
	}
	names := make([]string, 0, len(gauges))
	values := make([]float64, 0, len(gauges))
	for name, value := range gauges {
		names = append(names, name)
		values = append(values, value)
	}
	_, err := q.Exec(ctx, `
        INSERT INTO gauges (name, value)
        SELECT * FROM UNNEST($1::text[], $2::float8[])
        ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
    `, names, values)
	return err
}

func insertCountersBatch(ctx context.Context, q querier, counters map[string]int64) error {
	if len(counters) == 0 {
		return nil
	}
	names := make([]string, 0, len(counters))
	values := make([]int64, 0, len(counters))
	for name, value := range counters {
		names = append(names, name)
		values = append(values, value)
	}
	_, err := q.Exec(ctx, `
        INSERT INTO counters (name, value)
        SELECT * FROM UNNEST($1::text[], $2::int8[])
        ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value, updated_at = NOW()
    `, names, values)
	return err
}
