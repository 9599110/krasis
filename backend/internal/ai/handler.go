package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AIServiceInterface defines the interface for AI service operations
type AIServiceInterface interface {
	Ask(ctx context.Context, userID string, req *AskRequest) (*AskResponse, error)
	AskStream(ctx context.Context, userID string, req *AskRequest) (<-chan string, error)
	CreateConversation(ctx context.Context, userID, title, modelID string) (*Conversation, error)
	ListConversations(ctx context.Context, userID string) ([]*Conversation, error)
	ListMessages(ctx context.Context, conversationID, userID string) ([]*Message, error)
	SaveMessage(ctx context.Context, msg *Message) error
	GetModelManager() *ModelConfigManager
}

type Handler struct {
	service AIServiceInterface
	logger  *zap.Logger
}

func NewHandler(service AIServiceInterface, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) Ask(c *gin.Context) {
	userID := c.GetString("user_id")

	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1001, "message": "参数错误"})
		return
	}
	if req.Question == "" {
		c.JSON(400, gin.H{"code": 1001, "message": "问题不能为空"})
		return
	}

	// Auto-create conversation if not provided
	conversationID := req.ConversationID
	if conversationID == "" {
		title := req.Question
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv, err := h.service.CreateConversation(c.Request.Context(), userID, title, req.ModelID)
		if err != nil {
			c.JSON(500, gin.H{"code": 3001, "message": "创建对话失败"})
			return
		}
		conversationID = conv.ID.String()
	}

	resp, err := h.service.Ask(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(500, gin.H{"code": 3001, "message": err.Error()})
		return
	}

	// Persist messages asynchronously with error logging
	go func() {
		bgCtx := context.Background()
		userMsg := &Message{
			ConversationID: uuid.MustParse(conversationID),
			Role:           "user",
			Content:        req.Question,
		}
		if err := h.service.SaveMessage(bgCtx, userMsg); err != nil {
			h.logger.Error("failed to save user message", zap.String("conversation_id", conversationID), zap.Error(err))
		}
		assistantMsg := &Message{
			ConversationID: uuid.MustParse(conversationID),
			Role:           "assistant",
			Content:        resp.Answer,
			TokenCount:     len([]rune(resp.Answer)) / 4,
		}
		if err := h.service.SaveMessage(bgCtx, assistantMsg); err != nil {
			h.logger.Error("failed to save assistant message", zap.String("conversation_id", conversationID), zap.Error(err))
		}
	}()

	resp.ConversationID = conversationID
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": resp})
}

func (h *Handler) AskStream(c *gin.Context) {
	userID := c.GetString("user_id")

	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1001, "message": "参数错误"})
		return
	}
	if req.Question == "" {
		c.JSON(400, gin.H{"code": 1001, "message": "问题不能为空"})
		return
	}

	// Auto-create conversation if not provided
	conversationID := req.ConversationID
	if conversationID == "" {
		title := req.Question
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv, err := h.service.CreateConversation(c.Request.Context(), userID, title, req.ModelID)
		if err != nil {
			c.JSON(500, gin.H{"code": 3001, "message": "创建对话失败"})
			return
		}
		conversationID = conv.ID.String()
	}

	// Save user message asynchronously with error logging
	go func() {
		bgCtx := context.Background()
		msg := &Message{
			ConversationID: uuid.MustParse(conversationID),
			Role:           "user",
			Content:        req.Question,
		}
		if err := h.service.SaveMessage(bgCtx, msg); err != nil {
			h.logger.Error("failed to save user message (stream)", zap.String("conversation_id", conversationID), zap.Error(err))
		}
	}()

	stream, err := h.service.AskStream(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(500, gin.H{"code": 3001, "message": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	var answerBuilder strings.Builder
	for token := range stream {
		answerBuilder.WriteString(token)
		if strings.Contains(token, "\n") {
			token = strings.ReplaceAll(token, "\n", "\\n")
		}
		fmt.Fprintf(c.Writer, "event: token\ndata: {\"token\":%q,\"conversation_id\":%q}\n\n", token, conversationID)
		c.Writer.Flush()
	}

	// Save assistant message asynchronously with error logging
	go func() {
		bgCtx := context.Background()
		answer := answerBuilder.String()
		msg := &Message{
			ConversationID: uuid.MustParse(conversationID),
			Role:           "assistant",
			Content:        answer,
			TokenCount:     len([]rune(answer)) / 4,
		}
		if err := h.service.SaveMessage(bgCtx, msg); err != nil {
			h.logger.Error("failed to save assistant message (stream)", zap.String("conversation_id", conversationID), zap.Error(err))
		}
	}()

	fmt.Fprintf(c.Writer, "event: done\ndata: {\"done\":true,\"conversation_id\":%q}\n\n", conversationID)
	c.Writer.Flush()
}

func (h *Handler) ListConversations(c *gin.Context) {
	userID := c.GetString("user_id")

	convs, err := h.service.ListConversations(c, userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 3001, "message": "获取对话列表失败"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": convs})
}

func (h *Handler) GetMessages(c *gin.Context) {
	userID := c.GetString("user_id")
	conversationID := c.Param("id")

	messages, err := h.service.ListMessages(c, conversationID, userID)
	if err != nil {
		c.JSON(404, gin.H{"code": 1004, "message": "对话不存在"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "message": "success", "data": messages})
}
