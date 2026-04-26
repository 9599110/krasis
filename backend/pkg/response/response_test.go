package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// M1.1 验收：统一错误处理 — 所有响应符合 {code, message, data} 格式（见 docs/验收标准.md 1.1）
func TestSuccessResponseEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Success(c, gin.H{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d", w.Code, http.StatusOK)
	}
	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Code != 0 || body.Message != "success" {
		t.Fatalf("envelope: %+v", body)
	}
	data, ok := body.Data.(map[string]interface{})
	if !ok || data["hello"] != "world" {
		t.Fatalf("data: %#v", body.Data)
	}
}

func TestErrorResponseEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Error(c, http.StatusBadRequest, ErrBadRequest, "bad")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", w.Code)
	}
	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Code != ErrBadRequest || body.Message != "bad" {
		t.Fatalf("envelope: %+v", body)
	}
	if body.Data != nil {
		t.Fatalf("data should be omitted for errors: %#v", body.Data)
	}
}

func TestSuccessPaginatedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	SuccessPaginated(c, []string{"a"}, 10, 1, 5)

	var body Response
	_ = json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&body)
	pd, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data type: %#v", body.Data)
	}
	if int(pd["total"].(float64)) != 10 || pd["has_more"].(bool) != true {
		t.Fatalf("pagination: %+v", pd)
	}
}
