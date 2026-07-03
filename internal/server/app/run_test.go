package app

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config/db"
	"github.com/stretchr/testify/require"
)

// freeAddr резервирует свободный TCP-порт и тут же его освобождает,
// возвращая адрес для последующего биндинга сервером.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// TestServerApp_RunAndShutdown поднимает приложение целиком (HTTP + gRPC),
// дожидается готовности и проверяет graceful shutdown: Run должен завершиться
// без падения после Shutdown.
func TestServerApp_RunAndShutdown(t *testing.T) {
	httpAddr := freeAddr(t)
	grpcAddr := freeAddr(t)

	cfg := &config.ServerConfig{
		ServerAddress: httpAddr,
		GRPCAddress:   grpcAddr,
		DB:            &db.Config{DatabaseDSN: ""},
	}

	app, err := NewServerApp(context.Background(), cfg)
	require.NoError(t, err)

	runDone := make(chan struct{})
	go func() {
		app.Run(context.Background())
		close(runDone)
	}()

	// Дожидаемся готовности HTTP-сервера.
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", httpAddr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 20*time.Millisecond, "HTTP server should start")

	// gRPC-порт тоже должен слушать.
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", grpcAddr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 20*time.Millisecond, "gRPC server should start")

	// Реальный HTTP-запрос на корень (List) — проверяем, что сервер обслуживает.
	resp, err := http.Get("http://" + httpAddr + "/")
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, app.Shutdown(ctx))

	app.Close(context.Background())

	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Shutdown")
	}
}
