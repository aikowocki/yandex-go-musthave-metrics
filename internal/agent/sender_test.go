package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestClient_SendMetric_Success(t *testing.T) {
	// Создаём фейковый HTTP-сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Эта функция вызывается когда клиент делает запрос

		// Проверяем что метод POST
		assert.Equal(t, "POST", r.Method)

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
	err := client.SendMetric(model.MetricTypeGauge, "test", "3.14")

	// Проверяем что ошибки нет
	assert.NoError(t, err)
}

func TestClient_SendMetric_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetric(model.MetricTypeGauge, "test", "3.14")

	assert.Error(t, err)
}

func TestClient_SendMetric_InvalidURL(t *testing.T) {
	client := NewClient("ht!tp://invalid") // невалидный URL
	err := client.SendMetric(model.MetricTypeGauge, "test", "3.14")
	assert.Error(t, err)
}

func TestReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage := NewLocalStorage()
	storage.SetGauge("test", 3.14)
	storage.AddCounter("count", 5)

	client := NewClient(server.URL)
	Report(storage, client)

	// Проверяем что счётчик сброшен
	_, ok := storage.GetCounter("count")
	assert.False(t, ok, "counter should be reset after report")
}

func TestClient_SendMetric_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // всегда ошибка
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetric(model.MetricTypeGauge, "test", "3.14")

	assert.Error(t, err)

}

func TestClient_SendMetrics_Success(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetrics([]model.MetricDTO{
		{ID: "cpu", MType: "gauge", Value: func() *float64 { v := 3.14; return &v }()},
		{ID: "hits", MType: "counter", Delta: func() *int64 { v := int64(5); return &v }()},
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
	err := client.SendMetrics([]model.MetricDTO{
		{ID: "cpu", MType: "gauge", Value: func() *float64 { v := 1.0; return &v }()},
	})

	assert.Error(t, err)
}

func TestReportBatch_SendsToUpdatesEndpoint(t *testing.T) {
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
	ReportBatch(storage, client)

	assert.Equal(t, "/updates", capturedPath)
}

func TestReportBatch_EmptyStorage_NoRequest(t *testing.T) {
	requestMade := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage := NewLocalStorage() // пустое хранилище
	client := NewClient(server.URL)
	ReportBatch(storage, client)

	assert.False(t, requestMade, "should not send request for empty storage")
}
