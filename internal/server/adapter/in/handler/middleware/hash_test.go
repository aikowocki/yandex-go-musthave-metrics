package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkghash "github.com/aikowocki/yandex-go-musthave-metrics/pkg/hash"
	"github.com/stretchr/testify/assert"
)

func TestWithHashValidation_EmptyKey_PassesThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	handler := WithHashValidation("")(inner)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestWithHashValidation_ValidHash(t *testing.T) {
	key := "secret"
	body := `{"id":"cpu","type":"gauge","value":1.0}`

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("accepted"))
	})
	handler := WithHashValidation(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(pkghash.HEADER, pkghash.ComputeHMAC(key, []byte(body)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Ответ должен содержать HMAC-заголовок.
	assert.NotEmpty(t, rec.Header().Get(pkghash.HEADER))
}

func TestWithHashValidation_InvalidHash(t *testing.T) {
	key := "secret"
	body := `{"id":"cpu","type":"gauge","value":1.0}`

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should not reach"))
	})
	handler := WithHashValidation(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(pkghash.HEADER, pkghash.ComputeHMAC("wrong-key", []byte(body)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWithHashValidation_NoHeader_PassesThrough(t *testing.T) {
	key := "secret"
	body := `{"data":"test"}`

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no hash required"))
	})
	handler := WithHashValidation(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	// Не устанавливаем HMAC-заголовок.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Ответ всё равно подписывается.
	assert.NotEmpty(t, rec.Header().Get(pkghash.HEADER))
}

func TestWithHashValidation_SignsResponse(t *testing.T) {
	key := "secret"
	responseBody := "hello"

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(responseBody))
	})
	handler := WithHashValidation(key)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Проверяем что ответ подписан корректно.
	expectedMAC := pkghash.ComputeHMAC(key, []byte(responseBody))
	assert.Equal(t, expectedMAC, rec.Header().Get(pkghash.HEADER))
}
