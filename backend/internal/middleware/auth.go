package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/krasis/krasis/internal/auth"
	"github.com/krasis/krasis/pkg/response"
)

func AuthMiddleware(jm *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, 401, response.ErrUnauthorized, "未认证")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, 401, response.ErrUnauthorized, "无效的认证格式")
			c.Abort()
			return
		}

		claims, err := jm.Validate(parts[1])
		if err != nil {
			response.Error(c, 401, response.ErrUnauthorized, "无效的 token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			response.Error(c, 403, response.ErrForbidden, "权限不足")
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}

		response.Error(c, 403, response.ErrForbidden, "权限不足")
		c.Abort()
	}
}
