package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logging логирует каждый входящий unary-запрос: метод, код ответа и длительность.
func Logging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		zap.S().Infow("gRPC request",
			"method", info.FullMethod,
			"code", code.String(),
			"duration", time.Since(start),
		)

		return resp, err
	}
}
