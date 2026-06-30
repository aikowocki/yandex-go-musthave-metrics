package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/grpcserver"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/usecase"
	pkgcrypto "github.com/aikowocki/yandex-go-musthave-metrics/pkg/crypto"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type ServerApp struct {
	server     *http.Server
	gRPCServer *grpcserver.Server
	closer     func(context.Context)
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

	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	zap.S().Infow("HTTP server configured", "address", srv.Addr)

	// gRPC (опционально)
	var gRPCServer *grpcserver.Server
	if cfg.GRPCAddress != "" {
		gRPCServer = grpcserver.New(cfg.GRPCAddress, metricUseCase, auditPublisher, trustedSubnet)
		zap.S().Infow("gRPC server configured", "address", gRPCServer.Addr)
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

	return &ServerApp{server: srv, gRPCServer: gRPCServer, closer: closer}, nil
}

func (a *ServerApp) Run(ctx context.Context) {
	g, _ := errgroup.WithContext(ctx)

	//HTTP
	g.Go(func() error {
		zap.S().Infow("HTTP server starting", "address", a.server.Addr)

		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// gRPC server (опциональный)
	if a.gRPCServer != nil {
		g.Go(func() error {
			zap.S().Infow("gRPC server starting", "address", a.gRPCServer.Addr)
			return a.gRPCServer.ListenAndServe()
		})
	}

	if err := g.Wait(); err != nil {
		zap.S().Fatalw("server error", "error", err)
	}
}

func (a *ServerApp) Shutdown(ctx context.Context) error {
	// Гасим HTTP и gRPC параллельно в рамках общего бюджета ctx,
	// чтобы медленный GracefulStop одного не съедал время другого.
	var wg sync.WaitGroup
	if a.gRPCServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.gRPCServer.Shutdown(ctx)
		}()
	}

	err := a.server.Shutdown(ctx)
	wg.Wait()
	return err
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
