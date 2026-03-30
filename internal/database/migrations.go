package database

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
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

func RunMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	migrateInst, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return err
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

func RunMigrationsFromPool(pool *pgxpool.Pool, migrationsPath string) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return RunMigrations(db, migrationsPath)
}
