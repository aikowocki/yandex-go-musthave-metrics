package audit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPObserver_PostsJSON(t *testing.T) {
	var received entity.AuditEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &received))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	event := entity.AuditEvent{
		Timestamp: 123456,
		Metrics:   []string{"Alloc", "Sys"},
		IPAddress: "192.168.1.1",
	}

	err := obs.Notify(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, event, received)
}

func TestHTTPObserver_ReturnsErrorOn4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	err := obs.Notify(context.Background(), entity.AuditEvent{Timestamp: 1, Metrics: []string{"x"}, IPAddress: "1.1.1.1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestHTTPObserver_ReturnsErrorOnUnreachable(t *testing.T) {
	obs := NewHTTPObserver("http://127.0.0.1:1") // nothing listens here
	err := obs.Notify(context.Background(), entity.AuditEvent{Timestamp: 1, Metrics: []string{"x"}, IPAddress: "1.1.1.1"})
	assert.Error(t, err)
}
