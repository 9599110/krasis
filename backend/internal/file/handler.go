package file

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/pkg/response"
)

// FileServiceInterface defines the interface for file service operations
type FileServiceInterface interface {
	GeneratePresignURL(ctx context.Context, userID uuid.UUID, fileName, fileType string, noteID *uuid.UUID, folderID *uuid.UUID) (*PresignResult, error)
	ConfirmUpload(ctx context.Context, fileID uuid.UUID) error
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
	ListByNote(ctx context.Context, noteID uuid.UUID) ([]*ListFileResult, error)
	ListByFolder(ctx context.Context, folderID uuid.UUID) ([]*ListFileResult, error)
	ListHiddenByFolder(ctx context.Context, folderID uuid.UUID) ([]*ListFileResult, error)
	UnhideFile(ctx context.Context, fileID uuid.UUID) error
	MarkHidden(ctx context.Context, fileID uuid.UUID) error
	GenerateGetURL(ctx context.Context, fileID uuid.UUID) (string, error)
	GenerateDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error)
	ListImages(ctx context.Context, userID uuid.UUID, page, size int) ([]*ListFileResult, int64, error)
}

type Handler struct {
	service FileServiceInterface
}

func NewHandler(service FileServiceInterface) *Handler {
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

	var folderID *uuid.UUID
	if fid := c.Query("folder_id"); fid != "" {
		id, err := uuid.Parse(fid)
		if err == nil {
			folderID = &id
		}
	}

	result, err := h.service.GeneratePresignURL(c, userID, fileName, fileType, noteID, folderID)
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

func (h *Handler) ListByFolder(c *gin.Context) {
	folderID, err := uuid.Parse(c.Param("folder_id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件夹 ID")
		return
	}

	files, err := h.service.ListByFolder(c, folderID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取文件列表失败")
		return
	}

	response.Success(c, files)
}

func (h *Handler) GetFileURL(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件 ID")
		return
	}

	url, err := h.service.GenerateGetURL(c, fileID)
	if err != nil {
		if err == ErrFileNotFound {
			response.Error(c, 404, response.ErrNotFound, "文件不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "获取文件 URL 失败")
		return
	}

	response.Success(c, gin.H{"url": url})
}

// ListImages returns paginated image files for the current user across all folders.
func (h *Handler) ListImages(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, 401, response.ErrUnauthorized, "未认证")
		return
	}

	page := 1
	size := 50
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if s := c.Query("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 200 {
			size = v
		}
	}

	files, total, err := h.service.ListImages(c, userID, page, size)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取图片列表失败")
		return
	}

	response.Success(c, gin.H{
		"items": files,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// ListHiddenByFolder returns files that are marked as hidden in a folder.
// Hidden files are NOT synced to the frontend by default; only visible via this explicit endpoint.
func (h *Handler) ListHiddenByFolder(c *gin.Context) {
	folderID, err := uuid.Parse(c.Param("folder_id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件夹 ID")
		return
	}

	files, err := h.service.ListHiddenByFolder(c, folderID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取隐藏文件列表失败")
		return
	}

	response.Success(c, files)
}

// UnhideFile makes a hidden file visible again (sets is_hidden = false).
// Only after unhiding will the file appear in normal ListByFolder/ListByNote results.
func (h *Handler) UnhideFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件 ID")
		return
	}

	if err := h.service.UnhideFile(c, fileID); err != nil {
		if err == ErrFileNotFound {
			response.Error(c, 404, response.ErrNotFound, "文件不存在或不是隐藏状态")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "取消隐藏失败")
		return
	}

	response.Success(c, gin.H{"status": "unhidden"})
}

// MarkHidden marks a file as hidden (sets is_hidden = true).
// The file will no longer appear in normal ListByFolder/ListByNote results.
func (h *Handler) MarkHidden(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件 ID")
		return
	}

	if err := h.service.MarkHidden(c, fileID); err != nil {
		if err == ErrFileNotFound {
			response.Error(c, 404, response.ErrNotFound, "文件不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "隐藏文件失败")
		return
	}

	response.Success(c, gin.H{"status": "hidden"})
}

func (h *Handler) DownloadFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的文件 ID")
		return
	}

	url, err := h.service.GenerateDownloadURL(c, fileID)
	if err != nil {
		if err == ErrFileNotFound {
			response.Error(c, 404, response.ErrNotFound, "文件不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "获取下载链接失败")
		return
	}

	response.Success(c, gin.H{"url": url})
}
