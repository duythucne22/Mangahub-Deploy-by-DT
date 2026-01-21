package http

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/internal/core"
	"mangahub/pkg/models"
)

// AuthMiddleware validates JWT token and sets user context
func AuthMiddleware(authSvc core.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		user, err := authSvc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Store user ID and full user in context
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Next()
	}
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUser retrieves the full authenticated user from the context
func GetUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	u, ok := user.(*models.User)
	return u, ok
}

// AdminMiddleware ensures the user has admin role
func AdminMiddleware(authSvc core.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
		user, exists := GetUser(c)
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Check if user is admin
		if user.Role != "admin" {
			c.JSON(403, gin.H{"error": "forbidden: admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
