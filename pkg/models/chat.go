package models

import (
	"time"
	"github.com/gorilla/websocket"
)

// ChatMessage represents a chat message - EXACTLY matches schema.sql
type ChatMessage struct {
	ID        string    `json:"id" db:"id"`
	MangaID   string    `json:"manga_id" db:"manga_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SendChatMessageRequest - includes manga_id for per-manga rooms
type SendChatMessageRequest struct {
	MangaID string `json:"manga_id" validate:"required"` // Can also be derived from connection
	Content string `json:"content" validate:"required,min=1,max=5000"`
}

// ChatUser - minimal user info (SPEC.md schema compliant)
type ChatUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// ChatMessageResponse represents chat message with minimal user info
type ChatMessageResponse struct {
	ID        string    `json:"id"`
	MangaID   string    `json:"manga_id"`
	User      ChatUser  `json:"user"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatHistoryResponse represents paginated chat history
type ChatHistoryResponse struct {
	Data    []ChatMessageResponse `json:"data"`
	Total   int                  `json:"total"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
	HasMore bool                 `json:"has_more"`
}

// ChatRoomInfo - for listing active chat rooms
type ChatRoomInfo struct {
	MangaID      string `json:"manga_id"`
	MangaTitle   string `json:"manga_title"`
	ActiveUsers  int    `json:"active_users"`
	LastMessage  string `json:"last_message,omitempty"`
	LastActivity time.Time `json:"last_activity"`
}

// ==== WEBSOCKET SPECIFIC MODELS (Critical for protocol implementation) ====

// WebSocketConnection manages individual WebSocket connections
type WebSocketConnection struct {
	ID          string
	UserID      string
	Username    string
	MangaID     string
	Connection  *websocket.Conn
	SendChannel chan []byte
	Closed      bool
	LastActive  time.Time
}

// ChatRoom manages per-manga chat rooms
type ChatRoom struct {
	MangaID   string
	Name      string
	Users     map[string]*WebSocketConnection // UserID -> Connection
	Broadcast chan []byte
	Register  chan *WebSocketConnection
	Unregister chan *WebSocketConnection
}

// UserPresence for presence tracking (SPEC.md requirement)
type UserPresence struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	MangaID    string    `json:"manga_id"`
	MangaTitle string    `json:"manga_title"`
	Status     string    `json:"status"` // "online", "away"
	LastActive time.Time `json:"last_active"`
}

// ChatActivityEvent for TCP Stats Service integration
type ChatActivityEvent struct {
	Type      string    `json:"type"` // "chat_message"
	MangaID   string    `json:"manga_id"`
	UserID    string    `json:"user_id"`
	MessageID string    `json:"message_id"`
	Content   string    `json:"content"` // For keyword analysis
	Timestamp time.Time `json:"timestamp"`
}

const MaxChatMessageLength = 5000