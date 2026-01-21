package utils

import (
	"regexp"
	"strings"
	"time"

	"mangahub/pkg/models"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	passwordRegex = regexp.MustCompile(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{12,128}$`)
)

// ValidateUsername checks if username meets SPEC.md requirements
func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return models.ErrInvalidInput
	}
	return nil
}

// ValidatePassword checks if password meets security requirements
func ValidatePassword(password string) error {
	if !passwordRegex.MatchString(password) {
		return models.ErrInvalidInput
	}
	return nil
}

// ValidateMangaTitle validates manga title
func ValidateMangaTitle(title string) error {
	if len(strings.TrimSpace(title)) < 2 {
		return models.ErrInvalidInput
	}
	if len(title) > 255 {
		return models.ErrInvalidInput
	}
	return nil
}

// IsRecentTime checks if time is within specified duration
func IsRecentTime(t time.Time, duration time.Duration) bool {
	return time.Since(t) <= duration
}