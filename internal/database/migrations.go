package database

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	migrateInst.Log = &MigrationLogger{
		logger: zap.S(),
	}

	if err := migrateInst.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
