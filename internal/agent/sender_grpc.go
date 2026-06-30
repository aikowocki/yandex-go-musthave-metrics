package agent

import (
	"context"

	pb "github.com/aikowocki/yandex-go-musthave-metrics/pkg/proto"
	"go.uber.org/zap"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type GRPCClient struct {
	client pb.MetricsClient
	ip     string
	conn   *grpc.ClientConn
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func NewGRPCClient(address string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		client: pb.NewMetricsClient(conn),
		ip:     outboundIP(address),
		conn:   conn,
	}, nil
}

func (c *GRPCClient) SendMetrics(ctx context.Context, dtos []api.MetricDTO) error {
	metrics := make([]*pb.Metric, 0, len(dtos))

	for _, dto := range dtos {
		m := &pb.Metric{Id: dto.ID}

		switch dto.MType {
		case api.MetricTypeGauge:
			if dto.Value == nil {
				zap.S().Warnw("skip gauge with nil value", "id", dto.ID)
				continue
			}
			m.Type = pb.Metric_GAUGE
			m.Payload = &pb.Metric_Value{Value: *dto.Value}
		case api.MetricTypeCounter:
			if dto.Delta == nil {
				zap.S().Warnw("skip counter with nil delta", "id", dto.ID)
				continue
			}
			m.Type = pb.Metric_COUNTER
			m.Payload = &pb.Metric_Delta{Delta: *dto.Delta}
		default:
			zap.S().Warnw("skip metric with unknown type", "id", dto.ID, "type", dto.MType)
			continue
		}
		metrics = append(metrics, m)
	}
	// IP в метаданные
	ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", c.ip)

	_, err := c.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{Metrics: metrics})
	return err
}
