package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"mangahub/pkg/models"
)

// CommentRepository handles comment data persistence with protocol integration
type CommentRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, comment *models.Comment) (*models.CommentResponse, error)
	GetByID(ctx context.Context, id string) (*models.Comment, error)
	ListByMangaID(ctx context.Context, mangaID string, limit, offset int) ([]*models.CommentResponse, int, error)
	LikeComment(ctx context.Context, commentID string, userID string) (*models.CommentResponse, error)
	Delete(ctx context.Context, id string) error
	
	// Protocol-specific methods
	GetCommentActivity(ctx context.Context, since time.Time) ([]*models.CommentActivityEvent, error)
	
	// Stats & Ranking integration
	GetCommentStats(ctx context.Context, mangaID string) (*models.MangaStats, error)
	LogCommentActivity(ctx context.Context, comment *models.Comment) error
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type commentRepository struct {
	pool *pgxpool.Pool
}

// NewCommentRepository creates a new PostgreSQL comment repository
func NewCommentRepository(pool *pgxpool.Pool) CommentRepository {
	return &commentRepository{pool: pool}
}

// Create inserts a new comment with activity logging and stats events
func (r *commentRepository) Create(ctx context.Context, comment *models.Comment) (*models.CommentResponse, error) {
	var response *models.CommentResponse
	
	err := r.WithTransaction(ctx, func(tx pgx.Tx) error {
		if comment.ID == "" {
			comment.ID = generateUUID("comm")
		}

		// Insert comment
		insertQuery := `
			INSERT INTO comments (id, manga_id, user_id, content, like_count, created_at)
			VALUES ($1, $2, $3, $4, $5, COALESCE($6, CURRENT_TIMESTAMP))
			RETURNING id, created_at
		`
		
		err := tx.QueryRow(ctx, insertQuery,
			comment.ID,
			comment.MangaID,
			comment.UserID,
			comment.Content,
			comment.LikeCount,
			comment.CreatedAt,
		).Scan(&comment.ID, &comment.CreatedAt)
		
		if err != nil {
			return r.mapDBError(err, "create_comment")
		}
		
		// Log to activity feed
		activity := &models.Activity{
			ID:        generateUUID("act"),
			Type:      models.ActivityTypeComment,
			UserID:    &comment.UserID,
			MangaID:   &comment.MangaID,
			CreatedAt: comment.CreatedAt,
		}
		
		activityQuery := `
			INSERT INTO activity_feed (id, type, user_id, manga_id, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`
		
		_, err = tx.Exec(ctx, activityQuery,
			activity.ID,
			activity.Type,
			activity.UserID,
			activity.MangaID,
			activity.CreatedAt,
		)
		if err != nil {
			return r.mapDBError(err, "log_comment_activity")
		}
		
		// Update manga stats
		statsQuery := `
			INSERT INTO manga_stats (manga_id, comment_count, weekly_score, updated_at)
			VALUES ($1, 1, 1, CURRENT_TIMESTAMP)
			ON CONFLICT (manga_id) DO UPDATE
			SET comment_count = manga_stats.comment_count + 1,
				weekly_score = manga_stats.weekly_score + 1,
				updated_at = CURRENT_TIMESTAMP
		`
		_, err = tx.Exec(ctx, statsQuery, comment.MangaID)
		if err != nil {
			return r.mapDBError(err, "update_comment_stats")
		}

		// Get user info for response
		userQuery := `SELECT username FROM users WHERE id = $1`
		var username string
		err = tx.QueryRow(ctx, userQuery, comment.UserID).Scan(&username)
		if err != nil {
			return r.mapDBError(err, "get_comment_user")
		}
		
		response = &models.CommentResponse{
			ID:        comment.ID,
			MangaID:   comment.MangaID,
			User:      models.CommentUser{ID: comment.UserID, Username: username},
			Content:   comment.Content,
			LikeCount: comment.LikeCount,
			CreatedAt: comment.CreatedAt,
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return response, nil
}

// GetByID retrieves a comment by ID
func (r *commentRepository) GetByID(ctx context.Context, id string) (*models.Comment, error) {
	query := `
		SELECT id, manga_id, user_id, content, like_count, created_at
		FROM comments
		WHERE id = $1
	`
	comment := &models.Comment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&comment.ID,
		&comment.MangaID,
		&comment.UserID,
		&comment.Content,
		&comment.LikeCount,
		&comment.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, r.mapDBError(err, "get_comment_by_id")
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_comment_by_id")
	}
	return comment, nil
}

// ListByMangaID retrieves comments for a manga with user info and pagination
func (r *commentRepository) ListByMangaID(ctx context.Context, mangaID string, limit, offset int) ([]*models.CommentResponse, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM comments WHERE manga_id = $1`
	err := r.pool.QueryRow(ctx, countQuery, mangaID).Scan(&total)
	if err != nil {
		return nil, 0, r.mapDBError(err, "count_comments")
	}
	
	// Get paginated results with user info
	query := `
		SELECT 
			c.id, c.manga_id, c.user_id, c.content, c.like_count, c.created_at,
			u.username
		FROM comments c
		INNER JOIN users u ON c.user_id = u.id
		WHERE c.manga_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.pool.Query(ctx, query, mangaID, limit, offset)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_comments")
	}
	defer rows.Close()
	
	var comments []*models.CommentResponse
	for rows.Next() {
		var comment models.CommentResponse
		var username string
		
		err := rows.Scan(
			&comment.ID,
			&comment.MangaID,
			&comment.User.ID,
			&comment.Content,
			&comment.LikeCount,
			&comment.CreatedAt,
			&username,
		)
		if err != nil {
			return nil, 0, r.mapDBError(err, "scan_comment")
		}
		
		comment.User.Username = username
		comments = append(comments, &comment)
	}
	
	return comments, total, nil
}

// LikeComment increments the like count for a comment with activity logging
func (r *commentRepository) LikeComment(ctx context.Context, commentID string, userID string) (*models.CommentResponse, error) {
    var response *models.CommentResponse

    err := r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Get current comment to log activity
		var mangaID string
		getQuery := `SELECT manga_id FROM comments WHERE id = $1 FOR UPDATE`
		err := tx.QueryRow(ctx, getQuery, commentID).Scan(&mangaID)
		if err == pgx.ErrNoRows {
			return r.mapDBError(err, "like_comment")
		}
		if err != nil {
			return r.mapDBError(err, "like_comment")
		}
		
		// Atomic like count increment
		updateQuery := `
			UPDATE comments
			SET like_count = like_count + 1
			WHERE id = $1
			RETURNING id, manga_id, user_id, content, like_count, created_at
		`
		
		comment := &models.Comment{}
		err = tx.QueryRow(ctx, updateQuery, commentID).Scan(
			&comment.ID,
			&comment.MangaID,
			&comment.UserID,
			&comment.Content,
			&comment.LikeCount,
			&comment.CreatedAt,
		)
		if err != nil {
			return r.mapDBError(err, "update_comment_likes")
		}
		
		// Update manga stats
		statsQuery := `
			INSERT INTO manga_stats (manga_id, like_count, weekly_score, updated_at)
			VALUES ($1, 1, 1, CURRENT_TIMESTAMP)
			ON CONFLICT (manga_id) DO UPDATE
			SET like_count = manga_stats.like_count + 1,
				weekly_score = manga_stats.weekly_score + 1,
				updated_at = CURRENT_TIMESTAMP
		`
		_, err = tx.Exec(ctx, statsQuery, mangaID)
		if err != nil {
			return r.mapDBError(err, "update_like_stats")
		}

		// Get user info for the comment author (not the liker)
		userQuery := `SELECT username FROM users WHERE id = $1`
		var username string
		err = tx.QueryRow(ctx, userQuery, comment.UserID).Scan(&username)
		if err != nil {
			return r.mapDBError(err, "get_comment_user_for_like")
		}
		
		response = &models.CommentResponse{
			ID:        comment.ID,
			MangaID:   comment.MangaID,
			User:      models.CommentUser{ID: comment.UserID, Username: username},
			Content:   comment.Content,
			LikeCount: comment.LikeCount,
			CreatedAt: comment.CreatedAt,
		}
		
		// Log like activity (schema type must be "comment")
        likeActivity := &models.Activity{
            ID:        generateUUID("act"),
            Type:      models.ActivityTypeComment,
            UserID:    &userID,          // liker
            MangaID:   &comment.MangaID,  // affected manga
            CreatedAt: time.Now(),
        }
        _, err = tx.Exec(ctx, `
            INSERT INTO activity_feed (id, type, user_id, manga_id, created_at)
            VALUES ($1, $2, $3, $4, $5)
        `,
            likeActivity.ID,
            likeActivity.Type,
            likeActivity.UserID,
            likeActivity.MangaID,
            likeActivity.CreatedAt,
        )
        if err != nil {
            return r.mapDBError(err, "log_comment_like_activity")
        }
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return response, nil
}

// GetCommentActivity gets recent comment activity for TCP Stats Service
func (r *commentRepository) GetCommentActivity(ctx context.Context, since time.Time) ([]*models.CommentActivityEvent, error) {
	query := `
		SELECT id, manga_id, user_id, created_at
		FROM comments
		WHERE created_at >= $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.pool.Query(ctx, query, since)
	if err != nil {
		return nil, r.mapDBError(err, "get_comment_activity")
	}
	defer rows.Close()
	
	var events []*models.CommentActivityEvent
	for rows.Next() {
		var event models.CommentActivityEvent
		
		err := rows.Scan(
			&event.CommentID,
			&event.MangaID,
			&event.UserID,
			&event.Timestamp,
		)
		if err != nil {
			return nil, r.mapDBError(err, "scan_comment_activity")
		}
		
		event.Type = "comment_created"
		events = append(events, &event)
	}
	
	return events, nil
}

// GetCommentStats gets comment statistics for a manga
func (r *commentRepository) GetCommentStats(ctx context.Context, mangaID string) (*models.MangaStats, error) {
	query := `
		SELECT comment_count, updated_at
		FROM manga_stats
		WHERE manga_id = $1
	`
	
	stats := &models.MangaStats{MangaID: mangaID}
	err := r.pool.QueryRow(ctx, query, mangaID).Scan(
		&stats.CommentCount,
		&stats.UpdatedAt,
	)
	
	if err == pgx.ErrNoRows {
		// Initialize stats if not exists
		stats.CommentCount = 0
		stats.UpdatedAt = time.Now()
		return stats, nil
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_comment_stats")
	}
	
	return stats, nil
}

// LogCommentActivity logs comment activity for TCP Stats Service
func (r *commentRepository) LogCommentActivity(ctx context.Context, comment *models.Comment) error {
	// This is handled in the Create and LikeComment methods via transaction
	// But we provide this method for direct activity logging if needed
	return nil
}

// Delete removes a comment and associated activity
func (r *commentRepository) Delete(ctx context.Context, id string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Get comment first to log proper activity
		var mangaID, userID string
		getQuery := `SELECT manga_id, user_id FROM comments WHERE id = $1`
		err := tx.QueryRow(ctx, getQuery, id).Scan(&mangaID, &userID)
		if err == pgx.ErrNoRows {
			return r.mapDBError(err, "delete_comment")
		}
		if err != nil {
			return r.mapDBError(err, "delete_comment")
		}
		
		// Delete comment
		deleteQuery := `DELETE FROM comments WHERE id = $1`
		result, err := tx.Exec(ctx, deleteQuery, id)
		if err != nil {
			return r.mapDBError(err, "delete_comment")
		}
		
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return r.mapDBError(pgx.ErrNoRows, "delete_comment")
		}
		
		// Update manga stats
		statsQuery := `
			UPDATE manga_stats
			SET comment_count = GREATEST(comment_count - 1, 0),
				weekly_score = GREATEST(weekly_score - 1, 0),
				updated_at = CURRENT_TIMESTAMP
			WHERE manga_id = $1
		`
		_, err = tx.Exec(ctx, statsQuery, mangaID)
		if err != nil {
			return r.mapDBError(err, "update_comment_stats")
		}

		return nil
	})
}

// WithTransaction executes a function within a database transaction
func (r *commentRepository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
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

// mapDBError maps database errors to application errors
func (r *commentRepository) mapDBError(err error, operation string) error {
	if err == pgx.ErrNoRows {
		switch operation {
		case "get_comment_by_id", "delete_comment", "like_comment":
			return fmt.Errorf("%s: %w", operation, models.ErrNotFound)
		default:
			return fmt.Errorf("resource not found: %w", err)
		}
	}
	
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			if operation == "create_comment" {
				return fmt.Errorf("invalid manga or user reference: %w", err)
			}
			return fmt.Errorf("foreign key violation: %w", err)
		case "22001": // string_data_right_truncation
			return fmt.Errorf("comment content too long: %w", err)
		case "23505": // unique_violation
			return fmt.Errorf("duplicate comment: %w", err)
		}
	}
	
	return fmt.Errorf("database error during %s: %w", operation, err)
}