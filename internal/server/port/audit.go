package port

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type AuditObserver interface {
	Notify(ctx context.Context, event entity.AuditEvent) error
	Name() string
}

type AuditPublisher interface {
	Publish(event entity.AuditEvent)
	Close(ctx context.Context) error
}
