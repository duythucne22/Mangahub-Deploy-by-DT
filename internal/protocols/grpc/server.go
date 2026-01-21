package grpc

import (
	"context"
	"fmt"
	"net"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "mangahub/internal/protocols/grpc/pb"
	"mangahub/internal/repository"
)

// Server represents the gRPC server
type Server struct {
	server *grpc.Server
	port   int
	health *health.Server
	stop   chan struct{}
}

// NewServer creates a new gRPC server with all required services
func NewServer(
	port int,
	pool *pgxpool.Pool,
	mangaRepo repository.MangaRepository,
	statsRepo repository.StatsRepository,
) *Server {
	// Setup logging
	logger := logrus.New()
	grpcLogger := logrus.NewEntry(logger)

	// Create health server
	healthServer := health.NewServer()
	healthServer.SetServingStatus("mangahub.v1.MangaService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Create gRPC server with middleware
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_logging.UnaryServerInterceptor(grpcLogger),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_logging.StreamServerInterceptor(grpcLogger),
			grpc_recovery.StreamServerInterceptor(),
		)),
	)

	// Register services
	mangaService := NewMangaServiceServer(pool, mangaRepo, statsRepo)
	pb.RegisterMangaServiceServer(server, mangaService)
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	reflection.Register(server)

	return &Server{
		server: server,
		port:   port,
		health: healthServer,
		stop:   make(chan struct{}),
	}
}

// Start begins listening for gRPC connections
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	go func() {
		logrus.Infof("✅ gRPC server starting on port %d", s.port)
		if err := s.server.Serve(listener); err != nil {
			logrus.Errorf("gRPC server stopped: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	logrus.Info("🛑 gRPC server stopping...")
	s.health.Shutdown()
	s.server.GracefulStop()
	close(s.stop)
	logrus.Info("✅ gRPC server stopped")
}

// WaitForShutdown blocks until server is stopped
func (s *Server) WaitForShutdown(ctx context.Context) {
	select {
	case <-ctx.Done():
		s.Stop()
	case <-s.stop:
	}
}