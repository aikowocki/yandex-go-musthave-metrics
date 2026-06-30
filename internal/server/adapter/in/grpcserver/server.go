package grpcserver

import (
	"context"
	"net"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	pb "github.com/aikowocki/yandex-go-musthave-metrics/pkg/proto"
	"google.golang.org/grpc"
)

type Server struct {
	srv  *grpc.Server
	Addr string
}

func New(address string, uc MetricUseCase, audit port.AuditPublisher, subnet *net.IPNet) *Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.Recovery(),
			interceptor.Logging(),
			interceptor.TrustedSubnet(subnet),
		))
	pb.RegisterMetricsServer(srv, NewMetricService(uc, audit))
	return &Server{srv: srv, Addr: address}
}

func (s *Server) ListenAndServe() error {
	lis, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.srv.Serve(lis)
}

func (s *Server) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		s.srv.Stop()
	}
}
