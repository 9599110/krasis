package file

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/krasis/krasis/pkg/types"
)

type Service struct {
	repo     *Repository
	minio    *minio.Client
	bucket   string
	presignTTL time.Duration
}

func NewService(repo *Repository, minioClient *minio.Client, bucket string, presignTTL time.Duration) *Service {
	return &Service{
		repo:     repo,
		minio:    minioClient,
		bucket:   bucket,
		presignTTL: presignTTL,
	}
}

func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	// If no endpoint is configured, return nil client (MinIO is optional)
	if endpoint == "" {
		return nil, nil
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio: %w", err)
	}
	return client, nil
}

func (s *Service) GeneratePresignURL(ctx context.Context, userID uuid.UUID, fileName, fileType string, noteID *uuid.UUID, folderID *uuid.UUID) (*PresignResult, error) {
	if s.minio == nil {
		return nil, fmt.Errorf("MinIO not configured: cannot generate presign URL")
	}
	fileID := uuid.New().String()
	ext := filepath.Ext(fileName)
	objectKey := fmt.Sprintf("uploads/%s/%s%s", fileID, fileID, ext)

	url, err := s.minio.PresignedPutObject(ctx, s.bucket, objectKey, s.presignTTL)
	if err != nil {
		return nil, fmt.Errorf("presign: %w", err)
	}

	var noteIDVal types.NullUUID
	if noteID != nil {
		noteIDVal = types.NullUUID{UUID: *noteID, Valid: true}
	}
	var folderIDVal types.NullUUID
	if folderID != nil {
		folderIDVal = types.NullUUID{UUID: *folderID, Valid: true}
	}

	file := &File{
		ID:          uuid.MustParse(fileID),
		NoteID:      noteIDVal,
		FolderID:    folderIDVal,
		UserID:      userID,
		FileName:    fileName,
		MimeType:    types.NullString{String: fileType, Valid: fileType != ""},
		StoragePath: objectKey,
		Bucket:      s.bucket,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		return nil, fmt.Errorf("create file record: %w", err)
	}

	return &PresignResult{
		FileID:    fileID,
		UploadURL: url.String(),
		ExpiresIn: int(s.presignTTL.Seconds()),
	}, nil
}

func (s *Service) ConfirmUpload(ctx context.Context, fileID uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, fileID, 1)
}

func (s *Service) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	f, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	// Delete from MinIO (if configured)
	if s.minio != nil {
		s.minio.RemoveObject(ctx, s.bucket, f.StoragePath, minio.RemoveObjectOptions{})
	}

	return s.repo.Delete(ctx, fileID)
}

// ListFileResult is the API response for a file in a listing
type ListFileResult struct {
	ID           uuid.UUID              `json:"id"`
	NoteID       types.NullUUID          `json:"note_id"`
	FolderID     types.NullUUID          `json:"folder_id"`
	UserID       uuid.UUID              `json:"user_id"`
	FileName     string                 `json:"file_name"`
	FileType     types.NullString        `json:"file_type"`
	MimeType     types.NullString        `json:"mime_type"`
	StoragePath  string                 `json:"storage_path"`
	Bucket       string                 `json:"bucket"`
	SizeBytes    types.NullInt64         `json:"size_bytes"`
	Width        types.NullInt32         `json:"width"`
	Height       types.NullInt32         `json:"height"`
	ThumbnailURL *string                `json:"thumbnail_url"`
	Metadata     map[string]interface{}  `json:"metadata"`
	Status       int16                   `json:"status"`
	CreatedAt    time.Time               `json:"created_at"`
}

var imageMimePrefixes = []string{"image/"}
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true, ".avif": true,
}

func isImageFile(mimeType string, fileName string) bool {
	for _, p := range imageMimePrefixes {
		if strings.HasPrefix(mimeType, p) {
			return true
		}
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	return imageExts[ext]
}

func (s *Service) toListResult(ctx context.Context, f *File) *ListFileResult {
	r := &ListFileResult{
		ID:          f.ID,
		NoteID:      f.NoteID,
		FolderID:    f.FolderID,
		UserID:      f.UserID,
		FileName:    f.FileName,
		FileType:    f.FileType,
		MimeType:    f.MimeType,
		StoragePath: f.StoragePath,
		Bucket:      f.Bucket,
		SizeBytes:   f.SizeBytes,
		Width:       f.Width,
		Height:      f.Height,
		Metadata:    f.Metadata,
		Status:      f.Status,
		CreatedAt:   f.CreatedAt,
	}
	// For image files, generate a presigned thumbnail URL
	mime := f.MimeType.String
	if mime == "" {
		// try to infer from extension
		if strings.HasPrefix(f.FileName, ".") {
			mime = "image/" + strings.TrimPrefix(f.FileName, ".")
		}
	}
	if s.minio != nil && isImageFile(mime, f.FileName) {
		u, err := s.minio.PresignedGetObject(ctx, s.bucket, f.StoragePath, s.presignTTL, nil)
		if err == nil {
			// Strip query params to keep thumbnail URL shorter; client adds its own if needed
			// But we need the presigned signature to access MinIO directly
			s := u.String()
			r.ThumbnailURL = &s
		}
	}
	return r
}

func (s *Service) ListByNote(ctx context.Context, noteID uuid.UUID) ([]*ListFileResult, error) {
	files, err := s.repo.ListByNote(ctx, noteID)
	if err != nil {
		return nil, err
	}
	results := make([]*ListFileResult, len(files))
	for i, f := range files {
		results[i] = s.toListResult(ctx, f)
	}
	return results, nil
}

func (s *Service) ListByFolder(ctx context.Context, folderID uuid.UUID) ([]*ListFileResult, error) {
	files, err := s.repo.ListByFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	results := make([]*ListFileResult, len(files))
	for i, f := range files {
		results[i] = s.toListResult(ctx, f)
	}
	return results, nil
}

func (s *Service) GenerateGetURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	if s.minio == nil {
		return "", fmt.Errorf("MinIO not configured: cannot generate file URL")
	}
	f, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	url, err := s.minio.PresignedGetObject(ctx, s.bucket, f.StoragePath, s.presignTTL, nil)
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}
	return url.String(), nil
}

// GenerateDownloadURL returns a presigned URL that forces download via Content-Disposition header.
func (s *Service) GenerateDownloadURL(ctx context.Context, fileID uuid.UUID) (string, error) {
	if s.minio == nil {
		return "", fmt.Errorf("MinIO not configured: cannot generate download URL")
	}
	f, err := s.repo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	// Request params to force browser to download instead of preview
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf(`attachment; filename="%s"`, f.FileName))

	url, err := s.minio.PresignedGetObject(ctx, s.bucket, f.StoragePath, s.presignTTL, reqParams)
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}
	return url.String(), nil
}

// ListHiddenByFolder returns files marked as hidden in a folder.
// Hidden files are not returned in normal ListByFolder/ListByNote calls.
func (s *Service) ListHiddenByFolder(ctx context.Context, folderID uuid.UUID) ([]*ListFileResult, error) {
	files, err := s.repo.ListHiddenByFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	results := make([]*ListFileResult, len(files))
	for i, f := range files {
		results[i] = s.toListResult(ctx, f)
	}
	return results, nil
}

// UnhideFile sets a file's is_hidden flag to false, making it visible in normal listings.
func (s *Service) UnhideFile(ctx context.Context, fileID uuid.UUID) error {
	return s.repo.UnhideFile(ctx, fileID)
}

// MarkHidden sets a file as hidden (is_hidden = true), hiding it from normal listings.
func (s *Service) MarkHidden(ctx context.Context, fileID uuid.UUID) error {
	return s.repo.MarkHidden(ctx, fileID)
}

// ListImages returns paginated image files for a user across all folders.
func (s *Service) ListImages(ctx context.Context, userID uuid.UUID, page, size int) ([]*ListFileResult, int64, error) {
	files, total, err := s.repo.ListImagesByUser(ctx, userID, page, size)
	if err != nil {
		return nil, 0, err
	}
	results := make([]*ListFileResult, len(files))
	for i, f := range files {
		results[i] = s.toListResult(ctx, f)
	}
	return results, total, nil
}
