package interceptor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestXRealIP(t *testing.T) {
	t.Run("extracts ip from x-real-ip", func(t *testing.T) {
		md := metadata.New(map[string]string{"x-real-ip": "203.0.113.7"})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		assert.Equal(t, "203.0.113.7", XRealIP(ctx))
	})

	t.Run("trims surrounding spaces", func(t *testing.T) {
		md := metadata.New(map[string]string{"x-real-ip": "  10.0.0.1  "})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		assert.Equal(t, "10.0.0.1", XRealIP(ctx))
	})

	t.Run("no metadata returns empty", func(t *testing.T) {
		assert.Equal(t, "", XRealIP(context.Background()))
	})

	t.Run("metadata without x-real-ip returns empty", func(t *testing.T) {
		md := metadata.New(map[string]string{"other-key": "value"})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		assert.Equal(t, "", XRealIP(ctx))
	})
}
