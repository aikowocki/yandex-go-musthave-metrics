package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestLogger подменяет глобальный zap dev-логгером на время теста.
func setupTestLogger(t *testing.T) {
	t.Helper()
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	zap.ReplaceGlobals(logger)
	t.Cleanup(func() { _ = logger.Sync() })
}

func TestWithLogging(t *testing.T) {
	setupTestLogger(t)

	t.Run("logs successful request", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hello, World!"))
		})

		middleware := WithLogging()(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, World!", rec.Body.String())
	})

	t.Run("logs error status", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		})

		middleware := WithLogging()(handler)

		req := httptest.NewRequest(http.MethodGet, "/missing", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "Not Found", rec.Body.String())
	})

	t.Run("tracks response size", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("123456789")) // 9 bytes
		})

		middleware := WithLogging()(handler)

		req := httptest.NewRequest(http.MethodPost, "/data", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		assert.Equal(t, 9, rec.Body.Len())
	})

	t.Run("handles empty response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		middleware := WithLogging()(handler)

		req := httptest.NewRequest(http.MethodDelete, "/resource", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String())
	})
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	t.Run("writes data and tracks size", func(t *testing.T) {
		rec := httptest.NewRecorder()
		responseData := &responseData{}
		lw := &loggingResponseWriter{
			ResponseWriter: rec,
			ResponseData:   responseData,
		}

		n, err := lw.Write([]byte("hello"))
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, 5, responseData.Size)
		assert.Equal(t, http.StatusOK, responseData.Status)
	})

	t.Run("accumulates size on multiple writes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		responseData := &responseData{}
		lw := &loggingResponseWriter{
			ResponseWriter: rec,
			ResponseData:   responseData,
		}

		_, _ = lw.Write([]byte("hello"))
		_, _ = lw.Write([]byte(" "))
		_, _ = lw.Write([]byte("world"))

		assert.Equal(t, 11, responseData.Size)
		assert.Equal(t, "hello world", rec.Body.String())
	})
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	responseData := &responseData{}
	lw := &loggingResponseWriter{
		ResponseWriter: rec,
		ResponseData:   responseData,
	}

	lw.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, responseData.Status)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestWithLogging_MultipleRequests(t *testing.T) {
	setupTestLogger(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad Request"))
		}
	})

	middleware := WithLogging()(handler)

	// Request 1
	req1 := httptest.NewRequest(http.MethodGet, "/success", nil)
	rec1 := httptest.NewRecorder()
	middleware.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Request 2
	req2 := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec2 := httptest.NewRecorder()
	middleware.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)
}
