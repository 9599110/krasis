package search

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/krasis/krasis/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Search(c *gin.Context) {
	q := c.Query("q")
	searchType := c.DefaultQuery("type", "notes")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	items, total, err := h.service.Search(c, q, searchType, page, size)
	if err != nil {
		if err == ErrEmptyQuery {
			response.Error(c, 400, response.ErrBadRequest, "搜索关键词不能为空")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "搜索失败")
		return
	}

	response.SuccessPaginated(c, items, total, page, size)
}
