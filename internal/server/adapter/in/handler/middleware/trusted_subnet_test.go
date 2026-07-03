package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProbeHandler возвращает handler, который ставит 200 и фиксирует факт вызова
// через переданный указатель — это позволяет проверить, что при 403 next НЕ вызывается.
func newProbeHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestWithTrustedSubnet(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string // Пустая строка → subnet nil (фильтрация выключена)
		realIP      string
		wantStatus  int
		wantNextHit bool
	}{
		{
			name:        "nil subnet passes through",
			cidr:        "",
			realIP:      "10.0.0.1",
			wantStatus:  http.StatusOK,
			wantNextHit: true,
		},
		{
			name:        "nil subnet passes through even without header",
			cidr:        "",
			realIP:      "",
			wantStatus:  http.StatusOK,
			wantNextHit: true,
		},
		{
			name:        "ip inside subnet allowed",
			cidr:        "192.168.1.0/24",
			realIP:      "192.168.1.42",
			wantStatus:  http.StatusOK,
			wantNextHit: true,
		},
		{
			name:        "subnet lower boundary allowed",
			cidr:        "192.168.1.0/24",
			realIP:      "192.168.1.0",
			wantStatus:  http.StatusOK,
			wantNextHit: true,
		},
		{
			name:        "subnet upper boundary allowed",
			cidr:        "192.168.1.0/24",
			realIP:      "192.168.1.255",
			wantStatus:  http.StatusOK,
			wantNextHit: true,
		},
		{
			name:        "ip outside subnet rejected",
			cidr:        "192.168.1.0/24",
			realIP:      "10.0.0.1",
			wantStatus:  http.StatusForbidden,
			wantNextHit: false,
		},
		{
			name:        "empty header with configured subnet rejected",
			cidr:        "192.168.1.0/24",
			realIP:      "",
			wantStatus:  http.StatusForbidden,
			wantNextHit: false,
		},
		{
			name:        "malformed ip rejected",
			cidr:        "192.168.1.0/24",
			realIP:      "not-an-ip",
			wantStatus:  http.StatusForbidden,
			wantNextHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Парсим CIDR внутри под-теста: опечатка в данных одного кейса
			// завалит только его, а не всю таблицу (FailNow бьёт точечно).
			var subnet *net.IPNet
			if tt.cidr != "" {
				_, parsed, err := net.ParseCIDR(tt.cidr)
				require.NoError(t, err)
				subnet = parsed
			}

			var nextCalled bool
			handler := WithTrustedSubnet(subnet)(newProbeHandler(&nextCalled))

			req := httptest.NewRequest(http.MethodPost, "/updates", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNextHit, nextCalled, "next handler invocation")
		})
	}
}

// TestWithTrustedSubnet_TrimsHeader проверяет, что окружающие пробелы в X-Real-IP
// не ломают проверку (заголовок мог прийти от прокси с пробелами).
func TestWithTrustedSubnet_TrimsHeader(t *testing.T) {
	_, subnet, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)

	var nextCalled bool
	handler := WithTrustedSubnet(subnet)(newProbeHandler(&nextCalled))

	req := httptest.NewRequest(http.MethodPost, "/updates", nil)
	req.Header.Set("X-Real-IP", "  10.1.2.3  ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, nextCalled)
}
