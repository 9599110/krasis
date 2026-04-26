package file

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

func (h *Handler) GetPresignURL(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	fileName := c.Query("file_name")
	fileType := c.Query("file_type")
	if fileName == "" || fileType == "" {
		response.Error(c, 400, response.ErrBadRequest, "file_name 和 file_type 为必填")
		return
	}

	var noteID *uuid.UUID
	if nid := c.Query("note_id"); nid != "" {
		id, err := uuid.Parse(nid)
		if err == nil {
			noteID = &id
		}
	}

	result, err := h.service.GeneratePresignURL(c, userID, fileName, fileType, noteID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "生成上传 URL 失败")
		return
	}

	response.Success(c, result)
}

func (h *Handler) ConfirmUpload(c *gin.Context) {
	var req struct {
		FileID   string                 `json:"file_id" binding:"required"`
		NoteID   string                 `json:"note_id"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的 file_id")
		return
	}

	if err := h.service.ConfirmUpload(c, fileID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "确认上传失败")
		return
	}

	response.Success(c, nil)
}

func (h *Handler) DeleteFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件 ID")
		return
	}

	if err := h.service.DeleteFile(c, fileID); err != nil {
		if err == ErrFileNotFound {
			response.Error(c, 404, response.ErrNotFound, "文件不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "删除文件失败")
		return
	}

	response.Success(c, nil)
}
