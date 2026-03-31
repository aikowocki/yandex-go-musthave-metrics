package metric

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	integration = flag.Bool("integration", false, "run integration tests")
	testDB      *sql.DB
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !*integration {
		os.Exit(m.Run())
	}
	ctx := context.Background()

	dbName := "test"
	dbUser := "test"
	dbPassword := "test"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = database.RunMigrations(dsn, "file://../../../migrations")
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())

}

func TestPostgresStorage(t *testing.T) {
	if !*integration {
		t.Skip("skipping integration test, use -integration flag")
	}
	runStorageTests(t, NewPostgresStorage(testDB))
}
