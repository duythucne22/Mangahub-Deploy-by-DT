package models

import (
	"time"
)

// Comment represents a manga comment - EXACTLY matches schema.sql
type Comment struct {
	ID        string    `json:"id" db:"id"`
	MangaID   string    `json:"manga_id" db:"manga_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	LikeCount int       `json:"like_count" db:"like_count"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CreateCommentRequest - includes manga_id as required by domain model
type CreateCommentRequest struct {
	MangaID string `json:"manga_id" validate:"required"` // Can also be in URL path
	Content string `json:"content" validate:"required,min=1,max=5000"`
}

// LikeCommentRequest - required for SPEC.md "Like comment" functionality
type LikeCommentRequest struct {
	CommentID string `json:"comment_id" validate:"required"`
}

// CommentUser - minimal user info for comment responses (SPEC.md compliant)
type CommentUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// CommentResponse represents a comment with user info for API responses
type CommentResponse struct {
	ID        string      `json:"id"`
	MangaID   string      `json:"manga_id"`
	User      CommentUser `json:"user"`
	Content   string      `json:"content"`
	LikeCount int         `json:"like_count"`
	CreatedAt time.Time   `json:"created_at"`
}

// CommentListResponse is paginated list of comments - standard format
type CommentListResponse struct {
	Data    []CommentResponse `json:"data"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	HasMore bool              `json:"has_more"`
}

// ActivityEvent - for emitting to TCP Stats Service (SPEC.md section 5.1)
type CommentActivityEvent struct {
	Type      string    `json:"type"` // "comment_created" or "comment_liked"
	CommentID string    `json:"comment_id"`
	MangaID   string    `json:"manga_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

const MaxCommentLength = 5000