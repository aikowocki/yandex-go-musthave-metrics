package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkgcrypto "github.com/aikowocki/yandex-go-musthave-metrics/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func TestWithDecryption_NilKey_PassesThrough(t *testing.T) {
	body := `{"id":"cpu","type":"gauge","value":1.0}`

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(b)
	})
	handler := WithDecryption(nil)(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, body, rec.Body.String())
}

func TestWithDecryption_NoHeader_PassesThrough(t *testing.T) {
	key := generateTestKey(t)
	body := `{"id":"cpu","type":"gauge","value":1.0}`

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(b)
	})
	handler := WithDecryption(key)(inner)

	// Запрос без заголовка X-Encrypted — должен пройти без расшифровки.
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, body, rec.Body.String())
}

func TestWithDecryption_ValidEncryptedBody(t *testing.T) {
	key := generateTestKey(t)
	originalBody := `{"id":"cpu","type":"gauge","value":42.5}`

	// Шифруем тело публичным ключом.
	encrypted, err := pkgcrypto.Encrypt(&key.PublicKey, []byte(originalBody))
	require.NoError(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(b)
	})
	handler := WithDecryption(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(string(encrypted)))
	req.Header.Set("X-Encrypted", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, originalBody, rec.Body.String())
}

func TestWithDecryption_InvalidData_Returns400(t *testing.T) {
	key := generateTestKey(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("не должен быть вызван"))
	})
	handler := WithDecryption(key)(inner)

	// Отправляем мусор с заголовком шифрования.
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("invalid-encrypted-data"))
	req.Header.Set("X-Encrypted", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWithDecryption_WrongKey_Returns400(t *testing.T) {
	encryptKey := generateTestKey(t)
	decryptKey := generateTestKey(t) // другой ключ

	originalBody := `{"id":"mem","type":"gauge","value":100.0}`

	// Шифруем одним ключом.
	encrypted, err := pkgcrypto.Encrypt(&encryptKey.PublicKey, []byte(originalBody))
	require.NoError(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("не должен быть вызван"))
	})
	// Расшифровываем другим ключом — должна быть ошибка.
	handler := WithDecryption(decryptKey)(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(string(encrypted)))
	req.Header.Set("X-Encrypted", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWithDecryption_RemovesHeader(t *testing.T) {
	key := generateTestKey(t)
	originalBody := `test`

	encrypted, err := pkgcrypto.Encrypt(&key.PublicKey, []byte(originalBody))
	require.NoError(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// После расшифровки заголовок X-Encrypted должен быть удалён.
		assert.Empty(t, r.Header.Get("X-Encrypted"))
		_, _ = w.Write([]byte("ok"))
	})
	handler := WithDecryption(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(encrypted)))
	req.Header.Set("X-Encrypted", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWithDecryption_UpdatesContentLength(t *testing.T) {
	key := generateTestKey(t)
	originalBody := `{"metrics":"data"}`

	encrypted, err := pkgcrypto.Encrypt(&key.PublicKey, []byte(originalBody))
	require.NoError(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ContentLength должен соответствовать расшифрованному телу.
		assert.Equal(t, int64(len(originalBody)), r.ContentLength)
		_, _ = w.Write([]byte("ok"))
	})
	handler := WithDecryption(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(encrypted)))
	req.Header.Set("X-Encrypted", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
