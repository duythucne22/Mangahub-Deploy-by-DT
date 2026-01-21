package http

import (
	"time"

	"github.com/gin-gonic/gin"

	"mangahub/pkg/models"
)

// register handles user registration
func (s *Server) register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid request body",
			Timestamp: time.Now(),
		})
		return
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "username and password are required",
			Timestamp: time.Now(),
		})
		return
	}

	// Register user
	user, err := s.authSvc.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(201, models.APIResponse{
		Success:   true,
		Message:   "User registered successfully",
		Data:      gin.H{"user": user},
		Timestamp: time.Now(),
	})
}

// login handles user authentication
func (s *Server) login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid request body",
			Timestamp: time.Now(),
		})
		return
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "username and password are required",
			Timestamp: time.Now(),
		})
		return
	}

	// Login user
	resp, err := s.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "invalid credentials",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Message:   "Login successful",
		Data:      resp,
		Timestamp: time.Now(),
	})
}

// updateUserRole allows admins to change user roles
func (s *Server) updateUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "user id is required",
			Timestamp: time.Now(),
		})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid request body",
			Timestamp: time.Now(),
		})
		return
	}

	// Validate role
	validRoles := map[string]bool{
		"user":      true,
		"moderator": true,
		"admin":     true,
	}

	if !validRoles[req.Role] {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid role: must be user, moderator, or admin",
			Timestamp: time.Now(),
		})
		return
	}

	// Update role
	err := s.authSvc.UpdateUserRole(c.Request.Context(), userID, req.Role)
	if err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Get updated user
	user, err := s.authSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(500, models.APIResponse{
			Success:   false,
			Error:     "role updated but failed to fetch user",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Message:   "User role updated successfully",
		Data:      gin.H{"user": user},
		Timestamp: time.Now(),
	})
}
