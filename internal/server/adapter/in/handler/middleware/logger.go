package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		Status int
		Size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		ResponseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	if r.ResponseData.Status == 0 {
		r.ResponseData.Status = http.StatusOK
	}
	size, err := r.ResponseWriter.Write(b)
	r.ResponseData.Size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseData.Status = statusCode
}

func WithLogging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			uri := r.RequestURI
			method := r.Method

			responseData := &responseData{
				Status: 0,
				Size:   0,
			}

			lw := loggingResponseWriter{
				ResponseWriter: w,
				ResponseData:   responseData,
			}
			next.ServeHTTP(&lw, r)

			duration := time.Since(start)

			zap.S().Infow("request completed",
				"uri", uri,
				"method", method,
				"duration", duration,
				"status", responseData.Status,
				"size", responseData.Size,
			)
		})
	}
}
