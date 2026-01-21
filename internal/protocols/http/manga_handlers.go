package http

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	tcpProtocol "mangahub/internal/protocols/tcp"
	udpProtocol "mangahub/internal/protocols/udp"
	"mangahub/pkg/models"
)

// createManga handles manga creation
func (s *Server) createManga(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}

	user, ok := GetUser(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}
	if user.Role != models.UserRoleAdmin {
		c.JSON(403, models.APIResponse{
			Success:   false,
			Error:     "forbidden: admin access required",
			Timestamp: time.Now(),
		})
		return
	}

	var req models.CreateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid request body",
			Timestamp: time.Now(),
		})
		return
	}

	manga, err := s.mangaSvc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// CROSS-PROTOCOL INTEGRATION:
	// 1. Record activity to activity_feed
	_ = s.activitySvc.CreateActivity(c.Request.Context(), "manga_update", &userID, &manga.ID)

	// 2. UDP broadcast notification (admin action broadcast)
	if s.udpServer != nil {
		mangaID := manga.ID
		notification := udpProtocol.Notification{
			Type:    "manga_update",
			MangaID: &mangaID,
			Title:   manga.Title,
			Message: fmt.Sprintf("New manga '%s' added by %s", manga.Title, user.Username),
		}
		s.udpServer.Broadcast(notification)
	}

	// 3. TCP stats event
	if s.tcpAddr != "" {
		event := tcpProtocol.StatsEvent{
			Type:      tcpProtocol.EventTypeUpdate,
			MangaID:   manga.ID,
			UserID:    &userID,
			EventTime: time.Now(),
			Weight:    5,
			Source:    "http",
		}
		_ = tcpProtocol.SendStatsEvent(s.tcpAddr, event)
	}

	c.JSON(201, models.APIResponse{
		Success:   true,
		Message:   "Manga created successfully",
		Data:      manga,
		Timestamp: time.Now(),
	})
}

// getManga retrieves a single manga by ID
func (s *Server) getManga(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "manga id is required",
			Timestamp: time.Now(),
		})
		return
	}

	manga, err := s.mangaSvc.GetWithGenres(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, models.APIResponse{
			Success:   false,
			Error:     "manga not found",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Data:      manga,
		Timestamp: time.Now(),
	})
}

// listManga returns a paginated list of manga
func (s *Server) listManga(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	limit := 20

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

	// Get filter parameters
	status := c.Query("status")
	genre := c.Query("genre")

	// Build search request
	req := models.MangaSearchRequest{
		Query:  "",
		Genres: []string{},
		Status: status,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	if genre != "" {
		req.Genres = []string{genre}
	}

	result, err := s.mangaSvc.List(c.Request.Context(), req)
	if err != nil {
		c.JSON(500, models.APIResponse{
			Success:   false,
			Error:     "failed to list manga",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Data:      result,
		Timestamp: time.Now(),
	})
}

// searchManga searches for manga by title
func (s *Server) searchManga(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "search query is required",
			Timestamp: time.Now(),
		})
		return
	}

	// Parse pagination
	page := 1
	limit := 20

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

	offset := (page - 1) * limit

	result, err := s.mangaSvc.Search(c.Request.Context(), query, limit, offset)
	if err != nil {
		c.JSON(500, models.APIResponse{
			Success:   false,
			Error:     "search failed",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Data:      result,
		Timestamp: time.Now(),
	})
}

// getTrendingManga returns trending manga (top manga by weekly score)
func (s *Server) getTrendingManga(c *gin.Context) {
	// Parse limit
	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	// Use GetTopManga from stats service which returns top manga by weekly score
	result, err := s.statsSvc.GetTopManga(c.Request.Context(), limit, 0)
	if err != nil {
		c.JSON(500, models.APIResponse{
			Success:   false,
			Error:     "failed to get trending manga",
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Data:      result,
		Timestamp: time.Now(),
	})
}

// updateManga updates manga information
func (s *Server) updateManga(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "manga id is required",
			Timestamp: time.Now(),
		})
		return
	}

	var req models.UpdateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid request body",
			Timestamp: time.Now(),
		})
		return
	}

	manga, err := s.mangaSvc.Update(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Record activity
	_ = s.activitySvc.CreateActivity(c.Request.Context(), "manga_update", &userID, &manga.ID)

	c.JSON(200, models.APIResponse{
		Success:   true,
		Message:   "Manga updated successfully",
		Data:      manga,
		Timestamp: time.Now(),
	})
}

// deleteManga deletes a manga
func (s *Server) deleteManga(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "manga id is required",
			Timestamp: time.Now(),
		})
		return
	}

	if err := s.mangaSvc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Record activity
	_ = s.activitySvc.CreateActivity(c.Request.Context(), "manga_update", &userID, &id)

	c.JSON(200, models.APIResponse{
		Success:   true,
		Message:   "Manga deleted successfully",
		Timestamp: time.Now(),
	})
}
