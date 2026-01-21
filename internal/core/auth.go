// Package core - Core Business Logic
// Protocol-agnostic authentication service
// Handles user registration, login, and JWT token management
package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidToken       = errors.New("invalid token")
)

// AuthService defines authentication operations
type AuthService interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(ctx context.Context, tokenString string) (*models.User, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	UpdateUserRole(ctx context.Context, userID string, newRole string) error
}

type authService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
	jwtIssuer string
	jwtExpiry time.Duration
}

// JWT claims structure
type jwtClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo repository.UserRepository, jwtSecret, jwtIssuer string, jwtExpiry time.Duration) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		jwtIssuer: jwtIssuer,
		jwtExpiry: jwtExpiry,
	}
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	// Validate input
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return nil, fmt.Errorf("username must be between 3 and 50 characters")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Check if username exists
	exists, err := s.userRepo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Clear password hash before returning
	user.PasswordHash = ""
	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate JWT token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.LoginResponse{
		Token: token,
		User: models.UserProfile{
			ID:        user.ID,
			Username:  user.Username,
			CreatedAt: user.CreatedAt,
		},
		ExpiresIn: int(time.Until(expiresAt).Seconds()),
	}, nil
}

// ValidateToken verifies a JWT token and returns the user
func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*models.User, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Get user from database
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *authService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	// Clear password hash
	user.PasswordHash = ""
	return user, nil
}

// UpdateUserRole updates a user's role (admin only)
func (s *authService) UpdateUserRole(ctx context.Context, userID string, newRole string) error {
	// Validate role
	validRoles := map[string]bool{
		"user":      true,
		"moderator": true,
		"admin":     true,
	}

	if !validRoles[newRole] {
		return fmt.Errorf("invalid role: %s (must be user, moderator, or admin)", newRole)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update role
	user.Role = models.UserRole(newRole)
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return nil
}

// generateToken creates a new JWT token for a user
func (s *authService) generateToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.jwtExpiry)

	claims := &jwtClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.jwtIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}
