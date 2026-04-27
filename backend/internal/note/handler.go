package note

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/pkg/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *NoteService
	logger  *zap.Logger
}

func NewHandler(service *NoteService, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) CreateNote(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Title    string  `json:"title"`
		Content  string  `json:"content"`
		FolderID *string `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	var folderID *uuid.UUID
	if req.FolderID != nil {
		id, err := uuid.Parse(*req.FolderID)
		if err != nil {
			response.Error(c, 400, response.ErrBadRequest, "无效的文件夹 ID")
			return
		}
		folderID = &id
	}

	note, err := h.service.Create(c, ownerID, req.Title, req.Content, folderID)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "创建笔记失败")
		return
	}

	response.Success(c, gin.H{
		"id":         note.ID,
		"title":      note.Title,
		"version":    note.Version,
		"created_at": note.CreatedAt,
	})
}

func (h *Handler) GetNote(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	note, err := h.service.GetByID(c, ownerID, noteID)
	if err == ErrPermissionDenied || err == ErrNoteNotFound || err == ErrNoteDeleted {
		response.Error(c, 404, response.ErrNotFound, "笔记不存在")
		return
	}
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取笔记失败")
		return
	}

	data := gin.H{
		"id":         note.ID,
		"title":      note.Title,
		"content":    note.Content,
		"owner_id":   note.OwnerID,
		"folder_id":  note.FolderID,
		"version":    note.Version,
		"is_public":  note.IsPublic,
		"created_at": note.CreatedAt,
		"updated_at": note.UpdatedAt,
	}

	if note.ContentHTML.Valid {
		data["content_html"] = note.ContentHTML.String
	}

	response.Success(c, data)
}

func (h *Handler) ListNotes(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	sort := c.DefaultQuery("sort", "updated_at")
	order := c.DefaultQuery("order", "desc")

	var folderID *uuid.UUID
	if fid := c.Query("folder_id"); fid != "" {
		id, err := uuid.Parse(fid)
		if err == nil {
			folderID = &id
		}
	}

	notes, total, err := h.service.List(c, ownerID, folderID, page, size, sort, order)
	if err != nil {
		h.logger.Error("ListNotes failed", zap.Error(err), zap.String("owner_id", ownerID.String()))
		response.Error(c, 500, response.ErrInternalServerError, "获取笔记列表失败")
		return
	}

	items := make([]gin.H, 0, len(notes))
	for _, n := range notes {
		item := gin.H{
			"id":             n.ID,
			"title":          n.Title,
			"content_preview": truncate(n.Content, 200),
			"owner_id":       n.OwnerID,
			"folder_id":      n.FolderID,
			"version":        n.Version,
			"is_public":      n.IsPublic,
			"created_at":     n.CreatedAt,
			"updated_at":     n.UpdatedAt,
		}
		items = append(items, item)
	}

	response.SuccessPaginated(c, items, total, page, size)
}

func (h *Handler) UpdateNote(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	versionStr := c.GetHeader("If-Match")
	var version int
	if versionStr != "" {
		version, _ = strconv.Atoi(versionStr)
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Version int    `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	// Use If-Match header version if provided, otherwise body version
	// If-Match header takes precedence (HTTP standard)
	if versionStr != "" && version > 0 {
		// If-Match header is authoritative
	} else {
		// Fall back to body version
		version = req.Version
	}

	note, err := h.service.Update(c, ownerID, noteID, req.Title, req.Content, version)
	if err != nil {
		if _, ok := err.(*ConflictError); ok {
			response.Error(c, 409, response.ErrConflict, "版本冲突")
			return
		}
		if err == ErrPermissionDenied || err == ErrNoteNotFound || err == ErrNoteDeleted {
			response.Error(c, 404, response.ErrNotFound, "笔记不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "更新笔记失败")
		return
	}

	response.Success(c, gin.H{
		"id":         note.ID,
		"title":      note.Title,
		"version":    note.Version,
		"updated_at": note.UpdatedAt,
	})
}

func (h *Handler) DeleteNote(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	permanent := c.DefaultQuery("permanent", "false") == "true"

	if err := h.service.Delete(c, ownerID, noteID, permanent); err != nil {
		if err == ErrPermissionDenied || err == ErrNoteNotFound {
			response.Error(c, 404, response.ErrNotFound, "笔记不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "删除笔记失败")
		return
	}

	response.Success(c, nil)
}

func (h *Handler) GetVersions(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	versions, err := h.service.GetVersions(c, ownerID, noteID)
	if err != nil {
		if err == ErrPermissionDenied || err == ErrNoteNotFound {
			response.Error(c, 404, response.ErrNotFound, "笔记不存在")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "获取版本失败")
		return
	}

	items := make([]gin.H, 0, len(versions))
	for _, v := range versions {
		item := gin.H{
			"id":         v.ID,
			"version":    v.Version,
			"created_at": v.CreatedAt,
		}
		if v.Title.Valid {
			item["title"] = v.Title.String
		}
		if v.ChangeSummary.Valid {
			item["change_summary"] = v.ChangeSummary.String
		}
		items = append(items, item)
	}

	response.Success(c, gin.H{"items": items})
}

func (h *Handler) RestoreVersion(c *gin.Context) {
	ownerID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	version, err := strconv.Atoi(c.Param("version"))
	if err != nil || version < 1 {
		response.Error(c, 400, response.ErrBadRequest, "无效的版本号")
		return
	}

	if err := h.service.RestoreVersion(c, ownerID, noteID, version); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "恢复版本失败")
		return
	}

	response.Success(c, nil)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
