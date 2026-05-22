package app

import (
	"context"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/audit"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"go.uber.org/zap"
)

func initAudit(cfg *config.ServerConfig) (port.AuditPublisher, func(), error) {
	var observers []port.AuditObserver
	var fileObs *audit.FileObserver
	var err error

	if cfg.AuditFile != "" {
		fileObs, err = audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, nil, err
		}
		observers = append(observers, fileObs)
	}
	if cfg.AuditURL != "" {
		observers = append(observers, audit.NewHTTPObserver(cfg.AuditURL))
	}
	if len(observers) == 0 {
		return nil, func() {}, nil
	}

	publisher := audit.NewPublisher(observers...)
	closer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := publisher.Close(ctx); err != nil {
			zap.S().Warnw("audit publisher close", "err", err)
		}
		if fileObs != nil {
			if err := fileObs.Close(); err != nil {
				zap.S().Warnw("audit file close", "err", err)
			}
		}
	}
	return publisher, closer, nil
}
