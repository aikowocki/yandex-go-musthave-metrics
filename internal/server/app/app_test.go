package app

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	// Инициализируем zap для тестов
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func TestInitStorage_Memory(t *testing.T) {
	t.Run("memory storage without file backup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := &config.ServerConfig{
			DB:              &db.Config{DatabaseDSN: ""},
			FileStoragePath: "",
			Restore:         false,
			StoreInterval:   0,
		}

		storage, err := initStorage(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.NotNil(t, storage.repos)
		assert.Nil(t, storage.txManager, "memory storage should not have txManager")
		assert.Nil(t, storage.pinger, "memory storage should not have pinger")

		// Cleanup
		storage.closer()
	})

	t.Run("memory storage with file backup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		tmpFile := t.TempDir() + "/metrics.json"
		cfg := &config.ServerConfig{
			DB:              &db.Config{DatabaseDSN: ""},
			FileStoragePath: tmpFile,
			Restore:         false,
			StoreInterval:   1,
		}

		storage, err := initStorage(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.NotNil(t, storage.repos)
		assert.NotNil(t, storage.waitBackup, "should have waitBackup for async backup")

		// Останавливаем фоновую backup-горутину явно, до очистки t.TempDir().
		// Порядок важен: cancel → waitBackup → closer.
		cancel()
		if storage.waitBackup != nil {
			waitCtx, waitCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer waitCancel()
			storage.waitBackup(waitCtx)
		}
		storage.closer()
	})
}

func TestInitAudit(t *testing.T) {
	t.Run("no audit configured", func(t *testing.T) {
		cfg := &config.ServerConfig{
			AuditFile: "",
			AuditURL:  "",
		}

		publisher, closer, err := initAudit(cfg)
		require.NoError(t, err)
		assert.Nil(t, publisher, "should return nil publisher when no audit configured")
		assert.NotNil(t, closer, "closer should not be nil")

		// Cleanup should not panic
		closer(context.Background())
	})

	t.Run("file audit only", func(t *testing.T) {
		tmpFile := t.TempDir() + "/audit.jsonl"
		cfg := &config.ServerConfig{
			AuditFile: tmpFile,
			AuditURL:  "",
		}

		publisher, closer, err := initAudit(cfg)
		require.NoError(t, err)
		require.NotNil(t, publisher)

		// Cleanup
		closer(context.Background())
	})

	t.Run("HTTP audit only", func(t *testing.T) {
		cfg := &config.ServerConfig{
			AuditFile: "",
			AuditURL:  "http://localhost:9999/audit",
		}

		publisher, closer, err := initAudit(cfg)
		require.NoError(t, err)
		require.NotNil(t, publisher)

		// Cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		closer(ctx)
	})

	t.Run("both file and HTTP audit", func(t *testing.T) {
		tmpFile := t.TempDir() + "/audit.jsonl"
		cfg := &config.ServerConfig{
			AuditFile: tmpFile,
			AuditURL:  "http://localhost:9999/audit",
		}

		publisher, closer, err := initAudit(cfg)
		require.NoError(t, err)
		require.NotNil(t, publisher)

		// Cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		closer(ctx)
	})

	t.Run("invalid file path returns error", func(t *testing.T) {
		cfg := &config.ServerConfig{
			AuditFile: "/invalid/nonexistent/path/audit.jsonl",
			AuditURL:  "",
		}

		_, _, err := initAudit(cfg)
		assert.Error(t, err)
	})
}

func TestNewServerApp(t *testing.T) {
	t.Run("minimal configuration", func(t *testing.T) {
		// Используем свободный порт
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := listener.Addr().String()
		listener.Close()

		cfg := &config.ServerConfig{
			ServerAddress:   addr,
			Key:             "",
			FileStoragePath: "",
			Restore:         false,
			StoreInterval:   0,
			DB:              &db.Config{DatabaseDSN: ""},
			TrustedSubnet:   "",
			CryptoKey:       "",
			GRPCAddress:     "",
			AuditFile:       "",
			AuditURL:        "",
		}

		app, err := NewServerApp(context.Background(), cfg)
		require.NoError(t, err)
		require.NotNil(t, app)
		assert.NotNil(t, app.server)
		assert.Nil(t, app.gRPCServer, "gRPC should not be enabled")

		// Cleanup
		app.Close(context.Background())
	})

	t.Run("with trusted subnet", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := listener.Addr().String()
		listener.Close()

		cfg := &config.ServerConfig{
			ServerAddress: addr,
			TrustedSubnet: "10.0.0.0/8",
			DB:            &db.Config{DatabaseDSN: ""},
		}

		app, err := NewServerApp(context.Background(), cfg)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Cleanup
		app.Close(context.Background())
	})

	t.Run("invalid trusted subnet fails", func(t *testing.T) {
		cfg := &config.ServerConfig{
			ServerAddress: "127.0.0.1:8080",
			TrustedSubnet: "invalid-cidr",
			DB:            &db.Config{DatabaseDSN: ""},
		}

		_, err := NewServerApp(context.Background(), cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid trusted_subnet")
	})

	t.Run("with gRPC enabled", func(t *testing.T) {
		httpListener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		httpAddr := httpListener.Addr().String()
		httpListener.Close()

		grpcListener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		grpcAddr := grpcListener.Addr().String()
		grpcListener.Close()

		cfg := &config.ServerConfig{
			ServerAddress: httpAddr,
			GRPCAddress:   grpcAddr,
			DB:            &db.Config{DatabaseDSN: ""},
		}

		app, err := NewServerApp(context.Background(), cfg)
		require.NoError(t, err)
		require.NotNil(t, app)
		assert.NotNil(t, app.server)
		assert.NotNil(t, app.gRPCServer, "gRPC should be enabled")

		// Cleanup
		app.Close(context.Background())
	})

	t.Run("with audit configured", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := listener.Addr().String()
		listener.Close()

		tmpFile := t.TempDir() + "/audit.jsonl"

		cfg := &config.ServerConfig{
			ServerAddress: addr,
			AuditFile:     tmpFile,
			DB:            &db.Config{DatabaseDSN: ""},
		}

		app, err := NewServerApp(context.Background(), cfg)
		require.NoError(t, err)
		require.NotNil(t, app)

		// Cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		app.Close(ctx)
	})
}

func TestServerApp_Close(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	// Отменяемый контекст — чтобы фоновая backup-горутина остановилась
	// до очистки t.TempDir().
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.ServerConfig{
		ServerAddress:   addr,
		FileStoragePath: t.TempDir() + "/metrics.json",
		StoreInterval:   1,
		DB:              &db.Config{DatabaseDSN: ""},
	}

	app, err := NewServerApp(ctx, cfg)
	require.NoError(t, err)

	// Close должен завершиться без паники
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer closeCancel()

	assert.NotPanics(t, func() {
		app.Close(closeCtx)
	})
}
