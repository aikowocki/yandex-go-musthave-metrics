package grpcserver

import (
	"context"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	pb "github.com/aikowocki/yandex-go-musthave-metrics/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricUseCase interface {
	UpdateBatch(ctx context.Context, metrics []entity.Metric) error
}

type MetricService struct {
	pb.UnimplementedMetricsServer
	uc    MetricUseCase
	audit port.AuditPublisher
}

func NewMetricService(uc MetricUseCase, audit port.AuditPublisher) *MetricService {
	return &MetricService{uc: uc, audit: audit}
}

func (s *MetricService) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics := make([]entity.Metric, 0, len(req.GetMetrics()))

	for _, m := range req.GetMetrics() {
		metric, err := protoToEntity(m)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		metrics = append(metrics, metric)
	}

	if err := s.uc.UpdateBatch(ctx, metrics); err != nil {
		zap.S().Errorw("gRPC UpdateMetrics failed", "error", err)
		return nil, status.Error(codes.Internal, "failed to update metrics")
	}

	s.publishAudit(ctx, metrics)

	return &pb.UpdateMetricsResponse{}, nil
}

func (s *MetricService) publishAudit(ctx context.Context, metrics []entity.Metric) {
	if s.audit == nil || len(metrics) == 0 {
		return
	}
	names := make([]string, len(metrics))
	for i, m := range metrics {
		names[i] = m.GetName()
	}
	s.audit.Publish(entity.AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   names,
		IPAddress: interceptor.XRealIP(ctx),
		Transport: "grpc",
	})
}

func protoToEntity(m *pb.Metric) (entity.Metric, error) {
	id := m.GetId()
	if id == "" {
		return nil, entity.ErrEmptyMetricName
	}

	switch m.GetType() {
	case pb.Metric_GAUGE:
		p, ok := m.GetPayload().(*pb.Metric_Value)
		if !ok {
			return nil, entity.ErrInvalidMetricValue
		}
		return entity.NewGaugeMetric(id, p.Value), nil

	case pb.Metric_COUNTER:
		p, ok := m.GetPayload().(*pb.Metric_Delta)
		if !ok {
			return nil, entity.ErrInvalidMetricValue
		}
		return entity.NewCounterMetric(id, p.Delta), nil

	default:
		return nil, entity.ErrInvalidMetricType
	}
}
