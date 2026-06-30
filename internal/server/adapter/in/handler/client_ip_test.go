package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		realIP     string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{
			name:   "X-Real-IP wins",
			realIP: "203.0.113.5",
			want:   "203.0.113.5",
		},
		{
			name:      "X-Forwarded-For single",
			forwarded: "198.51.100.7",
			want:      "198.51.100.7",
		},
		{
			name:      "X-Forwarded-For takes first of chain",
			forwarded: "198.51.100.7, 10.0.0.1, 10.0.0.2",
			want:      "198.51.100.7",
		},
		{
			name:       "RemoteAddr fallback strips port",
			remoteAddr: "192.0.2.10:54321",
			want:       "192.0.2.10",
		},
		{
			name:       "RemoteAddr without port returned as-is",
			remoteAddr: "192.0.2.10",
			want:       "192.0.2.10",
		},
		{
			name:   "X-Real-IP trimmed",
			realIP: "  203.0.113.9  ",
			want:   "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}

			assert.Equal(t, tt.want, clientIP(req))
		})
	}
}
