// Package core - Manga Business Logic
// Protocol-agnostic manga management service
package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// MangaService defines manga operations
type MangaService interface {
	Create(ctx context.Context, req models.CreateMangaRequest) (*models.Manga, error)
	GetByID(ctx context.Context, id string) (*models.Manga, error)
	GetWithGenres(ctx context.Context, id string) (*models.MangaWithGenres, error)
	List(ctx context.Context, req models.MangaSearchRequest) (*models.MangaListResponse, error)
	Search(ctx context.Context, query string, limit, offset int) (*models.MangaListResponse, error)
	Update(ctx context.Context, id string, req models.UpdateMangaRequest) (*models.Manga, error)
	Delete(ctx context.Context, id string) error
}

type mangaService struct {
	mangaRepo repository.MangaRepository
}

// NewMangaService creates a new manga service
func NewMangaService(mangaRepo repository.MangaRepository) MangaService {
	return &mangaService{
		mangaRepo: mangaRepo,
	}
}

// Create creates a new manga
func (s *mangaService) Create(ctx context.Context, req models.CreateMangaRequest) (*models.Manga, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Status == "" {
		req.Status = "ongoing"
	}
	if req.Status != "ongoing" && req.Status != "completed" && req.Status != "hiatus" {
		return nil, fmt.Errorf("invalid status: must be one of [ongoing, completed, hiatus]")
	}

	manga := &models.Manga{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		CoverURL:    req.CoverURL,
		Status:      req.Status,
		CreatedAt:   time.Now(),
	}

	if err := s.mangaRepo.Create(ctx, manga, req.GenreIDs); err != nil {
		return nil, fmt.Errorf("failed to create manga: %w", err)
	}

	return manga, nil
}

// GetByID retrieves a manga by ID
func (s *mangaService) GetByID(ctx context.Context, id string) (*models.Manga, error) {
	manga, err := s.mangaRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("manga not found: %w", err)
	}
	return manga, nil
}

// GetWithGenres retrieves a manga with its genres
func (s *mangaService) GetWithGenres(ctx context.Context, id string) (*models.MangaWithGenres, error) {
	manga, err := s.mangaRepo.GetWithGenres(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("manga not found: %w", err)
	}
	return manga, nil
}

// List retrieves manga with pagination
func (s *mangaService) List(ctx context.Context, req models.MangaSearchRequest) (*models.MangaListResponse, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	if strings.TrimSpace(req.Query) != "" {
		results, total, err := s.mangaRepo.SearchManga(ctx, req.Query, req.Limit, req.Offset)
		if err != nil {
			return nil, fmt.Errorf("failed to search manga: %w", err)
		}

		out := make([]models.MangaWithGenres, 0, len(results))
		for _, r := range results {
			out = append(out, models.MangaWithGenres{
				Manga:  r.Manga,
				Genres: []models.Genre{},
			})
		}

		return &models.MangaListResponse{
			Data:    out,
			Total:   total,
			Limit:   req.Limit,
			Offset:  req.Offset,
			HasMore: req.Offset+req.Limit < total,
		}, nil
	}

	if req.Status != "" || len(req.Genres) > 0 {
		list, total, err := s.mangaRepo.ListFiltered(ctx, req.Limit, req.Offset, req.Status, req.Genres)
		if err != nil {
			return nil, fmt.Errorf("failed to list manga: %w", err)
		}

		out := make([]models.MangaWithGenres, 0, len(list))
		for _, m := range list {
			out = append(out, models.MangaWithGenres{
				Manga:  m,
				Genres: []models.Genre{},
			})
		}

		return &models.MangaListResponse{
			Data:    out,
			Total:   total,
			Limit:   req.Limit,
			Offset:  req.Offset,
			HasMore: req.Offset+req.Limit < total,
		}, nil
	}

	list, total, err := s.mangaRepo.List(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list manga: %w", err)
	}

	out := make([]models.MangaWithGenres, 0, len(list))
	for _, m := range list {
		out = append(out, models.MangaWithGenres{
			Manga:  m,
			Genres: []models.Genre{},
		})
	}

	return &models.MangaListResponse{
		Data:    out,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
		HasMore: req.Offset+req.Limit < total,
	}, nil
}

// Search performs full-text search on manga
func (s *mangaService) Search(ctx context.Context, query string, limit, offset int) (*models.MangaListResponse, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	results, total, err := s.mangaRepo.SearchManga(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search manga: %w", err)
	}

	out := make([]models.MangaWithGenres, 0, len(results))
	for _, r := range results {
		out = append(out, models.MangaWithGenres{
			Manga:  r.Manga,
			Genres: []models.Genre{},
		})
	}

	return &models.MangaListResponse{
		Data:    out,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}, nil
}

// Update updates manga information
func (s *mangaService) Update(ctx context.Context, id string, req models.UpdateMangaRequest) (*models.Manga, error) {
	if req.Status != nil {
		status := *req.Status
		if status != "ongoing" && status != "completed" && status != "hiatus" {
			return nil, fmt.Errorf("invalid status: must be one of [ongoing, completed, hiatus]")
		}
	}

	if err := s.mangaRepo.Update(ctx, id, &req); err != nil {
		return nil, fmt.Errorf("failed to update manga: %w", err)
	}

	return s.mangaRepo.GetByID(ctx, id)
}

// Delete removes a manga
func (s *mangaService) Delete(ctx context.Context, id string) error {
	if err := s.mangaRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete manga: %w", err)
	}
	return nil
}
