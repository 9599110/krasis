package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/krasis/krasis/pkg/response"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered", zap.Any("error", err))
				response.Error(c, http.StatusInternalServerError, response.ErrInternalServerError, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}
