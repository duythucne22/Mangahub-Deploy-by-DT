// Package core - Activity Feed Business Logic
// Protocol-agnostic activity feed management service
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// ActivityService defines activity feed operations
type ActivityService interface {
	CreateActivity(ctx context.Context, activityType string, userID *string, mangaID *string) error
	GetGlobalFeed(ctx context.Context, limit, offset int) (*models.ActivityFeedResponse, error)
	GetUserFeed(ctx context.Context, userID string, limit, offset int) (*models.ActivityFeedResponse, error)
	GetMangaFeed(ctx context.Context, mangaID string, limit, offset int) (*models.ActivityFeedResponse, error)
}

type activityService struct {
	activityRepo repository.ActivityRepository
}

// NewActivityService creates a new activity service
func NewActivityService(
	activityRepo repository.ActivityRepository,
) ActivityService {
	return &activityService{
		activityRepo: activityRepo,
	}
}

// CreateActivity creates a new activity entry
func (s *activityService) CreateActivity(ctx context.Context, activityType string, userID *string, mangaID *string) error {
	switch activityType {
	case models.ActivityTypeComment, models.ActivityTypeChat, models.ActivityTypeMangaUpdate:
	default:
		return fmt.Errorf("invalid activity type: must be one of [comment, chat, manga_update]")
	}

	activity := &models.Activity{
		ID:        uuid.New().String(),
		Type:      activityType,
		UserID:    userID,
		MangaID:   mangaID,
		CreatedAt: time.Now(),
	}

	return s.activityRepo.Create(ctx, activity)
}

// GetGlobalFeed retrieves the global activity feed
func (s *activityService) GetGlobalFeed(ctx context.Context, limit, offset int) (*models.ActivityFeedResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	activities, total, err := s.activityRepo.ListGlobal(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get global feed: %w", err)
	}

	responses := make([]models.ActivityResponse, 0, len(activities))
	for _, a := range activities {
		if a != nil {
			responses = append(responses, *a)
		}
	}

	return &models.ActivityFeedResponse{
		Data:    responses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}

// GetUserFeed retrieves activity feed for a specific user
func (s *activityService) GetUserFeed(ctx context.Context, userID string, limit, offset int) (*models.ActivityFeedResponse, error) {
	// Set defaults
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	activities, total, err := s.activityRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feed: %w", err)
	}

	responses := make([]models.ActivityResponse, 0, len(activities))
	for _, a := range activities {
		if a != nil {
			responses = append(responses, *a)
		}
	}

	return &models.ActivityFeedResponse{
		Data:    responses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}

// GetMangaFeed retrieves activity feed for a specific manga
func (s *activityService) GetMangaFeed(ctx context.Context, mangaID string, limit, offset int) (*models.ActivityFeedResponse, error) {
	// Set defaults
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	activities, total, err := s.activityRepo.ListByMangaID(ctx, mangaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get manga feed: %w", err)
	}

	responses := make([]models.ActivityResponse, 0, len(activities))
	for _, a := range activities {
		if a != nil {
			responses = append(responses, *a)
		}
	}

	return &models.ActivityFeedResponse{
		Data:    responses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}
