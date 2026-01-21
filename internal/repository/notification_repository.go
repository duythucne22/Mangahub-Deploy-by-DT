package repository

import (
    "context"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
    "mangahub/pkg/models"
)

// NotificationRepository handles notification persistence
type NotificationRepository interface {
    Create(ctx context.Context, notification *models.Notification) error
    GetNewNotifications(ctx context.Context, lastID string) ([]*models.Notification, error)
}

type notificationRepository struct {
    pool *pgxpool.Pool
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
    return &notificationRepository{pool: pool}
}

// Create inserts a notification record
func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
    query := `
        INSERT INTO notifications (id, message, created_at)
        VALUES ($1, $2, COALESCE($3, CURRENT_TIMESTAMP))
        RETURNING id, created_at
    `

    err := r.pool.QueryRow(ctx, query,
        notification.ID,
        notification.Message,
        notification.CreatedAt,
    ).Scan(&notification.ID, &notification.CreatedAt)
    if err != nil {
        return r.mapDBError(err, "create_notification")
    }
    return nil
}

func (r *notificationRepository) GetNewNotifications(ctx context.Context, lastID string) ([]*models.Notification, error) {
    if lastID == "" {
        lastID = "notif-0"
    }

    query := `
        SELECT id, message, created_at
        FROM notifications
        WHERE id > $1
        ORDER BY created_at ASC
    `
    rows, err := r.pool.Query(ctx, query, lastID)
    if err != nil {
        return nil, r.mapDBError(err, "get_new_notifications")
    }
    defer rows.Close()

    var out []*models.Notification
    for rows.Next() {
        var n models.Notification
        if err := rows.Scan(&n.ID, &n.Message, &n.CreatedAt); err != nil {
            return nil, r.mapDBError(err, "scan_notification")
        }
        out = append(out, &n)
    }

    return out, nil
}

func (r *notificationRepository) mapDBError(err error, operation string) error {
    if err == pgx.ErrNoRows {
        return models.NewHTTPError(models.ErrCodeNotFound, "resource not found", 404, err)
    }

    if pgErr, ok := err.(*pgconn.PgError); ok {
        switch pgErr.Code {
        case "23503": // foreign_key_violation
            return models.NewHTTPError(models.ErrCodeBadRequest, "invalid relationship", 400, err)
        case "22P02": // invalid_text_representation
            return models.NewHTTPError(models.ErrCodeBadRequest, "invalid input format", 400, err)
        }
    }

    return models.NewHTTPError(models.ErrCodeInternal, "database error during "+operation, 500, err)
}