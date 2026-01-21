// Package core - Chat Business Logic
// Protocol-agnostic chat message management service
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// ChatService defines chat operations
type ChatService interface {
	SendMessage(ctx context.Context, mangaID, userID string, req models.SendChatMessageRequest) (*models.ChatMessageResponse, error)
	GetHistory(ctx context.Context, mangaID string, limit, offset int) (*models.ChatHistoryResponse, error)
	DeleteMessage(ctx context.Context, id, userID string) error
}

type chatService struct {
	chatRepo repository.ChatRepository
	userRepo repository.UserRepository
}

// NewChatService creates a new chat service
func NewChatService(chatRepo repository.ChatRepository, userRepo repository.UserRepository) ChatService {
	return &chatService{
		chatRepo: chatRepo,
		userRepo: userRepo,
	}
}

// SendMessage sends a new chat message
func (s *chatService) SendMessage(ctx context.Context, mangaID, userID string, req models.SendChatMessageRequest) (*models.ChatMessageResponse, error) {
	// Validate input
	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(req.Content) > 5000 {
		return nil, fmt.Errorf("content exceeds maximum length of 5000 characters")
	}

	// Create message
	message := &models.ChatMessage{
		ID:        uuid.New().String(),
		MangaID:   mangaID,
		UserID:    userID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	return s.chatRepo.Create(ctx, message)
}

// GetHistory retrieves chat history for a manga with pagination
func (s *chatService) GetHistory(ctx context.Context, mangaID string, limit, offset int) (*models.ChatHistoryResponse, error) {
	// Set defaults
	if limit <= 0 || limit > 100 {
		limit = 50 // Higher default for chat
	}
	if offset < 0 {
		offset = 0
	}

	messages, total, err := s.chatRepo.ListByMangaID(ctx, mangaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	// Build response with user info
	responses := make([]models.ChatMessageResponse, 0, len(messages))
	for _, m := range messages {
		if m != nil {
			responses = append(responses, *m)
		}
	}

	return &models.ChatHistoryResponse{
		Data:    responses,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}

// DeleteMessage removes a chat message (only by owner or admin)
func (s *chatService) DeleteMessage(ctx context.Context, id, userID string) error {
	// Get message to verify ownership
	message, err := s.chatRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	// Check if user is owner
	if message.UserID != userID {
		// Get user to check if admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user.Role != "admin" {
			return fmt.Errorf("permission denied: only message owner or admin can delete")
		}
	}

	if err := s.chatRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	
	return nil
}
