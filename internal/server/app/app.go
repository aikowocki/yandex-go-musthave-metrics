package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/usecase"
	"go.uber.org/zap"
)

type ServerApp struct {
	server *http.Server
	closer func()
}

func (a *ServerApp) Close() {
	a.closer()
}

func NewServerApp(ctx context.Context, cfg *config.ServerConfig) (*ServerApp, error) {
	storage, err := initStorage(ctx, cfg)

	if err != nil {
		return nil, err
	}

	auditPublisher, auditClose, err := initAudit(cfg)
	if err != nil {
		storage.closer()
		return nil, err
	}

	metricUseCase := usecase.NewMetricUseCase(storage.repos.MetricRepo())
	metricHandler := handler.NewMetricHandler(metricUseCase, auditPublisher)
	metricJSONHandler := handler.NewMetricJSONHandler(metricUseCase, auditPublisher)

	healthHandler := handler.NewHealthHandler(storage.pinger)

	r := handler.NewRouter(metricHandler, metricJSONHandler, healthHandler, cfg.Key)

	zap.S().Infow(
		"Server starting",
		"address", cfg.ServerAddress,
	)

	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	closer := func() {
		auditClose()
		storage.closer()
	}

	return &ServerApp{server: srv, closer: closer}, nil
}

func (a *ServerApp) Run(ctx context.Context) {
	zap.S().Infow("server starting", "address", a.server.Addr)
	if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		zap.S().Fatalw("server failed", "error", err)
	}
}

func (a *ServerApp) Shutdown(ctx context.Context) error {

	return a.server.Shutdown(ctx)
}
