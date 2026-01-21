package models

import (
	"errors"
	"time"
)

// UserRole represents valid user roles 
type UserRole string

const (
	UserRoleUser      UserRole = "user"
	UserRoleModerator UserRole = "moderator"
	UserRoleAdmin     UserRole = "admin"
)

// User represents a system user
type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         UserRole  `json:"role" db:"role" validate:"required,oneof=user moderator admin"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// RegisterRequest 
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password string `json:"password" validate:"required,min=12,max=128,containsany=@$!%*#?&"`
}

// LoginRequest 
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// UserProfile - public-facing profile, NO sensitive data
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResponse
type LoginResponse struct {
	Token    string      `json:"token"`
	User     UserProfile `json:"user"`
	ExpiresIn int         `json:"expires_in"` // seconds (client-friendly)
}

// ValidateRegisterRequest adds additional validation beyond struct tags
func ValidateRegisterRequest(req *RegisterRequest) error {
	if len(req.Password) < 12 {
		return errors.New("password must be at least 12 characters with special characters")
	}
	if len(req.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	return nil
}

// HasRole checks if user has required role (for middleware)
func (u *User) HasRole(requiredRole UserRole) bool {
	switch requiredRole {
	case UserRoleAdmin:
		return u.Role == UserRoleAdmin
	case UserRoleModerator:
		return u.Role == UserRoleModerator || u.Role == UserRoleAdmin
	default: // UserRoleUser
		return true 
	}
}