package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/internal/user"
	"github.com/krasis/krasis/pkg/response"
)

type Handler struct {
	oauthManager   *OAuthManager
	jwtManager     *JWTManager
	sessionManager *SessionManager
	userService    *user.UserService
}

func NewHandler(om *OAuthManager, jm *JWTManager, sm *SessionManager, us *user.UserService) *Handler {
	return &Handler{
		oauthManager:   om,
		jwtManager:     jm,
		sessionManager: sm,
		userService:    us,
	}
}

// GitHubLogin redirects to GitHub OAuth.
func (h *Handler) GitHubLogin(c *gin.Context) {
	state := uuid.New().String()

	url, err := h.oauthManager.GetAuthURL(ProviderGitHub, state)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取 OAuth 地址失败")
		return
	}

	c.Redirect(http.StatusFound, url)
}

// GitHubCallback handles GitHub OAuth callback.
func (h *Handler) GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Error(c, 400, response.ErrBadRequest, "缺少授权码")
		return
	}

	tokenResp, err := h.oauthManager.ExchangeCode(c, ProviderGitHub, code)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "交换 Token 失败")
		return
	}

	userInfo, err := h.oauthManager.GetUserInfo(c, ProviderGitHub, tokenResp.AccessToken)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取用户信息失败")
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

// GoogleLogin redirects to Google OAuth.
func (h *Handler) GoogleLogin(c *gin.Context) {
	state := uuid.New().String()

	url, err := h.oauthManager.GetAuthURL(ProviderGoogle, state)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取 OAuth 地址失败")
		return
	}

	c.Redirect(http.StatusFound, url)
}

// GoogleCallback handles Google OAuth callback.
func (h *Handler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Error(c, 400, response.ErrBadRequest, "缺少授权码")
		return
	}

	tokenResp, err := h.oauthManager.ExchangeCode(c, ProviderGoogle, code)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "交换 Token 失败")
		return
	}

	userInfo, err := h.oauthManager.GetUserInfo(c, ProviderGoogle, tokenResp.AccessToken)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取用户信息失败")
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

func (h *Handler) handleOAuthLogin(c *gin.Context, userInfo *UserInfo) {
	userModel, err := h.userService.GetOrCreateByOAuth(
		c, userInfo.Email, userInfo.Username, userInfo.AvatarURL,
	)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, fmt.Sprintf("创建用户失败: %v", err))
		return
	}

	role, err := h.userService.GetRole(c, userModel.ID)
	if err != nil {
		role = "member"
	}

	info := &SessionInfo{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
	session, err := h.sessionManager.Create(c, userModel.ID.String(), info)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建 Session 失败")
		return
	}

	token, err := h.jwtManager.Generate(userModel.ID.String(), role, session.SessionID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "生成 Token 失败")
		return
	}

	response.Success(c, gin.H{
		// keep OAuth response for existing clients, but also return "token" for web app
		"access_token": token,
		"token":        token,
		"token_type":   "Bearer",
		"expires_in":   604800,
		"user": gin.H{
			"id":         userModel.ID,
			"email":      userModel.Email,
			"username":   userModel.Username,
			"name":       userModel.Username,
			"avatar_url": userModel.AvatarURL,
			"role":       role,
		},
	})
}

// Register handles local registration with username, email and password.
func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	userModel, err := h.userService.RegisterLocal(c, req.Email, req.Username, req.Password)
	if err != nil {
		// best-effort mapping
		if err.Error() == "email already exists" {
			response.Error(c, 409, response.ErrConflict, "邮箱已注册")
			return
		}
		if err.Error() == "password too short" {
			response.Error(c, 422, response.ErrUnprocessableEntity, "密码至少 6 位")
			return
		}
		response.Error(c, 400, response.ErrBadRequest, err.Error())
		return
	}

	role, err := h.userService.GetRole(c, userModel.ID)
	if err != nil {
		role = "member"
	}

	info := &SessionInfo{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
	session, err := h.sessionManager.Create(c, userModel.ID.String(), info)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建 Session 失败")
		return
	}

	token, err := h.jwtManager.Generate(userModel.ID.String(), role, session.SessionID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "生成 Token 失败")
		return
	}

	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         userModel.ID,
			"email":      userModel.Email,
			"username":   userModel.Username,
			"name":       userModel.Username,
			"avatar_url": userModel.AvatarURL,
			"role":       role,
			"created_at": userModel.CreatedAt,
		},
	})
}

// Login handles local username/password login.
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	userModel, err := h.userService.AuthenticateByUsername(c, req.Username, req.Password)
	if err != nil {
		response.Error(c, 401, response.ErrUnauthorized, "账号或密码错误")
		return
	}

	role, err := h.userService.GetRole(c, userModel.ID)
	if err != nil {
		role = "member"
	}

	info := &SessionInfo{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
	session, err := h.sessionManager.Create(c, userModel.ID.String(), info)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建 Session 失败")
		return
	}

	token, err := h.jwtManager.Generate(userModel.ID.String(), role, session.SessionID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "生成 Token 失败")
		return
	}

	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         userModel.ID,
			"email":      userModel.Email,
			"username":   userModel.Username,
			"name":       userModel.Username,
			"avatar_url": userModel.AvatarURL,
			"role":       role,
			"created_at": userModel.CreatedAt,
		},
	})
}

func (h *Handler) Logout(c *gin.Context) {
	sessionID := c.GetString("session_id")
	if sessionID == "" {
		response.Success(c, nil)
		return
	}

	if err := h.sessionManager.Delete(context.Background(), sessionID); err != nil {
		// Session may already be deleted; continue anyway
	}

	response.Success(c, nil)
}
