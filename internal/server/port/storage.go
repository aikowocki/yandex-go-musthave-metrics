package port

import (
	"context"
)

type Repositories interface {
	MetricRepo() MetricRepository
}
type PGStorage interface {
	Repositories
	DB() DB
	TxManager() TxManager
}

// опционально — только postgres реализует
type Pinger interface {
	Ping(ctx context.Context) error
}
