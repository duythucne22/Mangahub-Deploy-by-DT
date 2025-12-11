// Package main - TCP Synchronization Server
// Điểm vào cho TCP server dùng để đồng bộ dữ liệu real-time
// Chức năng:
//   - Lắng nghe kết nối TCP từ nhiều clients
//   - Broadcast progress updates đến tất cả clients đã kết nối
//   - Xử lý concurrent connections với goroutines
//   - JSON message protocol cho communication
//
// Port: 9090
package main

import (
	"os"
	"os/signal"
	"syscall"

	"mangahub/internal/tcp"
	"mangahub/pkg/config"
	"mangahub/pkg/logger"
)

func main() {
	cfg, err := config.Load("./configs/development.yaml")
	if err != nil {
		panic(err)
	}

	logger.Init(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})

	server := tcp.NewProgressSyncServer(cfg.TCP.Host, cfg.TCP.Port)

	go func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("TCP server error: %v", err)
		}
	}()

	logger.Infof("TCP Progress Sync Server started on %s:%d", cfg.TCP.Host, cfg.TCP.Port)

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down TCP server...")
	if err := server.Stop(); err != nil {
		logger.Errorf("error stopping TCP server: %v", err)
	}
	logger.Info("TCP server stopped.")
}
