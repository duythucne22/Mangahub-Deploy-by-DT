// Package main - UDP Notification Server
// Điểm vào cho UDP server dùng để gửi push notifications
// Chức năng:
//   - Nhận datagram từ clients (REGISTER/UNREGISTER)
//   - Gửi chapter release notifications đến subscribers
//   - Connectionless protocol - không cần maintain connections
//   - Broadcast notifications đến nhiều clients
//
// Port: 9091
package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"mangahub/internal/udp"
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

	server := udp.NewNotificationServer(cfg.UDP.Host, cfg.UDP.Port)

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("UDP server error: %v", err)
		}
	}()

	logger.Infof("UDP Notification Server started on %s:%d", cfg.UDP.Host, cfg.UDP.Port)

	// Demo: Send test notifications periodically
	go func() {
		time.Sleep(5 * time.Second) // Wait for clients to register
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				notification := udp.NewChapterNotification(
					"one-piece",
					"New chapter released: One Piece Chapter 1100!",
				)
				server.SendNotification(notification)
				logger.Info("Demo notification sent")
			}
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down UDP server...")
	if err := server.Stop(); err != nil {
		logger.Errorf("error stopping UDP server: %v", err)
	}
	logger.Info("UDP server stopped.")
}
