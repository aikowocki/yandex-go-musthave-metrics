package db

// Config хранит настройки подключения к базе данных.
// Флаги и переменные окружения привязываются родительским конфигом сервера.
type Config struct {
	DatabaseDSN    string `env:"DATABASE_DSN"`
	PostgresDriver string `env:"POSTGRES_DRIVER"`
}
