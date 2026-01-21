package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"mangahub/internal/core"
	pb "mangahub/internal/protocols/grpc/pb"
	httpProtocol "mangahub/internal/protocols/http"
	grpcProtocol "mangahub/internal/protocols/grpc"
	tcpProtocol "mangahub/internal/protocols/tcp"
	udpProtocol "mangahub/internal/protocols/udp"
	wsProtocol "mangahub/internal/protocols/websocket"
	"mangahub/internal/repository"
	"mangahub/pkg/config"
	"mangahub/pkg/database"
	"mangahub/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load("./configs/development.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	loggerCfg := logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	}
	logger.Init(loggerCfg)

	logger.Info("Starting Mangahub server...")

	// Connect to database
	dbCfg := database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		Timeout:         cfg.Database.Timeout,
	}

	pool, err := database.NewPGXPool(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	logger.Info("Connected to PostgreSQL database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(pool)
	mangaRepo := repository.NewMangaRepository(pool)
	commentRepo := repository.NewCommentRepository(pool)
	chatRepo := repository.NewChatRepository(pool)
	activityRepo := repository.NewActivityRepository(pool)
	statsRepo := repository.NewStatsRepository(pool)
	notificationRepo := repository.NewNotificationRepository(pool)

	logger.Info("Initialized all repositories")

	// Initialize core services
	authSvc := core.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.Expiration)
	mangaSvc := core.NewMangaService(mangaRepo)
	commentSvc := core.NewCommentService(commentRepo, userRepo)
	chatSvc := core.NewChatService(chatRepo, userRepo)
	activitySvc := core.NewActivityService(activityRepo)
	statsSvc := core.NewStatsService(statsRepo, mangaRepo)

	logger.Info("Initialized all core services")

	// Create protocol servers
	// 1. HTTP REST API Server
	httpServer := httpProtocol.NewServer(
		cfg,
		authSvc,
		mangaSvc,
		commentSvc,
		chatSvc,
		activitySvc,
		statsSvc,
	)

	// 2. gRPC Search Server
	grpcServer := grpc.NewServer()
	grpcSearchSvc := grpcProtocol.NewMangaServiceServer(pool, mangaRepo, statsRepo)
	pb.RegisterMangaServiceServer(grpcServer, grpcSearchSvc)

	// 3. WebSocket Chat Server
	wsHub := wsProtocol.NewHub(chatRepo, activityRepo)
	wsHandler := wsProtocol.NewHandler(
		wsHub,
		authSvc,
		mangaRepo,
		chatRepo,
		activityRepo,
		statsSvc,
		[]string{"*"},
	)

	// Register WebSocket routes on HTTP server
	httpServer.Router().GET("/ws/manga/:manga_id", wsHandler.HandleWebSocket)
	httpServer.Router().GET("/ws/manga/:manga_id/status", wsHandler.GetRoomStatus)

	// 4. UDP Notification Server
	udpServer := udpProtocol.NewServer(cfg.UDP.Host, cfg.UDP.Port, notificationRepo)

	// 5. TCP Stats Aggregator Server
	tcpServer := tcpProtocol.NewServer(cfg.TCP.Host, cfg.TCP.Port, statsRepo, activityRepo)

	// CROSS-PROTOCOL INTEGRATION: Wire up server references
	tcpAddr := fmt.Sprintf("%s:%d", cfg.TCP.Host, cfg.TCP.Port)
	httpServer.SetCrossProtocolServers(udpServer, tcpAddr)
	wsHub.SetStatsAddr(tcpAddr)

	logger.Info("Cross-protocol event flows configured")

	// Start all protocol servers concurrently with panic recovery

	// Start HTTP server
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("HTTP server panic recovered: %v", r)
			}
		}()
		httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info(fmt.Sprintf("Starting HTTP server on %s", httpAddr))
		if err := httpServer.Start(httpAddr); err != nil {
			logger.Errorf("HTTP server error (non-fatal): %v", err)
		}
	}()

	// Start gRPC server
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("gRPC server panic recovered: %v", r)
			}
		}()
		grpcAddr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		listener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.Errorf("gRPC listen error (non-fatal): %v", err)
			return
		}
		logger.Info(fmt.Sprintf("Starting gRPC server on %s", grpcAddr))
		if err := grpcServer.Serve(listener); err != nil {
			logger.Errorf("gRPC server error (non-fatal): %v", err)
		}
	}()

	// Start UDP notification server (optional for production)
	if os.Getenv("ENABLE_UDP") != "false" {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("UDP server panic recovered: %v", r)
				}
			}()
			logger.Info(fmt.Sprintf("Starting UDP server on %s:%d", cfg.UDP.Host, cfg.UDP.Port))
			if err := udpServer.Start(); err != nil {
				logger.Errorf("UDP server error (non-fatal): %v", err)
			}
		}()
	} else {
		logger.Info("UDP server disabled (ENABLE_UDP=false)")
	}

	// Start TCP stats aggregator (optional for production)
	if os.Getenv("ENABLE_TCP") != "false" {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("TCP server panic recovered: %v", r)
				}
			}()
			logger.Info(fmt.Sprintf("Starting TCP server on %s:%d", cfg.TCP.Host, cfg.TCP.Port))
			if err := tcpServer.Start(); err != nil {
				logger.Errorf("TCP server error (non-fatal): %v", err)
			}
		}()
	} else {
		logger.Info("TCP server disabled (ENABLE_TCP=false)")
	}

	logger.Info("All protocol servers started successfully")
	logger.Info("Press Ctrl+C to shutdown")

	// Wait for shutdown signal only (servers handle their own errors)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal
	sig := <-sigChan
	logger.Info(fmt.Sprintf("Received signal: %v", sig))

	// Graceful shutdown
	logger.Info("Shutting down servers...")

	// Stop protocol servers
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop gRPC server
	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped")

	// Stop UDP server
	udpServer.Stop()
	logger.Info("UDP server stopped")

	// Stop TCP server
	tcpServer.Stop()
	logger.Info("TCP server stopped")

	// HTTP server (would need shutdown method, but Gin doesn't support it easily)
	// For now, it will be terminated when the program exits

	// Wait a moment for cleanup
	<-shutdownCtx.Done()

	logger.Info("Shutdown complete")
}
