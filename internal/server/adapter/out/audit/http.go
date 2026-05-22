package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

const httpTimeout = 5 * time.Second

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: httpTimeout},
	}
}

func (o *HTTPObserver) Name() string { return "http:" + o.url }

func (o *HTTPObserver) Notify(ctx context.Context, event entity.AuditEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit http marshal: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, o.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("audit http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("audit http do: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("audit http status: %d", resp.StatusCode)
	}
	return nil
}
