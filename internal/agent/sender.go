package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/middleware"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/retry"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Client struct {
	restyClient *resty.Client
	serverURL   string
}

func NewClient(serverURL string) *Client {
	client := resty.New().
		SetHeader(constants.HeaderContentType, constants.ContentTypeJSON).
		SetHeader(constants.HeaderContentEncoding, middleware.EncodingGzip).
		SetHeader(constants.HeaderAcceptEncoding, middleware.EncodingGzip).
		SetTimeout(1 * time.Second).
		SetPreRequestHook(func(c *resty.Client, r *http.Request) error {
			if r.Body == nil {
				return nil
			}
			var buf bytes.Buffer
			w := gzip.NewWriter(&buf)
			_, err := io.Copy(w, r.Body)
			if err != nil {
				return err
			}
			if err := w.Close(); err != nil {
				return err
			}
			r.Body = io.NopCloser(&buf)
			r.ContentLength = int64(buf.Len())
			return nil
		})
	return &Client{
		restyClient: client,
		serverURL:   serverURL,
	}
}

// Deprecated: use ReportJSON
func Report(storage MetricStorage, client *Client) {
	zap.S().Debugw("reporting metrics to server")

	storage.ForEachGauge(func(name string, value float64) {
		err := client.SendMetric(model.MetricTypeGauge, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err != nil {
			zap.S().Errorw("failed to send gauge", "name", name, zap.Error(err))
		}
	})

	for name, value := range storage.SnapshotCounters() {
		err := client.SendMetric(model.MetricTypeCounter, name, strconv.FormatInt(value, 10))
		if err != nil {
			zap.S().Errorw("failed to send counter", "name", name, zap.Error(err))
		}
	}
}

func ReportJSON(storage MetricStorage, client *Client) {
	zap.S().Debugw("reporting metrics to server")

	storage.ForEachGauge(func(name string, value float64) {
		v := value
		err := client.SendMetricJSON(model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeGauge),
			Value: &v,
		})
		if err != nil {
			zap.S().Errorw("failed to send gauge", "name", name, zap.Error(err))
		}
	})

	for name, value := range storage.SnapshotCounters() {
		v := value
		err := client.SendMetricJSON(model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeCounter),
			Delta: &v,
		})
		if err != nil {
			zap.S().Errorw("failed to send counter", "name", name, zap.Error(err))
		}
	}
}

func ReportBatch(storage MetricStorage, client *Client) {
	zap.S().Debugw("reporting metrics to server")
	var metrics []model.MetricDTO
	for name, value := range storage.SnapshotGauges() {
		v := value
		metrics = append(metrics, model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeGauge),
			Value: &v,
		})
	}

	for name, value := range storage.SnapshotCounters() {
		v := value
		metrics = append(metrics, model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeCounter),
			Delta: &v,
		})
	}

	if len(metrics) > 0 {
		send := func() error { return client.SendMetrics(metrics) }
		retrierCondition := func(err error) bool {
			var netErr *net.OpError
			return errors.As(err, &netErr)
		}
		if err := retry.Do(context.TODO(), send, retry.WithRetryIf(retrierCondition)); err != nil {
			zap.S().Errorw("failed to send metrics", zap.Error(err))
		}
	}
}

// Deprecated: use SendMetricJSON
func (c *Client) SendMetric(metricType model.MetricType, name, value string) error {
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

func (c *Client) SendMetricJSON(dto model.MetricDTO) error {
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

func (c *Client) SendMetrics(dtos []model.MetricDTO) error {
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
