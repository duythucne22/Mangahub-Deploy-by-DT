package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// EventType represents type of activity event (MUST match schema activity_feed.type)
type EventType string

const (
	EventTypeComment EventType = "comment"       // Matches schema: CHECK (type IN ('comment', 'chat', 'manga_update'))
	EventTypeChat    EventType = "chat"          // Matches schema activity_feed.type
	EventTypeUpdate  EventType = "manga_update"  // Matches schema activity_feed.type
)

// StatsEvent represents a stats aggregation event (schema-aligned)
type StatsEvent struct {
	Type      EventType `json:"type"`           // Must match schema activity_feed.type
	MangaID   string    `json:"manga_id"`       // Matches manga_stats.manga_id FK
	UserID    *string   `json:"user_id"`        // Nullable like activity_feed.user_id
	EventTime time.Time `json:"event_time"`     // Timestamp for scoring
	Weight    int       `json:"weight"`         // Scoring weight (comment=1, chat=2, update=5)
	Source    string    `json:"source"`         // "http", "websocket", "admin"
}

// Server manages TCP stats aggregation server
type Server struct {
	addr      string
	listener  net.Listener
	statsRepo repository.StatsRepository
	activityRepo repository.ActivityRepository
	connMu    sync.Mutex
	stop      chan struct{}
	stopped   chan struct{}
}

// NewServer creates a new TCP stats aggregator server
func NewServer(host string, port int, statsRepo repository.StatsRepository, activityRepo repository.ActivityRepository) *Server {
	return &Server{
		addr:        fmt.Sprintf("%s:%d", host, port),
		statsRepo:   statsRepo,
		activityRepo: activityRepo,
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
	}
}

// Start starts the TCP stats aggregator server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("tcp listen failed on %s: %w", s.addr, err)
	}

	s.listener = listener
	fmt.Printf("✅ TCP Stats Aggregator started on %s\n", s.addr)

	go s.acceptLoop()
	return nil
}

// Stop stops the TCP server gracefully
func (s *Server) Stop() {
	fmt.Println("🛑 TCP Stats Aggregator stopping...")

	s.connMu.Lock()
	if s.listener != nil {
		s.listener.Close()
	}
	s.connMu.Unlock()

	close(s.stop)

	// Wait for accept loop to exit
	select {
	case <-s.stopped:
		fmt.Println("✅ TCP Stats Aggregator stopped cleanly")
	case <-time.After(5 * time.Second):
		fmt.Println("⚠️ TCP Stats Aggregator forced stop after timeout")
	}
}

// acceptLoop accepts incoming TCP connections
func (s *Server) acceptLoop() {
	defer close(s.stopped)

	fmt.Println("👂 TCP Stats Aggregator accepting connections...")

	for {
		select {
		case <-s.stop:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stop:
					return
				default:
					if !isTemporaryError(err) {
						fmt.Printf("❌ TCP accept error: %v\n", err)
					}
					continue
				}
			}

			// Set connection deadlines
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			clientAddr := conn.RemoteAddr().String()
			fmt.Printf("🔌 TCP client connected from %s\n", clientAddr)

			go s.handleConnection(conn, clientAddr)
		}
	}
}

// handleConnection processes a TCP connection with custom protocol framing
func (s *Server) handleConnection(conn net.Conn, clientAddr string) {
	defer func() {
		conn.Close()
		fmt.Printf("🔌 TCP client disconnected: %s\n", clientAddr)
	}()

	reader := bufio.NewReader(conn)
	ctx := context.Background()

	for {
		select {
		case <-s.stop:
			return
		default:
			// Read 4-byte length prefix (big-endian)
			var length uint32
			if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
				if err != io.EOF {
					fmt.Printf("❌ TCP read length error from %s: %v\n", clientAddr, err)
				}
				return
			}

			// Validation: Check frame size (prevent attacks)
			const maxFrameSize = 1024 // 1KB max per SPEC.md for internal events
			if length == 0 {
				fmt.Printf("❌ TCP invalid frame length 0 from %s\n", clientAddr)
				return
			}
			if length > maxFrameSize {
				fmt.Printf("❌ TCP frame too large from %s: %d bytes (max %d)\n", 
					clientAddr, length, maxFrameSize)
				return
			}

			// Read message data
			data := make([]byte, length)
			if _, err := io.ReadFull(reader, data); err != nil {
				fmt.Printf("❌ TCP read data error from %s: %v\n", clientAddr, err)
				return
			}

			// Parse event
			var event StatsEvent
			if err := json.Unmarshal(data, &event); err != nil {
				fmt.Printf("❌ TCP parse error from %s: %v\n", clientAddr, err)
				// Send error response back to client
				s.sendError(conn, fmt.Sprintf("Invalid event format: %v", err))
				continue
			}

			// Fill defaults for missing fields
			if event.EventTime.IsZero() {
				event.EventTime = time.Now()
			}
			if event.Source == "" {
				event.Source = "system"
			}
			if event.Weight == 0 {
				switch event.Type {
				case EventTypeComment:
					event.Weight = 1
				case EventTypeChat:
					event.Weight = 2
				case EventTypeUpdate:
					event.Weight = 5
				default:
					event.Weight = 1
				}
			}

			// Validate event
			if err := s.validateEvent(&event, clientAddr); err != nil {
				fmt.Printf("❌ TCP validation error from %s: %v\n", clientAddr, err)
				s.sendError(conn, err.Error())
				continue
			}

			// Process event
			if err := s.processEvent(ctx, &event); err != nil {
				fmt.Printf("❌ TCP processing error for event %s from %s: %v\n", 
					event.Type, clientAddr, err)
				s.sendError(conn, fmt.Sprintf("Processing failed: %v", err))
				continue
			}

			// Send success acknowledgment
			s.sendSuccess(conn, "Event processed successfully")
		}
	}
}

// validateEvent validates incoming stats events against schema constraints
func (s *Server) validateEvent(event *StatsEvent, clientAddr string) error {
	// Validate event type against schema CHECK constraints
	validTypes := map[EventType]bool{
		EventTypeComment: true,
		EventTypeChat:    true,
		EventTypeUpdate:  true,
	}
	
	if !validTypes[event.Type] {
		return fmt.Errorf("invalid event type: %s (must be 'comment', 'chat', or 'manga_update')", event.Type)
	}
	
	// Validate manga_id exists (basic check)
	if event.MangaID == "" {
		return fmt.Errorf("manga_id is required")
	}
	
	// Validate weight ranges
	if event.Weight < 1 || event.Weight > 10 {
		return fmt.Errorf("invalid weight: %d (must be 1-10)", event.Weight)
	}
	
	// Validate source types
	validSources := map[string]bool{
		"http":      true,
		"websocket": true,
		"admin":     true,
		"system":    true,
	}
	
	if !validSources[event.Source] {
		return fmt.Errorf("invalid source: %s (must be 'http', 'websocket', 'admin', or 'system')", event.Source)
	}
	
	return nil
}

// processEvent handles a stats event with atomic database operations
func (s *Server) processEvent(ctx context.Context, event *StatsEvent) error {
	fmt.Printf("📊 Processing event: type=%s manga_id=%s weight=%d source=%s\n",
		event.Type, event.MangaID, event.Weight, event.Source)

	// 1. Log to activity feed first (for audit trail)
	// Avoid duplicates when HTTP layer already logs activity
	if event.Source != "http" {
		activity := &models.Activity{
			ID:        generateActivityID(),
			Type:      string(event.Type),
			UserID:    event.UserID,
			MangaID:   &event.MangaID,
			CreatedAt: event.EventTime,
		}
		
		if err := s.activityRepo.Create(ctx, activity); err != nil {
			return fmt.Errorf("failed to log activity: %w", err)
		}
	}

	// 2. Update stats based on event type
	switch event.Type {
	case EventTypeComment:
		return s.processCommentEvent(ctx, event)
	case EventTypeChat:
		return s.processChatEvent(ctx, event)
	case EventTypeUpdate:
		return s.processUpdateEvent(ctx, event)
	default:
		// Default to comment processing for unknown types
		return s.processCommentEvent(ctx, event)
	}
}

// processCommentEvent handles comment-related statistics
func (s *Server) processCommentEvent(ctx context.Context, event *StatsEvent) error {
	// Atomic comment count increment
	if err := s.statsRepo.IncrementCommentCount(ctx, event.MangaID); err != nil {
		return fmt.Errorf("failed to increment comment count: %w", err)
	}
	
	// Update weekly score with comment weight
	if event.Weight > 0 {
		// Get current stats to calculate new score
		stats, err := s.statsRepo.GetByMangaID(ctx, event.MangaID)
		if err != nil {
			return fmt.Errorf("failed to get stats: %w", err)
		}
		
		newScore := stats.WeeklyScore + event.Weight
		if err := s.statsRepo.UpdateWeeklyScore(ctx, event.MangaID, newScore); err != nil {
			return fmt.Errorf("failed to update weekly score: %w", err)
		}
	}
	
	return nil
}

// processChatEvent handles chat-related statistics
func (s *Server) processChatEvent(ctx context.Context, event *StatsEvent) error {
	// Atomic chat count increment (weighted higher than comments)
	if err := s.statsRepo.IncrementChatCount(ctx, event.MangaID); err != nil {
		return fmt.Errorf("failed to increment chat count: %w", err)
	}

	return nil
}

// processUpdateEvent handles manga update statistics (admin actions)
func (s *Server) processUpdateEvent(ctx context.Context, event *StatsEvent) error {
	// Admin updates get bonus points
	stats, err := s.statsRepo.GetByMangaID(ctx, event.MangaID)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}
	
	// Admin updates get high weight (5x normal comments)
	newScore := stats.WeeklyScore + event.Weight
	if err := s.statsRepo.UpdateWeeklyScore(ctx, event.MangaID, newScore); err != nil {
		return fmt.Errorf("failed to update weekly score: %w", err)
	}
	
	return nil
}

// sendError sends an error response back to client
func (s *Server) sendError(conn net.Conn, message string) {
	response := map[string]string{
		"status":  "error",
		"message": message,
	}
	
	s.sendResponse(conn, response)
}

// sendSuccess sends a success response back to client
func (s *Server) sendSuccess(conn net.Conn, message string) {
	response := map[string]string{
		"status":  "success",
		"message": message,
	}
	
	s.sendResponse(conn, response)
}

// sendResponse sends a JSON response with proper framing
func (s *Server) sendResponse(conn net.Conn, data interface{}) {
	responseBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("❌ TCP response marshal error: %v\n", err)
		return
	}
	
	// Write length prefix
	if err := binary.Write(conn, binary.BigEndian, uint32(len(responseBytes))); err != nil {
		fmt.Printf("❌ TCP response length write error: %v\n", err)
		return
	}
	
	// Write response data
	if _, err := conn.Write(responseBytes); err != nil {
		fmt.Printf("❌ TCP response write error: %v\n", err)
	}
}

// SendStatsEvent is a helper for other services to send stats events
// This should be used by HTTP, WebSocket, and admin services
func SendStatsEvent(addr string, event StatsEvent) error {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("dial tcp: %w", err)
	}
	defer conn.Close()
	
	// Set deadlines
	conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	
	// Marshal event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	
	// Write length prefix
	length := uint32(len(data))
	if err := binary.Write(conn, binary.BigEndian, length); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	
	// Write data
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	
	// Read response (optional, but good for error handling)
	var responseLength uint32
	if err := binary.Read(conn, binary.BigEndian, &responseLength); err != nil {
		return fmt.Errorf("read response length: %w", err)
	}
	
	responseData := make([]byte, responseLength)
	if _, err := io.ReadFull(conn, responseData); err != nil {
		return fmt.Errorf("read response data: %w", err)
	}
	
	var response map[string]string
	if err := json.Unmarshal(responseData, &response); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	
	if response["status"] != "success" {
		return fmt.Errorf("server error: %s", response["message"])
	}
	
	return nil
}

// generateActivityID creates a unique activity ID (simplified for example)
func generateActivityID() string {
	return fmt.Sprintf("act-%d", time.Now().UnixNano())
}

// isTemporaryError checks if an error is temporary
func isTemporaryError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary()
	}
	return false
}