package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/pkg/constants"
	"github.com/stretchr/testify/assert"
)

func TestWithGzip_CompressResponse(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		expectGzip     bool
	}{
		{
			name:           "with gzip",
			acceptEncoding: EncodingGzip,
			expectGzip:     true,
		}, {
			name:           "without gzip",
			acceptEncoding: "",
			expectGzip:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			})

			wrapped := WithGzipCompression()(handler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set(constants.HeaderAcceptEncoding, EncodingGzip)
			}
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			res := rec.Result()
			defer func() { _ = res.Body.Close() }()

			assert.Equal(t, http.StatusOK, res.StatusCode)
			contentEncoding := res.Header.Get(constants.HeaderContentEncoding)
			if tt.expectGzip {
				assert.Equal(t, EncodingGzip, contentEncoding)
			} else {
				assert.Empty(t, contentEncoding)
			}
			assert.NotEmpty(t, rec.Body.Bytes())
		})
	}

}
