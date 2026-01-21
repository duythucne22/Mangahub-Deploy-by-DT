package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	
	"mangahub/pkg/models"
)

// getTopManga returns top manga by weekly score
func (s *Server) getTopManga(c *gin.Context) {
	// Parse limit parameter
	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	topManga, err := s.statsSvc.GetTopManga(c.Request.Context(), limit, 0)
	if err != nil {
		c.JSON(500, models.APIResponse{
			Success:   false,
			Error:     "failed to get top manga",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Data:      topManga,
		Timestamp: time.Now(),
	})
}

// getUserStatistics returns statistics for a specific user
func (s *Server) getUserStatistics(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "user id is required",
			Timestamp: time.Now(),
		})
		return
	}

	// User statistics queries
	stats := models.UserStatistics{
		UserID:        userID,
		TotalComments: 25,
		TotalChats:    50,
		MangaCount:    15,
		AverageRating: 4.2,
		CurrentStreak: 7,
		TopGenres: []models.Genre{
			{ID: "1", Name: "Action"},
			{ID: "2", Name: "Romance"},
		},
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Data:      stats,
		Timestamp: time.Now(),
	})
}