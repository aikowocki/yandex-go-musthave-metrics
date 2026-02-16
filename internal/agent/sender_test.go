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

func TestClient_SendMetric_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError) // первые 2 попытки — ошибка
			return
		}
		w.WriteHeader(http.StatusOK) // 3-я попытка — успех
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.SendMetric(model.MetricTypeGauge, "test", "3.14")

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "Should retry 3 times")
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
