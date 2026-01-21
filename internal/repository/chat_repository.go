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

// ChatRepository handles chat message data persistence with protocol integration
type ChatRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, message *models.ChatMessage) (*models.ChatMessageResponse, error)
	GetByID(ctx context.Context, id string) (*models.ChatMessage, error)
	ListByMangaID(ctx context.Context, mangaID string, limit, offset int) ([]*models.ChatMessageResponse, int, error)
	Delete(ctx context.Context, id string) error
	
	// Protocol-specific methods
	StreamMessages(ctx context.Context, mangaID string, lastMessageID *string) (<-chan *models.ChatMessageResponse, error)
	GetRoomPresence(ctx context.Context, mangaID string) ([]*models.UserPresence, error)
	BroadcastMessage(ctx context.Context, message *models.ChatMessage) error
	
	// Stats & Activity integration
	LogChatActivity(ctx context.Context, message *models.ChatMessage) error
	GetChatStats(ctx context.Context, mangaID string) (*models.MangaStats, error)
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type chatRepository struct {
	pool *pgxpool.Pool
}

// NewChatRepository creates a new PostgreSQL chat repository
func NewChatRepository(pool *pgxpool.Pool) ChatRepository {
	return &chatRepository{pool: pool}
}

// Create inserts a new chat message with activity logging and stats events
func (r *chatRepository) Create(ctx context.Context, message *models.ChatMessage) (*models.ChatMessageResponse, error) {
	var response *models.ChatMessageResponse
	
	err := r.WithTransaction(ctx, func(tx pgx.Tx) error {
		if message.ID == "" {
			message.ID = generateUUID("chat")
		}

		// Insert chat message
		insertQuery := `
			INSERT INTO chat_messages (id, manga_id, user_id, content, created_at)
			VALUES ($1, $2, $3, $4, COALESCE($5, CURRENT_TIMESTAMP))
			RETURNING id, created_at
		`
		
		err := tx.QueryRow(ctx, insertQuery,
			message.ID,
			message.MangaID,
			message.UserID,
			message.Content,
			message.CreatedAt,
		).Scan(&message.ID, &message.CreatedAt)
		
		if err != nil {
			return r.mapDBError(err, "create_chat_message")
		}
		
		// Log to activity feed
		activity := &models.Activity{
			ID:        generateUUID("act"),
			Type:      models.ActivityTypeChat,
			UserID:    &message.UserID,
			MangaID:   &message.MangaID,
			CreatedAt: message.CreatedAt,
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
			return r.mapDBError(err, "log_chat_activity")
		}
		
		statsQuery := `
			INSERT INTO manga_stats (manga_id, chat_count, weekly_score, updated_at)
			VALUES ($1, 1, 2, CURRENT_TIMESTAMP)
			ON CONFLICT (manga_id) DO UPDATE
			SET chat_count = manga_stats.chat_count + 1,
				weekly_score = manga_stats.weekly_score + 2,
				updated_at = CURRENT_TIMESTAMP
		`
		_, err = tx.Exec(ctx, statsQuery, message.MangaID)
		if err != nil {
			return r.mapDBError(err, "update_chat_stats")
		}

		// Get user info for response
		userQuery := `SELECT username FROM users WHERE id = $1`
		var username string
		err = tx.QueryRow(ctx, userQuery, message.UserID).Scan(&username)
		if err != nil {
			return r.mapDBError(err, "get_chat_user")
		}
		
		response = &models.ChatMessageResponse{
			ID:        message.ID,
			MangaID:   message.MangaID,
			User:      models.ChatUser{ID: message.UserID, Username: username},
			Content:   message.Content,
			CreatedAt: message.CreatedAt,
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return response, nil
}

// GetByID retrieves a chat message by ID
func (r *chatRepository) GetByID(ctx context.Context, id string) (*models.ChatMessage, error) {
	query := `
		SELECT id, manga_id, user_id, content, created_at
		FROM chat_messages
		WHERE id = $1
	`
	message := &models.ChatMessage{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&message.ID,
		&message.MangaID,
		&message.UserID,
		&message.Content,
		&message.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, r.mapDBError(err, "get_chat_message_by_id")
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_chat_message_by_id")
	}
	return message, nil
}

// ListByMangaID retrieves chat messages for a manga with user info and pagination
func (r *chatRepository) ListByMangaID(ctx context.Context, mangaID string, limit, offset int) ([]*models.ChatMessageResponse, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM chat_messages WHERE manga_id = $1`
	err := r.pool.QueryRow(ctx, countQuery, mangaID).Scan(&total)
	if err != nil {
		return nil, 0, r.mapDBError(err, "count_chat_messages")
	}
	
	// Get paginated results with user info
	query := `
		SELECT 
			cm.id, cm.manga_id, cm.user_id, cm.content, cm.created_at,
			u.username
		FROM chat_messages cm
		INNER JOIN users u ON cm.user_id = u.id
		WHERE cm.manga_id = $1
		ORDER BY cm.created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.pool.Query(ctx, query, mangaID, limit, offset)
	if err != nil {
		return nil, 0, r.mapDBError(err, "list_chat_messages")
	}
	defer rows.Close()
	
	var messages []*models.ChatMessageResponse
	for rows.Next() {
		var msg models.ChatMessageResponse
		var username string
		
		err := rows.Scan(
			&msg.ID,
			&msg.MangaID,
			&msg.User.ID,
			&msg.Content,
			&msg.CreatedAt,
			&username,
		)
		if err != nil {
			return nil, 0, r.mapDBError(err, "scan_chat_message")
		}
		
		msg.User.Username = username
		messages = append(messages, &msg)
	}
	
	return messages, total, nil
}

// StreamMessages provides a channel for real-time message streaming (WebSocket optimized)
func (r *chatRepository) StreamMessages(ctx context.Context, mangaID string, lastMessageID *string) (<-chan *models.ChatMessageResponse, error) {
	// This would typically use LISTEN/NOTIFY or a cursor-based approach
	// For simplicity, we'll create a channel that can be used by WebSocket hub
	messageChan := make(chan *models.ChatMessageResponse, 100)
	
	// In production, this would set up a PostgreSQL LISTEN on a channel
	// For now, we'll close the channel immediately as this is repository-only
	go func() {
		defer close(messageChan)
		// In production: listen for new messages and push to channel
	}()
	
	return messageChan, nil
}

// GetRoomPresence gets current user presence for a manga chat room
func (r *chatRepository) GetRoomPresence(ctx context.Context, mangaID string) ([]*models.UserPresence, error) {
	// In production, this would query a presence cache or session table
	// For SPEC.md compliance, we'll return a placeholder with active users from recent chat
	
	query := `
		SELECT DISTINCT ON (cm.user_id) 
			cm.user_id, u.username, cm.created_at as last_active
		FROM chat_messages cm
		INNER JOIN users u ON cm.user_id = u.id
		WHERE cm.manga_id = $1
			AND cm.created_at >= NOW() - INTERVAL '5 minutes'
		ORDER BY cm.user_id, cm.created_at DESC
		LIMIT 50
	`
	
	rows, err := r.pool.Query(ctx, query, mangaID)
	if err != nil {
		return nil, r.mapDBError(err, "get_room_presence")
	}
	defer rows.Close()
	
	var presences []*models.UserPresence
	for rows.Next() {
		var presence models.UserPresence
		var lastActive time.Time
		
		err := rows.Scan(
			&presence.UserID,
			&presence.Username,
			&lastActive,
		)
		if err != nil {
			return nil, r.mapDBError(err, "scan_presence")
		}
		
		presence.MangaID = mangaID
		presence.Status = "[ONLINE]"
		presence.LastActive = lastActive
		presences = append(presences, &presence)
	}
	
	return presences, nil
}

// BroadcastMessage broadcasts a message to all connected clients (admin use)
func (r *chatRepository) BroadcastMessage(ctx context.Context, message *models.ChatMessage) error {
	_, err := r.Create(ctx, message)
	return err
}

// LogChatActivity logs chat activity for TCP Stats Service
func (r *chatRepository) LogChatActivity(ctx context.Context, message *models.ChatMessage) error {
	// No-op: handled atomically in Create
	return nil
}

// GetChatStats gets chat statistics for a manga
func (r *chatRepository) GetChatStats(ctx context.Context, mangaID string) (*models.MangaStats, error) {
	query := `
		SELECT chat_count, updated_at
		FROM manga_stats
		WHERE manga_id = $1
	`
	
	stats := &models.MangaStats{MangaID: mangaID}
	err := r.pool.QueryRow(ctx, query, mangaID).Scan(
		&stats.ChatCount,
		&stats.UpdatedAt,
	)
	
	if err == pgx.ErrNoRows {
		// Initialize stats if not exists
		stats.ChatCount = 0
		stats.UpdatedAt = time.Now()
		return stats, nil
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_chat_stats")
	}
	
	return stats, nil
}

// Delete removes a chat message and associated activity
func (r *chatRepository) Delete(ctx context.Context, id string) error {
	return r.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Get message first to log proper activity
		var mangaID, userID string
		getQuery := `SELECT manga_id, user_id FROM chat_messages WHERE id = $1`
		err := tx.QueryRow(ctx, getQuery, id).Scan(&mangaID, &userID)
		if err == pgx.ErrNoRows {
			return r.mapDBError(err, "delete_chat_message")
		}
		if err != nil {
			return r.mapDBError(err, "delete_chat_message")
		}
		
		// Delete chat message
		deleteQuery := `DELETE FROM chat_messages WHERE id = $1`
		result, err := tx.Exec(ctx, deleteQuery, id)
		if err != nil {
			return r.mapDBError(err, "delete_chat_message")
		}
		
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return r.mapDBError(pgx.ErrNoRows, "delete_chat_message")
		}
		
		statsQuery := `
			UPDATE manga_stats
			SET chat_count = GREATEST(chat_count - 1, 0),
				weekly_score = GREATEST(weekly_score - 2, 0),
				updated_at = CURRENT_TIMESTAMP
			WHERE manga_id = $1
		`
		_, err = tx.Exec(ctx, statsQuery, mangaID)
		if err != nil {
 			return r.mapDBError(err, "update_chat_stats")
		}

		return nil
	})
}

// WithTransaction executes a function within a database transaction
func (r *chatRepository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
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
func (r *chatRepository) mapDBError(err error, operation string) error {
	if err == pgx.ErrNoRows {
		switch operation {
		case "get_chat_message_by_id", "delete_chat_message":
			return fmt.Errorf("%s: %w", operation, models.ErrNotFound)
		default:
			return fmt.Errorf("resource not found: %w", err)
		}
	}
	
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			if operation == "create_chat_message" {
				return fmt.Errorf("invalid manga or user reference: %w", err)
			}
			return fmt.Errorf("foreign key violation: %w", err)
		case "22001": // string_data_right_truncation
			return fmt.Errorf("message content too long: %w", err)
		}
	}
	
	return fmt.Errorf("database error during %s: %w", operation, err)
}