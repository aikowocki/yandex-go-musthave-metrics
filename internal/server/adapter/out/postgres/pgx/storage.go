package postgres_pgx

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
)

type Storage struct {
	db         *DB
	txManager  *TxManager
	metricRepo *MetricRepo
}

func NewStorage(ctx context.Context, dsn string) (*Storage, error) {
	db, err := NewPool(ctx, dsn)
	if err != nil {
		return nil, err
	}

	txm := NewTxManager(db)
	return &Storage{
		db:         db,
		txManager:  txm,
		metricRepo: NewMetricRepo(txm),
	}, nil
}

func (s *Storage) DB() port.DB {
	return s.db
}

func (s *Storage) TxManager() port.TxManager {
	return s.txManager
}

func (s *Storage) MetricRepo() port.MetricRepository {
	return s.metricRepo
}
