package http

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	tcpProtocol "mangahub/internal/protocols/tcp"
	"mangahub/pkg/models"
)

// createComment creates a new comment
func (s *Server) createComment(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}

	mangaID := c.Param("id")  // Changed from manga_id to id
	if mangaID == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "manga_id is required",
			Timestamp: time.Now(),
		})
		return
	}

	var req models.CreateCommentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "invalid request body",
			Timestamp: time.Now(),
		})
		return
	}

	comment, err := s.commentSvc.Create(c.Request.Context(), mangaID, userID, req)
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
	_ = s.activitySvc.CreateActivity(c.Request.Context(), "comment", &userID, &mangaID)

	// 2. Emit TCP stats event for real-time aggregation
	if s.tcpAddr != "" {
		go func() {
			event := tcpProtocol.StatsEvent{
				Type:      tcpProtocol.EventTypeComment,
				MangaID:   mangaID,
				UserID:    &userID,
				EventTime: time.Now(),
				Weight:    1,
				Source:    "http",
			}
			if err := tcpProtocol.SendStatsEvent(s.tcpAddr, event); err != nil {
				// Log error but don't fail the request
				fmt.Printf("Failed to emit TCP stats event: %v\n", err)
			}
		}()
	}

	c.JSON(201, models.APIResponse{
		Success:   true,
		Message:   "Comment created successfully",
		Data:      comment,
		Timestamp: time.Now(),
	})
}

// listComments returns comments for a manga
func (s *Server) listComments(c *gin.Context) {
	mangaID := c.Param("id")  // Changed from manga_id to id
	if mangaID == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "manga_id is required",
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

	result, err := s.commentSvc.ListByMangaID(c.Request.Context(), mangaID, limit, offset)
	if err != nil {
		c.JSON(500, models.APIResponse{
			Success:   false,
			Error:     "failed to list comments",
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

// likeComment increments the like count for a comment
func (s *Server) likeComment(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}

	commentID := c.Param("comment_id")
	if commentID == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "comment_id is required",
			Timestamp: time.Now(),
		})
		return
	}

	comment, err := s.commentSvc.IncrementLikes(c.Request.Context(), commentID, userID)
	if err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Record activity
	_ = s.activitySvc.CreateActivity(c.Request.Context(), "comment", &userID, nil)

	c.JSON(200, models.APIResponse{
		Success:   true,
		Message:   "Comment liked successfully",
		Data:      comment,
		Timestamp: time.Now(),
	})
}

// deleteComment deletes a comment (only owner can delete)
func (s *Server) deleteComment(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(401, models.APIResponse{
			Success:   false,
			Error:     "unauthorized",
			Timestamp: time.Now(),
		})
		return
	}

	commentID := c.Param("comment_id")
	if commentID == "" {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     "comment_id is required",
			Timestamp: time.Now(),
		})
		return
	}

	// Get comment to verify ownership
	comment, err := s.commentSvc.GetByID(c.Request.Context(), commentID)
	if err != nil {
		c.JSON(404, models.APIResponse{
			Success:   false,
			Error:     "comment not found",
			Timestamp: time.Now(),
		})
		return
	}

	if comment.UserID != userID {
		c.JSON(403, models.APIResponse{
			Success:   false,
			Error:     "you can only delete your own comments",
			Timestamp: time.Now(),
		})
		return
	}

	if err := s.commentSvc.Delete(c.Request.Context(), commentID, userID); err != nil {
		c.JSON(400, models.APIResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(200, models.APIResponse{
		Success:   true,
		Message:   "Comment deleted successfully",
		Timestamp: time.Now(),
	})
}
