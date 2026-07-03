package agent

import (
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	pkgcrypto "github.com/aikowocki/yandex-go-musthave-metrics/pkg/crypto"
	pkghash "github.com/aikowocki/yandex-go-musthave-metrics/pkg/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest хранит то, что реально дошло до сервера: путь,
// HMAC-заголовок, флаг шифрования и распакованное (при необходимости
// расшифрованное) тело запроса.
type capturedRequest struct {
	path       string
	hashHeader string
	encrypted  bool
	body       []byte
}

func newCapturingServer(t *testing.T, privateKey *rsa.PrivateKey) (*httptest.Server, *[]capturedRequest, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	captured := make([]capturedRequest, 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковываем gzip.
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "bad gzip", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(gz)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		encrypted := r.Header.Get("X-Encrypted") == "1"
		if encrypted && privateKey != nil {
			body, err = pkgcrypto.Decrypt(privateKey, body)
			if err != nil {
				http.Error(w, "decrypt error", http.StatusBadRequest)
				return
			}
		}

		mu.Lock()
		captured = append(captured, capturedRequest{
			path:       r.URL.Path,
			hashHeader: r.Header.Get(pkghash.HEADER),
			encrypted:  encrypted,
			body:       body,
		})
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &captured, &mu
}

func TestClient_SendMetrics_GzipAndHMAC(t *testing.T) {
	const key = "secret-key"
	srv, captured, mu := newCapturingServer(t, nil)

	client := NewClient(srv.URL, "", WithServerKey(key))

	delta := int64(5)
	value := 3.14
	dtos := []api.MetricDTO{
		{ID: "hits", MType: api.MetricTypeCounter, Delta: &delta},
		{ID: "cpu", MType: api.MetricTypeGauge, Value: &value},
	}

	err := client.SendMetrics(t.Context(), dtos)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, *captured, 1)
	req := (*captured)[0]

	assert.Equal(t, "/updates", req.path)
	assert.NotEmpty(t, req.hashHeader, "HMAC header must be set when server key configured")

	// HMAC должен валидироваться от распакованного тела.
	assert.True(t, pkghash.ValidateHMAC(key, req.body, req.hashHeader),
		"server-side HMAC validation must pass")

	// Тело — корректный JSON с обеими метриками.
	var got []api.MetricDTO
	require.NoError(t, json.Unmarshal(req.body, &got))
	assert.Len(t, got, 2)
}

func TestClient_SendMetricJSON(t *testing.T) {
	srv, captured, mu := newCapturingServer(t, nil)
	client := NewClient(srv.URL, "")

	value := 42.0
	err := client.SendMetricJSON(api.MetricDTO{
		ID:    "temp",
		MType: api.MetricTypeGauge,
		Value: &value,
	})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, *captured, 1)
	assert.Equal(t, "/update", (*captured)[0].path)
	assert.Empty(t, (*captured)[0].hashHeader, "no HMAC without server key")
}

func TestClient_SendMetric_PathBased(t *testing.T) {
	srv, captured, mu := newCapturingServer(t, nil)
	client := NewClient(srv.URL, "")

	err := client.SendMetric(api.MetricTypeGauge, "load", "1.5")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, *captured, 1)
	assert.Equal(t, "/update/gauge/load/1.5", (*captured)[0].path)
}

func TestClient_RSAEncryption(t *testing.T) {
	// Генерируем пару и пишем публичный ключ во временный файл.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubFile := t.TempDir() + "/public.pem"
	pubBytes := x509.MarshalPKCS1PublicKey(&priv.PublicKey)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubBytes})
	require.NoError(t, os.WriteFile(pubFile, pemData, 0600))

	srv, captured, mu := newCapturingServer(t, priv)

	client := NewClient(srv.URL, "", WithCryptoKey(pubFile))

	value := 99.9
	err = client.SendMetricJSON(api.MetricDTO{
		ID:    "encrypted_metric",
		MType: api.MetricTypeGauge,
		Value: &value,
	})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, *captured, 1)
	req := (*captured)[0]

	assert.True(t, req.encrypted, "X-Encrypted header must be set")

	// Расшифрованное на сервере тело — валидный JSON с нашей метрикой.
	var got api.MetricDTO
	require.NoError(t, json.Unmarshal(req.body, &got))
	assert.Equal(t, "encrypted_metric", got.ID)
}

func TestReport_And_ReportJSON(t *testing.T) {
	t.Run("Report (path-based)", func(t *testing.T) {
		srv, captured, mu := newCapturingServer(t, nil)
		client := NewClient(srv.URL, "")

		storage := NewLocalStorage()
		storage.SetGauge("g1", 1.0)
		storage.AddCounter("c1", 7)

		Report(storage, client)

		mu.Lock()
		defer mu.Unlock()
		// 1 gauge + 1 counter = 2 запроса.
		assert.Len(t, *captured, 2)
	})

	t.Run("ReportJSON", func(t *testing.T) {
		srv, captured, mu := newCapturingServer(t, nil)
		client := NewClient(srv.URL, "")

		storage := NewLocalStorage()
		storage.SetGauge("g1", 2.0)
		storage.AddCounter("c1", 3)

		ReportJSON(storage, client)

		mu.Lock()
		defer mu.Unlock()
		assert.Len(t, *captured, 2)
		for _, req := range *captured {
			assert.Equal(t, "/update", req.path)
		}
	})
}
