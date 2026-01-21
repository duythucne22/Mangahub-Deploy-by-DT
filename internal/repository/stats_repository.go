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

// StatsRepository handles manga statistics data persistence with TCP service integration
type StatsRepository interface {
	// Core stats operations
	GetByMangaID(ctx context.Context, mangaID string) (*models.MangaStats, error)
	IncrementCommentCount(ctx context.Context, mangaID string) error
	IncrementLikeCount(ctx context.Context, mangaID string) error
	IncrementChatCount(ctx context.Context, mangaID string) error
	UpdateWeeklyScore(ctx context.Context, mangaID string, score int) error
	
	// Ranking & Stats endpoints
	GetTopByWeeklyScore(ctx context.Context, limit, offset int) ([]*models.MangaStats, int, error)
	GetHotManga(ctx context.Context, limit, offset int) ([]*models.MangaStats, int, error)
	GetTrendingManga(ctx context.Context, hours int, limit int) ([]*models.TrendingManga, error)
	
	// TCP Stats Service integration
	ProcessActivityEvent(ctx context.Context, event *models.ActivityEvent) error
	RecalculateWeeklyScores(ctx context.Context, decayFactor float64) error
	GetRecentActivityForStats(ctx context.Context, hours int) ([]*models.ActivityEvent, error)
	
	// Batch operations
	BatchUpdateStats(ctx context.Context, updates []models.StatsUpdate) error
	RebuildAllStats(ctx context.Context) error
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type statsRepository struct {
	pool *pgxpool.Pool
}

// NewStatsRepository creates a new PostgreSQL stats repository
func NewStatsRepository(pool *pgxpool.Pool) StatsRepository {
	return &statsRepository{pool: pool}
}

// GetByMangaID retrieves statistics for a manga
func (r *statsRepository) GetByMangaID(ctx context.Context, mangaID string) (*models.MangaStats, error) {
	query := `
		SELECT manga_id, comment_count, like_count, chat_count, weekly_score, updated_at
		FROM manga_stats
		WHERE manga_id = $1
	`
	stats := &models.MangaStats{}
	err := r.pool.QueryRow(ctx, query, mangaID).Scan(
		&stats.MangaID,
		&stats.CommentCount,
		&stats.LikeCount,
		&stats.ChatCount,
		&stats.WeeklyScore,
		&stats.UpdatedAt,
	)
	
	if err == pgx.ErrNoRows {
		// Initialize stats if not exists
		stats.MangaID = mangaID
		stats.CommentCount = 0
		stats.LikeCount = 0
		stats.ChatCount = 0
		stats.WeeklyScore = 0
		stats.UpdatedAt = time.Now()
		
		// Try to insert initial stats
		insertQuery := `
			INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (manga_id) DO NOTHING
			RETURNING *
		`
		
		err := r.pool.QueryRow(ctx, insertQuery,
			mangaID,
			stats.CommentCount,
			stats.LikeCount,
			stats.ChatCount,
			stats.WeeklyScore,
			stats.UpdatedAt,
		).Scan(
			&stats.MangaID,
			&stats.CommentCount,
			&stats.LikeCount,
			&stats.ChatCount,
			&stats.WeeklyScore,
			&stats.UpdatedAt,
		)
		
		if err != nil && err != pgx.ErrNoRows {
			return nil, r.mapDBError(err, "initialize_manga_stats")
		}
		
		return stats, nil
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_manga_stats")
	}
	return stats, nil
}

// IncrementCommentCount increments the comment count for a manga
func (r *statsRepository) IncrementCommentCount(ctx context.Context, mangaID string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Get current stats to calculate new weekly score
		stats, err := r.getStatsForUpdate(ctx, tx, mangaID)
		if err != nil {
			return err
		}
		
		// Increment comment count and update weekly score
		newWeeklyScore := stats.WeeklyScore + 1 // Each comment adds 1 point
		newCommentCount := stats.CommentCount + 1
		
		query := `
			UPDATE manga_stats
			SET comment_count = $2,
			    weekly_score = $3,
			    updated_at = CURRENT_TIMESTAMP
			WHERE manga_id = $1
		`
		
		result, err := tx.Exec(ctx, query, mangaID, newCommentCount, newWeeklyScore)
		if err != nil {
			return r.mapDBError(err, "increment_comment_count")
		}
		
		if result.RowsAffected() == 0 {
			return r.mapDBError(pgx.ErrNoRows, "increment_comment_count")
		}
		
		return nil
	})
}

// IncrementLikeCount increments the like count for a manga
func (r *statsRepository) IncrementLikeCount(ctx context.Context, mangaID string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		stats, err := r.getStatsForUpdate(ctx, tx, mangaID)
		if err != nil {
			return err
		}
		
		newWeeklyScore := stats.WeeklyScore + 1 // Each like adds 1 point
		newLikeCount := stats.LikeCount + 1
		
		query := `
			UPDATE manga_stats
			SET like_count = $2,
			    weekly_score = $3,
			    updated_at = CURRENT_TIMESTAMP
			WHERE manga_id = $1
		`
		
		result, err := tx.Exec(ctx, query, mangaID, newLikeCount, newWeeklyScore)
		if err != nil {
			return r.mapDBError(err, "increment_like_count")
		}
		
		if result.RowsAffected() == 0 {
			return r.mapDBError(pgx.ErrNoRows, "increment_like_count")
		}
		
		return nil
	})
}

// IncrementChatCount increments the chat count for a manga
func (r *statsRepository) IncrementChatCount(ctx context.Context, mangaID string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		stats, err := r.getStatsForUpdate(ctx, tx, mangaID)
		if err != nil {
			return err
		}
		
		newWeeklyScore := stats.WeeklyScore + 2 // Each chat message adds 2 points (higher weight)
		newChatCount := stats.ChatCount + 1
		
		query := `
			UPDATE manga_stats
			SET chat_count = $2,
			    weekly_score = $3,
			    updated_at = CURRENT_TIMESTAMP
			WHERE manga_id = $1
		`
		
		result, err := tx.Exec(ctx, query, mangaID, newChatCount, newWeeklyScore)
		if err != nil {
			return r.mapDBError(err, "increment_chat_count")
		}
		
		if result.RowsAffected() == 0 {
			return r.mapDBError(pgx.ErrNoRows, "increment_chat_count")
		}
		
		return nil
	})
}

// UpdateWeeklyScore updates the weekly score for a manga
func (r *statsRepository) UpdateWeeklyScore(ctx context.Context, mangaID string, score int) error {
	query := `
		UPDATE manga_stats
		SET weekly_score = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE manga_id = $1
		RETURNING weekly_score
	`
	
	var updatedScore int
	err := r.pool.QueryRow(ctx, query, mangaID, score).Scan(&updatedScore)
	if err == pgx.ErrNoRows {
		return r.mapDBError(err, "update_weekly_score")
	}
	if err != nil {
		return r.mapDBError(err, "update_weekly_score")
	}
	return nil
}

// GetTopByWeeklyScore retrieves top manga stats by weekly score
func (r *statsRepository) GetTopByWeeklyScore(ctx context.Context, limit, offset int) ([]*models.MangaStats, int, error) {
    var total int
    countQuery := `SELECT COUNT(*) FROM manga_stats`
    if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
        return nil, 0, r.mapDBError(err, "count_hot_manga")
    }

    query := `
        SELECT 
            s.manga_id, s.comment_count, s.like_count, s.chat_count, s.weekly_score, s.updated_at
        FROM manga_stats s
        ORDER BY s.weekly_score DESC, s.updated_at DESC
        LIMIT $1 OFFSET $2
    `

    rows, err := r.pool.Query(ctx, query, limit, offset)
    if err != nil {
        return nil, 0, r.mapDBError(err, "get_hot_manga")
    }
    defer rows.Close()

    var statsList []*models.MangaStats
    for rows.Next() {
        var stats models.MangaStats
        if err := rows.Scan(
            &stats.MangaID,
            &stats.CommentCount,
            &stats.LikeCount,
            &stats.ChatCount,
            &stats.WeeklyScore,
            &stats.UpdatedAt,
        ); err != nil {
            return nil, 0, r.mapDBError(err, "scan_hot_manga")
        }
        statsList = append(statsList, &stats)
    }

    return statsList, total, nil
}

// GetHotManga retrieves hot manga (alias for GetTopByWeeklyScore)
func (r *statsRepository) GetHotManga(ctx context.Context, limit, offset int) ([]*models.MangaStats, int, error) {
    return r.GetTopByWeeklyScore(ctx, limit, offset)
}

// GetTrendingManga gets trending manga based on recent activity
func (r *statsRepository) GetTrendingManga(ctx context.Context, hours int, limit int) ([]*models.TrendingManga, error) {
	query := `
		SELECT 
			a.manga_id,
			COUNT(*) as activity_count
		FROM activity_feed a
		WHERE a.created_at >= NOW() - INTERVAL '1 hour' * $1
			AND a.type IN ('comment', 'chat')
			AND a.manga_id IS NOT NULL
		GROUP BY a.manga_id
		ORDER BY activity_count DESC
		LIMIT $2
	`
	
	rows, err := r.pool.Query(ctx, query, hours, limit)
	if err != nil {
		return nil, r.mapDBError(err, "get_trending_manga")
	}
	defer rows.Close()
	
	var trending []*models.TrendingManga
	for rows.Next() {
		var manga models.TrendingManga
		err := rows.Scan(
			&manga.MangaID,
			&manga.ActivityCount,
		)
		if err != nil {
			return nil, r.mapDBError(err, "scan_trending_manga")
		}
		trending = append(trending, &manga)
	}
	
	return trending, nil
}

// ProcessActivityEvent processes an activity event for stats aggregation
func (r *statsRepository) ProcessActivityEvent(ctx context.Context, event *models.ActivityEvent) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		if event.MangaID == nil {
			return nil
		}

		stats, err := r.getStatsForUpdate(ctx, tx, *event.MangaID)
		if err != nil {
			return err
		}

		switch event.Type {
		case "comment":
			stats.CommentCount++
			stats.WeeklyScore += 1
		case "chat":
			stats.ChatCount++
			stats.WeeklyScore += 2
		case "manga_update":
			stats.WeeklyScore += 5
		default:
			stats.WeeklyScore += event.Weight
		}

		updateQuery := `
			UPDATE manga_stats
			SET comment_count = $2,
			    like_count = $3,
			    chat_count = $4,
			    weekly_score = $5,
			    updated_at = CURRENT_TIMESTAMP
			WHERE manga_id = $1
		`
		_, err = tx.Exec(ctx, updateQuery,
			stats.MangaID,
			stats.CommentCount,
			stats.LikeCount,
			stats.ChatCount,
			stats.WeeklyScore,
		)
		if err != nil {
			return r.mapDBError(err, "process_activity_event")
		}
		return nil
	})
}

// RecalculateWeeklyScores recalculates all weekly scores with decay factor
func (r *statsRepository) RecalculateWeeklyScores(ctx context.Context, decayFactor float64) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Apply decay to all scores first
		decayQuery := `
			UPDATE manga_stats
			SET weekly_score = GREATEST(weekly_score * $1, 0),
			    updated_at = CURRENT_TIMESTAMP
		`
		
		_, err := tx.Exec(ctx, decayQuery, decayFactor)
		if err != nil {
			return r.mapDBError(err, "apply_score_decay")
		}
		
		// Get recent activity to add back fresh points
		activityQuery := `
			SELECT 
				a.manga_id,
				a.type,
				COUNT(*) as count
			FROM activity_feed a
			WHERE a.created_at >= NOW() - INTERVAL '7 days'
				AND a.manga_id IS NOT NULL
			GROUP BY a.manga_id, a.type
		`
		
		rows, err := tx.Query(ctx, activityQuery)
		if err != nil {
			return r.mapDBError(err, "get_recent_activity")
		}
		defer rows.Close()
		
		// Process each manga's activity
		for rows.Next() {
			var mangaID string
			var activityType string
			var count int
			
			err := rows.Scan(&mangaID, &activityType, &count)
			if err != nil {
				return r.mapDBError(err, "scan_activity_row")
			}
			
			// Get current stats
			stats, err := r.getStatsForUpdate(ctx, tx, mangaID)
			if err != nil {
				return err
			}
			
			// Add points based on activity type
			pointsToAdd := 0
			switch activityType {
			case "comment":
				pointsToAdd = count * 1
			case "chat":
				pointsToAdd = count * 2
			case "manga_update":
				pointsToAdd = count * 5
			}
			
			newScore := stats.WeeklyScore + pointsToAdd
			
			// Update the score
			updateQuery := `
				UPDATE manga_stats
				SET weekly_score = $2,
				    updated_at = CURRENT_TIMESTAMP
				WHERE manga_id = $1
			`
			
			_, err = tx.Exec(ctx, updateQuery, mangaID, newScore)
			if err != nil {
				return r.mapDBError(err, "update_recalculated_score")
			}
		}
		
		return nil
	})
}

// GetRecentActivityForStats gets recent activity for TCP Stats Service processing
func (r *statsRepository) GetRecentActivityForStats(ctx context.Context, hours int) ([]*models.ActivityEvent, error) {
	query := `
		SELECT 
			id, type, user_id, manga_id, created_at as timestamp
		FROM activity_feed
		WHERE created_at >= NOW() - INTERVAL '1 hour' * $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.pool.Query(ctx, query, hours)
	if err != nil {
		return nil, r.mapDBError(err, "get_recent_activity_for_stats")
	}
	defer rows.Close()
	
	var events []*models.ActivityEvent
	for rows.Next() {
		var event models.ActivityEvent
		var userID, mangaID *string
		var eventType string
		
		err := rows.Scan(
			&event.ID,
			&eventType,
			&userID,
			&mangaID,
			&event.Timestamp,
		)
		if err != nil {
			return nil, r.mapDBError(err, "scan_activity_event_for_stats")
		}
		
		event.Type = eventType
		event.UserID = userID
		event.MangaID = mangaID
		
		// Determine weight and event type
		switch eventType {
		case "comment":
			event.Weight = 1
			event.EventType = "create"
		case "chat":
			event.Weight = 2
			event.EventType = "create"
		case "manga_update":
			event.Weight = 5
			event.EventType = "update"
		default:
			event.Weight = 1
			event.EventType = "unknown"
		}
		
		event.Source = "tcp_stats_consumer"
		events = append(events, &event)
	}
	
	return events, nil
}

// BatchUpdateStats performs batch updates for performance
func (r *statsRepository) BatchUpdateStats(ctx context.Context, updates []models.StatsUpdate) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		for _, update := range updates {
			query := `
				UPDATE manga_stats
				SET comment_count = COALESCE($2, comment_count),
				    like_count = COALESCE($3, like_count),
				    chat_count = COALESCE($4, chat_count),
				    weekly_score = COALESCE($5, weekly_score),
				    updated_at = CURRENT_TIMESTAMP
				WHERE manga_id = $1
			`
			
			_, err := tx.Exec(ctx, query,
				update.MangaID,
				update.CommentCount,
				update.LikeCount,
				update.ChatCount,
				update.WeeklyScore,
			)
			if err != nil {
				return r.mapDBError(err, "batch_update_stats")
			}
		}
		return nil
	})
}

// RebuildAllStats recalculates all stats from scratch
func (r *statsRepository) RebuildAllStats(ctx context.Context) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Reset all stats to zero
		resetQuery := `
			UPDATE manga_stats
			SET comment_count = 0, like_count = 0, chat_count = 0, weekly_score = 0, updated_at = CURRENT_TIMESTAMP
		`
		_, err := tx.Exec(ctx, resetQuery)
		if err != nil {
			return r.mapDBError(err, "reset_all_stats")
		}
		
		// Count comments per manga
		commentQuery := `
			SELECT manga_id, COUNT(*) as count
			FROM comments
			GROUP BY manga_id
		`
		commentRows, err := tx.Query(ctx, commentQuery)
		if err != nil {
			return r.mapDBError(err, "count_comments_for_rebuild")
		}
		defer commentRows.Close()
		
		for commentRows.Next() {
			var mangaID string
			var count int
			err := commentRows.Scan(&mangaID, &count)
			if err != nil {
				return r.mapDBError(err, "scan_comment_count")
			}
			
			updateQuery := `
				UPDATE manga_stats
				SET comment_count = $2, weekly_score = weekly_score + $2
				WHERE manga_id = $1
			`
			_, err = tx.Exec(ctx, updateQuery, mangaID, count)
			if err != nil {
				return r.mapDBError(err, "update_comment_count")
			}
		}
		
		// Count chat messages per manga
		chatQuery := `
			SELECT manga_id, COUNT(*) as count
			FROM chat_messages
			GROUP BY manga_id
		`
		chatRows, err := tx.Query(ctx, chatQuery)
		if err != nil {
			return r.mapDBError(err, "count_chats_for_rebuild")
		}
		defer chatRows.Close()
		
		for chatRows.Next() {
			var mangaID string
			var count int
			err := chatRows.Scan(&mangaID, &count)
			if err != nil {
				return r.mapDBError(err, "scan_chat_count")
			}
			
			updateQuery := `
				UPDATE manga_stats
				SET chat_count = $2, weekly_score = weekly_score + ($2 * 2)
				WHERE manga_id = $1
			`
			_, err = tx.Exec(ctx, updateQuery, mangaID, count)
			if err != nil {
				return r.mapDBError(err, "update_chat_count")
			}
		}
		
		return nil
	})
}

// WithTransaction executes a function within a database transaction
func (r *statsRepository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
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

// getStatsForUpdate gets stats with row locking for updates
func (r *statsRepository) getStatsForUpdate(ctx context.Context, tx pgx.Tx, mangaID string) (*models.MangaStats, error) {
	query := `
		SELECT manga_id, comment_count, like_count, chat_count, weekly_score, updated_at
		FROM manga_stats
		WHERE manga_id = $1
		FOR UPDATE
	`
	
	stats := &models.MangaStats{}
	err := tx.QueryRow(ctx, query, mangaID).Scan(
		&stats.MangaID,
		&stats.CommentCount,
		&stats.LikeCount,
		&stats.ChatCount,
		&stats.WeeklyScore,
		&stats.UpdatedAt,
	)
	
	if err == pgx.ErrNoRows {
		// Initialize new stats record
		stats.MangaID = mangaID
		stats.CommentCount = 0
		stats.LikeCount = 0
		stats.ChatCount = 0
		stats.WeeklyScore = 0
		stats.UpdatedAt = time.Now()
		
		insertQuery := `
			INSERT INTO manga_stats (manga_id, comment_count, like_count, chat_count, weekly_score, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING *
		`
		
		err := tx.QueryRow(ctx, insertQuery,
			stats.MangaID,
			stats.CommentCount,
			stats.LikeCount,
			stats.ChatCount,
			stats.WeeklyScore,
			stats.UpdatedAt,
		).Scan(
			&stats.MangaID,
			&stats.CommentCount,
			&stats.LikeCount,
			&stats.ChatCount,
			&stats.WeeklyScore,
			&stats.UpdatedAt,
		)
		
		if err != nil {
			return nil, r.mapDBError(err, "initialize_stats_for_update")
		}
		return stats, nil
	}
	
	if err != nil {
		return nil, r.mapDBError(err, "get_stats_for_update")
	}
	
	return stats, nil
}

// mapDBError maps database errors to application errors
func (r *statsRepository) mapDBError(err error, operation string) error {
	if err == pgx.ErrNoRows {
		return fmt.Errorf("%s: %w", operation, models.ErrNotFound)
	}
	
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return fmt.Errorf("invalid manga reference: %w", err)
		case "40001": // serialization_failure
			return fmt.Errorf("concurrent update conflict - please retry: %w", err)
		}
	}
	
	return fmt.Errorf("database error during %s: %w", operation, err)
}