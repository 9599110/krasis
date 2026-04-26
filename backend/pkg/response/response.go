package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PaginatedData struct {
	Items   interface{} `json:"items"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
	HasMore bool        `json:"has_more"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func SuccessPaginated(c *gin.Context, items interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PaginatedData{
			Items:   items,
			Total:   total,
			Page:    page,
			Size:    size,
			HasMore: int64(page*size) < total,
		},
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	resp := Response{
		Code:    code,
		Message: message,
	}
	c.JSON(httpStatus, resp)
}

const (
	ErrBadRequest          = 1001
	ErrUnauthorized        = 1002
	ErrForbidden           = 1003
	ErrNotFound            = 1004
	ErrConflict            = 1005
	ErrUnprocessableEntity = 1006
	ErrTooManyRequests     = 2001
	ErrInternalServerError = 3001
	ErrServiceUnavailable  = 3002
)
