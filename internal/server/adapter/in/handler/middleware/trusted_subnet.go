package middleware

import (
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// WithTrustedSubnet возвращает middleware, отклоняющий запросы, чей IP из заголовка
// X-Real-IP не входит в доверенную подсеть subnet. Если subnet == nil — проверка
// отключена и запросы проходят без ограничений. Парсинг CIDR выполняется на этапе
// инициализации приложения (fail-fast), поэтому сюда подсеть приходит уже валидной.
func WithTrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	if subnet == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := strings.TrimSpace(r.Header.Get("X-Real-IP"))
			ip := net.ParseIP(ipStr)
			if ip == nil || !subnet.Contains(ip) {
				zap.S().Warnw("request from untrusted subnet rejected",
					"ip", ipStr, "subnet", subnet.String(), "path", r.URL.Path)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
