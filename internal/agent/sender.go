package agent

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
)

type Client struct {
	serverURL string
}

func NewClient(serverURL string) *Client {
	return &Client{serverURL: serverURL}
}

func Report(storage MetricStorage, сlient *Client) {
	fmt.Println("Reporting metrics to server...")
	for name, value := range storage.GetGauges() {
		err := сlient.SendMetric(model.Gauge, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err != nil {
			fmt.Printf("Failed to send gauge %s: %v\n", name, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for name, value := range storage.GetCounters() {
		err := сlient.SendMetric(model.Counter, name, strconv.FormatInt(value, 10))
		if err != nil {
			fmt.Printf("Failed to send counter %s: %v\n", name, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	storage.ResetCounters()
}

func (s *Client) SendMetric(metricType model.MetricType, name string, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", s.serverURL, metricType, name, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Retry
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			err = fmt.Errorf("server returned %d", resp.StatusCode)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		resp.Body.Close()
		fmt.Printf("Sent %s (%s) = %s\n", metricType, name, value)
		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, err)

}
