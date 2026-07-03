package interceptor

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TrustedSubnet(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if subnet == nil {
			return handler(ctx, req)
		}

		ip := net.ParseIP(XRealIP(ctx))
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "request from untrusted subnet")
		}

		return handler(ctx, req)
	}
}
