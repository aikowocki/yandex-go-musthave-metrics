package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	pb "github.com/aikowocki/yandex-go-musthave-metrics/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// --- protoToEntity tests ---

func TestProtoToEntity_Gauge(t *testing.T) {
	m := &pb.Metric{
		Id:      "cpu",
		Type:    pb.Metric_GAUGE,
		Payload: &pb.Metric_Value{Value: 42.5},
	}
	result, err := protoToEntity(m)
	require.NoError(t, err)

	gauge, ok := result.(*entity.GaugeMetric)
	require.True(t, ok)
	assert.Equal(t, "cpu", gauge.GetName())
	assert.Equal(t, 42.5, gauge.Value)
}

func TestProtoToEntity_Counter(t *testing.T) {
	m := &pb.Metric{
		Id:      "hits",
		Type:    pb.Metric_COUNTER,
		Payload: &pb.Metric_Delta{Delta: 10},
	}
	result, err := protoToEntity(m)
	require.NoError(t, err)

	counter, ok := result.(*entity.CounterMetric)
	require.True(t, ok)
	assert.Equal(t, "hits", counter.GetName())
	assert.Equal(t, int64(10), counter.Value)
}

func TestProtoToEntity_EmptyID(t *testing.T) {
	m := &pb.Metric{
		Id:      "",
		Type:    pb.Metric_GAUGE,
		Payload: &pb.Metric_Value{Value: 1.0},
	}
	_, err := protoToEntity(m)
	assert.ErrorIs(t, err, entity.ErrEmptyMetricName)
}

func TestProtoToEntity_UnspecifiedType(t *testing.T) {
	m := &pb.Metric{
		Id:      "test",
		Type:    pb.Metric_UNSPECIFIED,
		Payload: &pb.Metric_Value{Value: 1.0},
	}
	_, err := protoToEntity(m)
	assert.ErrorIs(t, err, entity.ErrInvalidMetricType)
}

func TestProtoToEntity_GaugeWithDeltaPayload(t *testing.T) {
	m := &pb.Metric{
		Id:      "cpu",
		Type:    pb.Metric_GAUGE,
		Payload: &pb.Metric_Delta{Delta: 5}, // wrong payload for gauge
	}
	_, err := protoToEntity(m)
	assert.ErrorIs(t, err, entity.ErrInvalidMetricValue)
}

func TestProtoToEntity_CounterWithValuePayload(t *testing.T) {
	m := &pb.Metric{
		Id:      "hits",
		Type:    pb.Metric_COUNTER,
		Payload: &pb.Metric_Value{Value: 5.0}, // wrong payload for counter
	}
	_, err := protoToEntity(m)
	assert.ErrorIs(t, err, entity.ErrInvalidMetricValue)
}

func TestProtoToEntity_NilPayload(t *testing.T) {
	m := &pb.Metric{
		Id:   "test",
		Type: pb.Metric_GAUGE,
		// Payload is nil
	}
	_, err := protoToEntity(m)
	assert.ErrorIs(t, err, entity.ErrInvalidMetricValue)
}

// --- MetricService.UpdateMetrics tests ---

type mockUseCase struct {
	called  bool
	metrics []entity.Metric
	err     error
}

func (m *mockUseCase) UpdateBatch(ctx context.Context, metrics []entity.Metric) error {
	m.called = true
	m.metrics = metrics
	return m.err
}

func TestUpdateMetrics_Success(t *testing.T) {
	uc := &mockUseCase{}
	svc := NewMetricService(uc, nil)

	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "cpu", Type: pb.Metric_GAUGE, Payload: &pb.Metric_Value{Value: 42.5}},
			{Id: "hits", Type: pb.Metric_COUNTER, Payload: &pb.Metric_Delta{Delta: 10}},
		},
	}

	resp, err := svc.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, uc.called)
	assert.Len(t, uc.metrics, 2)
}

func TestUpdateMetrics_EmptyBatch(t *testing.T) {
	uc := &mockUseCase{}
	svc := NewMetricService(uc, nil)

	resp, err := svc.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, uc.called)
	assert.Empty(t, uc.metrics)
}

func TestUpdateMetrics_InvalidMetric(t *testing.T) {
	uc := &mockUseCase{}
	svc := NewMetricService(uc, nil)

	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "", Type: pb.Metric_GAUGE, Payload: &pb.Metric_Value{Value: 1.0}}, // empty id
		},
	}

	resp, err := svc.UpdateMetrics(context.Background(), req)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, uc.called, "usecase should not be called on invalid input")
}

func TestUpdateMetrics_UseCaseError(t *testing.T) {
	uc := &mockUseCase{err: errors.New("db connection lost")}
	svc := NewMetricService(uc, nil)

	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "cpu", Type: pb.Metric_GAUGE, Payload: &pb.Metric_Value{Value: 1.0}},
		},
	}

	resp, err := svc.UpdateMetrics(context.Background(), req)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// --- MetricService audit tests ---

// fakeAuditPublisher фиксирует опубликованные события аудита.
type fakeAuditPublisher struct {
	events []entity.AuditEvent
}

func (f *fakeAuditPublisher) Publish(e entity.AuditEvent) { f.events = append(f.events, e) }
func (f *fakeAuditPublisher) Close(context.Context) error { return nil }

// TestUpdateMetrics_PublishesAudit проверяет полный путь publishAudit:
// при наличии паблишера событие публикуется с именами метрик, IP из metadata
// и транспортом "grpc".
func TestUpdateMetrics_PublishesAudit(t *testing.T) {
	uc := &mockUseCase{}
	audit := &fakeAuditPublisher{}
	svc := NewMetricService(uc, audit)

	md := metadata.New(map[string]string{"x-real-ip": "203.0.113.42"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "cpu", Type: pb.Metric_GAUGE, Payload: &pb.Metric_Value{Value: 1.0}},
			{Id: "hits", Type: pb.Metric_COUNTER, Payload: &pb.Metric_Delta{Delta: 2}},
		},
	}

	_, err := svc.UpdateMetrics(ctx, req)
	require.NoError(t, err)

	require.Len(t, audit.events, 1)
	ev := audit.events[0]
	assert.Equal(t, []string{"cpu", "hits"}, ev.Metrics)
	assert.Equal(t, "203.0.113.42", ev.IPAddress)
	assert.Equal(t, "grpc", ev.Transport)
}

// TestUpdateMetrics_EmptyBatchWithAudit: при пустом батче publishAudit
// должен сделать ранний выход и НЕ публиковать событие, даже если паблишер задан.
func TestUpdateMetrics_EmptyBatchWithAudit(t *testing.T) {
	uc := &mockUseCase{}
	audit := &fakeAuditPublisher{}
	svc := NewMetricService(uc, audit)

	_, err := svc.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})
	require.NoError(t, err)

	assert.Empty(t, audit.events, "no audit event for empty batch")
}
