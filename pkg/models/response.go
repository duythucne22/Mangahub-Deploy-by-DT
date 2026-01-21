package models

import "time"

// ✅ GENERIC API RESPONSE 
type APIResponse struct {
    Success   bool        `json:"success"`
    Message   string      `json:"message,omitempty"`
    Data      interface{} `json:"data,omitempty"`
    Error     string      `json:"error,omitempty"` // Simplified error
    Timestamp time.Time   `json:"timestamp"`
}

// ✅ AUTH RESPONSE 
type AuthResponse struct {
    Token string `json:"token"`
    User  struct {
        ID       string `json:"id"`
        Username string `json:"username"`
        Role     string `json:"role"`
    } `json:"user"`
    ExpiresIn int `json:"expires_in,omitempty"` // seconds
}

// ✅ PAGINATION (Match database capabilities)
type PaginationMeta struct {
    Total   int  `json:"total"`
    Limit   int  `json:"limit"`
    Offset  int  `json:"offset"`
    HasMore bool `json:"has_more"`
}

// ✅ COMMUNITY ACTIVITY RESPONSE
type CommunityStats struct {
    TotalActiveManga int `json:"total_active_manga"`
    TotalComments    int `json:"total_comments"`
    TotalChatMessages int `json:"total_chat_messages"`
    ActiveUsers      int `json:"active_users"`
}

// ✅ ACTIVITY FEED ITEM (Matches schema.activity_feed)
type ActivityFeedItem struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"` // 'comment', 'chat', 'manga_update'
    Username  *string   `json:"username,omitempty"`
    MangaTitle *string  `json:"manga_title,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

// ✅ GENERIC PAGINATED RESPONSE
type PaginatedResponse[T any] struct {
    Data []T           `json:"data"`
    Meta PaginationMeta `json:"meta"`
}

// NewPaginationMeta builds pagination metadata consistently
func NewPaginationMeta(total, limit, offset int) PaginationMeta {
	return PaginationMeta{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}
}