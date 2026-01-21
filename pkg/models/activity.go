package models

import (
	"time"
)

// Activity types constants - ENFORCES schema CHECK constraint
const (
	ActivityTypeComment     = "comment"
	ActivityTypeChat        = "chat"
	ActivityTypeMangaUpdate = "manga_update"
)

// Activity represents an activity feed entry - EXACTLY matches schema.sql
type Activity struct {
	ID        string    `json:"id" db:"id"`
	Type      string    `json:"type" db:"type" validate:"required,oneof=comment chat manga_update"`
	UserID    *string   `json:"user_id,omitempty" db:"user_id"`
	MangaID   *string   `json:"manga_id,omitempty" db:"manga_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ActivityUser - minimal user info for activity feed 
type ActivityUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// ActivityManga - minimal manga info for activity feed
type ActivityManga struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ActivityResponse represents an activity with resolved user/manga info
type ActivityResponse struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	User      *ActivityUser  `json:"user,omitempty"`
	Manga     *ActivityManga `json:"manga,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ActivityFeedResponse represents paginated activity feed
type ActivityFeedResponse struct {
	Data    []ActivityResponse `json:"data"`
	Total   int                `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
	HasMore bool               `json:"has_more"`
}

// ==== PROTOCOL INTEGRATION MODELS ====

// ActivityEvent for TCP Stats Service
type ActivityEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "comment_created", "chat_message", "manga_update"
	MangaID   *string   `json:"manga_id,omitempty"`
	UserID    *string   `json:"user_id,omitempty"`
	EventType string    `json:"event_type"` // "create", "like", "update"
	Weight    int       `json:"weight"`     // comment=1, chat=2, like=3, manga_update=5
	Source    string    `json:"source"`     // "http", "websocket", "admin"
	Timestamp time.Time `json:"timestamp"`
}

// NotificationEvent for UDP Broadcast Service
type NotificationEvent struct {
	Type      string    `json:"type"`      // "new_manga", "system_announcement"
	Message   string    `json:"message"`   // Human-readable message
	MangaID   *string   `json:"manga_id,omitempty"`
	Priority  string    `json:"priority"`  // "low", "medium", "high"
	Icon      string    `json:"icon,omitempty"` // "📚", "💬", "🔔"
	Timestamp time.Time `json:"timestamp"`
}

// ==== CONVERSION METHODS (Bridge between services) ====

// ToActivityEvent converts an Activity to TCP Stats Service event
func (a *Activity) ToActivityEvent(eventType string) *ActivityEvent {
	weight := 1 // default weight
	switch a.Type {
	case ActivityTypeChat:
		weight = 2
	case ActivityTypeMangaUpdate:
		weight = 5
	}
	
	return &ActivityEvent{
		Type:      a.Type,
		MangaID:   a.MangaID,
		UserID:    a.UserID,
		EventType: eventType,
		Weight:    weight,
		Source:    "http", // default, can be overridden
		Timestamp: a.CreatedAt,
	}
}

// ToNotificationEvent converts manga update to UDP notification
func MangaUpdateToNotification(mangaTitle string, mangaID string) *NotificationEvent {
	return &NotificationEvent{
		Type:     "manga_update",
		Message:  "📚 New manga: " + mangaTitle,
		MangaID:  &mangaID,
		Priority: "high",
		Icon:     "📚",
		Timestamp: time.Now(),
	}
}