package middleware

import (
	"net/http"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/helper"
	"go.uber.org/zap"
)

func WithLogging(sugar *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			uri := r.RequestURI
			method := r.Method

			responseData := &helper.ResponseData{
				Status: 0,
				Size:   0,
			}

			lw := helper.LoggingResponseWriter{
				ResponseWriter: w,
				ResponseData:   responseData,
			}
			next.ServeHTTP(&lw, r)

			duration := time.Since(start)

			sugar.Infoln(
				"uri", uri,
				"method", method,
				"duration", duration,
				"status", responseData.Status,
				"size", responseData.Size,
			)
		})
	}
}
