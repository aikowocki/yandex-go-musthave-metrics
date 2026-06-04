package integration

import (
	"context"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/postgres"
	postgrespgx "github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/postgres/pgx"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(tb testing.TB) string {
	tb.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		tb.Fatal(err)
	}

	if err := postgres.RunMigrations(dsn); err != nil {
		tb.Fatal(err)
	}
	return dsn
}

func setupPGXStorage(tb testing.TB) port.PGStorage {
	tb.Helper()
	dsn := setupTestDB(tb)
	storage, err := postgrespgx.NewStorage(context.Background(), dsn)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		storage.DB().Close()
	})
	return storage
}
