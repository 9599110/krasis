//go:build integration

package share

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/krasis/krasis/internal/auditlog"
	"github.com/krasis/krasis/internal/middleware"
	"github.com/krasis/krasis/internal/note"
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
		postgres.WithOrderedInitScripts(
			filepath.Join(dir, "000001_create_users_table.up.sql"),
			filepath.Join(dir, "000002_create_oauth_table.up.sql"),
			filepath.Join(dir, "000003_create_roles_table.up.sql"),
			filepath.Join(dir, "000004_create_notes_and_folders.up.sql"),
			filepath.Join(dir, "000005_create_note_versions_and_shares.up.sql"),
			filepath.Join(dir, "000006_create_search_tables.up.sql"),
			filepath.Join(dir, "000009_create_admin_tables.up.sql"),
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
			t.Fatalf("truncate test data (did you run migrations?): %v", err)
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
		INSERT INTO notes (id, title, content, user_id, version) VALUES ($1, $2, $3, $4, 1)
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

func bearerTokenRole(t *testing.T, jm *auth.JWTManager, userID uuid.UUID, role string) string {
	t.Helper()
	tok, err := jm.Generate(userID.String(), role, "sess-"+userID.String()[:8])
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

func newTestRouter(pool *pgxpool.Pool, rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)

	noteRepo := note.NewNoteRepository(pool)
	shareRepo := NewShareRepository(pool)
	shareSvc := NewService(shareRepo, noteRepo)
	auditRepo := auditlog.NewRepository(pool)
	h := NewHandler(shareSvc, auditRepo)

	jm := auth.NewJWTManager("integration-test-secret", time.Hour, "krasis-test", rdb)
	authMW := middleware.AuthMiddleware(jm)

	r := gin.New()
	// Authenticated note routes
	notes := r.Group("/notes")
	notes.Use(authMW)
	{
		notes.POST("/:id/share", h.CreateShare)
		notes.GET("/:id/share", h.GetShareStatus)
		notes.DELETE("/:id/share", h.DeleteShare)
	}
	// Public share access
	r.GET("/share/:token", h.AccessShare)
	// Admin share routes
	admin := r.Group("/admin/shares")
	admin.Use(authMW)
	{
		admin.GET("/pending", h.GetPendingList)
		admin.GET("/stats", h.GetShareStats)
		admin.POST("/batch/review", h.BatchReview)
		admin.GET("/:id", h.GetShareDetail)
		admin.POST("/:id/approve", h.ApproveShare)
		admin.POST("/:id/reject", h.RejectShare)
		admin.POST("/:id/re-review", h.ReReviewShare)
		admin.DELETE("/:id/revoke", h.RevokeShare)
	}
	return r
}

func TestIntegration_Share_Create_Access_Delete(t *testing.T) {
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
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a note
	noteID := createTestNote(t, pool, userID, "Shared Note", "This is shared content")

	// Create share
	createBody := `{"share_type":"link","permission":"read"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes/"+noteID.String()+"/share", bytes.NewReader([]byte(createBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authTok)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create share %d %s", res.StatusCode, string(b))
	}
	createWrap := decodeResp(t, b)
	data, _ := createWrap.Data.(map[string]interface{})
	token, _ := data["share_token"].(string)
	if token == "" {
		t.Fatalf("missing share_token: %#v", data)
	}
	status, _ := data["status"].(string)
	if status != "pending" {
		t.Fatalf("share status want pending got %s", status)
	}
	// Access share should fail (pending review)
	reqAccess, _ := http.NewRequest(http.MethodGet, ts.URL+"/share/"+token, nil)
	resAccess, err := client.Do(reqAccess)
	if err != nil {
		t.Fatal(err)
	}
	bAccess, _ := io.ReadAll(resAccess.Body)
	resAccess.Body.Close()
	if resAccess.StatusCode != http.StatusForbidden {
		t.Fatalf("access pending: want 403 got %d %s", resAccess.StatusCode, string(bAccess))
	}

	// Admin approves the share
	adminID := uuid.New()
	seedUser(t, pool, adminID, "admin@test.local", "admin")
	// Set role to admin via roles table
	ctx := context.Background()
	_, err = pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, 1)
	`, adminID)
	if err != nil {
		t.Fatal(err)
	}
	adminTok := bearerTokenRole(t, jm, adminID, "admin")

	// Get share UUID from pending list
	reqPending, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin/shares/pending", nil)
	reqPending.Header.Set("Authorization", adminTok)
	resPending, _ := client.Do(reqPending)
	bPending, _ := io.ReadAll(resPending.Body)
	resPending.Body.Close()
	if resPending.StatusCode != http.StatusOK {
		t.Fatalf("pending list: want 200 got %d %s", resPending.StatusCode, string(bPending))
	}
	var pendWrap response.Response
	_ = json.Unmarshal(bPending, &pendWrap)
	pd, _ := pendWrap.Data.(map[string]interface{})
	items, _ := pd["items"].([]interface{})
	if len(items) == 0 {
		t.Fatal("pending list empty")
	}
	firstShare, _ := items[0].(map[string]interface{})
	shareUUID, _ := firstShare["id"].(string)

	reqApprove, _ := http.NewRequest(http.MethodPost, ts.URL+"/admin/shares/"+shareUUID+"/approve", nil)
	reqApprove.Header.Set("Authorization", adminTok)
	resApprove, err := client.Do(reqApprove)
	if err != nil {
		t.Fatal(err)
	}
	defer resApprove.Body.Close()
	if resApprove.StatusCode != http.StatusOK {
		bA, _ := io.ReadAll(resApprove.Body)
		t.Fatalf("approve share: want 200 got %d %s", resApprove.StatusCode, string(bA))
	}

	// Now access should succeed
	reqAccess2, _ := http.NewRequest(http.MethodGet, ts.URL+"/share/"+token, nil)
	resAccess2, err := client.Do(reqAccess2)
	if err != nil {
		t.Fatal(err)
	}
	bAccess2, _ := io.ReadAll(resAccess2.Body)
	resAccess2.Body.Close()
	if resAccess2.StatusCode != http.StatusOK {
		t.Fatalf("access approved share: want 200 got %d %s", resAccess2.StatusCode, string(bAccess2))
	}
	var accessWrap response.Response
	_ = json.Unmarshal(bAccess2, &accessWrap)
	ad, _ := accessWrap.Data.(map[string]interface{})
	note, _ := ad["note"].(map[string]interface{})
	if note["content"] != "This is shared content" {
		t.Fatalf("access content: want 'This is shared content' got %#v", note["content"])
	}

	// Delete share
	reqDel, _ := http.NewRequest(http.MethodDelete, ts.URL+"/notes/"+noteID.String()+"/share", nil)
	reqDel.Header.Set("Authorization", authTok)
	resDel, err := client.Do(reqDel)
	if err != nil {
		t.Fatal(err)
	}
	defer resDel.Body.Close()
	if resDel.StatusCode != http.StatusOK {
		bD, _ := io.ReadAll(resDel.Body)
		t.Fatalf("delete share: want 200 got %d %s", resDel.StatusCode, string(bD))
	}

	// Access after delete should fail
	reqAccess3, _ := http.NewRequest(http.MethodGet, ts.URL+"/share/"+token, nil)
	resAccess3, _ := client.Do(reqAccess3)
	if resAccess3.StatusCode != http.StatusNotFound {
		t.Fatalf("access deleted share: want 404 got %d", resAccess3.StatusCode)
	}
}

func TestIntegration_Share_PasswordProtection(t *testing.T) {
	pool, cleanup := setupIntegrationPool(t)
	defer cleanup()

	userID := uuid.New()
	seedUser(t, pool, userID, "bob@test.local", "bob")
	adminID := uuid.New()
	seedUser(t, pool, adminID, "admin2@test.local", "admin2")
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 1)`, adminID)

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
	adminTok := bearerTokenRole(t, jm, adminID, "admin")
	client := &http.Client{Timeout: 30 * time.Second}

	noteID := createTestNote(t, pool, userID, "Password Note", "secret content")

	// Create share with password
	expiry := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	createBody := fmt.Sprintf(`{"share_type":"link","permission":"read","password":"testpass123","expires_at":"%s"}`, expiry)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes/"+noteID.String()+"/share", bytes.NewReader([]byte(createBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authTok)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create share %d %s", res.StatusCode, string(b))
	}
	var createWrap response.Response
	_ = json.Unmarshal(b, &createWrap)
	data, _ := createWrap.Data.(map[string]interface{})
	token, _ := data["share_token"].(string)
	if token == "" {
		t.Fatalf("missing share_token")
	}

	// Admin approves - get UUID from pending list
	reqPending, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin/shares/pending", nil)
	reqPending.Header.Set("Authorization", adminTok)
	resPending, _ := client.Do(reqPending)
	bPending, _ := io.ReadAll(resPending.Body)
	resPending.Body.Close()
	var pendWrap response.Response
	_ = json.Unmarshal(bPending, &pendWrap)
	pd, _ := pendWrap.Data.(map[string]interface{})
	pItems, _ := pd["items"].([]interface{})
	if len(pItems) == 0 {
		t.Fatal("pending list empty")
	}
	first, _ := pItems[0].(map[string]interface{})
	shareUUID, _ := first["id"].(string)

	reqApprove, _ := http.NewRequest(http.MethodPost, ts.URL+"/admin/shares/"+shareUUID+"/approve", nil)
	reqApprove.Header.Set("Authorization", adminTok)
	resApprove, _ := client.Do(reqApprove)
	resApprove.Body.Close()

	// Access without password should fail
	reqNoPw, _ := http.NewRequest(http.MethodGet, ts.URL+"/share/"+token, nil)
	resNoPw, _ := client.Do(reqNoPw)
	if resNoPw.StatusCode != http.StatusUnauthorized {
		t.Fatalf("access without password: want 401 got %d", resNoPw.StatusCode)
	}

	// Access with wrong password should fail
	reqBadPw, _ := http.NewRequest(http.MethodGet, ts.URL+"/share/"+token, nil)
	reqBadPw.Header.Set("X-Share-Password", "wrongpassword")
	resBadPw, _ := client.Do(reqBadPw)
	if resBadPw.StatusCode != http.StatusUnauthorized {
		t.Fatalf("access with wrong password: want 401 got %d", resBadPw.StatusCode)
	}

	// Access with correct password should succeed
	reqGoodPw, _ := http.NewRequest(http.MethodGet, ts.URL+"/share/"+token, nil)
	reqGoodPw.Header.Set("X-Share-Password", "testpass123")
	resGoodPw, _ := client.Do(reqGoodPw)
	if resGoodPw.StatusCode != http.StatusOK {
		t.Fatalf("access with correct password: want 200 got %d", resGoodPw.StatusCode)
	}
}

func TestIntegration_Share_DuplicateCreate(t *testing.T) {
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

	noteID := createTestNote(t, pool, userID, "Dup Note", "content")

	// First create
	req1, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes/"+noteID.String()+"/share", bytes.NewReader([]byte(`{"share_type":"link"}`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", authTok)
	res1, _ := client.Do(req1)
	res1.Body.Close()
	if res1.StatusCode != http.StatusOK {
		t.Fatalf("first create: want 200 got %d", res1.StatusCode)
	}

	// Second create should conflict
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes/"+noteID.String()+"/share", bytes.NewReader([]byte(`{"share_type":"link"}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", authTok)
	res2, _ := client.Do(req2)
	res2.Body.Close()
	if res2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate create: want 409 got %d", res2.StatusCode)
	}
}

func TestIntegration_Admin_BatchReview_ShareList(t *testing.T) {
	pool, cleanup := setupIntegrationPool(t)
	defer cleanup()

	userID := uuid.New()
	seedUser(t, pool, userID, "dave@test.local", "dave")
	adminID := uuid.New()
	seedUser(t, pool, adminID, "superadmin@test.local", "superadmin")
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, 1)`, adminID)

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
	adminTok := bearerTokenRole(t, jm, adminID, "admin")
	client := &http.Client{Timeout: 30 * time.Second}

	// Create two notes with shares
	noteA := createTestNote(t, pool, userID, "Note A", "content A")
	noteB := createTestNote(t, pool, userID, "Note B", "content B")

	for _, nid := range []uuid.UUID{noteA, noteB} {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/notes/"+nid.String()+"/share", bytes.NewReader([]byte(`{"share_type":"link"}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authTok)
		res, _ := client.Do(req)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("create share: want 200 got %d", res.StatusCode)
		}
	}

	// Get pending list to obtain share UUIDs
	reqList, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin/shares/pending", nil)
	reqList.Header.Set("Authorization", adminTok)
	resList, _ := client.Do(reqList)
	bList, _ := io.ReadAll(resList.Body)
	resList.Body.Close()
	if resList.StatusCode != http.StatusOK {
		t.Fatalf("list shares: want 200 got %d %s", resList.StatusCode, string(bList))
	}
	var listWrap response.Response
	_ = json.Unmarshal(bList, &listWrap)
	ld, _ := listWrap.Data.(map[string]interface{})
	lItems, _ := ld["items"].([]interface{})
	if len(lItems) < 2 {
		t.Fatalf("pending list want >=2 items got %d", len(lItems))
	}
	var shareIDs []string
	for _, item := range lItems[:2] {
		m, _ := item.(map[string]interface{})
		shareIDs = append(shareIDs, m["id"].(string))
	}

	// Batch approve
	batchBody := fmt.Sprintf(`{"share_ids":["%s","%s"],"action":"approve"}`, shareIDs[0], shareIDs[1])
	reqBatch, _ := http.NewRequest(http.MethodPost, ts.URL+"/admin/shares/batch/review", bytes.NewReader([]byte(batchBody)))
	reqBatch.Header.Set("Content-Type", "application/json")
	reqBatch.Header.Set("Authorization", adminTok)
	resBatch, _ := client.Do(reqBatch)
	defer resBatch.Body.Close()
	if resBatch.StatusCode != http.StatusOK {
		bB, _ := io.ReadAll(resBatch.Body)
		t.Fatalf("batch review: want 200 got %d %s", resBatch.StatusCode, string(bB))
	}

	// Get share stats
	reqStats, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin/shares/stats", nil)
	reqStats.Header.Set("Authorization", adminTok)
	resStats, _ := client.Do(reqStats)
	bStats, _ := io.ReadAll(resStats.Body)
	resStats.Body.Close()
	if resStats.StatusCode != http.StatusOK {
		t.Fatalf("stats: want 200 got %d %s", resStats.StatusCode, string(bStats))
	}
	var statsWrap response.Response
	_ = json.Unmarshal(bStats, &statsWrap)
	sd, _ := statsWrap.Data.(map[string]interface{})
	if int(sd["approved"].(float64)) != 2 {
		t.Fatalf("stats approved count want 2 got %.0f", sd["approved"])
	}
}
