package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// M1.3 验收：JWT 含 user_id、role、session_id，签名可校验（见 docs/验收标准.md 1.3）
func TestJWTManager_GenerateValidate_Claims(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)

	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	m := NewJWTManager("unit-test-secret", time.Hour, "krasis-test", rdb)
	token, err := m.Generate("user-1", "admin", "sess-9")
	if err != nil {
		t.Fatal(err)
	}

	claims, err := m.Validate(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.Role != "admin" || claims.SessionID != "sess-9" {
		t.Fatalf("claims: %+v", claims)
	}
	if claims.JTI == "" || claims.Issuer != "krasis-test" {
		t.Fatalf("registered: %+v", claims.RegisteredClaims)
	}
}

func TestJWTManager_Blacklist(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)

	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	m := NewJWTManager("unit-test-secret", time.Hour, "krasis-test", rdb)
	token, err := m.Generate("u", "member", "s")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Validate(token)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Blacklist(context.Background(), claims.JTI, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Validate(token); err == nil {
		t.Fatal("expected revoked token error")
	}
}
