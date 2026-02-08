package agent

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/go-resty/resty/v2"
)

type Client struct {
	restyClient *resty.Client
	serverURL   string
}

func NewClient(serverURL string) *Client {
	client := resty.New().
		SetTimeout(1*time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1*time.Second).
		SetHeader("Content-Type", "text/plain").
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry на сетевые ошибки или 5xx статусы
			return err != nil || (r != nil && r.StatusCode() >= 500)
		})

	return &Client{
		restyClient: client,
		serverURL:   serverURL,
	}
}

func Report(storage MetricStorage, сlient *Client) {
	fmt.Println("Reporting metrics to server...")
	for name, value := range storage.GetGauges() {
		err := сlient.SendMetric(model.MetricTypeGauge, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err != nil {
			fmt.Printf("Failed to send gauge %s: %v\n", name, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for name, value := range storage.GetCounters() {
		err := сlient.SendMetric(model.MetricTypeCounter, name, strconv.FormatInt(value, 10))
		if err != nil {
			fmt.Printf("Failed to send counter %s: %v\n", name, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	storage.ResetCounters()
}

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
