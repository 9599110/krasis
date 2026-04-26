package folder

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListFolders(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))

	folders, err := h.service.List(c, ownerID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取文件夹列表失败")
		return
	}

	response.Success(c, gin.H{"items": folders})
}

func (h *Handler) CreateFolder(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Name     string  `json:"name" binding:"required"`
		ParentID *string `json:"parent_id"`
		Color    string  `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			response.Error(c, 400, response.ErrBadRequest, "无效的父文件夹 ID")
			return
		}
		parentID = &id
	}

	folder, err := h.service.Create(c, ownerID, req.Name, parentID, req.Color)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建文件夹失败")
		return
	}

	response.Success(c, folder)
}

func (h *Handler) UpdateFolder(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件夹 ID")
		return
	}

	var req struct {
		Name      string  `json:"name"`
		ParentID  *string `json:"parent_id"`
		Color     string  `json:"color"`
		SortOrder int     `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		uid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			response.Error(c, 400, response.ErrBadRequest, "无效的父文件夹 ID")
			return
		}
		parentID = &uid
	}

	if err := h.service.Update(c, ownerID, id, req.Name, parentID, req.Color, req.SortOrder); err != nil {
		if err == ErrFolderNotFound || err == ErrPermissionDenied {
			response.Error(c, 404, response.ErrNotFound, "文件夹不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "更新文件夹失败")
		return
	}

	response.Success(c, nil)
}

func (h *Handler) DeleteFolder(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件夹 ID")
		return
	}

	if err := h.service.Delete(c, ownerID, id); err != nil {
		if err == ErrFolderNotFound {
			response.Error(c, 404, response.ErrNotFound, "文件夹不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "删除文件夹失败")
		return
	}

	response.Success(c, nil)
}
