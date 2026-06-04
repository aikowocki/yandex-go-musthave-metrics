package middleware

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
)

func BenchmarkGzipCompression_Response(b *testing.B) {
	payload := `{"id":"cpu","type":"gauge","value":3.14}`
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
		w.Write([]byte(payload))
	})
	handler := WithGzipCompression()(inner)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(constants.HeaderAcceptEncoding, EncodingGzip)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkGzipDecompression_Request(b *testing.B) {
	payload := `{"id":"cpu","type":"gauge","value":3.14}`
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	gz.Write([]byte(payload))
	gz.Close()
	compressedBytes := compressed.Bytes()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	handler := WithGzipCompression()(inner)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(compressedBytes))
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkHashValidation(b *testing.B) {
	key := "test-secret-key"
	payload := `{"id":"cpu","type":"gauge","value":3.14}`

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	handler := WithHashValidation(key)(inner)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(payload))
		// без хеш-заголовка — просто проверяем подпись ответа
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkGzipCompression_LargePayload(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 40; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"id":"metric_`)
		sb.WriteString(strings.Repeat("x", 10))
		sb.WriteString(`","type":"gauge","value":3.14}`)
	}
	sb.WriteString("]")
	payload := sb.String()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
		w.Write([]byte(payload))
	})
	handler := WithGzipCompression()(inner)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(constants.HeaderAcceptEncoding, EncodingGzip)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
