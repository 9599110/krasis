package note

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

	"github.com/krasis/krasis/pkg/response"
	"github.com/krasis/krasis/pkg/types"
)

type stubNoteRepo struct {
	createNote  *Note
	getNote     *Note
	listNotes   []*Note
	total       int64
	listErr     error
	getErr      error
	createErr   error
	updateNote  *Note
	updateErr   error
	deleteErr   error
	versions    []*NoteVersion
	versionsErr error
	restoreErr  error
}

var testUserID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

func (r *stubNoteRepo) Create(ctx context.Context, ownerID uuid.UUID, title, content string, folderID *uuid.UUID) (*Note, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	return r.createNote, nil
}
func (r *stubNoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*Note, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.getNote, nil
}
func (r *stubNoteRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, page, size int, sort, order string) ([]*Note, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	return r.listNotes, r.total, nil
}
func (r *stubNoteRepo) UpdateWithOptimisticLock(ctx context.Context, id uuid.UUID, title, content string, version int) (*Note, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return r.updateNote, nil
}
func (r *stubNoteRepo) SoftDelete(ctx context.Context, id uuid.UUID) error { return r.deleteErr }
func (r *stubNoteRepo) PermanentDelete(ctx context.Context, id uuid.UUID) error { return r.deleteErr }
func (r *stubNoteRepo) SaveVersion(ctx context.Context, noteID, changedBy uuid.UUID, title, content string, version int, summary string) error {
	return nil
}
func (r *stubNoteRepo) GetVersions(ctx context.Context, noteID uuid.UUID) ([]*NoteVersion, error) {
	return r.versions, r.versionsErr
}
func (r *stubNoteRepo) GetVersion(ctx context.Context, noteID uuid.UUID, version int) (*NoteVersion, error) {
	if r.versionsErr != nil {
		return nil, r.versionsErr
	}
	if len(r.versions) > 0 {
		return r.versions[0], nil
	}
	return nil, ErrNoteNotFound
}
func (r *stubNoteRepo) RestoreVersion(ctx context.Context, noteID uuid.UUID, version int, title, content string) error {
	return r.restoreErr
}
func (r *stubNoteRepo) Count(ctx context.Context) (int64, error) { return 0, nil }
func (r *stubNoteRepo) CountToday(ctx context.Context, action string) (int64, error) { return 0, nil }
func (r *stubNoteRepo) TotalStorageUsed(ctx context.Context) (int64, error) { return 0, nil }

func TestHandler_CreateNote(t *testing.T) {
	gin.SetMode(gin.TestMode)
	noteID := uuid.New()
	now := time.Now()
	repo := &stubNoteRepo{createNote: &Note{ID: noteID, Title: "Test", Content: "body", Version: 1, CreatedAt: now, UpdatedAt: types.NullTime{Time: now, Valid: true}}}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(`{"title":"Test","content":"body"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	h.CreateNote(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var body response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 {
		t.Fatalf("code %d msg %s", body.Code, body.Message)
	}
}

func TestHandler_CreateNote_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubNoteRepo{}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(`invalid`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	h.CreateNote(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
}

func TestHandler_ListNotes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	noteID := uuid.New()
	now := time.Now()
	repo := &stubNoteRepo{
		listNotes: []*Note{{ID: noteID, Title: "Test", Content: "body", Version: 1, OwnerID: testUserID, CreatedAt: now, UpdatedAt: types.NullTime{Time: now, Valid: true}}},
		total:     1,
	}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/notes?page=1&size=20", nil)
	c.Set("user_id", testUserID.String())
	h.ListNotes(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var body response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 {
		t.Fatalf("code %d", body.Code)
	}
	data, _ := body.Data.(map[string]interface{})
	items, _ := data["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items count %d", len(items))
	}
}

func TestHandler_GetNote_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubNoteRepo{getErr: ErrNoteNotFound}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	noteID := uuid.New()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String(), nil)
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: noteID.String()}}
	h.GetNote(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestHandler_GetNote_PermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubNoteRepo{getErr: ErrPermissionDenied}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	noteID := uuid.New()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String(), nil)
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: noteID.String()}}
	h.GetNote(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestHandler_GetNote_Deleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	noteID := uuid.New()
	repo := &stubNoteRepo{getNote: &Note{ID: noteID, OwnerID: testUserID, IsDeleted: true, CreatedAt: now, UpdatedAt: types.NullTime{Time: now, Valid: true}}}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String(), nil)
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: noteID.String()}}
	h.GetNote(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestHandler_UpdateNote_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	noteID := uuid.New()
	repo := &stubNoteRepo{
		getNote:   &Note{ID: noteID, OwnerID: testUserID, Version: 2, CreatedAt: time.Now(), UpdatedAt: types.NullTime{Time: time.Now(), Valid: true}},
		updateErr: ErrVersionConflict,
	}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/notes/"+noteID.String(), strings.NewReader(`{"title":"T","content":"C","version":1}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("If-Match", "1")
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: noteID.String()}}
	h.UpdateNote(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("status %d want 409", w.Code)
	}
}

func TestHandler_DeleteNote_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubNoteRepo{}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/notes/invalid-id", nil)
	c.Set("user_id", testUserID.String())
	h.DeleteNote(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_GetVersions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	noteID := uuid.New()
	repo := &stubNoteRepo{
		getNote:  &Note{ID: noteID, OwnerID: testUserID, CreatedAt: now, UpdatedAt: types.NullTime{Time: now, Valid: true}},
		versions: []*NoteVersion{{ID: uuid.New(), Version: 1, CreatedAt: now}, {ID: uuid.New(), Version: 2, CreatedAt: now}},
	}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String()+"/versions", nil)
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: noteID.String()}}
	h.GetVersions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var body response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body.Data.(map[string]interface{})
	items, _ := data["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("versions count %d", len(items))
	}
}

func TestHandler_RestoreVersion_InvalidVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubNoteRepo{}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/notes/"+uuid.New().String()+"/versions/abc/restore", nil)
	c.Set("user_id", testUserID.String())
	h.RestoreVersion(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_CreateNote_EmptyTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	noteID := uuid.New()
	now := time.Now()
	repo := &stubNoteRepo{createNote: &Note{ID: noteID, Title: "Untitled", Content: "body", Version: 1, CreatedAt: now}}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(`{"content":"body"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	h.CreateNote(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestHandler_DeleteNote_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	noteID := uuid.New()
	repo := &stubNoteRepo{
		getNote:   &Note{ID: noteID, OwnerID: testUserID, CreatedAt: time.Now(), UpdatedAt: types.NullTime{Time: time.Now(), Valid: true}},
		deleteErr: ErrNoteNotFound,
	}
	svc := NewNoteService(repo, nil)
	h := NewHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/notes/"+noteID.String(), nil)
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: noteID.String()}}
	h.DeleteNote(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Fatal("short string should not be truncated")
	}
	if truncate("this is a longer string", 10) != "this is a ..." {
		t.Fatal("long string should be truncated")
	}
}

func TestNoteService_Create_DefaultTitle(t *testing.T) {
	repo := &stubNoteRepo{createNote: &Note{ID: uuid.New(), Title: "Untitled", Version: 1, CreatedAt: time.Now()}}
	svc := NewNoteService(repo, nil)

	note, err := svc.Create(context.Background(), testUserID, "", "content", nil)
	if err != nil {
		t.Fatal(err)
	}
	if note.Title != "Untitled" {
		t.Fatalf("title want 'Untitled' got %s", note.Title)
	}
}

func TestNoteService_GetByID_WrongOwner(t *testing.T) {
	noteID := uuid.New()
	otherUser := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	repo := &stubNoteRepo{getNote: &Note{ID: noteID, OwnerID: otherUser, CreatedAt: time.Now(), UpdatedAt: types.NullTime{Time: time.Now(), Valid: true}}}
	svc := NewNoteService(repo, nil)

	_, err := svc.GetByID(context.Background(), testUserID, noteID)
	if err != ErrPermissionDenied {
		t.Fatalf("want ErrPermissionDenied got %v", err)
	}
}

func TestNoteService_List_PageDefaults(t *testing.T) {
	repo := &stubNoteRepo{listNotes: []*Note{}, total: 0}
	svc := NewNoteService(repo, nil)

	notes, total, err := svc.List(context.Background(), testUserID, nil, 0, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("want 0 notes got %d", len(notes))
	}
	if total != 0 {
		t.Fatalf("want 0 total got %d", total)
	}
}

func TestConflictError(t *testing.T) {
	err := &ConflictError{CurrentVersion: 5}
	if err.Error() != "version conflict" {
		t.Fatalf("want 'version conflict' got %s", err.Error())
	}
	if err.CurrentVersion != 5 {
		t.Fatalf("want version 5 got %d", err.CurrentVersion)
	}
}
