package models

import (
	"time"
)

// MangaStatus represents valid manga status values
type MangaStatus string

const (
	MangaStatusOngoing  MangaStatus = "ongoing"
	MangaStatusCompleted MangaStatus = "completed"
	MangaStatusHiatus   MangaStatus = "hiatus"
)

// Manga represents a manga
type Manga struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	CoverURL    string    `json:"cover_url" db:"cover_url"`
	Status      string    `json:"status" db:"status"` // ongoing, completed, hiatus
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Genre represents a manga genre
type Genre struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// MangaWithGenres represents manga with populated genres for API responses
type MangaWithGenres struct {
	Manga
	Genres []Genre `json:"genres"`
}

// MangaSearchResult adds search relevance scoring for gRPC service
type MangaSearchResult struct {
	Manga
	RelevanceScore float64 `json:"relevance_score"`
}

// MangaSearchRequest represents search parameters
type MangaSearchRequest struct {
	Query  string   `json:"query" form:"query"`
	Genres []string `json:"genres" form:"genres"`
	Status string   `json:"status" form:"status"`
	Limit  int      `json:"limit" form:"limit" validate:"min=1,max=100"`
	Offset int      `json:"offset" form:"offset" validate:"min=0"`
}

// MangaListResponse represents paginated manga results
type MangaListResponse struct {
	Data    []MangaWithGenres `json:"data"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	HasMore bool              `json:"has_more"`
}

// CreateMangaRequest represents a request to create new manga
type CreateMangaRequest struct {
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description"`
	CoverURL    string   `json:"cover_url"`
	Status      string   `json:"status" validate:"oneof=ongoing completed hiatus"`
	GenreIDs    []string `json:"genre_ids"`
}

// UpdateMangaRequest represents a request to update manga
type UpdateMangaRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url"`
	Status      *string  `json:"status" validate:"omitempty,oneof=ongoing completed hiatus"`
	GenreIDs    []string `json:"genre_ids"`
}

// ValidateMangaSearch validates manga search request
func ValidateMangaSearch(req *MangaSearchRequest) error {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	return nil
}

// IsValidMangaStatus validates status against schema constraints
func IsValidMangaStatus(status string) bool {
	switch MangaStatus(status) {
	case MangaStatusOngoing, MangaStatusCompleted, MangaStatusHiatus:
		return true
	default:
		return false
	}
}
