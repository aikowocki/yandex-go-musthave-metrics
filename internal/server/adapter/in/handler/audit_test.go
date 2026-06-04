package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyPublisher храним все опубликованные события.
type spyPublisher struct {
	mu     sync.Mutex
	events []entity.AuditEvent
}

func (s *spyPublisher) Publish(e entity.AuditEvent) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *spyPublisher) Close(_ context.Context) error { return nil }

func (s *spyPublisher) got() []entity.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]entity.AuditEvent, len(s.events))
	copy(out, s.events)
	return out
}

func TestAudit_UpdatePath(t *testing.T) {
	spy := &spyPublisher{}
	h := NewMetricHandler(newTestUseCase(), spy)
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/3.14", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	events := spy.got()
	require.Len(t, events, 1)
	assert.Equal(t, []string{"cpu"}, events[0].Metrics)
	assert.Equal(t, "192.168.1.10", events[0].IPAddress)
	assert.NotZero(t, events[0].Timestamp)
}

func TestAudit_UpdateJSON(t *testing.T) {
	spy := &spyPublisher{}
	h := NewMetricJSONHandler(newTestUseCase(), spy)
	r := chi.NewRouter()
	r.Post("/update", h.Update)

	body := `{"id":"mem","type":"gauge","value":128.5}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.5:9999"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	events := spy.got()
	require.Len(t, events, 1)
	assert.Equal(t, []string{"mem"}, events[0].Metrics)
	assert.Equal(t, "10.0.0.5", events[0].IPAddress)
}

func TestAudit_BatchUpdate(t *testing.T) {
	spy := &spyPublisher{}
	h := NewMetricJSONHandler(newTestUseCase(), spy)
	r := chi.NewRouter()
	r.Post("/updates", h.BatchUpdate)

	body := `[{"id":"cpu","type":"gauge","value":1.1},{"id":"hits","type":"counter","delta":5}]`
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "172.17.0.1:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	events := spy.got()
	require.Len(t, events, 1)
	assert.Equal(t, []string{"cpu", "hits"}, events[0].Metrics)
	assert.Equal(t, "172.17.0.1", events[0].IPAddress)
}

func TestAudit_NotPublishedOnError(t *testing.T) {
	spy := &spyPublisher{}
	h := NewMetricHandler(newTestUseCase(), spy)
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	// invalid type → 400, no audit
	req := httptest.NewRequest(http.MethodPost, "/update/unknown/x/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, spy.got())
}

func TestAudit_NilPublisher(t *testing.T) {
	h := NewMetricHandler(newTestUseCase(), nil)
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/x/1", nil)
	w := httptest.NewRecorder()
	// Should not panic
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
