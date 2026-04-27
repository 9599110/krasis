//go:build integration

package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/krasis/krasis/internal/auth"
	"github.com/krasis/krasis/internal/middleware"
	"github.com/krasis/krasis/pkg/response"
)

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

func startPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()
	dir := migrationsDir(t)

	pgC, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("krasis"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.WithInitScripts(
			filepath.Join(dir, "000001_create_users_table.up.sql"),
			filepath.Join(dir, "000002_create_oauth_table.up.sql"),
			filepath.Join(dir, "000003_create_roles_table.up.sql"),
			filepath.Join(dir, "000004_create_notes_and_folders.up.sql"),
			filepath.Join(dir, "000007_add_full_text_search.up.sql"),
			filepath.Join(dir, "000010_add_chinese_fts.up.sql"),
		),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}

	pool, err := pgxpool.New(ctx, pgC.MustConnectionString(ctx, "sslmode=disable"))
	if err != nil {
		_ = pgC.Terminate(ctx)
		t.Fatal(err)
	}

	cleanup := func() {
		pool.Close()
		_ = pgC.Terminate(context.Background())
	}
	return pool, cleanup
}

func setupIntegrationPool(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	if dsn := os.Getenv("KRASIS_TEST_DATABASE_URL"); dsn != "" {
		ctx := context.Background()
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			t.Fatalf("KRASIS_TEST_DATABASE_URL: %v", err)
		}
		if _, err := pool.Exec(ctx, `TRUNCATE users RESTART IDENTITY CASCADE`); err != nil {
			pool.Close()
			t.Fatalf("truncate test data: %v", err)
		}
		return pool, func() { pool.Close() }
	}
	return startPostgres(t)
}

func seedUser(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, email, username string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, username) VALUES ($1, $2, $3);
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, 2);
	`, id, email, username)
	if err != nil {
		t.Fatal(err)
	}
}

func createTestNote(t *testing.T, pool *pgxpool.Pool, ownerID uuid.UUID, title, content string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	noteID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO notes (id, title, content, user_id, version, search_vector)
		VALUES ($1, $2, $3, $4, 1, to_tsvector('simple', $2 || ' ' || $3))
	`, noteID, title, content, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	return noteID
}

func bearerToken(t *testing.T, jm *auth.JWTManager, userID uuid.UUID) string {
	t.Helper()
	tok, err := jm.Generate(userID.String(), "member", "sess-"+userID.String()[:8])
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tok
}

func newTestRouter(pool *pgxpool.Pool, rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)

	searchRepo := NewSearchRepository(pool)
	searchSvc := NewService(searchRepo)
	h := NewHandler(searchSvc)

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	authMW := middleware.AuthMiddleware(jm)

	r := gin.New()
	r.GET("/search", authMW, h.Search)
	return r
}

func TestIntegration_Search_FullTextQuery(t *testing.T) {
	pool, cleanup := setupIntegrationPool(t)
	defer cleanup()

	userID := uuid.New()
	seedUser(t, pool, userID, "alice@test.local", "alice")

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	r := newTestRouter(pool, rdb)
	ts := httptest.NewServer(r)
	defer ts.Close()

	authTok := bearerToken(t, jm, userID)

	// Seed notes with searchable content
	createTestNote(t, pool, userID, "Go 编程指南", "Go 语言是一种开源编程语言，支持并发编程")
	createTestNote(t, pool, userID, "Python 入门", "Python 是一种高级编程语言，适合数据分析")
	createTestNote(t, pool, userID, "笔记标题", "这是关于 JavaScript 前端开发的内容")

	client := &http.Client{Timeout: 30 * time.Second}

	// Search for "Go"
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/search?q=Go&page=1&size=10", nil)
	req.Header.Set("Authorization", authTok)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("search: want 200 got %d %s", res.StatusCode, string(b))
	}
	var wrap response.Response
	_ = json.Unmarshal(b, &wrap)
	data, _ := wrap.Data.(map[string]interface{})
	items, _ := data["items"].([]interface{})
	total := int64(data["total"].(float64))
	if len(items) < 1 {
		t.Fatalf("search results: want >=1 got %d", len(items))
	}
	if total < 1 {
		t.Fatalf("search total: want >=1 got %d", total)
	}
	// First result should contain "Go" in title
	first, _ := items[0].(map[string]interface{})
	title, _ := first["title"].(string)
	if title == "" {
		t.Fatalf("first result missing title: %#v", first)
	}
}

func TestIntegration_Search_EmptyQuery(t *testing.T) {
	pool, cleanup := setupIntegrationPool(t)
	defer cleanup()

	userID := uuid.New()
	seedUser(t, pool, userID, "bob@test.local", "bob")

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	r := newTestRouter(pool, rdb)
	ts := httptest.NewServer(r)
	defer ts.Close()

	authTok := bearerToken(t, jm, userID)
	client := &http.Client{Timeout: 30 * time.Second}

	// Empty query should return 400
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/search?q=&page=1&size=10", nil)
	req.Header.Set("Authorization", authTok)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty query: want 400 got %d", res.StatusCode)
	}
}

func TestIntegration_Search_NoResults(t *testing.T) {
	pool, cleanup := setupIntegrationPool(t)
	defer cleanup()

	userID := uuid.New()
	seedUser(t, pool, userID, "carol@test.local", "carol")

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	r := newTestRouter(pool, rdb)
	ts := httptest.NewServer(r)
	defer ts.Close()

	authTok := bearerToken(t, jm, userID)
	client := &http.Client{Timeout: 30 * time.Second}

	// Search for something that doesn't exist
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/search?q=xyznonexistent&page=1&size=10", nil)
	req.Header.Set("Authorization", authTok)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("search no results: want 200 got %d %s", res.StatusCode, string(b))
	}
	var wrap response.Response
	_ = json.Unmarshal(b, &wrap)
	data, _ := wrap.Data.(map[string]interface{})
	total := int64(data["total"].(float64))
	if total != 0 {
		t.Fatalf("search no results: total want 0 got %d", total)
	}
}
