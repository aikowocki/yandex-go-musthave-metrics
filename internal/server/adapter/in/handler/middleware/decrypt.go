package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	pkgcrypto "github.com/aikowocki/yandex-go-musthave-metrics/pkg/crypto"
	"go.uber.org/zap"
)

// WithDecryption middleware для расшифровки входящих запросов с помощью RSA приватного ключа.
func WithDecryption(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privateKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Расшифровываем только если агент отправил заголовок шифрования.
			if r.Header.Get("X-Encrypted") == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "не удалось прочитать тело запроса", http.StatusInternalServerError)
				return
			}

			decrypted, err := pkgcrypto.Decrypt(privateKey, body)
			if err != nil {
				zap.S().Warnw("не удалось расшифровать тело запроса", "error", err, "path", r.URL.Path)
				http.Error(w, "ошибка расшифровки", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decrypted))
			r.ContentLength = int64(len(decrypted))
			r.Header.Del("X-Encrypted")

			next.ServeHTTP(w, r)
		})
	}
}
