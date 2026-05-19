package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	pkghash "github.com/aikowocki/yandex-go-musthave-metrics/pkg/hash"
	"go.uber.org/zap"
)

type hashResponseWriter struct {
	http.ResponseWriter
	buf bytes.Buffer
}

func (w *hashResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func WithHashValidation(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Проверка хеша запроса
			if headerHash := r.Header.Get(pkghash.HEADER); headerHash != "" {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
						http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
						return
					}
					http.Error(w, "failed to read body", http.StatusInternalServerError)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))

				if !pkghash.ValidateHMAC(key, body, headerHash) {
					zap.S().Warnw("hash mismatch", "path", r.URL.Path)
					http.Error(w, "hash mismatch", http.StatusBadRequest)
					return
				}
				zap.S().Debugw("hash validated successfully")
			}

			// Подпись ответа
			hw := &hashResponseWriter{ResponseWriter: w}
			next.ServeHTTP(hw, r)

			responseBody := hw.buf.Bytes()
			w.Header().Set(pkghash.HEADER, pkghash.ComputeHMAC(key, responseBody))
			w.Write(responseBody)
		})
	}
}
