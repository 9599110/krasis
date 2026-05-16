package keys

import (
	"time"

	"github.com/google/uuid"
	"github.com/krasis/krasis/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

type UpsertRequest struct {
	EncryptedPrivateKey string `json:"encrypted_private_key" binding:"required"`
	Salt                string `json:"salt" binding:"required"`
	IV                  string `json:"iv" binding:"required"`
	PublicKeyRaw        string `json:"public_key_raw" binding:"required"`
}

// UpsertKey handles PUT /keys — upload/sync the encrypted key package to the server.
func (h *Handler) UpsertKey(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, 401, response.ErrUnauthorized, "未认证")
		return
	}

	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, response.ErrBadRequest, "参数错误")
		return
	}

	key := &UserKey{
		UserID:              userID,
		EncryptedPrivateKey: req.EncryptedPrivateKey,
		Salt:                req.Salt,
		IV:                  req.IV,
		PublicKeyRaw:        req.PublicKeyRaw,
	}

	if err := h.repo.Upsert(c, key); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "保存密钥失败")
		return
	}

	response.Success(c, gin.H{"status": "synced"})
}

// GetKey handles GET /keys — fetch the user's encrypted key package from the server.
func (h *Handler) GetKey(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, 401, response.ErrUnauthorized, "未认证")
		return
	}

	key, err := h.repo.GetByUserID(c, userID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "未找到密钥")
		return
	}

	response.Success(c, gin.H{
		"encrypted_private_key": key.EncryptedPrivateKey,
		"salt":                  key.Salt,
		"iv":                    key.IV,
		"public_key_raw":        key.PublicKeyRaw,
		"created_at":            key.CreatedAt,
		"updated_at":            key.UpdatedAt,
	})
}

// DeleteKey handles DELETE /keys — remove the user's key from the server.
func (h *Handler) DeleteKey(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, 401, response.ErrUnauthorized, "未认证")
		return
	}

	if err := h.repo.Delete(c, userID); err != nil {
		response.Error(c, 500, response.ErrInternalServerError, "删除密钥失败")
		return
	}

	response.Success(c, nil)
}

// GetQRPackage handles GET /keys/qr — get the key package in compact JSON for QR code display.
func (h *Handler) GetQRPackage(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Error(c, 401, response.ErrUnauthorized, "未认证")
		return
	}

	key, err := h.repo.GetByUserID(c, userID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "未找到密钥")
		return
	}

	// Compact key package for QR encoding
	now := time.Now().Unix()
	pkg := gin.H{
		"v":  1,                          // version
		"t":  now,                        // timestamp
		"ek": key.EncryptedPrivateKey,    // encrypted private key
		"s":  key.Salt,                   // salt
		"i":  key.IV,                     // iv
		"pk": key.PublicKeyRaw,           // public key
	}

	response.Success(c, pkg)
}

// GetPublicKey handles GET /keys/:user_id/public — get another user's public key.
func (h *Handler) GetPublicKey(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.Error(c, 400, response.ErrBadRequest, "无效的用户 ID")
		return
	}

	pub, err := h.repo.GetPublicKey(c, targetID)
	if err != nil {
		response.Error(c, 404, response.ErrNotFound, "未找到该用户的公钥")
		return
	}

	response.Success(c, gin.H{
		"user_id":         targetID,
		"public_key_raw":  pub,
	})
}
