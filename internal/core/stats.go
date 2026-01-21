package core

import (
	"context"
	"fmt"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// StatsService defines statistics operations
type StatsService interface {
	GetMangaStats(ctx context.Context, mangaID string) (*models.MangaStats, error)
	IncrementCommentCount(ctx context.Context, mangaID string) error
	IncrementLikeCount(ctx context.Context, mangaID string) error
	IncrementChatCount(ctx context.Context, mangaID string) error
	GetTopManga(ctx context.Context, limit, offset int) (*models.RankedMangaResponse, error)
	CalculateWeeklyScore(ctx context.Context, mangaID string) error
}

type statsService struct {
	statsRepo repository.StatsRepository
	mangaRepo repository.MangaRepository
}

// NewStatsService creates a new statistics service
func NewStatsService(
	statsRepo repository.StatsRepository,
	mangaRepo repository.MangaRepository,
) StatsService {
	return &statsService{
		statsRepo: statsRepo,
		mangaRepo: mangaRepo,
	}
}

// GetMangaStats retrieves statistics for a manga
func (s *statsService) GetMangaStats(ctx context.Context, mangaID string) (*models.MangaStats, error) {
	stats, err := s.statsRepo.GetByMangaID(ctx, mangaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manga stats: %w", err)
	}
	return stats, nil
}

// IncrementCommentCount increments the comment count for a manga
func (s *statsService) IncrementCommentCount(ctx context.Context, mangaID string) error {
	if err := s.statsRepo.IncrementCommentCount(ctx, mangaID); err != nil {
		return fmt.Errorf("failed to increment comment count: %w", err)
	}
	
	// Recalculate weekly score
	if err := s.CalculateWeeklyScore(ctx, mangaID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to recalculate weekly score: %v\n", err)
	}
	
	return nil
}

// IncrementLikeCount increments the like count for a manga
func (s *statsService) IncrementLikeCount(ctx context.Context, mangaID string) error {
	if err := s.statsRepo.IncrementLikeCount(ctx, mangaID); err != nil {
		return fmt.Errorf("failed to increment like count: %w", err)
	}
	
	// Recalculate weekly score
	if err := s.CalculateWeeklyScore(ctx, mangaID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to recalculate weekly score: %v\n", err)
	}
	
	return nil
}

// IncrementChatCount increments the chat count for a manga
func (s *statsService) IncrementChatCount(ctx context.Context, mangaID string) error {
	if err := s.statsRepo.IncrementChatCount(ctx, mangaID); err != nil {
		return fmt.Errorf("failed to increment chat count: %w", err)
	}
	
	// Recalculate weekly score
	if err := s.CalculateWeeklyScore(ctx, mangaID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to recalculate weekly score: %v\n", err)
	}
	
	return nil
}

// GetTopManga retrieves top ranked manga by weekly score
func (s *statsService) GetTopManga(ctx context.Context, limit, offset int) (*models.RankedMangaResponse, error) {
	// Set defaults
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	statsList, total, err := s.statsRepo.GetTopByWeeklyScore(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get top manga: %w", err)
	}

	var rankedManga []models.RankedManga
	rank := offset + 1
	
	for _, stats := range statsList {
		manga, err := s.mangaRepo.GetByID(ctx, stats.MangaID)
		if err != nil {
			continue
		}

		rankedManga = append(rankedManga, models.RankedManga{
			Manga: *manga,
			Stats: *stats,
			Rank:  rank,
		})
		rank++
	}

	return &models.RankedMangaResponse{
		Data:    rankedManga,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}

// CalculateWeeklyScore calculates and updates the weekly score for a manga
// Formula (UseCase): comments * 1 + likes * 1 + chats * 2
func (s *statsService) CalculateWeeklyScore(ctx context.Context, mangaID string) error {
	stats, err := s.statsRepo.GetByMangaID(ctx, mangaID)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	weeklyScore := (stats.CommentCount * 1) + (stats.LikeCount * 1) + (stats.ChatCount * 2)

	if err := s.statsRepo.UpdateWeeklyScore(ctx, mangaID, weeklyScore); err != nil {
		return fmt.Errorf("failed to update weekly score: %w", err)
	}

	return nil
}
