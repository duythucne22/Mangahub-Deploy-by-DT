package udp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// Notification represents a UDP notification (schema-aligned)
type Notification struct {
	Message   string    `json:"message"`      // Matches notifications.message field
	Timestamp time.Time `json:"timestamp"`    // Matches notifications.created_at
	Type      string    `json:"type"`         // Internal type for routing (not in schema)
	MangaID   *string   `json:"manga_id,omitempty"` // For context, not stored in schema
	Title     string    `json:"title,omitempty"`    // Optional display hint (not stored in schema)
}

// Server manages UDP notification broadcasting
type Server struct {
	addr        string
	conn        *net.UDPConn
	broadcast   chan Notification
	stop        chan struct{}
	rateLimiter *rate.Limiter // Rate limiter: 100 packets/second
	notificationRepo repository.NotificationRepository
	stats       struct {
		mu              sync.RWMutex
		packetsReceived uint64
		packetsDropped  uint64
		packetsSent     uint64
	}
	broadcastAddr *net.UDPAddr // Single broadcast address (no client registration)
}

const (
	maxPacketSize    = 1024                  // 1KB max packet size (SPEC.md requirement)
	packetsPerSecond = 100                   // Rate limit: 100 packets/sec (SPEC.md requirement)
	burstSize        = 50                    // Burst allowance
)

// NewServer creates a new UDP notification server
func NewServer(host string, port int, notificationRepo repository.NotificationRepository) *Server {
	// Resolve broadcast address (all clients listen on same port)
	broadcastAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", "255.255.255.255", port))

	return &Server{
		addr:             fmt.Sprintf("%s:%d", host, port),
		broadcast:        make(chan Notification, 256),
		stop:             make(chan struct{}),
		rateLimiter:      rate.NewLimiter(rate.Limit(packetsPerSecond), burstSize),
		notificationRepo: notificationRepo,
		broadcastAddr:    broadcastAddr,
	}
}

// Start starts the UDP notification server
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return fmt.Errorf("resolve udp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}

	// Enable broadcast on socket
	if err := conn.SetWriteBuffer(maxPacketSize); err != nil {
		fmt.Printf("⚠️ UDP: Failed to set write buffer: %v\n", err)
	}

	s.conn = conn
	fmt.Printf("✅ UDP Notification Server started on %s (broadcast mode)\n", s.addr)

	// Start broadcast goroutine
	go s.broadcastLoop()

	// Start database polling for notifications
	go s.pollDatabaseForNotifications()

	return nil
}

// Stop stops the UDP server
func (s *Server) Stop() {
	fmt.Println("🛑 UDP Notification Server stopping...")
	close(s.stop)
	if s.conn != nil {
		s.conn.Close()
	}
	fmt.Println("✅ UDP Notification Server stopped")
}

// broadcastLoop sends notifications via UDP broadcast
func (s *Server) broadcastLoop() {
	fmt.Println("🔊 UDP broadcast loop started")

	for {
		select {
		case notification := <-s.broadcast:
			if err := s.broadcastNotification(notification); err != nil {
				fmt.Printf("❌ UDP broadcast error: %v\n", err)
			}
		case <-s.stop:
			return
		}
	}
}

// broadcastNotification sends a single notification via UDP broadcast
func (s *Server) broadcastNotification(notification Notification) error {
	// Rate limiting (SPEC.md requirement: 100 packets/second)
	if !s.rateLimiter.Allow() {
		s.stats.mu.Lock()
		s.stats.packetsDropped++
		s.stats.mu.Unlock()
		return fmt.Errorf("rate limit exceeded")
	}

	// Set timestamp if not set
	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}

	// Marshal to JSON
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	// VALIDATION: Check packet size (SPEC.md requirement: max 1KB)
	if len(data) > maxPacketSize {
		fmt.Printf("⚠️ UDP: Notification too large (%d bytes), truncating\n", len(data))
		data = data[:maxPacketSize]
	}

	// Send broadcast packet (fire-and-forget)
	s.conn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
	_, err = s.conn.WriteToUDP(data, s.broadcastAddr)
	if err != nil {
		// Don't fail on send errors - UDP is fire-and-forget
		fmt.Printf("_UDP send error (ignored): %v\n", err)
	}

	// Update stats
	s.stats.mu.Lock()
	s.stats.packetsSent++
	s.stats.mu.Unlock()

	// Log successful broadcast for debugging
	fmt.Printf("📤 UDP broadcasted: '%s' (%d bytes)\n", 
		notification.Message[:min(len(notification.Message), 50)], len(data))

	return nil
}

// pollDatabaseForNotifications polls database for new notifications
func (s *Server) pollDatabaseForNotifications() {
	fmt.Println("🔍 UDP database polling started")

	var lastID string
	ctx := context.Background()

	for {
		select {
		case <-s.stop:
			return
		case <-time.After(1 * time.Second): // Poll every second
			// Get new notifications since last check
			notifications, err := s.notificationRepo.GetNewNotifications(ctx, lastID)
			if err != nil {
				fmt.Printf("❌ UDP database error: %v\n", err)
				continue
			}

			for _, notif := range notifications {
				// Convert database notification to broadcast format
				broadcastNotif := Notification{
					Message:   notif.Message,
					Timestamp: notif.CreatedAt,
					Type:      "system", // Default type for database notifications
				}

				// Broadcast to all clients without re-logging to DB
				select {
				case s.broadcast <- broadcastNotif:
				default:
					s.stats.mu.Lock()
					s.stats.packetsDropped++
					s.stats.mu.Unlock()
					fmt.Println("❌ UDP: Broadcast channel full, dropping notification")
				}

				// Update last ID for next poll
				if notif.ID > lastID {
					lastID = notif.ID
				}
			}
		}
	}
}

// Broadcast queues a notification for broadcast (fire-and-forget)
func (s *Server) Broadcast(notification Notification) {
	// Log notification to database first (SPEC.md requirement)
	if notification.Message != "" {
		notif := &models.Notification{
			ID:        generateNotificationID(),
			Message:   notification.Message,
			CreatedAt: notification.Timestamp,
		}
		
		ctx := context.Background()
		if err := s.notificationRepo.Create(ctx, notif); err != nil {
			fmt.Printf("❌ UDP database log error: %v\n", err)
		}
	}

	// Queue for broadcast (non-blocking)
	select {
	case s.broadcast <- notification:
	default:
		// Channel full, drop notification (fire-and-forget semantics per SPEC.md)
		s.stats.mu.Lock()
		s.stats.packetsDropped++
		s.stats.mu.Unlock()
		fmt.Println("❌ UDP: Broadcast channel full, dropping notification")
	}
}

// SendSystemNotification sends a system notification (admin use)
func (s *Server) SendSystemNotification(message string) {
	s.Broadcast(Notification{
		Message:   message,
		Timestamp: time.Now(),
		Type:      "system",
	})
}

// SendMangaUpdateNotification sends notification for manga updates
func (s *Server) SendMangaUpdateNotification(mangaID, title string) {
	msg := fmt.Sprintf("📚 New manga: %s", title)
	
	s.Broadcast(Notification{
		Message:   msg,
		Timestamp: time.Now(),
		Type:      "manga_update",
		MangaID:   &mangaID,
	})
}

// GetStats returns server statistics
func (s *Server) GetStats() (received, dropped, sent uint64) {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	return s.stats.packetsReceived, s.stats.packetsDropped, s.stats.packetsSent
}

// Helper function to generate notification ID
func generateNotificationID() string {
	return fmt.Sprintf("notif-%d", time.Now().UnixNano())
}

// Helper function for string truncation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}