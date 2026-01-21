package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"mangahub/pkg/models"
)

// MangaRepository handles manga data persistence with protocol-aware methods
type MangaRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, manga *models.Manga, genreIDs []string) error
	GetByID(ctx context.Context, id string) (*models.Manga, error)
	GetWithGenres(ctx context.Context, id string) (*models.MangaWithGenres, error)
	List(ctx context.Context, limit, offset int) ([]models.Manga, int, error)
	ListFiltered(ctx context.Context, limit, offset int, status string, genres []string) ([]models.Manga, int, error)
	Update(ctx context.Context, mangaID string, update *models.UpdateMangaRequest) error
	Delete(ctx context.Context, id string) error

	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error

	// Search and Trending
	SearchManga(ctx context.Context, query string, limit, offset int) ([]*models.MangaSearchResult, int, error)
	GetTrendingManga(ctx context.Context, limit int) ([]*models.HotManga, error)
}

type mangaRepository struct {
	pool *pgxpool.Pool
}

// NewMangaRepository creates a new PostgreSQL manga repository
func NewMangaRepository(pool *pgxpool.Pool) MangaRepository {
	return &mangaRepository{pool: pool}
}

// Create inserts a new manga with genres and initializes stats
func (r *mangaRepository) Create(ctx context.Context, manga *models.Manga, genreIDs []string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Insert manga - note: updated_at will be set by trigger or default
		mangaQuery := `
			INSERT INTO manga (id, title, description, cover_url, status, created_at)
			VALUES ($1, $2, $3, $4, $5, COALESCE($6, CURRENT_TIMESTAMP))
			RETURNING id, created_at, updated_at
		`

		err := tx.QueryRow(ctx, mangaQuery,
			manga.ID,
			manga.Title,
			manga.Description,
			manga.CoverURL,
			string(manga.Status),
			manga.CreatedAt,
		).Scan(&manga.ID, &manga.CreatedAt, &manga.UpdatedAt)

		if err != nil {
			return r.mapDBError(err, "create_manga")
		}

		// Insert genres
		if err := r.updateGenresForManga(ctx, tx, manga.ID, genreIDs); err != nil {
			return err
		}

		// Initialize stats
		statsQuery := `
			INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score, updated_at)
			VALUES ($1, 0, 0, 0, 0, CURRENT_TIMESTAMP)
		`
		_, err = tx.Exec(ctx, statsQuery, manga.ID)
		if err != nil {
			return r.mapDBError(err, "initialize_manga_stats")
		}

		return nil
	})
}

// GetByID retrieves a manga by ID
func (r *mangaRepository) GetByID(ctx context.Context, id string) (*models.Manga, error) {
	query := `
		SELECT id, title, description, cover_url, status, created_at, updated_at
		FROM manga
		WHERE id = $1
	`
	manga := &models.Manga{}
	var statusStr string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&manga.ID,
		&manga.Title,
		&manga.Description,
		&manga.CoverURL,
		&statusStr,
		&manga.CreatedAt,
		&manga.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, r.mapDBError(err, "get_manga_by_id")
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_manga_by_id")
	}

	manga.Status = statusStr
	return manga, nil
}

// GetWithGenres retrieves a manga with its associated genres
func (r *mangaRepository) GetWithGenres(ctx context.Context, id string) (*models.MangaWithGenres, error) {
	manga, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get genres
	genresQuery := `
		SELECT g.id, g.name
		FROM genres g
		INNER JOIN manga_genres mg ON g.id = mg.genre_id
		WHERE mg.manga_id = $1
		ORDER BY g.name
	`
	rows, err := r.pool.Query(ctx, genresQuery, id)
	if err != nil {
		return nil, r.mapDBError(err, "get_manga_genres")
	}
	defer rows.Close()

	var genres []models.Genre
	for rows.Next() {
		var genre models.Genre
		err := rows.Scan(&genre.ID, &genre.Name)
		if err != nil {
			return nil, r.mapDBError(err, "scan_genre")
		}
		genres = append(genres, genre)
	}

	return &models.MangaWithGenres{
		Manga:  *manga,
		Genres: genres,
	}, nil
}

// List retrieves manga with pagination
func (r *mangaRepository) List(ctx context.Context, limit, offset int) ([]models.Manga, int, error) {
	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM manga").Scan(&total)
	if err != nil {
		return nil, 0, r.mapDBError(err, "count_manga")
	}

	// Get paginated results
	query := `
		SELECT id, title, description, cover_url, status, created_at, updated_at
		FROM manga
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_manga")
	}
	defer rows.Close()

	var mangaList []models.Manga
	for rows.Next() {
		var manga models.Manga
		var statusStr string
		err := rows.Scan(
			&manga.ID,
			&manga.Title,
			&manga.Description,
			&manga.CoverURL,
			&statusStr,
			&manga.CreatedAt,
			&manga.UpdatedAt,
		)
		if err != nil {
			return nil, 0, r.mapDBError(err, "scan_manga")
		}
		manga.Status = statusStr
		mangaList = append(mangaList, manga)
	}

	return mangaList, total, nil
}

// ListFiltered retrieves manga with optional status and genre filters
func (r *mangaRepository) ListFiltered(ctx context.Context, limit, offset int, status string, genres []string) ([]models.Manga, int, error) {
	baseQuery := `
		FROM manga m
	`
	args := []interface{}{}
	filters := []string{}
	param := 1

	if len(genres) > 0 {
		placeholders := make([]string, len(genres))
		for i, g := range genres {
			placeholders[i] = fmt.Sprintf("$%d", param)
			args = append(args, g)
			param++
		}
		filters = append(filters, fmt.Sprintf("m.id IN (SELECT manga_id FROM manga_genres WHERE genre_id IN (%s))", strings.Join(placeholders, ",")))
	}

	if status != "" {
		filters = append(filters, fmt.Sprintf("m.status = $%d", param))
		args = append(args, status)
		param++
	}

	where := ""
	if len(filters) > 0 {
		where = " WHERE " + strings.Join(filters, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(DISTINCT m.id) " + baseQuery + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, r.mapDBError(err, "count_manga_filtered")
	}

	// List results
	selectQuery := `
		SELECT DISTINCT m.id, m.title, m.description, m.cover_url, m.status, m.created_at, m.updated_at
	` + baseQuery + where + `
		ORDER BY m.created_at DESC
		LIMIT $%d OFFSET $%d
	`

	args = append(args, limit, offset)
	selectQuery = fmt.Sprintf(selectQuery, param, param+1)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_manga_filtered")
	}
	defer rows.Close()

	var mangaList []models.Manga
	for rows.Next() {
		var manga models.Manga
		var statusStr string
		if err := rows.Scan(
			&manga.ID,
			&manga.Title,
			&manga.Description,
			&manga.CoverURL,
			&statusStr,
			&manga.CreatedAt,
			&manga.UpdatedAt,
		); err != nil {
			return nil, 0, r.mapDBError(err, "scan_manga_filtered")
		}
		manga.Status = statusStr
		mangaList = append(mangaList, manga)
	}

	return mangaList, total, nil
}

// Update updates manga information and genres
func (r *mangaRepository) Update(ctx context.Context, mangaID string, update *models.UpdateMangaRequest) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Build update query dynamically
		var updates []string
		var args []interface{}
		args = append(args, mangaID)
		argCount := 2 // Starting from $2

		if update.Title != nil {
			updates = append(updates, fmt.Sprintf("title = $%d", argCount))
			args = append(args, *update.Title)
			argCount++
		}

		if update.Description != nil {
			updates = append(updates, fmt.Sprintf("description = $%d", argCount))
			args = append(args, *update.Description)
			argCount++
		}

		if update.CoverURL != nil {
			updates = append(updates, fmt.Sprintf("cover_url = $%d", argCount))
			args = append(args, *update.CoverURL)
			argCount++
		}

		if update.Status != nil {
			updates = append(updates, fmt.Sprintf("status = $%d", argCount))
			args = append(args, string(*update.Status))
			argCount++
		}

		if len(updates) == 0 {
			return nil // Nothing to update
		}

		query := fmt.Sprintf(`
			UPDATE manga
			SET %s, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING id
		`, strings.Join(updates, ", "))

		var updatedID string
		err := tx.QueryRow(ctx, query, args...).Scan(&updatedID)
		if err != nil {
			return r.mapDBError(err, "update_manga")
		}

		// Update genres if provided
		if update.GenreIDs != nil {
			if err := r.updateGenresForManga(ctx, tx, mangaID, update.GenreIDs); err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete removes a manga and all related data
func (r *mangaRepository) Delete(ctx context.Context, id string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Delete will cascade to manga_genres, comments, chat_messages, manga_stats
		query := `DELETE FROM manga WHERE id = $1 RETURNING id`
		var deletedID string
		err := tx.QueryRow(ctx, query, id).Scan(&deletedID)
		if err == pgx.ErrNoRows {
			return r.mapDBError(err, "delete_manga")
		}
		if err != nil {
			return r.mapDBError(err, "delete_manga")
		}
		return nil
	})
}

// WithTransaction executes a function within a database transaction
func (r *mangaRepository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return r.mapDBError(err, "begin_transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// updateGenresForManga updates the genres for a manga (helper method)
func (r *mangaRepository) updateGenresForManga(ctx context.Context, tx pgx.Tx, mangaID string, genreIDs []string) error {
	// Delete existing genres
	_, err := tx.Exec(ctx, "DELETE FROM manga_genres WHERE manga_id = $1", mangaID)
	if err != nil {
		return r.mapDBError(err, "delete_manga_genres")
	}

	// Insert new genres
	if len(genreIDs) > 0 {
		// Check if genres exist, create if needed
		for _, genreID := range genreIDs {
			// Try to insert genre (ignore if exists)
			_, err := tx.Exec(ctx, `
				INSERT INTO genres (id, name) 
				VALUES ($1, $2) 
				ON CONFLICT (id) DO NOTHING
			`, genreID, strings.Title(strings.ReplaceAll(genreID, "-", " ")))
			if err != nil {
				return r.mapDBError(err, "ensure_genre_exists")
			}

			// Insert manga_genre relationship
			_, err = tx.Exec(ctx, `
				INSERT INTO manga_genres (manga_id, genre_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, mangaID, genreID)
			if err != nil {
				return r.mapDBError(err, "insert_manga_genre")
			}
		}
	}

	return nil
}

// SearchManga searches for manga by title or description
func (r *mangaRepository) SearchManga(ctx context.Context, query string, limit, offset int) ([]*models.MangaSearchResult, int, error) {
    searchQuery := strings.TrimSpace(query)
    if searchQuery == "" {
        return []*models.MangaSearchResult{}, 0, nil
    }

    countSQL := `
        SELECT COUNT(*)
        FROM manga
        WHERE search_vector @@ websearch_to_tsquery('english', $1)
    `
    var total int
    if err := r.pool.QueryRow(ctx, countSQL, searchQuery).Scan(&total); err != nil {
        return nil, 0, r.mapDBError(err, "count_search_results")
    }
    if total == 0 {
        return []*models.MangaSearchResult{}, 0, nil
    }

    searchSQL := `
        SELECT 
            m.id, m.title, m.description, m.cover_url, m.status, m.created_at, m.updated_at,
            ts_rank_cd(m.search_vector, websearch_to_tsquery('english', $1)) as relevance_score
        FROM manga m
        WHERE m.search_vector @@ websearch_to_tsquery('english', $1)
        ORDER BY relevance_score DESC, m.created_at DESC
        LIMIT $2 OFFSET $3
    `
    rows, err := r.pool.Query(ctx, searchSQL, searchQuery, limit, offset)
    if err != nil {
        return nil, 0, r.mapDBError(err, "search_manga")
    }
    defer rows.Close()

    var results []*models.MangaSearchResult
    for rows.Next() {
        var manga models.Manga
        var statusStr string
        var relevanceScore float64
        if err := rows.Scan(
            &manga.ID,
            &manga.Title,
            &manga.Description,
            &manga.CoverURL,
            &statusStr,
            &manga.CreatedAt,
            &manga.UpdatedAt,
            &relevanceScore,
        ); err != nil {
            return nil, 0, r.mapDBError(err, "scan_search_result")
        }
		manga.Status = statusStr
        results = append(results, &models.MangaSearchResult{
            Manga:          manga,
            RelevanceScore: relevanceScore,
        })
    }
    return results, total, nil
}

// GetTrendingManga retrieves the trending manga based on weekly score
func (r *mangaRepository) GetTrendingManga(ctx context.Context, limit int) ([]*models.HotManga, error) {
	query := `
		SELECT 
			m.id, m.title, m.cover_url,
			s.weekly_score,
			RANK() OVER (ORDER BY s.weekly_score DESC) as rank
		FROM manga m
		INNER JOIN manga_stats s ON m.id = s.manga_id
		ORDER BY s.weekly_score DESC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, r.mapDBError(err, "get_trending_manga")
	}
	defer rows.Close()

	var results []*models.HotManga
	rank := 1
	for rows.Next() {
		var manga models.HotManga
		if err := rows.Scan(
			&manga.MangaID,
			&manga.Title,
			&manga.CoverURL,
			&manga.WeeklyScore,
			&rank,
		); err != nil {
			return nil, r.mapDBError(err, "scan_trending_manga")
		}
		manga.Rank = rank
		results = append(results, &manga)
		rank++
	}
	return results, nil
}

// mapDBError maps database errors to application errors
func (r *mangaRepository) mapDBError(err error, operation string) error {
	if err == pgx.ErrNoRows {
		return fmt.Errorf("%s: %w", operation, models.ErrMangaNotFound)
	}

	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("duplicate manga entry: %w", err)
		case "23503": // foreign_key_violation
			return fmt.Errorf("invalid genre reference: %w", err)
		case "22P02": // invalid_text_representation
			return fmt.Errorf("invalid manga status: %w", err)
		}
	}

	return fmt.Errorf("database error during %s: %w", operation, err)
}