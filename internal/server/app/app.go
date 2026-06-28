package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/usecase"
	pkgcrypto "github.com/aikowocki/yandex-go-musthave-metrics/pkg/crypto"
	"go.uber.org/zap"
)

type ServerApp struct {
	server *http.Server
	closer func(context.Context)
}

// Close освобождает ресурсы приложения (audit publisher, storage),
// используя ctx как общий бюджет времени на graceful shutdown.
func (a *ServerApp) Close(ctx context.Context) {
	a.closer(ctx)
}

func NewServerApp(ctx context.Context, cfg *config.ServerConfig) (*ServerApp, error) {
	// Парсим доверенную подсеть до выделения ресурсов: невалидный CIDR должен
	// прерывать старт (fail-fast), а не молча отключать фильтрацию в рантайме.
	trustedSubnet, err := parseTrustedSubnet(cfg.TrustedSubnet)
	if err != nil {
		return nil, err
	}

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

	var routerOpts handler.RouterOptions
	if cfg.CryptoKey != "" {
		privateKey, cryptoErr := pkgcrypto.LoadPrivateKey(cfg.CryptoKey)
		if cryptoErr != nil {
			storage.closer()
			return nil, cryptoErr
		}
		routerOpts.CryptoPrivateKey = privateKey
		zap.S().Infow("RSA decryption enabled", "key", cfg.CryptoKey)
	}

	r := handler.NewRouter(metricHandler, metricJSONHandler, healthHandler, cfg.Key, trustedSubnet, routerOpts)

	zap.S().Infow(
		"Server starting",
		"address", cfg.ServerAddress,
	)

	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	closer := func(ctx context.Context) {
		// Сначала дожидаемся, пока фоновая горутина бэкапа выполнит финальный
		// Save (в рамках бюджета ctx), и только потом освобождаем ресурсы.
		if storage.waitBackup != nil {
			storage.waitBackup(ctx)
		}
		auditClose(ctx)
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

// parseTrustedSubnet разбирает CIDR из конфига в *net.IPNet.
// Пустая строка означает выключенную фильтрацию и возвращает (nil, nil).
// Непустое, но некорректное значение — ошибка, прерывающая старт сервера.
func parseTrustedSubnet(cidr string) (*net.IPNet, error) {
	if cidr == "" {
		return nil, nil
	}
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid trusted_subnet %q: %w", cidr, err)
	}
	return subnet, nil
}
