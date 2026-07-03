package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerApp_Smoke(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &config.ServerConfig{
		ServerAddress:   "localhost:18080",
		FileStoragePath: "",
		Restore:         false,
		StoreInterval:   0,
		DB:              &db.Config{},
		PprofAddress:    "localhost:16060",
	}

	application, err := NewServerApp(ctx, cfg)
	require.NoError(t, err)

	go application.Run(ctx)

	baseURL := "http://localhost:18080"

	// Детерминированно дожидаемся готовности сервера вместо фиксированного sleep:
	// опрашиваем эндпоинт, пока он не начнёт принимать соединения.
	require.Eventually(t, func() bool {
		resp, getErr := http.Get(baseURL + "/")
		if getErr != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond, "server did not become ready in time")

	// Тест 1: POST /update — сохранить gauge через JSON
	body := `{"id":"cpu","type":"gauge","value":42.5}`
	resp, err := http.Post(baseURL+"/update", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	// Тест 2: POST /value — получить сохранённую метрику
	body = `{"id":"cpu","type":"gauge"}`
	resp, err = http.Post(baseURL+"/value", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var dto api.MetricDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&dto))
	_ = resp.Body.Close()
	assert.Equal(t, "cpu", dto.ID)
	require.NotNil(t, dto.Value)
	assert.Equal(t, 42.5, *dto.Value)

	// Тест 3: GET / — список метрик (HTML)
	resp, err = http.Get(baseURL + "/")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	// Тест 4: POST /updates — batch update
	batch := `[{"id":"mem","type":"gauge","value":128.5},{"id":"hits","type":"counter","delta":10}]`
	resp, err = http.Post(baseURL+"/updates", "application/json", strings.NewReader(batch))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	// Тест 5: Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	err = application.Shutdown(shutdownCtx)
	assert.NoError(t, err)
	application.Close(shutdownCtx)
}
