package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	pkgcrypto "github.com/aikowocki/yandex-go-musthave-metrics/pkg/crypto"
	pkghash "github.com/aikowocki/yandex-go-musthave-metrics/pkg/hash"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/pool"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/retry"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var gzipWriterPool = pool.NewFunc(
	func() *gzip.Writer { return gzip.NewWriter(io.Discard) },
	func(gz *gzip.Writer) { gz.Reset(io.Discard) },
)

type ClientOption func(*Client)

func WithServerKey(key string) ClientOption {
	return func(c *Client) {
		c.serverKey = key
	}
}

// WithCryptoKey задаёт путь к файлу с RSA публичным ключом для шифрования запросов.
func WithCryptoKey(path string) ClientOption {
	return func(c *Client) {
		c.cryptoKeyPath = path
	}
}

// Client — HTTP-клиент агента для отправки метрик на сервер.
// Поддерживает gzip-сжатие, HMAC-подпись и RSA-шифрование запросов.
type Client struct {
	restyClient   *resty.Client
	serverURL     string
	serverKey     string
	cryptoKeyPath string
	publicKey     *rsa.PublicKey
}

// NewClient создаёт нового клиента для отправки метрик на указанный сервер.
func NewClient(serverURL string, opts ...ClientOption) *Client {
	c := &Client{
		serverURL: serverURL,
	}
	for _, opt := range opts {
		opt(c)
	}

	// Загружаем RSA публичный ключ если задан путь.
	if c.cryptoKeyPath != "" {
		pub, err := pkgcrypto.LoadPublicKey(c.cryptoKeyPath)
		if err != nil {
			zap.S().Fatalw("filed to load RSA public key", "path", c.cryptoKeyPath, "error", err)
		}
		c.publicKey = pub
		zap.S().Infow("RSA encryption enabled", "key", c.cryptoKeyPath)
	}

	restyClient := resty.New().
		SetHeader(constants.HeaderContentType, constants.ContentTypeJSON).
		SetHeader(constants.HeaderContentEncoding, constants.EncodingGzip).
		SetHeader(constants.HeaderAcceptEncoding, constants.EncodingGzip).
		SetTimeout(1 * time.Second).
		SetPreRequestHook(func(_ *resty.Client, r *http.Request) error {
			if r.Body == nil {
				return nil
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}

			// HMAC от оригинального (незашифрованного) тела.
			if c.serverKey != "" {
				r.Header.Set(pkghash.HEADER, pkghash.ComputeHMAC(c.serverKey, body))
			}

			// RSA шифрование (перед gzip).
			if c.publicKey != nil {
				body, err = pkgcrypto.Encrypt(c.publicKey, body)
				if err != nil {
					return fmt.Errorf("rsa encrypt: %w", err)
				}
				r.Header.Set("X-Encrypted", "1")
			}

			// Gzip сжатие.
			var buf bytes.Buffer

			gz := gzipWriterPool.Get()
			gz.Reset(&buf)
			defer gzipWriterPool.Put(gz)

			if _, err := gz.Write(body); err != nil {
				_ = gz.Close()
				return err
			}
			if err := gz.Close(); err != nil {
				return err
			}

			r.Body = io.NopCloser(&buf)
			r.ContentLength = int64(buf.Len())
			return nil
		})
	c.restyClient = restyClient

	// X-Real-IP — исходящий IP-адрес хоста агента, по которому сервер
	// проверяет принадлежность к доверенной подсети.
	if ip := outboundIP(serverURL); ip != "" {
		// по идее выставление этого заголовка в проде на плечах nginx/Traefik/...
		restyClient.SetHeader("X-Real-IP", ip)
	}

	return c
}

// outboundIP определяет исходящий IP-адрес хоста, который ОС выберет для
// соединения с сервером. UDP-«соединение» реально пакеты не отправляет,
// но заставляет ядро выбрать сетевой интерфейс и локальный адрес.
func outboundIP(serverURL string) string {
	host := serverURL
	if u, err := url.Parse(serverURL); err == nil && u.Host != "" {
		host = u.Host
	}

	conn, err := net.Dial("udp", host)
	if err != nil {
		zap.S().Warnw("failed to determine outbound IP", "host", host, "error", err)
		return ""
	}
	defer func() { _ = conn.Close() }()

	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		ip := addr.IP.String()
		zap.S().Infow("determined outbound IP", "ip", ip)
		return ip
	}
	return ""
}

// Deprecated: use ReportJSON
func Report(storage MetricStorage, client *Client) {
	zap.S().Debugw("reporting metrics to server")

	storage.ForEachGauge(func(name string, value float64) {
		err := client.SendMetric(api.MetricTypeGauge, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err != nil {
			zap.S().Errorw("failed to send gauge", "name", name, zap.Error(err))
		}
	})

	for name, value := range storage.SnapshotCounters() {
		err := client.SendMetric(api.MetricTypeCounter, name, strconv.FormatInt(value, 10))
		if err != nil {
			zap.S().Errorw("failed to send counter", "name", name, zap.Error(err))
		}
	}
}

func ReportJSON(storage MetricStorage, client *Client) {
	zap.S().Debugw("reporting metrics to server")

	storage.ForEachGauge(func(name string, value float64) {
		v := value
		err := client.SendMetricJSON(api.MetricDTO{
			ID:    name,
			MType: api.MetricTypeGauge,
			Value: &v,
		})
		if err != nil {
			zap.S().Errorw("failed to send gauge", "name", name, zap.Error(err))
		}
	})

	for name, value := range storage.SnapshotCounters() {
		v := value
		err := client.SendMetricJSON(api.MetricDTO{
			ID:    name,
			MType: api.MetricTypeCounter,
			Delta: &v,
		})
		if err != nil {
			zap.S().Errorw("failed to send counter", "name", name, zap.Error(err))
		}
	}
}

// CollectBatch формирует пачку метрик из storage для отправки на сервер.
// Возвращает слайс MetricDTO, готовый к отправке через SendBatch.
// Вызов SnapshotCounters сбрасывает счётчики в storage.
func CollectBatch(storage MetricStorage) []api.MetricDTO {
	zap.S().Debugw("collecting metrics batch")
	gauges := storage.SnapshotGauges()
	counters := storage.SnapshotCounters()
	metrics := make([]api.MetricDTO, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		v := value
		metrics = append(metrics, api.MetricDTO{
			ID:    name,
			MType: api.MetricTypeGauge,
			Value: &v,
		})
	}

	for name, value := range counters {
		v := value
		metrics = append(metrics, api.MetricDTO{
			ID:    name,
			MType: api.MetricTypeCounter,
			Delta: &v,
		})
	}
	return metrics
}

// SendBatch отправляет пачку метрик на сервер с retry-логикой при сетевых ошибках.
func SendBatch(ctx context.Context, client *Client, metrics []api.MetricDTO) {
	zap.S().Debugw("sending metrics batch to server")
	if len(metrics) > 0 {
		send := func() error { return client.SendMetrics(metrics) }
		retrierCondition := func(err error) bool {
			var netErr *net.OpError
			return errors.As(err, &netErr)
		}
		if err := retry.Do(ctx, send, retry.WithRetryIf(retrierCondition)); err != nil {
			zap.S().Errorw("failed to send metrics", zap.Error(err))
		}
	}
}

// Deprecated: use SendMetricJSON
func (c *Client) SendMetric(metricType string, name, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", c.serverURL, metricType, name, value)

	resp, err := c.restyClient.R().Post(url)
	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

func (c *Client) SendMetricJSON(dto api.MetricDTO) error {
	url := fmt.Sprintf("%s/update", c.serverURL)

	resp, err := c.restyClient.R().SetBody(dto).Post(url)
	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

func (c *Client) SendMetrics(dtos []api.MetricDTO) error {
	url := fmt.Sprintf("%s/updates", c.serverURL)

	resp, err := c.restyClient.R().SetBody(dtos).Post(url)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}
