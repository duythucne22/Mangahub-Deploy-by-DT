package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"mangahub/pkg/models"
)

// ActivityRepository handles activity feed data persistence with protocol integration
type ActivityRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, activity *models.Activity) error
	GetByID(ctx context.Context, id string) (*models.Activity, error)
	ListGlobal(ctx context.Context, limit, offset int) ([]*models.ActivityResponse, int, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.ActivityResponse, int, error)
	ListByMangaID(ctx context.Context, mangaID string, limit, offset int) ([]*models.ActivityResponse, int, error)
	Delete(ctx context.Context, id string) error
	
	// Protocol-specific methods
	LogActivityEvent(ctx context.Context, event *models.ActivityEvent) error
	LogNotificationEvent(ctx context.Context, notification *models.NotificationEvent) error
	
	// Stats & Ranking integration
	GetRecentActivity(ctx context.Context, hours int) ([]*models.ActivityEvent, error)
	GetDailyActivityCounts(ctx context.Context, days int) (map[string]int, error)
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
	
	// TUI Feed methods
	GetGlobalFeed(ctx context.Context, limit, offset int) ([]*models.ActivityResponse, int, error)
	GetPersonalFeed(ctx context.Context, userID string, limit, offset int) ([]*models.ActivityResponse, int, error)
}

type activityRepository struct {
	pool *pgxpool.Pool
}

// NewActivityRepository creates a new PostgreSQL activity repository
func NewActivityRepository(pool *pgxpool.Pool) ActivityRepository {
	return &activityRepository{pool: pool}
}

// Create inserts a new activity with proper null handling
func (r *activityRepository) Create(ctx context.Context, activity *models.Activity) error {
	query := `
		INSERT INTO activity_feed (id, type, user_id, manga_id, created_at)
		VALUES ($1, $2, $3, $4, COALESCE($5, CURRENT_TIMESTAMP))
		RETURNING id, created_at
	`
	
	var userID, mangaID *string
	if activity.UserID != nil {
		temp := *activity.UserID
		userID = &temp
	}
	if activity.MangaID != nil {
		temp := *activity.MangaID
		mangaID = &temp
	}
	
	err := r.pool.QueryRow(ctx, query,
		activity.ID,
		activity.Type,
		userID,
		mangaID,
		activity.CreatedAt,
	).Scan(&activity.ID, &activity.CreatedAt)
	
	if err != nil {
		return r.mapDBError(err, "create_activity")
	}
	return nil
}

// GetByID retrieves an activity by ID with proper null handling
func (r *activityRepository) GetByID(ctx context.Context, id string) (*models.Activity, error) {
	query := `
		SELECT id, type, user_id, manga_id, created_at
		FROM activity_feed
		WHERE id = $1
	`
	activity := &models.Activity{}
	var userID, mangaID *string
	
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&activity.ID,
		&activity.Type,
		&userID,
		&mangaID,
		&activity.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, r.mapDBError(err, "get_activity_by_id")
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_activity_by_id")
	}
	
	activity.UserID = userID
	activity.MangaID = mangaID
	return activity, nil
}

// ListGlobal retrieves global activity feed with resolved user/manga info
func (r *activityRepository) ListGlobal(ctx context.Context, limit, offset int) ([]*models.ActivityResponse, int, error) {
	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM activity_feed").Scan(&total)
	if err != nil {
		return nil, 0, r.mapDBError(err, "count_global_activities")
	}
	
	// Get paginated results with joins
	query := `
		SELECT 
			a.id, a.type, a.user_id, a.manga_id, a.created_at,
			u.username as user_username,
			m.title as manga_title
		FROM activity_feed a
		LEFT JOIN users u ON a.user_id = u.id
		LEFT JOIN manga m ON a.manga_id = m.id
		ORDER BY a.created_at DESC
		LIMIT $1 OFFSET $2
	`
	
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_global_activities")
	}
	defer rows.Close()
	
	var activities []*models.ActivityResponse
	for rows.Next() {
		var activity models.ActivityResponse
		var userID, mangaID, userUsername, mangaTitle *string
		var activityType string
		
		err := rows.Scan(
			&activity.ID,
			&activityType,
			&userID,
			&mangaID,
			&activity.CreatedAt,
			&userUsername,
			&mangaTitle,
		)
		if err != nil {
			return nil, 0, r.mapDBError(err, "scan_global_activity")
		}
		
		activity.Type = activityType
		
		if userID != nil && userUsername != nil {
			activity.User = &models.ActivityUser{
				ID:       *userID,
				Username: *userUsername,
			}
		}
		
		if mangaID != nil && mangaTitle != nil {
			activity.Manga = &models.ActivityManga{
				ID:    *mangaID,
				Title: *mangaTitle,
			}
		}
		
		activities = append(activities, &activity)
	}
	
	return activities, total, nil
}

// ListByUserID retrieves activities for a specific user with resolved info
func (r *activityRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.ActivityResponse, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM activity_feed WHERE user_id = $1`
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, r.mapDBError(err, "count_user_activities")
	}
	
	// Get paginated results
	query := `
		SELECT 
			a.id, a.type, a.user_id, a.manga_id, a.created_at,
			u.username as user_username,
			m.title as manga_title
		FROM activity_feed a
		LEFT JOIN users u ON a.user_id = u.id
		LEFT JOIN manga m ON a.manga_id = m.id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_user_activities")
	}
	defer rows.Close()
	
	var activities []*models.ActivityResponse
	for rows.Next() {
		var activity models.ActivityResponse
		var dbUserID, mangaID, userUsername, mangaTitle *string
		var activityType string
		
		err := rows.Scan(
			&activity.ID,
			&activityType,
			&dbUserID,
			&mangaID,
			&activity.CreatedAt,
			&userUsername,
			&mangaTitle,
		)
		if err != nil {
			return nil, 0, r.mapDBError(err, "scan_user_activity")
		}
		
		activity.Type = activityType
		
		// Initialize User struct properly
		if dbUserID != nil && userUsername != nil {
			activity.User = &models.ActivityUser{
				ID:       *dbUserID,
				Username: *userUsername,
			}
		}
		
		if mangaID != nil && mangaTitle != nil {
			activity.Manga = &models.ActivityManga{
				ID:    *mangaID,
				Title: *mangaTitle,
			}
		}
		
		activities = append(activities, &activity)
	}
	
	return activities, total, nil
}

// ListByMangaID retrieves activities for a specific manga with resolved info
func (r *activityRepository) ListByMangaID(ctx context.Context, mangaID string, limit, offset int) ([]*models.ActivityResponse, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM activity_feed WHERE manga_id = $1`
	err := r.pool.QueryRow(ctx, countQuery, mangaID).Scan(&total)
	if err != nil {
		return nil, 0, r.mapDBError(err, "count_manga_activities")
	}
	
	// Get paginated results
	query := `
		SELECT 
			a.id, a.type, a.user_id, a.manga_id, a.created_at,
			u.username as user_username,
			m.title as manga_title
		FROM activity_feed a
		LEFT JOIN users u ON a.user_id = u.id
		LEFT JOIN manga m ON a.manga_id = m.id
		WHERE a.manga_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.pool.Query(ctx, query, mangaID, limit, offset)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_manga_activities")
	}
	defer rows.Close()
	
	var activities []*models.ActivityResponse
	for rows.Next() {
		var activity models.ActivityResponse
		var userID, dbMangaID, userUsername, mangaTitle *string
		var activityType string
		
		err := rows.Scan(
			&activity.ID,
			&activityType,
			&userID,
			&dbMangaID,
			&activity.CreatedAt,
			&userUsername,
			&mangaTitle,
		)
		if err != nil {
			return nil, 0, r.mapDBError(err, "scan_manga_activity")
		}
		
		activity.Type = activityType
		
		if userID != nil && userUsername != nil {
			activity.User = &models.ActivityUser{
				ID:       *userID,
				Username: *userUsername,
			}
		}
		
		// Initialize Manga struct properly
		if dbMangaID != nil && mangaTitle != nil {
			activity.Manga = &models.ActivityManga{
				ID:    *dbMangaID,
				Title: *mangaTitle,
			}
		}
		
		activities = append(activities, &activity)
	}
	
	return activities, total, nil
}

// Delete removes an activity
func (r *activityRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM activity_feed 
		WHERE id = $1
		RETURNING id
	`
	var deletedID string
	err := r.pool.QueryRow(ctx, query, id).Scan(&deletedID)
	if err == pgx.ErrNoRows {
		return r.mapDBError(err, "delete_activity")
	}
	if err != nil {
		return r.mapDBError(err, "delete_activity")
	}
	return nil
}

// normalizeActivityType maps event types to schema-safe values
func normalizeActivityType(t string) string {
	switch t {
	case "comment", "comment_created", "comment_liked":
		return models.ActivityTypeComment
	case "chat", "chat_message", "chat_join", "chat_leave", "chat_connection":
		return models.ActivityTypeChat
	case "manga_update", "manga_created", "manga_updated", "manga_deleted":
		return models.ActivityTypeMangaUpdate
	default:
		// Fallback to comment to avoid CHECK constraint violations
		return models.ActivityTypeComment
	}
}

// LogActivityEvent logs an activity event for TCP Stats Service consumption
func (r *activityRepository) LogActivityEvent(ctx context.Context, event *models.ActivityEvent) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Create activity feed entry
		activity := &models.Activity{
			ID:        generateUUID("act"),
			Type:      normalizeActivityType(event.Type),
			UserID:    event.UserID,
			MangaID:   event.MangaID,
			CreatedAt: event.Timestamp,
		}
		
		insertQuery := `
			INSERT INTO activity_feed (id, type, user_id, manga_id, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`
		
		_, err := tx.Exec(ctx, insertQuery,
			activity.ID,
			activity.Type,
			activity.UserID,
			activity.MangaID,
			activity.CreatedAt,
		)
		if err != nil {
			return r.mapDBError(err, "log_activity_event")
		}
		
		// This event will be consumed by TCP Stats Service
		// No need to emit here - Stats Service polls activity_feed
		return nil
	})
}

// LogNotificationEvent logs a notification event and triggers UDP broadcast
func (r *activityRepository) LogNotificationEvent(ctx context.Context, notification *models.NotificationEvent) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Log to notifications table
		notifQuery := `
			INSERT INTO notifications (id, message, created_at)
			VALUES ($1, $2, $3)
			RETURNING id
		`
		
		var notifID string
		err := tx.QueryRow(ctx, notifQuery,
			generateUUID("notif"),
			notification.Message,
			notification.Timestamp,
		).Scan(&notifID)
		if err != nil {
			return r.mapDBError(err, "log_notification_event")
		}
		
		// Also log to activity feed for stats
		if notification.MangaID != nil {
			activity := &models.Activity{
				ID:        generateUUID("act"),
				Type:      models.ActivityTypeMangaUpdate,
				UserID:    nil,
				MangaID:   notification.MangaID,
				CreatedAt: notification.Timestamp,
			}
			
			activityQuery := `
				INSERT INTO activity_feed (id, type, user_id, manga_id, created_at)
				VALUES ($1, $2, $3, $4, $5)
			`
			
			_, err := tx.Exec(ctx, activityQuery,
				activity.ID,
				activity.Type,
				activity.UserID,
				activity.MangaID,
				activity.CreatedAt,
			)
			if err != nil {
				return r.mapDBError(err, "log_notification_activity")
			}
		}
		
		// UDP broadcast happens in service layer, not repository
		return nil
	})
}

// GetRecentActivity returns recent activity events for TCP Stats Service
func (r *activityRepository) GetRecentActivity(ctx context.Context, hours int) ([]*models.ActivityEvent, error) {
	query := `
		SELECT id, type, user_id, manga_id, created_at
		FROM activity_feed
		WHERE created_at >= NOW() - INTERVAL '1 hour' * $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.pool.Query(ctx, query, hours)
	if err != nil {
		return nil, r.mapDBError(err, "get_recent_activity")
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
			return nil, r.mapDBError(err, "scan_activity_event")
		}
		
		event.Type = eventType
		event.UserID = userID
		event.MangaID = mangaID
		
		// Determine weight based on activity type
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
		
		events = append(events, &event)
	}
	
	return events, nil
}

// GetDailyActivityCounts returns activity counts by day for stats
func (r *activityRepository) GetDailyActivityCounts(ctx context.Context, days int) (map[string]int, error) {
	query := `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM-DD') as date,
			COUNT(*) as count
		FROM activity_feed
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY TO_CHAR(created_at, 'YYYY-MM-DD')
		ORDER BY date DESC
	`
	
	rows, err := r.pool.Query(ctx, query, days)
	if err != nil {
		return nil, r.mapDBError(err, "get_daily_activity_counts")
	}
	defer rows.Close()
	
	counts := make(map[string]int)
	for rows.Next() {
		var date string
		var count int
		err := rows.Scan(&date, &count)
		if err != nil {
			return nil, r.mapDBError(err, "scan_daily_count")
		}
		counts[date] = count
	}
	
	return counts, nil
}

// WithTransaction executes a function within a database transaction
func (r *activityRepository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
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
func (r *activityRepository) mapDBError(err error, operation string) error {
	if err == pgx.ErrNoRows {
		return fmt.Errorf("%s: %w", operation, models.ErrNotFound)
	}
	
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "23502": // not_null_violation
			return fmt.Errorf("required field missing in activity: %w", err)
		case "23514": // check_violation
			return fmt.Errorf("invalid activity type: %w", err)
		}
	}
	
	return fmt.Errorf("database error during %s: %w", operation, err)
}

// GetGlobalFeed is an alias for ListGlobal (TUI)
func (r *activityRepository) GetGlobalFeed(ctx context.Context, limit, offset int) ([]*models.ActivityResponse, int, error) {
    return r.ListGlobal(ctx, limit, offset)
}

// GetPersonalFeed is an alias for ListByUserID (TUI)
func (r *activityRepository) GetPersonalFeed(ctx context.Context, userID string, limit, offset int) ([]*models.ActivityResponse, int, error) {
    return r.ListByUserID(ctx, userID, limit, offset)
}