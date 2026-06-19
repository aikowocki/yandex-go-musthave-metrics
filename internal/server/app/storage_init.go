package app

import (
	"context"
	"fmt"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/memory"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/postgres"
	postgres_pgx "github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/postgres/pgx"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"go.uber.org/zap"
)

type storageResult struct {
	repos      port.Repositories
	txManager  port.TxManager // nil для memory
	closer     func()
	waitBackup func(context.Context) // nil, если фонового бэкапа нет (PG / sync / без файла)
	pinger     port.Pinger           // nil для memory
}

func initStorage(ctx context.Context, cfg *config.ServerConfig) (*storageResult, error) {
	if cfg.DB.DatabaseDSN != "" {
		var pgStorage port.PGStorage
		var err error

		switch cfg.DB.PostgresDriver {
		//case "gorm":
		//	dbStorage, err = postgres_gorm.NewStorage(cfg.DB.DatabaseDSN)
		//	zap.S().Infow("Used gorm driver")
		default:
			zap.S().Infow("Used pgx driver")
			pgStorage, err = postgres_pgx.NewStorage(ctx, cfg.DB.DatabaseDSN)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		if err := postgres.RunMigrations(cfg.DB.DatabaseDSN); err != nil {
			pgStorage.DB().Close()
			return nil, fmt.Errorf("failed to run migration: %w", err)
		}
		return &storageResult{
			repos:     pgStorage,
			txManager: pgStorage.TxManager(),
			closer:    pgStorage.DB().Close,
			pinger:    pgStorage.DB(),
		}, nil
	}

	memStorage := memory.NewMemoryStorage(ctx, cfg.FileStoragePath, cfg.Restore, time.Duration(cfg.StoreInterval))
	return &storageResult{
		repos:      memStorage,
		closer:     func() {},
		waitBackup: memStorage.WaitBackup,
	}, nil
}
