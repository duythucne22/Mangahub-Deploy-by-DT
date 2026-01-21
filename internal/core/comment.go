// Package core - Comment Business Logic
// Protocol-agnostic comment management service
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// CommentService defines comment operations
type CommentService interface {
	Create(ctx context.Context, mangaID, userID string, req models.CreateCommentRequest) (*models.CommentResponse, error)
	GetByID(ctx context.Context, id string) (*models.Comment, error)
	ListByMangaID(ctx context.Context, mangaID string, limit, offset int) (*models.CommentListResponse, error)
	IncrementLikes(ctx context.Context, id, userID string) (*models.CommentResponse, error)
	Delete(ctx context.Context, id, userID string) error
}

type commentService struct {
	commentRepo repository.CommentRepository
	userRepo    repository.UserRepository
}

// NewCommentService creates a new comment service
func NewCommentService(commentRepo repository.CommentRepository, userRepo repository.UserRepository) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		userRepo:    userRepo,
	}
}

// Create creates a new comment
func (s *commentService) Create(ctx context.Context, mangaID, userID string, req models.CreateCommentRequest) (*models.CommentResponse, error) {
	// Validate input
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(req.Content) > 5000 {
		return nil, fmt.Errorf("content exceeds maximum length of 5000 characters")
	}

	// Create comment
	comment := &models.Comment{
		ID:        uuid.New().String(),
		MangaID:   mangaID,
		UserID:    userID,
		Content:   req.Content,
		LikeCount: 0,
		CreatedAt: time.Now(),
	}

	return s.commentRepo.Create(ctx, comment)
}

// GetByID retrieves a comment by ID
func (s *commentService) GetByID(ctx context.Context, id string) (*models.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("comment not found: %w", err)
	}
	return comment, nil
}

// ListByMangaID retrieves comments for a manga with pagination
func (s *commentService) ListByMangaID(ctx context.Context, mangaID string, limit, offset int) (*models.CommentListResponse, error) {
	// Set defaults
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	comments, total, err := s.commentRepo.ListByMangaID(ctx, mangaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	// Repository already returns CommentResponse with user info
	responses := make([]models.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		responses = append(responses, *comment)
	}

	return &models.CommentListResponse{
		Data:    responses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}

// IncrementLikes increments the like count for a comment
func (s *commentService) IncrementLikes(ctx context.Context, id, userID string) (*models.CommentResponse, error) {
	return s.commentRepo.LikeComment(ctx, id, userID)
}

// Delete removes a comment (only by owner or admin)
func (s *commentService) Delete(ctx context.Context, id, userID string) error {
	// Get comment to verify ownership
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("comment not found: %w", err)
	}

	// Check if user is owner
	if comment.UserID != userID {
		// Get user to check if admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user.Role != "admin" {
			return fmt.Errorf("permission denied: only comment owner or admin can delete")
		}
	}

	if err := s.commentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	
	return nil
}
