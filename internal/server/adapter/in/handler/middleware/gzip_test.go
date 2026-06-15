package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	"github.com/stretchr/testify/assert"
)

func TestWithGzip_CompressResponse(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		expectGzip     bool
	}{
		{
			name:           "with gzip",
			acceptEncoding: EncodingGzip,
			expectGzip:     true,
		}, {
			name:           "without gzip",
			acceptEncoding: "",
			expectGzip:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			})

			wrapped := WithGzipCompression()(handler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set(constants.HeaderAcceptEncoding, EncodingGzip)
			}
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			res := rec.Result()
			defer func() { _ = res.Body.Close() }()

			assert.Equal(t, http.StatusOK, res.StatusCode)
			contentEncoding := res.Header.Get(constants.HeaderContentEncoding)
			if tt.expectGzip {
				assert.Equal(t, EncodingGzip, contentEncoding)
			} else {
				assert.Empty(t, contentEncoding)
			}
			assert.NotEmpty(t, rec.Body.Bytes())
		})
	}

}

func TestWithGzip_DecompressRequest(t *testing.T) {
	var receivedBody []byte
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	})

	wrapped := WithGzipCompression()(inner)

	// Подготавливаем gzip-сжатое тело.
	original := `{"id":"cpu","type":"gauge","value":3.14}`
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(original))
	assert.NoError(t, err)
	assert.NoError(t, gz.Close())

	req := httptest.NewRequest(http.MethodPost, "/update", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, original, string(receivedBody))
}

func TestWithGzip_InvalidGzipBody(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := WithGzipCompression()(inner)

	// Отправляем невалидный gzip.
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("not gzip at all"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
