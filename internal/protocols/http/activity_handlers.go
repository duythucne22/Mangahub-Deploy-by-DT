package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// getGlobalFeed returns global activity feed
func (s *Server) getGlobalFeed(c *gin.Context) {
	// Parse pagination
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	result, err := s.activitySvc.GetGlobalFeed(c.Request.Context(), limit, (page-1)*limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get global feed"})
		return
	}

	c.JSON(200, result)
}

// getUserFeed returns activity feed for a specific user
func (s *Server) getUserFeed(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id is required"})
		return
	}

	// Parse pagination
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	result, err := s.activitySvc.GetUserFeed(c.Request.Context(), userID, limit, (page-1)*limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get user feed"})
		return
	}

	c.JSON(200, result)
}

// getMangaFeed returns activity feed for a specific manga
func (s *Server) getMangaFeed(c *gin.Context) {
	mangaID := c.Param("manga_id")
	if mangaID == "" {
		c.JSON(400, gin.H{"error": "manga_id is required"})
		return
	}

	// Parse pagination
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	result, err := s.activitySvc.GetMangaFeed(c.Request.Context(), mangaID, limit, (page-1)*limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get manga feed"})
		return
	}

	c.JSON(200, result)
}
