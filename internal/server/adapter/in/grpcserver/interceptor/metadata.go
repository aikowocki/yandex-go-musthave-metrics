package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

// XRealIPKey — ключ метаданных, в котором агент передаёт свой IP-адрес.
// Ключи метаданных gRPC всегда в нижнем регистре.
const XRealIPKey = "x-real-ip"

// XRealIP извлекает IP-адрес клиента из метаданных входящего gRPC-запроса.
// Возвращает пустую строку, если метаданные или ключ отсутствуют.
// Единый источник для interceptor проверки подсети и аудита.
func XRealIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(XRealIPKey)
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}
