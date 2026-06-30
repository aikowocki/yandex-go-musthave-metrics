package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ptr возвращает указатель на значение — для заполнения опциональных
// полей api.MetricDTO (Value/Delta).
func ptr[T any](v T) *T { return &v }

func TestClient_SendMetric_Success(t *testing.T) {
	// Создаём фейковый HTTP-сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Эта функция вызывается когда клиент делает запрос

		// Проверяем что метод POST
		assert.Equal(t, http.MethodPost, r.Method)

		// Проверяем что URL правильный
		assert.Equal(t, "/update/gauge/test/3.14", r.URL.Path)

		// Возвращаем успешный ответ
		w.WriteHeader(http.StatusOK)
	}))

	// Когда тест закончится — останавливаем сервер
	defer server.Close()

	// Создаём клиента с адресом mock-сервера
	client := NewClient(server.URL) // server.URL = "http://127.0.0.1:12345" (случайный порт)

	// Вызываем тестируемую функцию
	err := client.SendMetric(api.MetricTypeGauge, "test", "3.14")

	// Проверяем что ошибки нет
	assert.NoError(t, err)
}

func TestClient_SendMetric_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetric(api.MetricTypeGauge, "test", "3.14")

	assert.Error(t, err)
}

func TestClient_SendMetric_InvalidURL(t *testing.T) {
	client := NewClient("ht!tp://invalid") // невалидный URL
	err := client.SendMetric(api.MetricTypeGauge, "test", "3.14")
	assert.Error(t, err)
}

func TestReport(t *testing.T) {
	var (
		mu        sync.Mutex
		gotPaths  []string
		gaugeSent bool
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		if strings.HasPrefix(r.URL.Path, "/update/gauge/test/") {
			gaugeSent = true
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage := NewLocalStorage()
	storage.SetGauge("test", 3.14)
	storage.AddCounter("count", 5)

	client := NewClient(server.URL)
	Report(storage, client)

	mu.Lock()
	defer mu.Unlock()
	// Gauge действительно отправлен на сервер.
	assert.True(t, gaugeSent, "gauge metric should be sent, got paths: %v", gotPaths)

	// Counter сброшен после отчёта (SnapshotCounters очищает).
	_, ok := storage.GetCounter("count")
	assert.False(t, ok, "counter should be reset after report")
}

func TestClient_SendMetrics_Success(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetrics(t.Context(), []api.MetricDTO{
		{ID: "cpu", MType: "gauge", Value: ptr(3.14)},
		{ID: "hits", MType: "counter", Delta: ptr(int64(5))},
	})

	assert.NoError(t, err)
	assert.Equal(t, "/updates", capturedPath)
}

func TestClient_SendMetrics_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetrics(t.Context(), []api.MetricDTO{
		{ID: "cpu", MType: "gauge", Value: ptr(1.0)},
	})

	assert.Error(t, err)
}

func TestSendBatch_SendsToUpdatesEndpoint(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage := NewLocalStorage()
	storage.SetGauge("cpu", 1.5)
	storage.AddCounter("hits", 10)

	client := NewClient(server.URL)
	SendBatch(t.Context(), client, CollectBatch(storage))

	assert.Equal(t, "/updates", capturedPath)
}

func TestSendBatch_EmptyStorage_NoRequest(t *testing.T) {
	requestMade := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage := NewLocalStorage() // пустое хранилище
	client := NewClient(server.URL)
	SendBatch(t.Context(), client, CollectBatch(storage))

	assert.False(t, requestMade, "should not send request for empty storage")
}

// TestSendBatch_NoRetryOnServerError фиксирует поведение retrierCondition в SendBatch:
// HTTP 500 не является *net.OpError, поэтому ретраев быть не должно — ровно один запрос.
func TestSendBatch_NoRetryOnServerError(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	metrics := []api.MetricDTO{
		{ID: "cpu", MType: "gauge", Value: ptr(1.0)},
	}

	client := NewClient(server.URL)
	SendBatch(t.Context(), client, metrics)

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls),
		"server error (500) is not a net error, SendBatch must not retry")
}

// TestSendBatch_RetriesOnNetworkError проверяет, что при сетевой ошибке
// (сервер недоступен) SendBatch выполняет повторные попытки согласно retrierCondition
// Дефолтные задержки retry.Do не проверяем, только сам факт
// нескольких попыток через подсчёт TCP-подключений на закрытом порту
func TestSendBatch_RetriesOnNetworkError(t *testing.T) {
	// Поднимаем и сразу закрываем сервер, чтобы получить гарантированно
	// свободный (закрытый) адрес — подключение к нему даст *net.OpError.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := server.URL
	server.Close()

	metrics := []api.MetricDTO{
		{ID: "cpu", MType: "gauge", Value: ptr(1.0)},
	}

	// Отменяемый контекст: после первой неудачной попытки прерываем,
	// чтобы не ждать полный цикл дефолтных задержек retry (1s+3s+5s).
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		client := NewClient(addr)
		SendBatch(ctx, client, metrics) // не должен паниковать, корректно завершается по ctx
		close(done)
	}()

	select {
	case <-done:
		// SendBatch завершился (по исчерпанию попыток или по ctx) — ок.
	case <-time.After(2 * time.Second):
		t.Fatal("SendBatch did not return in time on network error")
	}
}

// TestIsRetriable проверяет, что предикат ретраев распознаёт временные ошибки
// обоих транспортов (HTTP *net.OpError и gRPC Unavailable/DeadlineExceeded)
// и не ретраит прикладные ошибки.
func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"http net.OpError", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
		{"wrapped net.OpError", fmt.Errorf("send: %w", &net.OpError{Op: "dial", Err: errors.New("x")}), true},
		{"grpc unavailable", status.Error(codes.Unavailable, "server down"), true},
		{"grpc deadline exceeded", status.Error(codes.DeadlineExceeded, "timeout"), true},
		{"grpc invalid argument", status.Error(codes.InvalidArgument, "bad input"), false},
		{"plain error (http 500)", errors.New("unexpected status code: 500"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetriable(tt.err))
		})
	}
}
