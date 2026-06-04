package postgres

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type MigrationLogger struct {
	logger *zap.SugaredLogger
}

func (l *MigrationLogger) Printf(format string, v ...interface{}) {
	l.logger.Infof(format, v...)
}

func (l *MigrationLogger) Verbose() bool {
	return true
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close() //nolint:errcheck

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	migrateInst, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}
	defer func() {
		if sourceErr, dbErr := migrateInst.Close(); sourceErr != nil || dbErr != nil {
			zap.S().Warnw("failed to close migrate", "sourceErr", sourceErr, "dbErr", dbErr)
		}
	}()

	migrateInst.Log = &MigrationLogger{
		logger: zap.S(),
	}

	if err := migrateInst.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
