package user

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/pkg/response"
)

type SessionManager interface {
	GetUserSessionsMap(ctx context.Context, userID, currentSessionID string) ([]map[string]interface{}, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteAllForUser(ctx context.Context, userID string) error
}

type Handler struct {
	service        *UserService
	sessionManager SessionManager
}

func NewHandler(service *UserService, sessionManager SessionManager) *Handler {
	return &Handler{
		service:        service,
		sessionManager: sessionManager,
	}
}

func (h *Handler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, 401, response.ErrUnauthorized, "未认证")
		return
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	user, err := h.service.GetByID(c, id)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取用户信息失败")
		return
	}

	response.Success(c, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"avatar_url": user.AvatarURL,
		"role":       c.GetString("role"),
		"created_at": user.CreatedAt,
	})
}

func (h *Handler) GetSessions(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.GetString("session_id")

	sessions, err := h.sessionManager.GetUserSessionsMap(c, userID, sessionID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取设备列表失败")
		return
	}

	response.Success(c, gin.H{
		"sessions": sessions,
	})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	if err := h.sessionManager.DeleteSession(c, sessionID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "下线设备失败")
		return
	}

	response.Success(c, nil)
}

func (h *Handler) DeleteAllSessions(c *gin.Context) {
	userID := c.GetString("user_id")

	if err := h.sessionManager.DeleteAllForUser(c, userID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "下线所有设备失败")
		return
	}

	response.Success(c, nil)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	id, err := uuid.Parse(userID)
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	var req struct {
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if err := h.service.UpdateProfile(c, id, req.Username, req.AvatarURL); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "更新资料失败")
		return
	}

	response.Success(c, nil)
}
