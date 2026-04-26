package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/krasis/krasis/internal/auth"
)

// M1.3 验收：JWT 校验中间件 — 合法 Bearer 通过，非法拒绝（见 docs/验收标准.md 1.3）
func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, _ := miniredis.Run()
	t.Cleanup(srv.Close)
	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	jm := auth.NewJWTManager("mw-secret", time.Hour, "iss", rdb)
	tok, err := jm.Generate("uid-1", "member", "sid-1")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.GET("/p", AuthMiddleware(jm), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":    c.GetString("user_id"),
			"role":       c.GetString("role"),
			"session_id": c.GetString("session_id"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_MissingAndInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, _ := miniredis.Run()
	t.Cleanup(srv.Close)
	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	jm := auth.NewJWTManager("mw-secret", time.Hour, "iss", rdb)

	r := gin.New()
	r.GET("/p", AuthMiddleware(jm), func(c *gin.Context) { c.Status(http.StatusOK) })

	for name, hdr := range map[string]string{
		"missing": "",
		"badfmt":  "Token x",
		"badjwt":  "Bearer not-a-jwt",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/p", nil)
			if hdr != "" {
				req.Header.Set("Authorization", hdr)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s: status %d", name, w.Code)
			}
		})
	}
}

func TestRequireRole_AllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/a", func(c *gin.Context) {
		c.Set("role", "admin")
	}, RequireRole("admin"), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/a", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}

func TestRequireRole_ForbidsMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/a", func(c *gin.Context) {
		c.Set("role", "member")
	}, RequireRole("admin"), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/a", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d", w.Code)
	}
}
