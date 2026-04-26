package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// M1.3 验收：GET /auth/github/login、GET /auth/google/login 跳转 OAuth 提供方（见 docs/验收标准.md 1.3）
func TestHandler_GitHubLogin_Redirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	om := NewOAuthManager(map[Provider]OAuthConfig{
		ProviderGitHub: {ClientID: "x", ClientSecret: "y", RedirectURI: "http://localhost/cb"},
	})
	h := NewHandler(om, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/github/login", nil)
	h.GitHubLogin(c)

	if w.Code != http.StatusFound {
		t.Fatalf("status %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc == "" || loc[:8] != "https://" {
		t.Fatalf("location: %q", loc)
	}
}

func TestHandler_GoogleLogin_Redirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	om := NewOAuthManager(map[Provider]OAuthConfig{
		ProviderGoogle: {ClientID: "x", ClientSecret: "y", RedirectURI: "http://localhost/cb"},
	})
	h := NewHandler(om, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	h.GoogleLogin(c)

	if w.Code != http.StatusFound {
		t.Fatalf("status %d", w.Code)
	}
	if w.Header().Get("Location") == "" {
		t.Fatal("missing Location")
	}
}

func TestHandler_GitHubLogin_ProviderDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	om := NewOAuthManager(map[Provider]OAuthConfig{})
	h := NewHandler(om, nil, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/github/login", nil)
	h.GitHubLogin(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", w.Code)
	}
}
