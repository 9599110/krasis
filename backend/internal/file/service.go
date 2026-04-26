package file

import (
	"context"
	"fmt"
	"path/filepath"
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
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio: %w", err)
	}
	return client, nil
}

func (s *Service) GeneratePresignURL(ctx context.Context, userID uuid.UUID, fileName, fileType string, noteID *uuid.UUID) (*PresignResult, error) {
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

	file := &File{
		ID:          uuid.MustParse(fileID),
		NoteID:      noteIDVal,
		UserID:      userID,
		FileName:    fileName,
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

	// Delete from MinIO
	s.minio.RemoveObject(ctx, s.bucket, f.StoragePath, minio.RemoveObjectOptions{})

	return s.repo.Delete(ctx, fileID)
}

func (s *Service) ListByNote(ctx context.Context, noteID uuid.UUID) ([]*File, error) {
	return s.repo.ListByNote(ctx, noteID)
}
