package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubPinger — управляемая заглушка Pinger для проверки веток Ping.
type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHealthHandler_Ping(t *testing.T) {
	tests := []struct {
		name       string
		pinger     Pinger
		wantStatus int
	}{
		{
			name:       "healthy pinger returns 200",
			pinger:     stubPinger{err: nil},
			wantStatus: http.StatusOK,
		},
		{
			name:       "failing pinger returns 500",
			pinger:     stubPinger{err: errors.New("db unavailable")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "nil pinger returns 500",
			pinger:     nil,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(tt.pinger)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			h.Ping(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
