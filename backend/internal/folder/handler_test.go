package folder

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

type folderStub struct {
	list   []*Folder
	get    *Folder
	getErr error
	update error
	del    error
}

func (f *folderStub) Create(ctx context.Context, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string) (*Folder, error) {
	return &Folder{ID: uuid.New(), Name: name, OwnerID: ownerID, CreatedAt: time.Now()}, nil
}
func (f *folderStub) GetByID(ctx context.Context, id uuid.UUID) (*Folder, error) {
	return f.get, f.getErr
}
func (f *folderStub) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*Folder, error) {
	return f.list, nil
}
func (f *folderStub) Update(ctx context.Context, id, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string, sortOrder int) error {
	return f.update
}
func (f *folderStub) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	return f.del
}

var testUserID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

func TestFolderService_Create(t *testing.T) {
	stub := &folderStub{}
	svc := NewService(stub)

	folder, err := svc.Create(context.Background(), testUserID, "Work", nil, "#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	if folder.Name != "Work" {
		t.Fatalf("name want Work got %s", folder.Name)
	}
}

func TestFolderService_List(t *testing.T) {
	stub := &folderStub{list: []*Folder{{ID: uuid.New(), Name: "Test", CreatedAt: time.Now()}}}
	svc := NewService(stub)

	folders, err := svc.List(context.Background(), testUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 {
		t.Fatalf("count want 1 got %d", len(folders))
	}
}

func TestFolderService_Update_NotFound(t *testing.T) {
	stub := &folderStub{update: ErrFolderNotFound}
	svc := NewService(stub)

	err := svc.Update(context.Background(), testUserID, uuid.New(), "New", nil, "", 0)
	if err != ErrFolderNotFound {
		t.Fatalf("want ErrFolderNotFound got %v", err)
	}
}

func TestFolderService_Delete_NotFound(t *testing.T) {
	stub := &folderStub{del: ErrFolderNotFound}
	svc := NewService(stub)

	err := svc.Delete(context.Background(), testUserID, uuid.New())
	if err != ErrFolderNotFound {
		t.Fatalf("want ErrFolderNotFound got %v", err)
	}
}

func TestHandler_ListFolders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &folderStub{list: []*Folder{{ID: uuid.New(), Name: "Work", CreatedAt: time.Now(), UpdatedAt: types.NullTime{Time: time.Now(), Valid: true}}}}
	svc := NewService(stub)
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/folders", nil)
	c.Set("user_id", testUserID.String())
	h.ListFolders(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var body response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body.Data.(map[string]interface{})
	items, _ := data["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items count %d", len(items))
	}
}

func TestHandler_CreateFolder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &folderStub{}
	svc := NewService(stub)
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/folders", strings.NewReader(`{"name":"Work","color":"#ff0000"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	h.CreateFolder(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateFolder_MissingName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &folderStub{}
	svc := NewService(stub)
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/folders", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	h.CreateFolder(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_CreateFolder_InvalidParentID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &folderStub{}
	svc := NewService(stub)
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/folders", strings.NewReader(`{"name":"Test","parent_id":"not-a-uuid"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	h.CreateFolder(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_UpdateFolder_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &folderStub{update: ErrFolderNotFound}
	svc := NewService(stub)
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/folders/"+uuid.New().String(), strings.NewReader(`{"name":"Renamed"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	h.UpdateFolder(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestHandler_DeleteFolder_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &folderStub{del: ErrFolderNotFound}
	svc := NewService(stub)
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/folders/"+uuid.New().String(), nil)
	c.Set("user_id", testUserID.String())
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	h.DeleteFolder(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}
