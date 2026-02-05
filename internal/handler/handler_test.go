package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		wantStatusCode int
	}{
		{
			name:           "success gauge",
			method:         "POST",
			url:            "/update/gauge/test/3.14",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "success counter",
			method:         "POST",
			url:            "/update/counter/test/5",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "missing metric name",
			method:         "POST",
			url:            "/update/gauge/",
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "invalid metric type",
			method:         "POST",
			url:            "/update/unknown/test/1",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid gauge value",
			method:         "POST",
			url:            "/update/gauge/test/abc",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid counter value",
			method:         "POST",
			url:            "/update/counter/test/3.14",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := repository.NewMemStorage()
			handler := NewHandler(storage)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Update(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}
