package agent

import (
	"context"
	"net"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	pb "github.com/aikowocki/yandex-go-musthave-metrics/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// mockMetricsServer реализует pb.MetricsServer для тестирования
type mockMetricsServer struct {
	pb.UnimplementedMetricsServer
	receivedMetrics []*pb.Metric
	receivedIP      string
}

func (m *mockMetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	m.receivedMetrics = req.Metrics

	// Извлекаем IP из metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ips := md.Get("x-real-ip"); len(ips) > 0 {
			m.receivedIP = ips[0]
		}
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// setupGRPCTestServer создаёт in-memory gRPC сервер для тестирования
func setupGRPCTestServer(t *testing.T) (*mockMetricsServer, func() *grpc.ClientConn) {
	t.Helper()

	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	mock := &mockMetricsServer{}
	pb.RegisterMetricsServer(server, mock)

	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	t.Cleanup(func() {
		server.Stop()
		listener.Close()
	})

	dialFunc := func() *grpc.ClientConn {
		conn, err := grpc.DialContext(
			context.Background(),
			"bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		return conn
	}

	return mock, dialFunc
}

func TestNewGRPCClient(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "valid address",
			address: "localhost:8080",
			wantErr: false,
		},
		{
			name:    "with http scheme",
			address: "http://localhost:8080",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewGRPCClient(tt.address, OutboundIP(tt.address))

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, client)
				assert.NotNil(t, client.client)
				assert.NotEmpty(t, client.ip)

				// Cleanup
				err := client.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRPCClient_SendMetrics(t *testing.T) {
	mock, dialFunc := setupGRPCTestServer(t)
	conn := dialFunc()

	client := &GRPCClient{
		client: pb.NewMetricsClient(conn),
		ip:     "192.168.1.100",
		conn:   conn,
	}
	t.Cleanup(func() { _ = client.Close() })

	t.Run("send gauge metrics", func(t *testing.T) {
		gaugeValue := 3.14
		dtos := []api.MetricDTO{
			{
				ID:    "cpu",
				MType: api.MetricTypeGauge,
				Value: &gaugeValue,
			},
		}

		err := client.SendMetrics(t.Context(), dtos)
		require.NoError(t, err)

		require.Len(t, mock.receivedMetrics, 1)
		assert.Equal(t, "cpu", mock.receivedMetrics[0].Id)
		assert.Equal(t, pb.Metric_GAUGE, mock.receivedMetrics[0].Type)
		assert.Equal(t, gaugeValue, mock.receivedMetrics[0].GetValue())
		assert.Equal(t, "192.168.1.100", mock.receivedIP)
	})

	t.Run("send counter metrics", func(t *testing.T) {
		counterDelta := int64(42)
		dtos := []api.MetricDTO{
			{
				ID:    "requests",
				MType: api.MetricTypeCounter,
				Delta: &counterDelta,
			},
		}

		err := client.SendMetrics(t.Context(), dtos)
		require.NoError(t, err)

		require.Len(t, mock.receivedMetrics, 1)
		assert.Equal(t, "requests", mock.receivedMetrics[0].Id)
		assert.Equal(t, pb.Metric_COUNTER, mock.receivedMetrics[0].Type)
		assert.Equal(t, counterDelta, mock.receivedMetrics[0].GetDelta())
	})

	t.Run("send mixed metrics", func(t *testing.T) {
		gaugeValue := 2.71
		counterDelta := int64(100)

		dtos := []api.MetricDTO{
			{
				ID:    "temperature",
				MType: api.MetricTypeGauge,
				Value: &gaugeValue,
			},
			{
				ID:    "hits",
				MType: api.MetricTypeCounter,
				Delta: &counterDelta,
			},
		}

		err := client.SendMetrics(t.Context(), dtos)
		require.NoError(t, err)

		require.Len(t, mock.receivedMetrics, 2)

		// Проверяем gauge
		assert.Equal(t, "temperature", mock.receivedMetrics[0].Id)
		assert.Equal(t, pb.Metric_GAUGE, mock.receivedMetrics[0].Type)
		assert.Equal(t, gaugeValue, mock.receivedMetrics[0].GetValue())

		// Проверяем counter
		assert.Equal(t, "hits", mock.receivedMetrics[1].Id)
		assert.Equal(t, pb.Metric_COUNTER, mock.receivedMetrics[1].Type)
		assert.Equal(t, counterDelta, mock.receivedMetrics[1].GetDelta())
	})

	t.Run("empty metrics list", func(t *testing.T) {
		err := client.SendMetrics(t.Context(), []api.MetricDTO{})
		require.NoError(t, err)
		assert.Empty(t, mock.receivedMetrics)
	})
}

func TestGRPCClient_Close(t *testing.T) {
	_, dialFunc := setupGRPCTestServer(t)

	conn := dialFunc()
	client := &GRPCClient{
		client: pb.NewMetricsClient(conn),
		ip:     "127.0.0.1",
		conn:   conn,
	}

	err := client.Close()
	assert.NoError(t, err)

	// После закрытия повторный Close должен вернуть ошибку
	err = client.Close()
	assert.Error(t, err)
}
