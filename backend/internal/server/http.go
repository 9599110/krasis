package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/krasis/krasis/internal/admin"
	"github.com/krasis/krasis/internal/ai"
	"github.com/krasis/krasis/internal/auth"
	"github.com/krasis/krasis/internal/auditlog"
	"github.com/krasis/krasis/internal/collab"
	"github.com/krasis/krasis/internal/config"
	"github.com/krasis/krasis/internal/file"
	folderpkg "github.com/krasis/krasis/internal/folder"
	"github.com/krasis/krasis/internal/group"
	"github.com/krasis/krasis/internal/middleware"
	"github.com/krasis/krasis/internal/note"
	"github.com/krasis/krasis/internal/oauthconfig"
	"github.com/krasis/krasis/internal/search"
	"github.com/krasis/krasis/internal/share"
	"github.com/krasis/krasis/internal/systemconfig"
	"github.com/krasis/krasis/internal/user"
	"github.com/krasis/krasis/pkg/vector"
)

// keywordSearchAdapter adapts search.SearchRepository to ai.KeywordSearcher
type keywordSearchAdapter struct {
	repo *search.SearchRepository
}

func (a *keywordSearchAdapter) SearchByKeyword(ctx context.Context, query, userID string, topK int) ([]*ai.FTSResult, error) {
	results, err := a.repo.SearchByKeyword(ctx, query, userID, topK)
	if err != nil {
		return nil, err
	}
	ftsResults := make([]*ai.FTSResult, len(results))
	for i, r := range results {
		ftsResults[i] = &ai.FTSResult{
			ID:         r.ID,
			Title:      r.Title,
			Highlights: r.Highlights,
			Score:      r.Score,
		}
	}
	return ftsResults, nil
}

type Server struct {
	engine *gin.Engine
	logger *zap.Logger
	cfg    *config.Config
}

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client, logger *zap.Logger) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.CORS())
	engine.Use(middleware.Logging(logger))
	engine.Use(middleware.Metrics())

	// Services
	userRepo := user.NewUserRepository(pool)
	userService := user.NewUserService(userRepo)

	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.Expiration)*time.Second,
		cfg.JWT.Issuer,
		rdb,
	)

	sessionManager := auth.NewSessionManager(
		rdb,
		time.Duration(cfg.JWT.Expiration)*time.Second,
	)

	oauthConfigs := make(map[auth.Provider]auth.OAuthConfig)
	if cfg.OAuth.GitHub.Enabled {
		oauthConfigs[auth.ProviderGitHub] = auth.OAuthConfig{
			ClientID:     cfg.OAuth.GitHub.ClientID,
			ClientSecret: cfg.OAuth.GitHub.ClientSecret,
			RedirectURI:  cfg.OAuth.GitHub.RedirectURI,
			Scopes:       []string{"user:email"},
		}
	}
	if cfg.OAuth.Google.Enabled {
		oauthConfigs[auth.ProviderGoogle] = auth.OAuthConfig{
			ClientID:     cfg.OAuth.Google.ClientID,
			ClientSecret: cfg.OAuth.Google.ClientSecret,
			RedirectURI:  cfg.OAuth.Google.RedirectURI,
			Scopes:       []string{"email", "profile"},
		}
	}

	oauthManager := auth.NewOAuthManager(oauthConfigs)
	authHandler := auth.NewHandler(oauthManager, jwtManager, sessionManager, userService)
	userHandler := user.NewHandler(userService, sessionManager)

	// Folder module
	folderRepo := folderpkg.NewRepository(pool)
	folderService := folderpkg.NewService(folderRepo)
	folderHandler := folderpkg.NewHandler(folderService)

	// File module
	minioClient, _ := file.NewMinioClient(
		cfg.Storage.MinIO.Endpoint,
		cfg.Storage.MinIO.AccessKey,
		cfg.Storage.MinIO.SecretKey,
		cfg.Storage.MinIO.UseSSL,
	)
	fileRepo := file.NewRepository(pool)
	fileService := file.NewService(fileRepo, minioClient, cfg.Storage.MinIO.Bucket, 15*time.Minute)
	fileHandler := file.NewHandler(fileService)

	// AI module (must be before note module for indexer injection)
	aiRepo := ai.NewRepository(pool)
	modelManager := ai.NewModelConfigManager(aiRepo)
	if err := modelManager.LoadModels(context.Background()); err != nil {
		logger.Warn("failed to load AI models on startup", zap.Error(err))
	}
	// Start hot reloader (30s interval)
	modelManager.StartReloader(context.Background(), 30*time.Second, logger)

	qdrantClient := vector.NewQdrantClient(
		cfg.Storage.AI.Qdrant.Endpoint,
		cfg.Storage.AI.Qdrant.APIKey,
		cfg.Storage.AI.Qdrant.Collection,
	)

	// Search module (needed for hybrid RAG retrieval)
	searchRepo := search.NewSearchRepository(pool)
	searchService := search.NewService(searchRepo)
	searchHandler := search.NewHandler(searchService)

	// Keyword search adapter for hybrid RAG retrieval
	keywordSearcher := &keywordSearchAdapter{repo: searchRepo}

	aiService := ai.NewAIService(aiRepo, modelManager, qdrantClient, keywordSearcher)
	aiHandler := ai.NewHandler(aiService, logger)

	// Note module (with AI indexer for RAG sync)
	noteRepo := note.NewNoteRepository(pool, logger)
	noteService := note.NewNoteService(noteRepo, aiService)
	noteHandler := note.NewHandler(noteService, logger)

	// Audit repo (needed by share handler for review audit logs)
	auditRepo := auditlog.NewRepository(pool)

	// Share module
	shareRepo := share.NewShareRepository(pool)
	shareService := share.NewService(shareRepo, noteRepo)
	shareHandler := share.NewHandler(shareService, auditRepo)

	// Admin module
	sysConfigRepo := systemconfig.NewRepository(pool)
	oauthRepo := oauthconfig.NewRepository(pool)
	groupRepo := group.NewRepository(pool)
	adminHandler := admin.NewHandler(userService, noteRepo, aiRepo, modelManager, sysConfigRepo, oauthRepo, groupRepo, auditRepo, rdb, logger)

	// Collaboration module
	collabHub := collab.NewHub(rdb, logger)
	collabHandler := collab.NewHandler(collabHub, jwtManager, logger)

	// Prometheus: register collab room count as a metric
	_ = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "collab_active_rooms",
			Help: "Number of active collaboration rooms",
		},
		func() float64 { return float64(collabHub.RoomCount()) },
	)

	authMiddleware := middleware.AuthMiddleware(jwtManager)

	// Routes
	engine.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := pool.Ping(c.Request.Context()); err != nil {
			dbStatus = fmt.Sprintf("error: %v", err)
		}

		redisStatus := "ok"
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = fmt.Sprintf("error: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
			"services": gin.H{
				"database": dbStatus,
				"redis":    redisStatus,
			},
		})
	})

	// Prometheus metrics endpoint
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Auth routes (public)
	authGroup := engine.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.GET("/github/login", authHandler.GitHubLogin)
		authGroup.GET("/github/callback", authHandler.GitHubCallback)
		authGroup.GET("/google/login", authHandler.GoogleLogin)
		authGroup.GET("/google/callback", authHandler.GoogleCallback)
		authGroup.POST("/logout", authMiddleware, authHandler.Logout)
		authGroup.GET("/me", authMiddleware, userHandler.GetMe)
	}

	// User routes (authenticated)
	userGroup := engine.Group("/user")
	userGroup.Use(authMiddleware)
	{
		userGroup.GET("/sessions", userHandler.GetSessions)
		userGroup.DELETE("/sessions/:session_id", userHandler.DeleteSession)
		userGroup.DELETE("/sessions", userHandler.DeleteAllSessions)
		userGroup.PUT("/profile", userHandler.UpdateProfile)
	}

	// Folder routes (authenticated)
	folderGroup := engine.Group("/folders")
	folderGroup.Use(authMiddleware)
	{
		folderGroup.GET("", folderHandler.ListFolders)
		folderGroup.POST("", folderHandler.CreateFolder)
		folderGroup.PUT("/:id", folderHandler.UpdateFolder)
		folderGroup.DELETE("/:id", folderHandler.DeleteFolder)
	}

	// Note routes (authenticated)
	noteGroup := engine.Group("/notes")
	noteGroup.Use(authMiddleware)
	{
		noteGroup.GET("", noteHandler.ListNotes)
		noteGroup.POST("", noteHandler.CreateNote)
		noteGroup.GET("/:id", noteHandler.GetNote)
		noteGroup.PUT("/:id", noteHandler.UpdateNote)
		noteGroup.DELETE("/:id", noteHandler.DeleteNote)
		noteGroup.GET("/:id/versions", noteHandler.GetVersions)
		noteGroup.POST("/:id/versions/:version/restore", noteHandler.RestoreVersion)

		// Share routes (authenticated, under notes)
		noteGroup.POST("/:id/share", shareHandler.CreateShare)
		noteGroup.GET("/:id/share", shareHandler.GetShareStatus)
		noteGroup.DELETE("/:id/share", shareHandler.DeleteShare)
	}

	// Public share access route
	shareGroup := engine.Group("/share")
	{
		shareGroup.GET("/:token", shareHandler.AccessShare)
	}

	// Search route (authenticated)
	engine.GET("/search", authMiddleware, middleware.StatsMiddleware(rdb), searchHandler.Search)

	// AI routes (authenticated + group rate limit)
	aiGroup := engine.Group("/ai")
	aiGroup.Use(authMiddleware, middleware.StatsMiddleware(rdb), middleware.GroupLimitMiddleware(pool, rdb, "ai_ask_limit"))
	{
		aiGroup.POST("/ask", aiHandler.Ask)
		aiGroup.POST("/ask/stream", aiHandler.AskStream)
		aiGroup.GET("/conversations", aiHandler.ListConversations)
		aiGroup.GET("/conversations/:id/messages", aiHandler.GetMessages)
	}

	// File routes (authenticated)
	fileGroup := engine.Group("/files")
	fileGroup.Use(authMiddleware, middleware.StatsMiddleware(rdb))
	{
		fileGroup.GET("/presign", fileHandler.GetPresignURL)
		fileGroup.POST("/confirm", fileHandler.ConfirmUpload)
		fileGroup.DELETE("/:id", fileHandler.DeleteFile)
	}

	// WebSocket collaboration route (authenticated via query param)
	engine.GET("/ws/collab", collabHandler.HandleCollab)

	// Admin routes
	adminGroup := engine.Group("/admin")
	adminGroup.Use(authMiddleware, middleware.RequireRole("admin"))
	{
		// Admin user management
		adminGroup.GET("/users", adminHandler.ListUsers)
		adminGroup.GET("/users/:id", adminHandler.GetUser)
		adminGroup.POST("/users", adminHandler.CreateUser)
		adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
		adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
		adminGroup.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		adminGroup.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
		adminGroup.POST("/users/batch/disable", adminHandler.BatchDisableUsers)
		adminGroup.POST("/users/export", adminHandler.ExportUsers)

		// Admin stats
		adminGroup.GET("/stats/overview", adminHandler.GetStatsOverview)
		adminGroup.GET("/stats/users", adminHandler.GetUserStats)
		adminGroup.GET("/stats/usage", adminHandler.GetUsageStats)

		// Admin share review routes
		shares := adminGroup.Group("/shares")
		{
			shares.GET("/pending", shareHandler.GetPendingList)
			shares.GET("/stats", shareHandler.GetShareStats)
			shares.POST("/batch/review", shareHandler.BatchReview)
			shares.GET("/:id", shareHandler.GetShareDetail)
			shares.POST("/:id/approve", shareHandler.ApproveShare)
			shares.POST("/:id/reject", shareHandler.RejectShare)
			shares.POST("/:id/re-review", shareHandler.ReReviewShare)
			shares.DELETE("/:id/revoke", shareHandler.RevokeShare)
		}

		// Admin AI model routes
		aiModels := adminGroup.Group("/ai")
		{
			aiModels.GET("/models", adminHandler.ListModels)
			aiModels.POST("/models", adminHandler.CreateModel)
			aiModels.PUT("/models/:id", adminHandler.UpdateModel)
			aiModels.DELETE("/models/:id", adminHandler.DeleteModel)
			aiModels.POST("/models/:id/test", adminHandler.TestModel)
			aiModels.GET("/config", adminHandler.GetAIConfig)
			aiModels.PUT("/config", adminHandler.UpdateAIConfig)
			aiModels.GET("/embedding-models", adminHandler.ListEmbeddingModels)
		}

		// System config routes
		adminGroup.GET("/config", adminHandler.GetSystemConfig)
		adminGroup.PUT("/config", adminHandler.UpdateSystemConfig)

		// OAuth config routes
		adminGroup.GET("/auth/oauth", adminHandler.GetOAuthConfig)
		adminGroup.PUT("/auth/oauth", adminHandler.UpdateOAuthConfig)

		// Group management routes
		adminGroup.GET("/groups", adminHandler.ListGroups)
		adminGroup.POST("/groups", adminHandler.CreateGroup)
		adminGroup.PUT("/groups/:id", adminHandler.UpdateGroup)
		adminGroup.DELETE("/groups/:id", adminHandler.DeleteGroup)
		adminGroup.GET("/groups/:id/features", adminHandler.GetGroupFeatures)
		adminGroup.PUT("/groups/:id/features", adminHandler.UpdateGroupFeatures)

		// Audit log routes
		adminGroup.GET("/logs", adminHandler.GetAuditLogs)
	}

	return &Server{
		engine: engine,
		logger: logger,
		cfg:    cfg,
	}
}

func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Server.Port)
	s.logger.Info("starting http server", zap.String("addr", addr))

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.logger.Info("shutting down http server")
	return srv.Shutdown(shutdownCtx)
}
