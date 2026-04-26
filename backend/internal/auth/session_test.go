package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// M1.3 验收：Session 写入 Redis，多 Session 可列举（见 docs/验收标准.md 1.3 / modules 2.1.2）
func TestSessionManager_CreateAndList(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)

	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	m := NewSessionManager(rdb, time.Hour)
	ctx := context.Background()

	s1, err := m.Create(ctx, "user-a", &SessionInfo{UserAgent: "Mozilla/5.0", IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := m.Create(ctx, "user-a", &SessionInfo{UserAgent: "Other", IPAddress: "10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}

	list, err := m.GetUserSessionsMap(ctx, "user-a", s1.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("sessions len %d", len(list))
	}
	var current int
	for _, row := range list {
		if row["is_current"].(bool) {
			current++
		}
	}
	if current != 1 {
		t.Fatalf("is_current count %d", current)
	}

	if err := m.DeleteSession(ctx, s2.SessionID); err != nil {
		t.Fatal(err)
	}
	list2, err := m.GetUserSessionsMap(ctx, "user-a", s1.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 1 {
		t.Fatalf("after delete len %d", len(list2))
	}
}
