package admin

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/internal/ai"
	"github.com/krasis/krasis/internal/auditlog"
	"github.com/krasis/krasis/internal/group"
	"github.com/krasis/krasis/internal/middleware"
	"github.com/krasis/krasis/internal/note"
	"github.com/krasis/krasis/internal/oauthconfig"
	"github.com/krasis/krasis/internal/systemconfig"
	"github.com/krasis/krasis/internal/user"
	"github.com/krasis/krasis/pkg/response"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Handler struct {
	userService    *user.UserService
	noteRepo       *note.NoteRepository
	aiRepo         *ai.Repository
	modelManager   *ai.ModelConfigManager
	sysConfigRepo  *systemconfig.Repository
	oauthRepo      *oauthconfig.Repository
	groupRepo      *group.Repository
	auditRepo      *auditlog.Repository
	redis          *redis.Client
	logger         *zap.Logger
}

func NewHandler(userService *user.UserService, noteRepo *note.NoteRepository, aiRepo *ai.Repository, modelManager *ai.ModelConfigManager,
	sysConfigRepo *systemconfig.Repository, oauthRepo *oauthconfig.Repository,
	groupRepo *group.Repository, auditRepo *auditlog.Repository, rdb *redis.Client, logger *zap.Logger) *Handler {
	return &Handler{
		userService:    userService,
		noteRepo:       noteRepo,
		aiRepo:         aiRepo,
		modelManager:   modelManager,
		sysConfigRepo:  sysConfigRepo,
		oauthRepo:      oauthRepo,
		groupRepo:      groupRepo,
		auditRepo:      auditRepo,
		redis:          rdb,
		logger:         logger,
	}
}

func (h *Handler) audit(c *gin.Context, action string, targetType string, targetID uuid.NullUUID, changes interface{}) {
	adminIDStr, _ := c.Get("user_id")
	if adminIDStr == nil {
		return
	}
	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		return
	}
	var changesJSON json.RawMessage
	if changes != nil {
		changesJSON, _ = json.Marshal(changes)
	}
	_ = h.auditRepo.Create(c.Request.Context(), &auditlog.AuditLog{
		Action:     action,
		TargetType: sql.NullString{String: targetType, Valid: targetType != ""},
		TargetID:   targetID,
		AdminID:    adminID,
		Changes:    changesJSON,
		IPAddress:  sql.NullString{String: c.ClientIP(), Valid: true},
		UserAgent:  sql.NullString{String: c.GetHeader("User-Agent"), Valid: true},
	})
}

func (h *Handler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	role := c.Query("role")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	items, total, err := h.userService.ListUsers(c, keyword, role, page, size)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取用户列表失败")
		return
	}

	response.SuccessPaginated(c, items, total, page, size)
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if err := h.userService.UpdateUserRole(c, userID, req.Role); err != nil {
		response.Error(c, 400, response.ErrBadRequest, err.Error())
		return
	}

	h.audit(c, "user.update_role", "user", uuid.NullUUID{UUID: userID, Valid: true}, map[string]interface{}{"role": req.Role})
	response.Success(c, nil)
}

func (h *Handler) UpdateUserStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	var req struct {
		Status int16 `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if err := h.userService.UpdateUserStatus(c, userID, req.Status); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "更新状态失败")
		return
	}

	h.audit(c, "user.update_status", "user", uuid.NullUUID{UUID: userID, Valid: true}, map[string]interface{}{"status": req.Status})
	response.Success(c, nil)
}

// AI Model management handlers
func (h *Handler) ListModels(c *gin.Context) {
	modelType := c.Query("type")
	models, err := h.aiRepo.ListModels(c, modelType)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取模型列表失败")
		return
	}
	response.Success(c, models)
}

func (h *Handler) ListEmbeddingModels(c *gin.Context) {
	models, err := h.aiRepo.ListModels(c, "embedding")
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取嵌入模型列表失败")
		return
	}
	response.Success(c, models)
}

func (h *Handler) CreateModel(c *gin.Context) {
	var req struct {
		Name        string          `json:"name" binding:"required"`
		Provider    string          `json:"provider" binding:"required"`
		ModelType   string          `json:"type" binding:"required"`
		Endpoint    string          `json:"endpoint"`
		APIKey      string          `json:"api_key"`
		ModelName   string          `json:"model_name" binding:"required"`
		APIVersion  string          `json:"api_version"`
		MaxTokens   int             `json:"max_tokens"`
		Temperature float64         `json:"temperature"`
		TopP        float64         `json:"top_p"`
		Dimensions  int             `json:"dimensions"`
		IsEnabled   bool            `json:"is_enabled"`
		IsDefault   bool            `json:"is_default"`
		Priority    int             `json:"priority"`
		Config      json.RawMessage `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	config := req.Config
	if len(config) == 0 {
		config = json.RawMessage("{}")
	}

	model := &ai.AIModel{
		Name: req.Name, Provider: req.Provider, ModelType: req.ModelType,
		Endpoint: req.Endpoint, APIKey: req.APIKey, ModelName: req.ModelName,
		APIVersion: req.APIVersion, MaxTokens: req.MaxTokens, Temperature: req.Temperature,
		TopP: req.TopP, Dimensions: req.Dimensions, IsEnabled: req.IsEnabled,
		IsDefault: req.IsDefault, Priority: req.Priority, Config: config,
	}

	if err := h.aiRepo.CreateModel(c, model); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, fmt.Sprintf("创建模型失败: %v", err))
		return
	}

	h.modelManager.LoadModels(c)
	h.audit(c, "ai_model.create", "model", uuid.NullUUID{UUID: model.ID, Valid: true}, gin.H{"name": model.Name, "provider": model.Provider})
	response.Success(c, model)
}

func (h *Handler) UpdateModel(c *gin.Context) {
	modelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的模型 ID")
		return
	}

	existing, err := h.aiRepo.GetModel(c, modelID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "模型不存在")
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Provider    string  `json:"provider"`
		ModelType   string  `json:"type"`
		Endpoint    string  `json:"endpoint"`
		APIKey      string  `json:"api_key"`
		ModelName   string  `json:"model_name"`
		APIVersion  string  `json:"api_version"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
		TopP        float64 `json:"top_p"`
		Dimensions  int     `json:"dimensions"`
		IsEnabled   bool    `json:"is_enabled"`
		IsDefault   bool    `json:"is_default"`
		Priority    int     `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Provider != "" {
		existing.Provider = req.Provider
	}
	if req.ModelType != "" {
		existing.ModelType = req.ModelType
	}
	if req.Endpoint != "" {
		existing.Endpoint = req.Endpoint
	}
	// Do not clear API key if the UI leaves it blank
	if req.APIKey != "" {
		existing.APIKey = req.APIKey
	}
	if req.ModelName != "" {
		existing.ModelName = req.ModelName
	}
	if req.APIVersion != "" {
		existing.APIVersion = req.APIVersion
	}
	if req.MaxTokens > 0 {
		existing.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		existing.Temperature = req.Temperature
	}
	existing.TopP = req.TopP
	if req.Dimensions > 0 {
		existing.Dimensions = req.Dimensions
	}
	existing.IsEnabled = req.IsEnabled
	existing.IsDefault = req.IsDefault
	if req.Priority > 0 {
		existing.Priority = req.Priority
	}

	if err := h.aiRepo.UpdateModel(c, modelID, existing); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "更新模型失败")
		return
	}

	h.modelManager.LoadModels(c)
	h.audit(c, "ai_model.update", "model", uuid.NullUUID{UUID: modelID, Valid: true}, nil)
	response.Success(c, existing)
}

func (h *Handler) DeleteModel(c *gin.Context) {
	modelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的模型 ID")
		return
	}

	if err := h.aiRepo.DeleteModel(c, modelID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "删除模型失败")
		return
	}

	h.modelManager.LoadModels(c)
	h.audit(c, "ai_model.delete", "model", uuid.NullUUID{UUID: modelID, Valid: true}, nil)
	response.Success(c, nil)
}

// Admin user CRUD handlers
func (h *Handler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	u, err := h.userService.GetByID(c, userID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "用户不存在")
		return
	}

	role, _ := h.userService.GetRole(c, userID)
	result := map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"avatar_url": u.AvatarURL,
		"status":     u.Status,
		"role":       role,
		"created_at": u.CreatedAt,
	}
	response.Success(c, result)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	u, err := h.userService.CreateUser(c, req.Email, req.Username, req.Password, req.Role)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建用户失败")
		return
	}

	h.audit(c, "user.create", "user", uuid.NullUUID{UUID: u.ID, Valid: true}, nil)
	response.Success(c, u)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Status   *int16 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if req.Username != "" {
		if err := h.userService.UpdateProfile(c, userID, req.Username, ""); err != nil {
			response.Error(c, 500, response.ErrInternalServerError, "更新用户失败")
			return
		}
	}
	if req.Status != nil {
		if err := h.userService.UpdateUserStatus(c, userID, *req.Status); err != nil {
			response.Error(c, 500, response.ErrInternalServerError, "更新状态失败")
			return
		}
	}

	h.audit(c, "user.update", "user", uuid.NullUUID{UUID: userID, Valid: true}, req)
	response.Success(c, nil)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	if err := h.userService.DeleteUser(c, userID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "删除用户失败")
		return
	}

	h.audit(c, "user.delete", "user", uuid.NullUUID{UUID: userID, Valid: true}, nil)
	response.Success(c, nil)
}

func (h *Handler) BatchDisableUsers(c *gin.Context) {
	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	var ids []uuid.UUID
	for _, idStr := range req.UserIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, 400, response.ErrBadRequest, fmt.Sprintf("无效的用户 ID: %s", idStr))
			return
		}
		ids = append(ids, id)
	}

	if err := h.userService.BatchUpdateStatus(c, ids, 0); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "批量禁用失败")
		return
	}

	h.audit(c, "user.batch_disable", "user", uuid.NullUUID{}, gin.H{"count": len(ids)})
	response.Success(c, gin.H{"disabled_count": len(ids)})
}

func (h *Handler) ExportUsers(c *gin.Context) {
	users, _, err := h.userService.ListUsers(c, "", "", 1, 10000)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取用户列表失败")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/csv")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=users.csv")
	c.Writer.WriteHeader(http.StatusOK)

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"id", "email", "username", "role", "status", "created_at"})
	for _, u := range users {
		w.Write([]string{
			u.ID.String(), u.Email, u.Username, u.Role,
			strconv.Itoa(int(u.Status)), u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()
}

// Stats handlers
type StatsOverview struct {
	TotalUsers    int64   `json:"total_users"`
	ActiveUsers   int64   `json:"active_users"`
	TotalNotes    int64   `json:"total_notes"`
	TotalShares   int64   `json:"total_shares"`
	PendingShares int64   `json:"pending_shares"`
	StorageUsed   float64 `json:"storage_used_gb"`
}

func (h *Handler) GetStatsOverview(c *gin.Context) {
	totalUsers, _ := h.userService.CountUsers(c)
	totalNotes, _ := h.noteRepo.Count(c.Request.Context())
	totalStorage, _ := h.noteRepo.TotalStorageUsed(c.Request.Context())

	stats := StatsOverview{
		TotalUsers:  totalUsers,
		ActiveUsers: totalUsers, // TODO: count active sessions
		TotalNotes:  totalNotes,
		StorageUsed: float64(totalStorage) / 1073741824,
	}

	response.Success(c, stats)
}

func (h *Handler) GetUserStats(c *gin.Context) {
	// Return simplified user stats - items with date, new users count, active users
	notesCreated, _ := h.noteRepo.CountToday(c.Request.Context(), "created")
	notesUpdated, _ := h.noteRepo.CountToday(c.Request.Context(), "updated")
	totalUsers, _ := h.userService.CountUsers(c)

	response.Success(c, gin.H{
		"items": []gin.H{
			{"date": time.Now().Format("2006-01-02"), "new_users": 0, "active_users": totalUsers, "notes_created": notesCreated, "notes_updated": notesUpdated},
		},
	})
}

func (h *Handler) GetUsageStats(c *gin.Context) {
	totalStorage, _ := h.noteRepo.TotalStorageUsed(c.Request.Context())
	notesCreated, _ := h.noteRepo.CountToday(c.Request.Context(), "created")
	notesUpdated, _ := h.noteRepo.CountToday(c.Request.Context(), "updated")

	response.Success(c, gin.H{
		"notes_created_today":   notesCreated,
		"notes_updated_today":   notesUpdated,
		"files_uploaded_today":  middleware.GetTodayCount(c, h.redis, "file"),
		"storage_used_gb":       float64(totalStorage) / 1073741824,
		"ai_requests_today":     middleware.GetTodayCount(c, h.redis, "ai"),
		"search_requests_today": middleware.GetTodayCount(c, h.redis, "search"),
		"api_requests_today":    middleware.GetTodayCount(c, h.redis, "api"),
	})
}

// Test model connection
func (h *Handler) TestModel(c *gin.Context) {
	modelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的模型 ID")
		return
	}

	model, err := h.aiRepo.GetModel(c, modelID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "模型不存在")
		return
	}

	// Test connection based on model type
	result := gin.H{
		"model":   model.ModelName,
		"status":  "ok",
		"message": "Connection successful",
	}

	// For LLM models, try a minimal API call
	if model.ModelType == "llm" {
		err = h.testLLMConnection(model)
		if err != nil {
			result["status"] = "error"
			result["message"] = err.Error()
		}
	} else if model.ModelType == "embedding" {
		err = h.testEmbeddingConnection(model)
		if err != nil {
			result["status"] = "error"
			result["message"] = err.Error()
		}
	}

	response.Success(c, result)
}

func (h *Handler) testLLMConnection(model *ai.AIModel) error {
	// Simple HTTP connectivity test to the endpoint
	resp, err := http.Get(model.Endpoint)
	if err != nil {
		return fmt.Errorf("cannot reach endpoint: %v", err)
	}
	resp.Body.Close()
	return nil
}

func (h *Handler) testEmbeddingConnection(model *ai.AIModel) error {
	// Simple HTTP connectivity test
	resp, err := http.Get(model.Endpoint)
	if err != nil {
		return fmt.Errorf("cannot reach endpoint: %v", err)
	}
	resp.Body.Close()
	return nil
}

func (h *Handler) GetAIConfig(c *gin.Context) {
	cfg, err := h.aiRepo.GetAIConfig(c)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取 AI 配置失败")
		return
	}
	response.Success(c, cfg)
}

func (h *Handler) UpdateAIConfig(c *gin.Context) {
	var req struct {
		ChunkSize        int     `json:"chunk_size"`
		ChunkOverlap     int     `json:"chunk_overlap"`
		TopK             int     `json:"top_k"`
		ScoreThreshold   float64 `json:"score_threshold"`
		EnableRAG        bool    `json:"enable_rag"`
		MaxContextTokens int     `json:"max_context_tokens"`
		SystemPrompt     string  `json:"system_prompt"`
		EnableStreaming  bool    `json:"enable_streaming"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	configs := map[string]interface{}{
		"chunk_size":         req.ChunkSize,
		"chunk_overlap":      req.ChunkOverlap,
		"top_k":              req.TopK,
		"score_threshold":    req.ScoreThreshold,
		"enable_rag":         req.EnableRAG,
		"max_context_tokens": req.MaxContextTokens,
		"system_prompt":      req.SystemPrompt,
		"enable_streaming":   req.EnableStreaming,
	}

	for key, value := range configs {
		valJSON, _ := json.Marshal(map[string]interface{}{"value": value})
		h.aiRepo.UpdateConfigValue(c, key, valJSON)
	}

	h.audit(c, "config.update", "ai_config", uuid.NullUUID{}, configs)
	response.Success(c, nil)
}

// --- System Config ---

func (h *Handler) GetSystemConfig(c *gin.Context) {
	cfg, err := h.sysConfigRepo.GetAsConfigData(c.Request.Context())
	if err != nil {
		h.logger.Error("GetSystemConfig failed", zap.Error(err))
		response.Error(c, 500, response.ErrInternalServerError, "获取系统配置失败: "+err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *Handler) UpdateSystemConfig(c *gin.Context) {
	var req struct {
		SiteName               string `json:"site_name"`
		AllowSignup            *bool  `json:"allow_signup"`
		RequireEmailVerification *bool `json:"require_email_verification"`
		DefaultRole            string `json:"default_role"`
		MaxNotesPerUser        *int   `json:"max_notes_per_user"`
		MaxStoragePerUserBytes *int64 `json:"max_storage_per_user_bytes"`
		MaxFileSizeBytes       *int64 `json:"max_file_size_bytes"`
		SessionDurationDays    *int   `json:"session_duration_days"`
		MaxDevicesPerUser      *int   `json:"max_devices_per_user"`
		EnableSharing          *bool  `json:"enable_sharing"`
		EnableAI               *bool  `json:"enable_ai"`
		MaintenanceMode        *bool  `json:"maintenance_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	values := make(map[string]interface{})
	if req.SiteName != "" {
		values["site_name"] = req.SiteName
	}
	if req.AllowSignup != nil {
		values["allow_signup"] = *req.AllowSignup
	}
	if req.RequireEmailVerification != nil {
		values["require_email_verification"] = *req.RequireEmailVerification
	}
	if req.DefaultRole != "" {
		values["default_role"] = req.DefaultRole
	}
	if req.MaxNotesPerUser != nil {
		values["max_notes_per_user"] = *req.MaxNotesPerUser
	}
	if req.MaxStoragePerUserBytes != nil {
		values["max_storage_per_user_bytes"] = *req.MaxStoragePerUserBytes
	}
	if req.MaxFileSizeBytes != nil {
		values["max_file_size_bytes"] = *req.MaxFileSizeBytes
	}
	if req.SessionDurationDays != nil {
		values["session_duration_days"] = *req.SessionDurationDays
	}
	if req.MaxDevicesPerUser != nil {
		values["max_devices_per_user"] = *req.MaxDevicesPerUser
	}
	if req.EnableSharing != nil {
		values["enable_sharing"] = *req.EnableSharing
	}
	if req.EnableAI != nil {
		values["enable_ai"] = *req.EnableAI
	}
	if req.MaintenanceMode != nil {
		values["maintenance_mode"] = *req.MaintenanceMode
	}

	if err := h.sysConfigRepo.UpdateBatch(c.Request.Context(), values); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "更新系统配置失败")
		return
	}
	h.audit(c, "config.update", "system_config", uuid.NullUUID{}, values)
	response.Success(c, nil)
}

// --- OAuth Config ---

func (h *Handler) GetOAuthConfig(c *gin.Context) {
	cfgs, err := h.oauthRepo.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取 OAuth 配置失败")
		return
	}
	for _, p := range cfgs {
		if p.ClientSecret != "" {
			p.ClientSecret = "***"
		}
	}
	response.Success(c, cfgs)
}

func (h *Handler) UpdateOAuthConfig(c *gin.Context) {
	var req struct {
		GitHub *OAuthUpdateReq `json:"github"`
		Google *OAuthUpdateReq `json:"google"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if req.GitHub != nil {
		if err := h.oauthRepo.Upsert(c.Request.Context(), "github", req.GitHub.Enabled, req.GitHub.ClientID, req.GitHub.ClientSecret, req.GitHub.RedirectURI); err != nil {
			response.Error(c, 500, response.ErrInternalServerError, "更新 GitHub OAuth 配置失败")
			return
		}
	}
	if req.Google != nil {
		if err := h.oauthRepo.Upsert(c.Request.Context(), "google", req.Google.Enabled, req.Google.ClientID, req.Google.ClientSecret, req.Google.RedirectURI); err != nil {
			response.Error(c, 500, response.ErrInternalServerError, "更新 Google OAuth 配置失败")
			return
		}
	}
	h.audit(c, "config.update", "oauth_config", uuid.NullUUID{}, gin.H{"providers": []string{"github", "google"}})
	response.Success(c, nil)
}

type OAuthUpdateReq struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

// --- Group Management ---

func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.groupRepo.List(c.Request.Context())
	if err != nil {
		h.logger.Error("ListGroups failed", zap.Error(err))
		response.Error(c, 500, response.ErrInternalServerError, "获取用户组列表失败: "+err.Error())
		return
	}
	response.Success(c, groups)
}

func (h *Handler) CreateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		IsDefault   bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}
	g, err := h.groupRepo.Create(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建用户组失败")
		return
	}
	h.audit(c, "group.create", "group", uuid.NullUUID{UUID: g.ID, Valid: true}, gin.H{"name": req.Name})
	response.Success(c, g)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的组 ID")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}
	if err := h.groupRepo.Update(c.Request.Context(), id, req.Name, req.Description); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "更新用户组失败")
		return
	}
	h.audit(c, "group.update", "group", uuid.NullUUID{UUID: id, Valid: true}, req)
	response.Success(c, nil)
}

func (h *Handler) DeleteGroup(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的组 ID")
		return
	}
	if err := h.groupRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "删除用户组失败")
		return
	}
	h.audit(c, "group.delete", "group", uuid.NullUUID{UUID: id, Valid: true}, nil)
	response.Success(c, nil)
}

func (h *Handler) GetGroupFeatures(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的组 ID")
		return
	}
	features, err := h.groupRepo.GetFeatures(c.Request.Context(), id)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取功能配置失败")
		return
	}
	response.Success(c, features)
}

func (h *Handler) UpdateGroupFeatures(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的组 ID")
		return
	}
	var req struct {
		Features map[string]json.RawMessage `json:"features" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}
	if err := h.groupRepo.UpdateFeatures(c.Request.Context(), id, req.Features); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "更新功能配置失败")
		return
	}
	h.audit(c, "group.update_features", "group", uuid.NullUUID{UUID: id, Valid: true}, nil)
	response.Success(c, nil)
}

// --- Audit Logs ---

func (h *Handler) GetAuditLogs(c *gin.Context) {
	action := c.Query("action")
	userID := c.Query("user_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	logs, total, err := h.auditRepo.List(c.Request.Context(), action, userID, startDate, endDate, page, size)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取操作日志失败")
		return
	}
	response.SuccessPaginated(c, logs, total, page, size)
}
