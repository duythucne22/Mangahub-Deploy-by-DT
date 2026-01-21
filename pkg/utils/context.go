package utils

import (
	"context"
	"time"
)

// DefaultTimeout is the standard timeout for database operations
const DefaultTimeout = 5 * time.Second

// WithTimeout creates context with default timeout
func WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultTimeout)
}

// WithLongTimeout creates context with longer timeout (for search, etc.)
func WithLongTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 10*time.Second)
}

// IsContextError checks if error is from context cancellation
func IsContextError(err error) bool {
	return err != nil && (err == context.Canceled || err == context.DeadlineExceeded)
}