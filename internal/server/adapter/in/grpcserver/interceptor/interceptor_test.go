package interceptor

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// fakeHandler returns a UnaryHandler that records whether it was called.
func fakeHandler(resp any) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		return resp, nil
	}
}

func fakeErrorHandler(err error) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		return nil, err
	}
}

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, subnet, err := net.ParseCIDR(cidr)
	require.NoError(t, err)
	return subnet
}

func ctxWithIP(ip string) context.Context {
	md := metadata.Pairs("x-real-ip", ip)
	return metadata.NewIncomingContext(context.Background(), md)
}

var dummyInfo = &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

// --- TrustedSubnet tests ---

func TestTrustedSubnet(t *testing.T) {
	tests := []struct {
		name     string
		subnet   *net.IPNet
		ctx      context.Context
		wantCode codes.Code
		wantPass bool
	}{
		{
			name:     "nil subnet passes",
			subnet:   nil,
			ctx:      context.Background(),
			wantPass: true,
		},
		{
			name:     "ip in subnet passes",
			subnet:   mustCIDR(t, "192.168.1.0/24"),
			ctx:      ctxWithIP("192.168.1.42"),
			wantPass: true,
		},
		{
			name:     "ip outside subnet rejected",
			subnet:   mustCIDR(t, "192.168.1.0/24"),
			ctx:      ctxWithIP("10.0.0.1"),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "no metadata rejected",
			subnet:   mustCIDR(t, "192.168.1.0/24"),
			ctx:      context.Background(),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "empty x-real-ip rejected",
			subnet:   mustCIDR(t, "192.168.1.0/24"),
			ctx:      ctxWithIP(""),
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "malformed ip rejected",
			subnet:   mustCIDR(t, "192.168.1.0/24"),
			ctx:      ctxWithIP("not-an-ip"),
			wantCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := TrustedSubnet(tt.subnet)
			resp, err := interceptor(tt.ctx, "req", dummyInfo, fakeHandler("ok"))

			if tt.wantPass {
				assert.NoError(t, err)
				assert.Equal(t, "ok", resp)
			} else {
				assert.Nil(t, resp)
				assert.Equal(t, tt.wantCode, status.Code(err))
			}
		})
	}
}

// --- Logging tests ---

func TestLogging_CallsHandler(t *testing.T) {
	interceptor := Logging()
	resp, err := interceptor(context.Background(), "req", dummyInfo, fakeHandler("response"))

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

func TestLogging_PropagatesError(t *testing.T) {
	expectedErr := status.Error(codes.NotFound, "not found")
	interceptor := Logging()
	resp, err := interceptor(context.Background(), "req", dummyInfo, fakeErrorHandler(expectedErr))

	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// --- Recovery tests ---

func TestRecovery_NoPanic(t *testing.T) {
	interceptor := Recovery()
	resp, err := interceptor(context.Background(), "req", dummyInfo, fakeHandler("ok"))

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestRecovery_CatchesPanic(t *testing.T) {
	panicHandler := func(ctx context.Context, req any) (any, error) {
		panic("something went wrong")
	}

	interceptor := Recovery()
	resp, err := interceptor(context.Background(), "req", dummyInfo, panicHandler)

	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRecovery_PropagatesNormalError(t *testing.T) {
	interceptor := Recovery()
	expectedErr := errors.New("normal error")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, expectedErr
	}

	resp, err := interceptor(context.Background(), "req", dummyInfo, handler)

	assert.Nil(t, resp)
	assert.ErrorIs(t, err, expectedErr)
}
