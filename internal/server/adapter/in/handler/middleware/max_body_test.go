package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithMaxBodySize(t *testing.T) {
	const limit int64 = 10

	// Хендлер пытается вычитать всё тело; при превышении лимита
	// MaxBytesReader вернёт ошибку чтения.
	readAllHandler := func(readErr *error) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			*readErr = err
			w.WriteHeader(http.StatusOK)
		})
	}

	t.Run("body within limit reads successfully", func(t *testing.T) {
		var readErr error
		h := WithMaxBodySize(limit)(readAllHandler(&readErr))

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("short"))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.NoError(t, readErr, "body within limit should read without error")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("body over limit triggers read error", func(t *testing.T) {
		var readErr error
		h := WithMaxBodySize(limit)(readAllHandler(&readErr))

		// 20 байт при лимите 10 — чтение должно прерваться.
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 20)))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		require.Error(t, readErr, "body exceeding limit must produce read error")
		assert.Contains(t, readErr.Error(), "too large")
	})

	t.Run("nil body passes through", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		h := WithMaxBodySize(limit)(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Body = nil
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.True(t, called, "next handler should be called even with nil body")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
