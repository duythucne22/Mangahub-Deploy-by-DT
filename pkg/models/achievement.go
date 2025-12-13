// Package models - Achievement and Gamification System
// Hệ thống achievements và challenges
// Chức năng:
//   - Achievement definitions với tiers
//   - User achievement tracking
//   - Challenges với time limits
//   - Points và leaderboard support
package models

import (
	"time"
)

// Achievement represents an unlockable achievement
type Achievement struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Description      string    `json:"description" db:"description"`
	Category         string    `json:"category" db:"category"` // reading, social, collection, streak
	Tier             string    `json:"tier" db:"tier"`         // bronze, silver, gold, platinum
	Points           int       `json:"points" db:"points"`
	IconURL          string    `json:"icon_url,omitempty" db:"icon_url"`
	RequirementType  string    `json:"requirement_type" db:"requirement_type"`   // chapters_read, manga_completed, days_streak, etc.
	RequirementValue int       `json:"requirement_value" db:"requirement_value"` // Number needed to unlock
	IsSecret         bool      `json:"is_secret" db:"is_secret"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// UserAchievement tracks a user's progress toward an achievement
type UserAchievement struct {
	ID            string     `json:"id" db:"id"`
	UserID        string     `json:"user_id" db:"user_id"`
	AchievementID string     `json:"achievement_id" db:"achievement_id"`
	Achievement   *Achievement `json:"achievement,omitempty" db:"-"` // Joined
	Progress      int        `json:"progress" db:"progress"`
	Unlocked      bool       `json:"unlocked" db:"unlocked"`
	UnlockedAt    *time.Time `json:"unlocked_at,omitempty" db:"unlocked_at"`
	Notified      bool       `json:"notified" db:"notified"` // Has user been notified?
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// Challenge represents a time-limited challenge
type Challenge struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Description      string    `json:"description" db:"description"`
	ChallengeType    string    `json:"challenge_type" db:"challenge_type"` // daily, weekly, monthly, event
	RequirementType  string    `json:"requirement_type" db:"requirement_type"`
	RequirementValue int       `json:"requirement_value" db:"requirement_value"`
	RewardPoints     int       `json:"reward_points" db:"reward_points"`
	RewardBadge      string    `json:"reward_badge,omitempty" db:"reward_badge"`
	StartsAt         time.Time `json:"starts_at" db:"starts_at"`
	EndsAt           time.Time `json:"ends_at" db:"ends_at"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// ChallengeProgress tracks a user's progress in a challenge
type ChallengeProgress struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	ChallengeID string     `json:"challenge_id" db:"challenge_id"`
	Challenge   *Challenge `json:"challenge,omitempty" db:"-"` // Joined
	Progress    int        `json:"progress" db:"progress"`
	Completed   bool       `json:"completed" db:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// UserStats tracks overall user statistics for achievements
type UserStats struct {
	UserID           string    `json:"user_id" db:"user_id"`
	TotalChaptersRead int      `json:"total_chapters_read" db:"total_chapters_read"`
	TotalMangaCompleted int    `json:"total_manga_completed" db:"total_manga_completed"`
	TotalMangaInLibrary int    `json:"total_manga_in_library" db:"total_manga_in_library"`
	CurrentStreak    int       `json:"current_streak" db:"current_streak"` // Days
	LongestStreak    int       `json:"longest_streak" db:"longest_streak"`
	TotalPoints      int       `json:"total_points" db:"total_points"`
	TotalAchievements int      `json:"total_achievements" db:"total_achievements"`
	LastReadAt       time.Time `json:"last_read_at" db:"last_read_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Achievement categories
const (
	CategoryReading    = "reading"
	CategorySocial     = "social"
	CategoryCollection = "collection"
	CategoryStreak     = "streak"
	CategoryExplorer   = "explorer"
)

// Achievement tiers
const (
	TierBronze   = "bronze"
	TierSilver   = "silver"
	TierGold     = "gold"
	TierPlatinum = "platinum"
)

// Requirement types
const (
	ReqChaptersRead    = "chapters_read"
	ReqMangaCompleted  = "manga_completed"
	ReqDaysStreak      = "days_streak"
	ReqMangaInLibrary  = "manga_in_library"
	ReqRatingsGiven    = "ratings_given"
	ReqReviewsWritten  = "reviews_written"
	ReqChatMessages    = "chat_messages"
	ReqFriendsAdded    = "friends_added"
	ReqGenresExplored  = "genres_explored"
)

// Challenge types
const (
	ChallengeDaily   = "daily"
	ChallengeWeekly  = "weekly"
	ChallengeMonthly = "monthly"
	ChallengeEvent   = "event"
)

// Points per tier
var TierPoints = map[string]int{
	TierBronze:   10,
	TierSilver:   25,
	TierGold:     50,
	TierPlatinum: 100,
}
