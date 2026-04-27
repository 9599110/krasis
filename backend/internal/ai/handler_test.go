package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubAIService struct {
	askErr             error
	askStreamTokens     []string
	askStreamErr       error
	createConv         *Conversation
	createConvErr      error
	listConvs          []*Conversation
	listConvsErr       error
	listMsgs           []*Message
	listMsgsErr        error
	saveMsgErr         error
}

func (s *stubAIService) Ask(ctx context.Context, userID string, req *AskRequest) (*AskResponse, error) {
	if s.askErr != nil {
		return nil, s.askErr
	}
	return &AskResponse{
		Answer: "This is a test answer",
	}, nil
}

func (s *stubAIService) AskStream(ctx context.Context, userID string, req *AskRequest) (<-chan string, error) {
	if s.askStreamErr != nil {
		return nil, s.askStreamErr
	}
	ch := make(chan string, len(s.askStreamTokens))
	for _, t := range s.askStreamTokens {
		ch <- t
	}
	close(ch)
	return ch, nil
}

func (s *stubAIService) CreateConversation(ctx context.Context, userID, title, modelID string) (*Conversation, error) {
	if s.createConvErr != nil {
		return nil, s.createConvErr
	}
	if s.createConv != nil {
		return s.createConv, nil
	}
	return &Conversation{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Title:     title,
		Model:     modelID,
		CreatedAt: time.Now(),
	}, nil
}

func (s *stubAIService) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	return s.listConvs, s.listConvsErr
}

func (s *stubAIService) ListMessages(ctx context.Context, conversationID, userID string) ([]*Message, error) {
	return s.listMsgs, s.listMsgsErr
}

func (s *stubAIService) SaveMessage(ctx context.Context, msg *Message) error {
	return s.saveMsgErr
}

func (s *stubAIService) GetModelManager() *ModelConfigManager {
	return nil
}

func TestHandler_Ask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{
		askStreamTokens: []string{"Hello", " world"},
	}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/ask", strings.NewReader(`{"question":"What is Go?"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New().String())
	h.AskStream(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
}

func TestHandler_Ask_EmptyQuestion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/ask", strings.NewReader(`{"question":""}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New().String())
	h.Ask(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_Ask_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/ask", strings.NewReader(`invalid`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New().String())
	h.Ask(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_Ask_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{askErr: ErrNoLLMConfigured}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/ask", strings.NewReader(`{"question":"What is Go?"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New().String())
	h.Ask(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d want 500", w.Code)
	}
}

func TestHandler_AskStream_EmptyQuestion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/ask/stream", strings.NewReader(`{"question":""}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New().String())
	h.AskStream(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_AskStream_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{
		askStreamTokens: []string{"Hello", " world"},
	}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ai/ask/stream", strings.NewReader(`{"question":"What is Go?"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New().String())
	h.AskStream(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
}

func TestHandler_ListConversations_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	convID := uuid.New()
	svc := &stubAIService{
		listConvs: []*Conversation{
			{ID: convID, Title: "Test", CreatedAt: time.Now()},
		},
	}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ai/conversations", nil)
	c.Set("user_id", uuid.New().String())
	h.ListConversations(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
}

func TestHandler_ListConversations_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{listConvsErr: ErrConversationNotFound}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ai/conversations", nil)
	c.Set("user_id", uuid.New().String())
	h.ListConversations(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d want 500", w.Code)
	}
}

func TestHandler_GetMessages_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	convID := uuid.New()
	msgID := uuid.New()
	svc := &stubAIService{
		listMsgs: []*Message{
			{ID: msgID, ConversationID: convID, Role: "user", Content: "Hello"},
			{ID: uuid.New(), ConversationID: convID, Role: "assistant", Content: "Hi there"},
		},
	}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ai/conversations/"+convID.String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: convID.String()}}
	c.Set("user_id", uuid.New().String())
	h.GetMessages(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
}

func TestHandler_GetMessages_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	svc := &stubAIService{listMsgsErr: ErrConversationNotFound}
	h := NewHandler(svc, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ai/conversations/"+uuid.New().String()+"/messages", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Set("user_id", uuid.New().String())
	h.GetMessages(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestConversation_JSON(t *testing.T) {
	conv := &Conversation{
		ID:        uuid.New(),
		UserID:    uuid.New().String(),
		Title:     "Test Conversation",
		ModelID:   "gpt-4",
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(conv)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Conversation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Title != conv.Title {
		t.Fatalf("title mismatch: want %s got %s", conv.Title, decoded.Title)
	}
}

func TestMessage_JSON(t *testing.T) {
	msg := &Message{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		Role:           "user",
		Content:        "What is the meaning of life?",
		TokenCount:     10,
		CreatedAt:      time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Role != msg.Role {
		t.Fatalf("role mismatch")
	}
	if decoded.Content != msg.Content {
		t.Fatalf("content mismatch")
	}
}
