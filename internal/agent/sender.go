package agent

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/middleware"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	"github.com/go-resty/resty/v2"
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
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry на сетевые ошибки или 5xx статусы
			return err != nil || (r != nil && r.StatusCode() >= 500)
		}).
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
	fmt.Println("Reporting metrics to server...")

	storage.ForEachGauge(func(name string, value float64) {
		err := client.SendMetric(model.MetricTypeGauge, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err != nil {
			fmt.Printf("Failed to send gauge %s: %v\n", name, err)
		}
	})

	for name, value := range storage.SnapshotCounters() {
		err := client.SendMetric(model.MetricTypeCounter, name, strconv.FormatInt(value, 10))
		if err != nil {
			fmt.Printf("Failed to send counter %s: %v\n", name, err)
		}
	}
}

func ReportJSON(storage MetricStorage, client *Client) {
	fmt.Println("Reporting metrics to server...")

	storage.ForEachGauge(func(name string, value float64) {
		v := value
		err := client.SendMetricJSON(model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeGauge),
			Value: &v,
		})
		if err != nil {
			fmt.Printf("Failed to send gauge %s: %v\n", name, err)
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
			fmt.Printf("Failed to send counter %s: %v\n", name, err)
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
