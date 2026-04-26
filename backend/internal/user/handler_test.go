package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/krasis/krasis/pkg/response"
)

type stubSessions struct {
	list  []map[string]interface{}
	delID string
	err   error
}

func (s *stubSessions) GetUserSessionsMap(ctx context.Context, userID, currentSessionID string) ([]map[string]interface{}, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.list, nil
}

func (s *stubSessions) DeleteSession(ctx context.Context, sessionID string) error {
	s.delID = sessionID
	return s.err
}

func (s *stubSessions) DeleteAllForUser(ctx context.Context, userID string) error {
	return s.err
}

// M1.3 验收：GET /user/sessions 返回设备列表；DELETE /user/sessions/:id 下线（见 docs/验收标准.md 1.3）
func TestHandler_GetSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubSessions{list: []map[string]interface{}{{"session_id": "s1", "is_current": true}}}
	h := NewHandler(nil, stub)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/sessions", nil)
	c.Set("user_id", "u1")
	c.Set("session_id", "s1")
	h.GetSessions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var body response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body.Data.(map[string]interface{})
	sessions, _ := data["sessions"].([]interface{})
	if len(sessions) != 1 {
		t.Fatalf("sessions: %#v", body.Data)
	}
}

func TestHandler_DeleteSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &stubSessions{}
	h := NewHandler(nil, stub)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/user/sessions/to-remove", nil)
	c.Params = gin.Params{gin.Param{Key: "session_id", Value: "to-remove"}}
	h.DeleteSession(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if stub.delID != "to-remove" {
		t.Fatalf("deleted id %q", stub.delID)
	}
}
