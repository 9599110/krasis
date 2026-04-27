package file

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
)

type stubFileService struct {
	presignResult *PresignResult
	presignErr    error
	confirmErr    error
	deleteErr     error
	deleteID      uuid.UUID
}

func (s *stubFileService) GeneratePresignURL(ctx context.Context, userID uuid.UUID, fileName, fileType string, noteID *uuid.UUID) (*PresignResult, error) {
	if s.presignErr != nil {
		return nil, s.presignErr
	}
	if s.presignResult != nil {
		return s.presignResult, nil
	}
	return &PresignResult{
		FileID:    uuid.New().String(),
		UploadURL: "https://minio.example.com/presigned",
		ExpiresIn: 3600,
	}, nil
}

func (s *stubFileService) ConfirmUpload(ctx context.Context, fileID uuid.UUID) error {
	return s.confirmErr
}

func (s *stubFileService) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	s.deleteID = fileID
	return s.deleteErr
}

func (s *stubFileService) ListByNote(ctx context.Context, noteID uuid.UUID) ([]*File, error) {
	return nil, nil
}

func TestHandler_GetPresignURL_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{
		presignResult: &PresignResult{
			FileID:    uuid.New().String(),
			UploadURL: "https://minio.example.com/presigned",
			ExpiresIn: 3600,
		},
	}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/presign?file_name=test.png&file_type=image/png", nil)
	c.Set("user_id", uuid.New().String())
	h.GetPresignURL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	data := resp["data"].(map[string]interface{})
	if data["upload_url"] != "https://minio.example.com/presigned" {
		t.Fatalf("upload_url mismatch")
	}
}

func TestHandler_GetPresignURL_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/presign", nil)
	c.Set("user_id", uuid.New().String())
	h.GetPresignURL(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_GetPresignURL_WithNoteID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	noteID := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/presign?file_name=test.png&file_type=image/png&note_id="+noteID.String(), nil)
	c.Set("user_id", uuid.New().String())
	h.GetPresignURL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
}

func TestHandler_ConfirmUpload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	fileID := uuid.New()
	body := `{"file_id":"` + fileID.String() + `"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/files/confirm", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ConfirmUpload(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
}

func TestHandler_ConfirmUpload_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/files/confirm", strings.NewReader(`invalid`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ConfirmUpload(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_ConfirmUpload_InvalidFileID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	body := `{"file_id":"not-a-uuid"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/files/confirm", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ConfirmUpload(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_ConfirmUpload_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{confirmErr: ErrFileNotFound}
	h := NewHandler(svc)

	fileID := uuid.New()
	body := `{"file_id":"` + fileID.String() + `"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/files/confirm", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ConfirmUpload(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d want 500", w.Code)
	}
}

func TestHandler_DeleteFile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	fileID := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/files/"+fileID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: fileID.String()}}
	h.DeleteFile(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d want 200", w.Code)
	}
	if svc.deleteID != fileID {
		t.Fatalf("deleted ID mismatch")
	}
}

func TestHandler_DeleteFile_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/files/not-a-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	h.DeleteFile(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d want 400", w.Code)
	}
}

func TestHandler_DeleteFile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubFileService{deleteErr: ErrFileNotFound}
	h := NewHandler(svc)

	fileID := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/files/"+fileID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: fileID.String()}}
	h.DeleteFile(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d want 404", w.Code)
	}
}

func TestFile_Model(t *testing.T) {
	file := &File{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		FileName:  "test.png",
		Bucket:    "krasis-files",
		Status:    1,
		CreatedAt: time.Now(),
	}

	if file.FileName != "test.png" {
		t.Fatalf("filename mismatch")
	}
	if file.Status != 1 {
		t.Fatalf("status mismatch")
	}
}

func TestPresignResult_JSON(t *testing.T) {
	result := &PresignResult{
		FileID:    uuid.New().String(),
		UploadURL: "https://minio.example.com/upload",
		ExpiresIn: 3600,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var decoded PresignResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ExpiresIn != 3600 {
		t.Fatalf("expires_in mismatch")
	}
}
