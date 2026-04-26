//go:build integration

package note

import (
	"bytes"
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

// 验收：docs/验收标准.md M2.1 笔记 CRUD、乐观锁、版本历史、软删除
//
// 运行方式：
//   1) Docker：go test -tags=integration ./internal/note/ -count=1 -timeout 6m
//   2) 已有库（需已执行 backend/migrations 000001–000005；会 TRUNCATE users CASCADE，请使用独立测试库）：
//      KRASIS_TEST_DATABASE_URL='postgres://.../krasis_test?sslmode=disable' go test -tags=integration ./internal/note/ -count=1

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
		postgres.WithOrderedInitScripts(
			filepath.Join(dir, "000001_create_users_table.up.sql"),
			filepath.Join(dir, "000002_create_oauth_table.up.sql"),
			filepath.Join(dir, "000003_create_roles_table.up.sql"),
			filepath.Join(dir, "000004_create_notes_and_folders.up.sql"),
			filepath.Join(dir, "000005_create_note_versions_and_shares.up.sql"),
		),
	)
	if err != nil {
		t.Fatalf("postgres container: %v (set KRASIS_TEST_DATABASE_URL or ensure Docker can pull images)", err)
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
			t.Fatalf("truncate test data (did you run migrations 000001–000005?): %v", err)
		}
		return pool, func() { pool.Close() }
	}
	return startPostgres(t)
}

func seedTwoUsers(t *testing.T, pool *pgxpool.Pool) (userA, userB uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	userA = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, username) VALUES ($1, 'alice@test.local', 'alice');
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, 2);
	`, userA)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, username) VALUES ($1, 'bob@test.local', 'bob');
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, 2);
	`, userB)
	if err != nil {
		t.Fatal(err)
	}
	return userA, userB
}

func newNoteTestRouter(pool *pgxpool.Pool, rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := NewNoteRepository(pool)
	svc := NewNoteService(repo)
	h := NewHandler(svc)

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	authMW := middleware.AuthMiddleware(jm)

	r := gin.New()
	notes := r.Group("/notes")
	notes.Use(authMW)
	{
		notes.GET("", h.ListNotes)
		notes.POST("", h.CreateNote)
		notes.GET("/:id", h.GetNote)
		notes.PUT("/:id", h.UpdateNote)
		notes.DELETE("/:id", h.DeleteNote)
		notes.GET("/:id/versions", h.GetVersions)
		notes.POST("/:id/versions/:version/restore", h.RestoreVersion)
	}
	return r
}

func bearerToken(t *testing.T, jm *auth.JWTManager, userID uuid.UUID) string {
	t.Helper()
	tok, err := jm.Generate(userID.String(), "member", "sess-"+userID.String()[:8])
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tok
}

func decodeResp(t *testing.T, body []byte) response.Response {
	t.Helper()
	var out response.Response
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("json: %v body=%s", err, string(body))
	}
	return out
}

func TestIntegration_NoteAPI_CRUD_OptimisticLock_Versions(t *testing.T) {
	pool, cleanup := setupIntegrationPool(t)
	defer cleanup()

	userA, userB := seedTwoUsers(t, pool)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	r := newNoteTestRouter(pool, rdb)
	ts := httptest.NewServer(r)
	defer ts.Close()

	authA := bearerToken(t, jm, userA)
	authB := bearerToken(t, jm, userB)
	client := &http.Client{Timeout: 30 * time.Second}

	// Create
	createBody := `{"title":"Hello","content":"body text"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes", bytes.NewReader([]byte(createBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authA)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status %d", res.StatusCode)
	}
	var createWrap response.Response
	if err := json.NewDecoder(res.Body).Decode(&createWrap); err != nil {
		t.Fatal(err)
	}
	if createWrap.Code != 0 {
		t.Fatalf("create code %d msg %s", createWrap.Code, createWrap.Message)
	}
	data, _ := createWrap.Data.(map[string]interface{})
	noteIDStr, _ := data["id"].(string)
	if noteIDStr == "" {
		t.Fatalf("missing id: %#v", data)
	}
	if int(data["version"].(float64)) != 1 {
		t.Fatalf("version want 1 got %#v", data["version"])
	}
	noteID := uuid.MustParse(noteIDStr)

	// List (pagination)
	for range 2 {
		body := `{"title":"Extra","content":"x"}`
		req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes", bytes.NewReader([]byte(body)))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", authA)
		res2, err := client.Do(req2)
		if err != nil {
			t.Fatal(err)
		}
		res2.Body.Close()
		if res2.StatusCode != http.StatusOK {
			t.Fatalf("create extra %d", res2.StatusCode)
		}
	}

	reqList, _ := http.NewRequest(http.MethodGet, ts.URL+"/notes?page=1&size=2&sort=updated_at&order=desc", nil)
	reqList.Header.Set("Authorization", authA)
	resList, err := client.Do(reqList)
	if err != nil {
		t.Fatal(err)
	}
	bList, err := io.ReadAll(resList.Body)
	resList.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resList.StatusCode != http.StatusOK {
		t.Fatalf("list %d %s", resList.StatusCode, string(bList))
	}
	var listWrap response.Response
	_ = json.Unmarshal(bList, &listWrap)
	pd, ok := listWrap.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("list data %#v", listWrap.Data)
	}
	if int64(pd["total"].(float64)) < 3 {
		t.Fatalf("total want >=3 got %#v", pd["total"])
	}

	// Other user cannot read
	reqDeny, _ := http.NewRequest(http.MethodGet, ts.URL+"/notes/"+noteID.String(), nil)
	reqDeny.Header.Set("Authorization", authB)
	resDeny, _ := client.Do(reqDeny)
	bDeny, err := io.ReadAll(resDeny.Body)
	resDeny.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resDeny.StatusCode != http.StatusNotFound {
		t.Fatalf("cross user get: %d %s", resDeny.StatusCode, string(bDeny))
	}

	// Get detail
	reqGet, _ := http.NewRequest(http.MethodGet, ts.URL+"/notes/"+noteID.String(), nil)
	reqGet.Header.Set("Authorization", authA)
	resGet, _ := client.Do(reqGet)
	bGet, err := io.ReadAll(resGet.Body)
	resGet.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resGet.StatusCode != http.StatusOK {
		t.Fatalf("get %d %s", resGet.StatusCode, string(bGet))
	}
	getWrap := decodeResp(t, bGet)
	gd, _ := getWrap.Data.(map[string]interface{})
	if gd["content"] != "body text" {
		t.Fatalf("content %#v", gd["content"])
	}

	// Update with If-Match
	up := `{"title":"Hello","content":"updated","version":1}`
	reqUp, _ := http.NewRequest(http.MethodPut, ts.URL+"/notes/"+noteID.String(), bytes.NewReader([]byte(up)))
	reqUp.Header.Set("Content-Type", "application/json")
	reqUp.Header.Set("Authorization", authA)
	reqUp.Header.Set("If-Match", "1")
	resUp, _ := client.Do(reqUp)
	bUp, err := io.ReadAll(resUp.Body)
	resUp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resUp.StatusCode != http.StatusOK {
		t.Fatalf("update %d %s", resUp.StatusCode, string(bUp))
	}
	var upWrap response.Response
	_ = json.Unmarshal(bUp, &upWrap)
	upd, _ := upWrap.Data.(map[string]interface{})
	if int(upd["version"].(float64)) != 2 {
		t.Fatalf("version after update want 2 got %#v", upd["version"])
	}

	// Stale version -> 409
	stale := `{"title":"X","content":"Y","version":1}`
	req409, _ := http.NewRequest(http.MethodPut, ts.URL+"/notes/"+noteID.String(), bytes.NewReader([]byte(stale)))
	req409.Header.Set("Content-Type", "application/json")
	req409.Header.Set("Authorization", authA)
	req409.Header.Set("If-Match", "1")
	res409, _ := client.Do(req409)
	b409, err := io.ReadAll(res409.Body)
	res409.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if res409.StatusCode != http.StatusConflict {
		t.Fatalf("conflict status %d body %s", res409.StatusCode, string(b409))
	}
	c409 := decodeResp(t, b409)
	if c409.Code != response.ErrConflict {
		t.Fatalf("conflict code %d", c409.Code)
	}

	time.Sleep(200 * time.Millisecond) // async SaveVersion after update

	// Versions
	reqVer, _ := http.NewRequest(http.MethodGet, ts.URL+"/notes/"+noteID.String()+"/versions", nil)
	reqVer.Header.Set("Authorization", authA)
	resVer, _ := client.Do(reqVer)
	bVer, err := io.ReadAll(resVer.Body)
	resVer.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resVer.StatusCode != http.StatusOK {
		t.Fatalf("versions %d %s", resVer.StatusCode, string(bVer))
	}
	var verWrap response.Response
	_ = json.Unmarshal(bVer, &verWrap)
	vd, _ := verWrap.Data.(map[string]interface{})
	items, _ := vd["items"].([]interface{})
	if len(items) < 2 {
		t.Fatalf("versions count %d", len(items))
	}

	// Restore v1
	reqRest, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes/"+noteID.String()+"/versions/1/restore", nil)
	reqRest.Header.Set("Authorization", authA)
	resRest, _ := client.Do(reqRest)
	bRest, err := io.ReadAll(resRest.Body)
	resRest.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resRest.StatusCode != http.StatusOK {
		t.Fatalf("restore %d %s", resRest.StatusCode, string(bRest))
	}

	reqGet2, _ := http.NewRequest(http.MethodGet, ts.URL+"/notes/"+noteID.String(), nil)
	reqGet2.Header.Set("Authorization", authA)
	resGet2, _ := client.Do(reqGet2)
	bGet2, err := io.ReadAll(resGet2.Body)
	resGet2.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resGet2.StatusCode != http.StatusOK {
		t.Fatalf("get after restore %d %s", resGet2.StatusCode, string(bGet2))
	}
	var get2 response.Response
	_ = json.Unmarshal(bGet2, &get2)
	g2, _ := get2.Data.(map[string]interface{})
	if g2["content"] != "body text" {
		t.Fatalf("after restore content want body text got %#v", g2["content"])
	}

	// Soft delete
	reqDel, _ := http.NewRequest(http.MethodDelete, ts.URL+"/notes/"+noteID.String(), nil)
	reqDel.Header.Set("Authorization", authA)
	resDel, _ := client.Do(reqDel)
	bDel, err := io.ReadAll(resDel.Body)
	resDel.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resDel.StatusCode != http.StatusOK {
		t.Fatalf("delete %d %s", resDel.StatusCode, string(bDel))
	}

	reqGone, _ := http.NewRequest(http.MethodGet, ts.URL+"/notes/"+noteID.String(), nil)
	reqGone.Header.Set("Authorization", authA)
	resGone, _ := client.Do(reqGone)
	bGone, err := io.ReadAll(resGone.Body)
	resGone.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resGone.StatusCode != http.StatusNotFound {
		t.Fatalf("get deleted want 404 got %d %s", resGone.StatusCode, string(bGone))
	}
}
