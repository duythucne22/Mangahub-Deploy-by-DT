package models

import (
	"time"
)

// MangaStats represents manga statistics - EXACTLY matches schema.sql
type MangaStats struct {
	MangaID    string    `json:"manga_id" db:"manga_id"`
	CommentCount int     `json:"comment_count" db:"comment_count"`
	LikeCount    int     `json:"like_count" db:"like_count"`
	ChatCount    int     `json:"chat_count" db:"chat_count"`
	WeeklyScore  int     `json:"weekly_score" db:"weekly_score"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// HotManga represents a manga with its hot score for ranking
type HotManga struct {
	MangaID    string `json:"manga_id"`
	Title      string `json:"title"`
	CoverURL   string `json:"cover_url,omitempty"`
	WeeklyScore int   `json:"weekly_score"`
	Rank       int    `json:"rank"`
}

// TrendingManga represents trending manga based on recent activity
type TrendingManga struct {
	MangaID     string `json:"manga_id"`
	Title       string `json:"title"`
	ActivityCount int  `json:"activity_count"`
	Timeframe   string `json:"timeframe"` // "24h", "7d", "30d"
}

// Notification represents a broadcast notification - EXACTLY matches schema.sql
type Notification struct {
	ID        string    `json:"id" db:"id"`
	Message   string    `json:"message" db:"message"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// StatsResponse provides aggregated statistics for the frontend
type StatsResponse struct {
	HotManga      []HotManga      `json:"hot_manga"`
	ActiveChats   []TrendingManga `json:"active_chats"`
	RecentActivity []ActivityEvent `json:"recent_activity"`
	TotalComments int             `json:"total_comments"`
	TotalChats    int             `json:"total_chats"`
}

// RankedManga represents a manga with ranking and full stats
type RankedManga struct {
    Manga Manga      `json:"manga"`
    Stats MangaStats `json:"stats"`
    Rank  int        `json:"rank"`
}

// RankedMangaResponse represents paginated ranked manga results
type RankedMangaResponse struct {
    Data    []RankedManga `json:"data"`
    Total   int           `json:"total"`
    Limit   int           `json:"limit"`
    Offset  int           `json:"offset"`
    HasMore bool          `json:"has_more"`
}

// LeaderboardEntry represents a user leaderboard row
type LeaderboardEntry struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Score    int    `json:"score"`
	Rank     int    `json:"rank"`
	Category string `json:"category"`
}

// UserStatistics represents aggregated stats for a user
type UserStatistics struct {
	UserID        string  `json:"user_id"`
	TotalComments int     `json:"total_comments"`
	TotalChats    int     `json:"total_chats"`
	MangaCount    int     `json:"manga_count"`
	AverageRating float64 `json:"average_rating"`
	CurrentStreak int     `json:"current_streak"`
	TopGenres     []Genre `json:"top_genres"`
}

// StatsUpdate represents partial updates to manga stats
type StatsUpdate struct {
	MangaID      string `json:"manga_id"`
	CommentCount *int   `json:"comment_count,omitempty"`
	LikeCount    *int   `json:"like_count,omitempty"`
	ChatCount    *int   `json:"chat_count,omitempty"`
	WeeklyScore  *int   `json:"weekly_score,omitempty"`
}
