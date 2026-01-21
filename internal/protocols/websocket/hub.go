// Package websocket - WebSocket Chat Protocol Handler
// Implements real-time per-manga chat rooms with message persistence
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"mangahub/internal/repository"
	tcpProtocol "mangahub/internal/protocols/tcp"
	"mangahub/pkg/models"
)

// Constants for performance and limits
const (
	maxMessageSize    = 1024                  // 1KB max message size per SPEC.md
	writeWait         = 10 * time.Second      // Time allowed to write a message
	pongWait          = 60 * time.Second      // Time allowed to read the next pong
	pingPeriod        = (pongWait * 9) / 10   // Send pings to client
	historyLimit      = 50                     // Max chat history messages to send
	maxRoomSize       = 1000                   // Max clients per room
	cleanupInterval   = 5 * time.Minute        // Room cleanup interval
)

// Hub manages all chat rooms and client connections
type Hub struct {
	roomsMu   sync.RWMutex
	rooms     map[string]*Room // manga_id -> Room
	chatRepo  repository.ChatRepository
	activityRepo repository.ActivityRepository
	statsAddr string // TCP Stats Service address
	stop      chan struct{}
	wg        sync.WaitGroup
}

// Room represents a chat room for a specific manga
type Room struct {
	mangaID    string
	clientsMu  sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	stopped    bool
	stop       chan struct{}
}

// Client represents a WebSocket client connection
type Client struct {
	hub     *Hub
	room    *Room
	conn    *websocket.Conn
	send    chan *Message
	userID  string
	username string
	mangaID string
	lastActive time.Time
	onDisconnect func()
}

// Message represents a chat message (schema-aligned)
type Message struct {
	Type      string    `json:"type"`      // "message", "join", "leave", "history"
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	MangaID   string    `json:"manga_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// NewHub creates a new chat hub with dependencies
func NewHub(chatRepo repository.ChatRepository, activityRepo repository.ActivityRepository) *Hub {
	hub := &Hub{
		rooms:      make(map[string]*Room),
		chatRepo:   chatRepo,
		activityRepo: activityRepo,
		stop:       make(chan struct{}),
	}
	
	// Start room cleanup routine
	hub.wg.Add(1)
	go hub.cleanupRooms()
	
	return hub
}

// SetStatsAddr sets the TCP Stats Service address
func (h *Hub) SetStatsAddr(addr string) {
	h.statsAddr = addr
}

// cleanupRooms periodically removes empty rooms
func (h *Hub) cleanupRooms() {
	defer h.wg.Done()
	
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.roomsMu.Lock()
			for mangaID, room := range h.rooms {
				room.clientsMu.RLock()
				clientCount := len(room.clients)
				room.clientsMu.RUnlock()
				
				if clientCount == 0 {
					close(room.stop)
					delete(h.rooms, mangaID)
					logrus.Infof("🧹 Cleaned up empty room: %s", mangaID)
				}
			}
			h.roomsMu.Unlock()
			
		case <-h.stop:
			return
		}
	}
}

// GetOrCreateRoom returns existing room or creates new one for manga
func (h *Hub) GetOrCreateRoom(mangaID string) *Room {
	h.roomsMu.Lock()
	defer h.roomsMu.Unlock()

	if room, exists := h.rooms[mangaID]; exists {
		return room
	}

	room := &Room{
		mangaID:    mangaID,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
		stopped:    false,
	}

	h.rooms[mangaID] = room
	go room.run()
	
	logrus.Infof("🆕 Created new chat room: %s", mangaID)
	return room
}

// run handles room operations
func (r *Room) run() {
	for {
		select {
		case client := <-r.register:
			r.handleRegister(client)
		case client := <-r.unregister:
			r.handleUnregister(client)
		case message := <-r.broadcast:
			r.handleBroadcast(message)
		case <-r.stop:
			r.handleStop()
			return
		}
	}
}

// handleRegister processes client registration
func (r *Room) handleRegister(client *Client) {
	if r.stopped {
		return
	}
	
	r.clientsMu.Lock()
	if len(r.clients) >= maxRoomSize {
		r.clientsMu.Unlock()
		logrus.Warnf("Room %s full, rejecting client %s", r.mangaID, client.userID)
		return
	}
	
	r.clients[client] = true
	r.clientsMu.Unlock()

	logrus.Debugf("✅ Client %s joined room %s", client.userID, r.mangaID)

	// Send join notification
	joinMsg := &Message{
		Type:      "join",
		UserID:    client.userID,
		Username:  client.username,
		MangaID:   r.mangaID,
		Timestamp: time.Now(),
	}
	r.broadcastToAll(joinMsg)
}

// handleUnregister processes client unregistration
func (r *Room) handleUnregister(client *Client) {
	if r.stopped {
		return
	}
	
	r.clientsMu.Lock()
	if _, ok := r.clients[client]; ok {
		delete(r.clients, client)
		close(client.send)
	}
	r.clientsMu.Unlock()

	logrus.Debugf("👋 Client %s left room %s", client.userID, r.mangaID)

	// Send leave notification
	leaveMsg := &Message{
		Type:      "leave",
		UserID:    client.userID,
		Username:  client.username,
		MangaID:   r.mangaID,
		Timestamp: time.Now(),
	}
	r.broadcastToAll(leaveMsg)
}

// handleBroadcast processes message broadcast
func (r *Room) handleBroadcast(message *Message) {
	if r.stopped {
		return
	}
	r.broadcastToAll(message)
}

// handleStop cleans up room resources
func (r *Room) handleStop() {
	r.stopped = true
	
	r.clientsMu.Lock()
	for client := range r.clients {
		close(client.send)
		client.conn.Close()
	}
	r.clients = nil
	r.clientsMu.Unlock()
	
	logrus.Infof("🛑 Room stopped: %s", r.mangaID)
}

// broadcastToAll sends message to all clients in room
func (r *Room) broadcastToAll(message *Message) {
	r.clientsMu.RLock()
	defer r.clientsMu.RUnlock()

	for client := range r.clients {
		select {
		case client.send <- message:
		default:
			// Client send buffer full, remove client
			logrus.Warnf("Client %s send buffer full, disconnecting", client.userID)
			r.unregister <- client
		}
	}
}

// logChatActivity logs chat message activity and emits TCP stats event
func (h *Hub) logChatActivity(mangaID, userID string, eventTime time.Time, weight int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	activity := &models.Activity{
		ID:        generateActivityID(),
		Type:      models.ActivityTypeChat,
		UserID:    &userID,
		MangaID:   &mangaID,
		CreatedAt: eventTime,
	}

	if err := h.activityRepo.Create(ctx, activity); err != nil {
		logrus.Errorf("Failed to log chat activity: %v", err)
	}

	if h.statsAddr != "" {
		event := tcpProtocol.StatsEvent{
			Type:      tcpProtocol.EventTypeChat,
			MangaID:   mangaID,
			UserID:    &userID,
			EventTime: eventTime,
			Weight:    weight,
			Source:    "websocket",
		}
		if err := tcpProtocol.SendStatsEvent(h.statsAddr, event); err != nil {
			logrus.Errorf("Failed to emit TCP stats event: %v", err)
		}
	}
}

// readPump reads messages from WebSocket connection
func (c *Client) readPump() {
	defer func() {
		if c.onDisconnect != nil {
			c.onDisconnect()
		}
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastActive = time.Now()
		return nil
	})

	for {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Warnf("WebSocket read error: %v", err)
			}
			break
		}

		// Validate message size
		if len(messageData) > maxMessageSize {
			logrus.Warnf("Message too large from client %s: %d bytes", c.userID, len(messageData))
			c.sendError("message_too_large", "Message exceeds 1KB limit")
			continue
		}

		var msg Message
		if err := json.Unmarshal(messageData, &msg); err != nil {
			logrus.Warnf("Invalid message format from client %s: %v", c.userID, err)
			c.sendError("invalid_format", "Invalid JSON format")
			continue
		}

		// Validate content
		if len(msg.Content) > 5000 { // Reasonable limit for chat messages
			c.sendError("content_too_long", "Message content too long")
			continue
		}

		// Set metadata
		msg.UserID = c.userID
		msg.Username = c.username
		msg.MangaID = c.mangaID
		msg.Timestamp = time.Now()
		msg.Type = "message"

		// Save message to database first (atomic operation)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		chatMsg := &models.ChatMessage{
			ID:        generateMessageID(),
			MangaID:   c.mangaID,
			UserID:    c.userID,
			Content:   msg.Content,
			CreatedAt: msg.Timestamp,
		}
		
		if _, err := c.hub.chatRepo.Create(ctx, chatMsg); err != nil {
			cancel()
			logrus.Errorf("Failed to save chat message: %v", err)
			c.sendError("database_error", "Failed to save message")
			continue
		}
		cancel()

		// Log activity + emit TCP stats
		c.hub.logChatActivity(c.mangaID, c.userID, msg.Timestamp, 2)

		// Broadcast to room
		c.room.broadcast <- &msg
	}
}

// writePump writes messages to WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Client was unregistered
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				logrus.Errorf("Failed to marshal message: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.room.stop:
			return
		}
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message string) {
	errMsg := &Message{
		Type:      "error",
		UserID:    "system",
		Username:  "System",
		MangaID:   c.mangaID,
		Content:   fmt.Sprintf("Error [%s]: %s", code, message),
		Timestamp: time.Now(),
	}
	
	select {
	case c.send <- errMsg:
	default:
		// Don't block if channel is full
	}
}

// ServeClient handles WebSocket connection for a client
func (h *Hub) ServeClient(conn *websocket.Conn, userID, username, mangaID string, onDisconnect func()) {
	room := h.GetOrCreateRoom(mangaID)

	client := &Client{
		hub:     h,
		room:    room,
		conn:    conn,
		send:    make(chan *Message, 256),
		userID:  userID,
		username: username,
		mangaID: mangaID,
		lastActive: time.Now(),
		onDisconnect: onDisconnect,
	}

	room.register <- client

	// Start goroutines for reading and writing
	h.wg.Add(2)
	go func() {
		defer h.wg.Done()
		client.writePump()
	}()
	go func() {
		defer h.wg.Done()
		client.readPump()
	}()

	// Send chat history to newly connected client
	go h.sendChatHistory(client)
}

// sendChatHistory sends recent chat messages to client
func (h *Hub) sendChatHistory(client *Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, _, err := h.chatRepo.ListByMangaID(ctx, client.mangaID, historyLimit, 0)
	if err != nil {
		logrus.Warnf("Failed to get chat history for %s: %v", client.mangaID, err)
		return
	}

	// Send history messages (newest first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		
		// Get username for the message
		username := "Anonymous"
		// In production, we'd join with users table or cache usernames
		if msg.User.ID == client.userID {
			username = client.username
		}

		historyMsg := &Message{
			Type:      "history",
			UserID:    msg.User.ID,
			Username:  username,
			MangaID:   msg.MangaID,
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
		}

		select {
		case client.send <- historyMsg:
		case <-time.After(2 * time.Second):
			// Timeout - client might be slow, stop sending history
			return
		}
	}

	logrus.Debugf("📨 Sent %d history messages to client %s", len(messages), client.userID)
}

// GetRoomClientCount returns number of clients in a room
func (h *Hub) GetRoomClientCount(mangaID string) int {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	if room, exists := h.rooms[mangaID]; exists {
		room.clientsMu.RLock()
		defer room.clientsMu.RUnlock()
		return len(room.clients)
	}
	return 0
}

// GetRoomPresence returns current user presence for a room
func (h *Hub) GetRoomPresence(mangaID string) ([]*models.UserPresence, error) {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	if room, exists := h.rooms[mangaID]; exists {
		room.clientsMu.RLock()
		defer room.clientsMu.RUnlock()
		
		presence := make([]*models.UserPresence, 0, len(room.clients))
		now := time.Now()
		
		for client := range room.clients {
			status := "online"
			if now.Sub(client.lastActive) > 5*time.Minute {
				status = "away"
			}
			
			presence = append(presence, &models.UserPresence{
				UserID:     client.userID,
				Username:   client.username,
				MangaID:    mangaID,
				Status:     status,
				LastActive: client.lastActive,
			})
		}
		
		return presence, nil
	}
	
	return []*models.UserPresence{}, nil
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	logrus.Info("🛑 Stopping WebSocket hub...")
	
	close(h.stop)
	
	h.roomsMu.Lock()
	for _, room := range h.rooms {
		close(room.stop)
	}
	h.roomsMu.Unlock()
	
	h.wg.Wait()
	logrus.Info("✅ WebSocket hub stopped")
}

// Helper functions
func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

func generateActivityID() string {
	return fmt.Sprintf("act-%d", time.Now().UnixNano())
}