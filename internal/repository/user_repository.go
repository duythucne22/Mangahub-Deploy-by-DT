package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
	"mangahub/pkg/models"
)

// UserRepository handles user data persistence with protocol-aware methods
type UserRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

// Create inserts a new user into the database with UUID generation
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, role, created_at)
		VALUES (COALESCE($1, uuid_generate_v4()::text), $2, $3, $4, COALESCE($5, CURRENT_TIMESTAMP))
		RETURNING id, created_at
	`
	
	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		string(user.Role),
		user.CreatedAt,
	).Scan(&user.ID, &user.CreatedAt)
	
	if err != nil {
		return r.mapDBError(err, "create_user")
	}
	return nil
}

// GetByID retrieves a user by ID with proper role type conversion
func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, role, created_at
		FROM users
		WHERE id = $1
	`
	user := &models.User{}
	var roleStr string
	
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&roleStr,
		&user.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, models.NewHTTPError(models.ErrCodeNotFound, "user not found", 404, err)
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_user_by_id")
	}
	
	user.Role = models.UserRole(roleStr)
	return user, nil
}

// GetByUsername retrieves a user by username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, role, created_at
		FROM users
		WHERE username = $1
	`
	user := &models.User{}
	var roleStr string
	
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&roleStr,
		&user.CreatedAt,
	)
	
	if err == pgx.ErrNoRows {
		return nil, models.NewHTTPError(models.ErrCodeNotFound, "user not found", 404, err)
	}
	if err != nil {
		return nil, r.mapDBError(err, "get_user_by_username")
	}
	
	user.Role = models.UserRole(roleStr)
	return user, nil
}

// UsernameExists checks if a username is already taken
func (r *userRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	
	err := r.pool.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, r.mapDBError(err, "check_username_exists")
	}
	return exists, nil
}

// Update updates user information with role validation
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, password_hash = $3, role = $4
		WHERE id = $1
		RETURNING created_at
	`
	
	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		string(user.Role),
	).Scan(&user.CreatedAt)
	
	if err != nil {
		return r.mapDBError(err, "update_user")
	}
	return nil
}

// Delete removes a user from the database
func (r *userRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM users WHERE id = $1
		RETURNING id
	`
	var deletedID string
	
	err := r.pool.QueryRow(ctx, query, id).Scan(&deletedID)
	if err == pgx.ErrNoRows {
		return models.NewHTTPError(models.ErrCodeNotFound, "user not found", 404, err)
	}
	if err != nil {
		return r.mapDBError(err, "delete_user")
	}
	return nil
}

// WithTransaction executes a function within a database transaction
func (r *userRepository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
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

// mapDBError maps database errors to protocol-specific error responses
func (r *userRepository) mapDBError(err error, operation string) error {
	if err == pgx.ErrNoRows {
		return models.NewHTTPError(models.ErrCodeNotFound, "resource not found", 404, err)
	}
	
	if pgErr, ok := err.(*pgconn.PgError); ok {
		switch pgErr.Code {
		case "23505": // unique_violation
			if operation == "create_user" {
				return models.NewHTTPError(models.ErrCodeConflict, "username already exists", 409, err)
			}
			return models.NewHTTPError(models.ErrCodeConflict, "resource already exists", 409, err)
		case "23503": // foreign_key_violation
			return models.NewHTTPError(models.ErrCodeBadRequest, "invalid relationship", 400, err)
		case "22P02": // invalid_text_representation
			return models.NewHTTPError(models.ErrCodeBadRequest, "invalid input format", 400, err)
		}
	}
	
	// Default to internal server error
	return models.NewHTTPError(models.ErrCodeInternal, "database error during "+operation, 500, err)
}
