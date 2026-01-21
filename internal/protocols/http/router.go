package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"mangahub/internal/core"
	udpProtocol "mangahub/internal/protocols/udp"
	"mangahub/pkg/config"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Server manages HTTP REST API server
type Server struct {
	router      *gin.Engine
	config      *config.Config
	authSvc     core.AuthService
	mangaSvc    core.MangaService
	commentSvc  core.CommentService
	chatSvc     core.ChatService
	activitySvc core.ActivityService
	statsSvc    core.StatsService
	udpServer   *udpProtocol.Server // For broadcasting admin events
	tcpAddr     string              // TCP server address for stats events
}

// NewServer creates a new HTTP server with all handlers
func NewServer(
	cfg *config.Config,
	authSvc core.AuthService,
	mangaSvc core.MangaService,
	commentSvc core.CommentService,
	chatSvc core.ChatService,
	activitySvc core.ActivityService,
	statsSvc core.StatsService,
) *Server {
	// Set Gin to release mode by default
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	
	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	
	s := &Server{
		router:      router,
		config:      cfg,
		authSvc:     authSvc,
		mangaSvc:    mangaSvc,
		commentSvc:  commentSvc,
		chatSvc:     chatSvc,
		activitySvc: activitySvc,
		statsSvc:    statsSvc,
	}

	s.setupRoutes()
	return s
}

// SetCrossProtocolServers sets UDP and TCP servers for cross-protocol event emission
func (s *Server) SetCrossProtocolServers(udpServer *udpProtocol.Server, tcpAddr string) {
	s.udpServer = udpServer
	s.tcpAddr = tcpAddr
}

// setupRoutes registers all HTTP routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.register)
			auth.POST("/login", s.login)
		}

		// Admin routes (requires admin role)
		admin := v1.Group("/admin", AuthMiddleware(s.authSvc), AdminMiddleware(s.authSvc))
		{
			admin.PUT("/users/:id/role", s.updateUserRole)  // Update user role
		}

		// Manga routes
		v1.GET("/manga", s.listManga)                  // Public: list manga
		v1.GET("/manga/search", s.searchManga)         // Public: search
		v1.GET("/manga/trending", s.getTrendingManga)  // Public: trending manga
		v1.GET("/manga/:id", s.getManga)               // Public: get single manga
		
		// Protected manga routes
		protected := v1.Group("", AuthMiddleware(s.authSvc))
		{
			protected.POST("/manga", s.createManga)            // Create manga
			protected.PUT("/manga/:id", s.updateManga)         // Update manga
			protected.DELETE("/manga/:id", s.deleteManga)      // Delete manga
		}

		// Comment routes (use same parameter name :id to avoid conflicts)
		v1.GET("/manga/:id/comments", s.listComments)        // Public: list comments
		
		protectedComments := v1.Group("", AuthMiddleware(s.authSvc))
		{
			protectedComments.POST("/manga/:id/comments", s.createComment)                     // Create comment
			protectedComments.POST("/manga/:id/comments/:comment_id/like", s.likeComment)      // Like comment
			protectedComments.DELETE("/manga/:id/comments/:comment_id", s.deleteComment)       // Delete own comment
		}

		// Activity routes
		activity := v1.Group("/activity")
		{
			activity.GET("/global", s.getGlobalFeed)           // Public: global feed
			activity.GET("/recent", s.getGlobalFeed)           // Alias for global feed
			activity.GET("/manga/:manga_id", s.getMangaFeed)   // Public: manga feed
			
			protected := activity.Group("", AuthMiddleware(s.authSvc))
			{
				protected.GET("/user/:user_id", s.getUserFeed)  // Get user feed
			}
		}

		// Stats routes (public)
		stats := v1.Group("/stats")
		{
			stats.GET("/top", s.getTopManga)  // Top manga by weekly score
		}

		// Statistics and leaderboard routes (public)
		v1.GET("/statistics/user/:id", s.getUserStatistics)  // User statistics
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}

// Router returns the gin router (for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}

// corsMiddleware handles CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
