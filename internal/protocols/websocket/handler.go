package websocket

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"mangahub/internal/core"
	"mangahub/internal/repository"
)

// WebSocket close codes
const (
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseNoStatusReceived        = 1005
	CloseAbnormalClosure         = 1006
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerError     = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
		// Enable compression for better performance
		EnableCompression: true,
		// Handle subprotocols for TUI client optimization
		Subprotocols: []string{"mangahub.tui-v1", "mangahub.web-v1"},
	}
)

// Handler manages WebSocket connections for chat rooms
type Handler struct {
	hub          *Hub
	authSvc      core.AuthService
	mangaRepo    repository.MangaRepository
	chatRepo     repository.ChatRepository
	activityRepo repository.ActivityRepository
	statsSvc     core.StatsService
	allowedOrigins []string
	metrics      struct {
		sync.Mutex
		totalConnections uint64
		activeRooms      map[string]int
	}
}

// NewHandler creates a new WebSocket handler with proper dependencies
func NewHandler(
	hub *Hub,
	authSvc core.AuthService,
	mangaRepo repository.MangaRepository,
	chatRepo repository.ChatRepository,
	activityRepo repository.ActivityRepository,
	statsSvc core.StatsService,
	allowedOrigins []string,
) *Handler {
	if allowedOrigins == nil {
		allowedOrigins = []string{"*"}
	}

	handler := &Handler{
		hub:          hub,
		authSvc:      authSvc,
		mangaRepo:    mangaRepo,
		chatRepo:     chatRepo,
		activityRepo: activityRepo,
		statsSvc:     statsSvc,
		allowedOrigins: allowedOrigins,
	}
	handler.metrics.activeRooms = make(map[string]int)
	return handler
}

// HandleWebSocket upgrades HTTP connection to WebSocket for manga chat
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Validate manga ID from URL
	mangaID := c.Param("manga_id")
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id parameter is required"})
		return
	}

	// Validate manga exists in database
	ctx := c.Request.Context()
	_, err := h.mangaRepo.GetByID(ctx, mangaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found"})
		return
	}

	// Get and validate authentication token
	token, err := extractToken(c)
	if err != nil {
		h.sendWebSocketError(c, http.StatusUnauthorized, "authentication_required", err.Error())
		return
	}

	// Validate token and get user
	user, err := h.authSvc.ValidateToken(ctx, token)
	if err != nil {
		h.sendWebSocketError(c, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}

	// Get client info and terminal capabilities
	clientInfo := h.parseClientInfo(c)
	logrus.Infof("WebSocket connection attempt: manga_id=%s user_id=%s client=%s",
		mangaID, user.ID, clientInfo.UserAgent)

	// Upgrade HTTP connection to WebSocket with proper error handling
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade failed: %v", err)
		// NOTE: gorilla/websocket writes its own HTTP response (often 403) when CheckOrigin fails.
		// Writing JSON here can cause confusing double-write behavior, so just return.
		return
	}

	// Set connection deadlines and keepalive
	conn.SetReadLimit(1024) // 1KB max message size (SPEC.md requirement)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Update metrics
	h.updateMetrics(mangaID, true)

	// Register client with hub and start goroutines
	h.hub.ServeClient(conn, user.ID, user.Username, mangaID, func() {
		h.updateMetrics(mangaID, false)
	})

	logrus.Infof("✅ WebSocket client connected: manga_id=%s user_id=%s",
		mangaID, user.ID)
}

// GetRoomStatus returns status of a chat room
func (h *Handler) GetRoomStatus(c *gin.Context) {
	mangaID := c.Param("manga_id")
	if mangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id parameter is required"})
		return
	}

	clientCount := h.hub.GetRoomClientCount(mangaID)
	
	// Get recent chat messages for preview
	ctx := c.Request.Context()
	messages, _, err := h.chatRepo.ListByMangaID(ctx, mangaID, 5, 0) // Last 5 messages
	if err != nil {
		logrus.Warnf("Failed to get chat preview for manga %s: %v", mangaID, err)
	}

	// Get room presence information
	presence, err := h.hub.GetRoomPresence(mangaID)
	if err != nil {
		logrus.Warnf("Failed to get room presence for manga %s: %v", mangaID, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"manga_id":     mangaID,
		"client_count": clientCount,
		"active":       clientCount > 0,
		"messages":     messages,
		"online_users": presence,
	})
}

// GetGlobalStatus returns global WebSocket statistics
func (h *Handler) GetGlobalStatus(c *gin.Context) {
	h.metrics.Lock()
	defer h.metrics.Unlock()
	
	// Get top active rooms
	topRooms := make([]gin.H, 0, 5)
	for mangaID, count := range h.metrics.activeRooms {
		if count > 0 {
			topRooms = append(topRooms, gin.H{
				"manga_id": mangaID,
				"clients":  count,
			})
		}
		if len(topRooms) >= 5 {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_connections": h.metrics.totalConnections,
		"active_rooms":      len(h.metrics.activeRooms),
		"top_active_rooms":  topRooms,
		"server_time":       time.Now().UTC(),
	})
}

// extractToken extracts authentication token from request
func extractToken(c *gin.Context) (string, error) {
	// Try query parameter first (for TUI clients)
	token := c.Query("token")
	if token != "" {
		return token, nil
	}

	// Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1], nil
		}
	}

	// Try cookie (for web clients)
	cookie, err := c.Request.Cookie("token")
	if err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	return "", fmt.Errorf("no authentication token provided")
}

// checkOrigin validates request origin against allowed origins
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// Non-browser clients may omit Origin; treat as allowed.
	if origin == "" {
		return true
	}

	// Always allow local development origins (TUI client).
	if u, err := url.Parse(origin); err == nil {
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" {
			return true
		}
	}

	// In development, allow all origins
	if gin.Mode() == gin.DebugMode {
		return true
	}

	// Production: strict origin checking
	allowed := []string{"https://mangahub.example.com", "https://app.mangahub.example.com"}
	for _, allowedOrigin := range allowed {
		if origin == allowedOrigin {
			return true
		}
	}
	return false
}

// sendWebSocketError sends proper WebSocket error with logging
func (h *Handler) sendWebSocketError(c *gin.Context, status int, code, message string) {
	logrus.Warnf("WebSocket error: status=%d code=%s message=%s",
		status, code, message)
	
	c.JSON(status, gin.H{
		"error": code,
		"message": message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// updateMetrics updates connection metrics
func (h *Handler) updateMetrics(mangaID string, connected bool) {
	h.metrics.Lock()
	defer h.metrics.Unlock()

	if connected {
		h.metrics.totalConnections++
		h.metrics.activeRooms[mangaID]++
	} else {
		if count, exists := h.metrics.activeRooms[mangaID]; exists {
			count--
			if count <= 0 {
				delete(h.metrics.activeRooms, mangaID)
			} else {
				h.metrics.activeRooms[mangaID] = count
			}
		}
	}
}

// ClientInfo represents client capabilities and metadata
type ClientInfo struct {
	UserAgent    string
	TerminalSize *TerminalSize
	ColorSupport bool
	Protocol     string // "tui-v1", "web-v1"
}

// TerminalSize represents terminal dimensions
type TerminalSize struct {
	Width  int
	Height int
}

// parseClientInfo extracts client information from request
func (h *Handler) parseClientInfo(c *gin.Context) *ClientInfo {
	userAgent := c.GetHeader("User-Agent")
	
	// Detect TUI client from user agent
	isTUI := strings.Contains(strings.ToLower(userAgent), "mangahub-tui") ||
	         strings.Contains(strings.ToLower(userAgent), "terminal")
	
	// Get terminal size from headers (TUI clients only)
	var termSize *TerminalSize
	if isTUI {
		width, _ := strconv.Atoi(c.GetHeader("X-Terminal-Width"))
		height, _ := strconv.Atoi(c.GetHeader("X-Terminal-Height"))
		if width > 0 && height > 0 {
			termSize = &TerminalSize{Width: width, Height: height}
		}
	}
	
	// Check color support
	colorSupport := true
	if strings.Contains(strings.ToLower(userAgent), "windows") {
		colorSupport = false
	}
	
	// Get negotiated protocol
	protocol := c.GetHeader("Sec-WebSocket-Protocol")
	if protocol == "" {
		protocol = "unknown"
	}
	
	return &ClientInfo{
		UserAgent:    userAgent,
		TerminalSize: termSize,
		ColorSupport: colorSupport,
		Protocol:     protocol,
	}
}
