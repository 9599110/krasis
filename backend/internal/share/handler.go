package share

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/krasis/krasis/internal/auditlog"
	"github.com/krasis/krasis/pkg/response"
)

type Handler struct {
	service   *Service
	auditRepo *auditlog.Repository
}

func NewHandler(service *Service, auditRepo *auditlog.Repository) *Handler {
	return &Handler{service: service, auditRepo: auditRepo}
}

func (h *Handler) audit(c *gin.Context, action string, targetType string, targetID uuid.NullUUID, changes interface{}) {
	adminIDStr, _ := c.Get("user_id")
	if adminIDStr == nil || h.auditRepo == nil {
		return
	}
	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		return
	}
	var changesJSON json.RawMessage
	if changes != nil {
		changesJSON, _ = json.Marshal(changes)
	}
	_ = h.auditRepo.Create(c.Request.Context(), &auditlog.AuditLog{
		Action:     action,
		TargetType: sql.NullString{String: targetType, Valid: targetType != ""},
		TargetID:   targetID,
		AdminID:    adminID,
		Changes:    changesJSON,
		IPAddress:  sql.NullString{String: c.ClientIP(), Valid: true},
		UserAgent:  sql.NullString{String: c.GetHeader("User-Agent"), Valid: true},
	})
}

func (h *Handler) CreateShare(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	var req struct {
		ShareType  string  `json:"share_type"`
		Permission string  `json:"permission"`
		Password   string  `json:"password"`
		ExpiresAt  *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	createReq := &CreateShareRequest{
		ShareType:  req.ShareType,
		Permission: req.Permission,
		Password:   req.Password,
	}
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			response.Error(c, 400, response.ErrBadRequest, "无效的过期时间")
			return
		}
		createReq.ExpiresAt = &t
	}

	share, err := h.service.CreateShare(c, userID, noteID, createReq)
	if err != nil {
		if err == ErrShareExists {
			response.Error(c, 409, response.ErrConflict, "该笔记已有分享链接")
			return
		}
		response.Error(c, 500, response.ErrInternalServerError, "创建分享失败")
		return
	}

	response.Success(c, gin.H{
		"share_token": share.ShareToken,
		"share_url":   "/share/" + share.ShareToken,
		"expires_at":  share.ExpiresAt,
		"status":      share.Status,
		"status_description": shareStatusDesc(share.Status),
	})
}

func (h *Handler) GetShareStatus(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	share, err := h.service.GetShareStatus(c, userID, noteID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "分享不存在")
		return
	}

	resp := gin.H{
		"share_token":        share.ShareToken,
		"share_url":          "/share/" + share.ShareToken,
		"permission":         share.Permission,
		"password_protected": share.PasswordHash.Valid,
		"expires_at":         share.ExpiresAt,
		"status":             share.Status,
		"status_description": shareStatusDesc(share.Status),
		"created_at":         share.CreatedAt,
	}

	if share.RejectionReason.Valid {
		resp["rejection_reason"] = share.RejectionReason.String
	}

	response.Success(c, resp)
}

func (h *Handler) AccessShare(c *gin.Context) {
	token := c.Param("token")
	password := c.GetHeader("X-Share-Password")

	noteItem, permission, err := h.service.AccessShare(c, token, password)
	if err != nil {
		switch err {
		case ErrSharePending:
			response.Error(c, 403, response.ErrForbidden, "分享待审核，暂不可访问")
		case ErrShareRejected:
			response.Error(c, 403, response.ErrForbidden, "分享未通过审核")
		case ErrShareExpired:
			response.Error(c, 410, response.ErrNotFound, "分享已过期")
		case ErrInvalidPassword:
			response.Error(c, 401, response.ErrUnauthorized, "需要密码访问")
		case ErrShareNotFound:
			response.Error(c, 404, response.ErrNotFound, "分享不存在")
		default:
			response.Error(c, 500, response.ErrInternalServerError, "获取分享失败")
		}
		return
	}

	response.Success(c, gin.H{
		"note": gin.H{
			"id":      noteItem.ID,
			"title":   noteItem.Title,
			"content": noteItem.Content,
		},
		"permission": permission,
	})
}

func (h *Handler) DeleteShare(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的笔记 ID")
		return
	}

	if err := h.service.DeleteShare(c, userID, noteID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "取消分享失败")
		return
	}

	response.Success(c, nil)
}

// Admin share handlers
func (h *Handler) GetPendingList(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	items, total, err := h.service.ListShares(c, status, keyword, page, size)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取分享列表失败")
		return
	}

	response.SuccessPaginated(c, items, total, page, size)
}

func (h *Handler) GetShareDetail(c *gin.Context) {
	shareID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的分享 ID")
		return
	}

	share, err := h.service.GetShareDetail(c, shareID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "分享不存在")
		return
	}

	response.Success(c, share)
}

func (h *Handler) GetShareStats(c *gin.Context) {
	stats, err := h.service.GetShareStats(c)
	if err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "获取分享统计失败")
		return
	}
	response.Success(c, stats)
}

func (h *Handler) BatchReview(c *gin.Context) {
	reviewerID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		ShareIDs []string `json:"share_ids" binding:"required"`
		Action   string   `json:"action" binding:"required"`
		Reason   string   `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	if req.Action != "approve" && req.Action != "reject" {
		response.Error(c, 400, response.ErrBadRequest, "action 必须为 approve 或 reject")
		return
	}

	var ids []uuid.UUID
	for _, idStr := range req.ShareIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, 400, response.ErrBadRequest, fmt.Sprintf("无效的分享 ID: %s", idStr))
			return
		}
		ids = append(ids, id)
	}

	if err := h.service.BatchReview(c, ids, reviewerID, req.Action, req.Reason); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "批量审核失败")
		return
	}

	response.Success(c, gin.H{"processed_count": len(ids)})
}

func (h *Handler) ApproveShare(c *gin.Context) {
	reviewerID, _ := uuid.Parse(c.GetString("user_id"))
	shareID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的分享 ID")
		return
	}

	if err := h.service.Approve(c, shareID, reviewerID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "审核失败")
		return
	}

	h.audit(c, "share.approve", "share", uuid.NullUUID{UUID: shareID, Valid: true}, nil)
	response.Success(c, nil)
}

func (h *Handler) RejectShare(c *gin.Context) {
	reviewerID, _ := uuid.Parse(c.GetString("user_id"))
	shareID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的分享 ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.service.Reject(c, shareID, reviewerID, req.Reason); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "审核失败")
		return
	}

	h.audit(c, "share.reject", "share", uuid.NullUUID{UUID: shareID, Valid: true}, gin.H{"reason": req.Reason})
	response.Success(c, nil)
}

func (h *Handler) ReReviewShare(c *gin.Context) {
	reviewerID, _ := uuid.Parse(c.GetString("user_id"))
	shareID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的分享 ID")
		return
	}

	if err := h.service.ReReview(c, shareID, reviewerID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "复审失败")
		return
	}

	h.audit(c, "share.rereview", "share", uuid.NullUUID{UUID: shareID, Valid: true}, nil)
	response.Success(c, nil)
}

func (h *Handler) RevokeShare(c *gin.Context) {
	reviewerID, _ := uuid.Parse(c.GetString("user_id"))
	shareID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的分享 ID")
		return
	}

	if err := h.service.Revoke(c, shareID, reviewerID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "撤回失败")
		return
	}

	h.audit(c, "share.revoke", "share", uuid.NullUUID{UUID: shareID, Valid: true}, nil)
	response.Success(c, nil)
}

func shareStatusDesc(status string) string {
	switch status {
	case "pending":
		return "待审核"
	case "approved":
		return "已通过"
	case "rejected":
		return "已拒绝"
	case "revoked":
		return "已撤回"
	}
	return status
}
