package share

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/krasis/krasis/internal/note"
)

type Service struct {
	shareRepo *ShareRepository
	noteRepo  *note.NoteRepository
}

func NewService(shareRepo *ShareRepository, noteRepo *note.NoteRepository) *Service {
	return &Service{
		shareRepo: shareRepo,
		noteRepo:  noteRepo,
	}
}

type CreateShareRequest struct {
	ShareType  string     `json:"share_type"`
	Permission string     `json:"permission"`
	Password   string     `json:"password"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func (s *Service) CreateShare(ctx context.Context, userID, noteID uuid.UUID, req *CreateShareRequest) (*NoteShare, error) {
	noteItem, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if noteItem.OwnerID != userID {
		return nil, note.ErrPermissionDenied
	}

	// Check existing share
	existing, err := s.shareRepo.GetByNoteID(ctx, noteID)
	if err == nil && existing.Status != "rejected" {
		return nil, ErrShareExists
	}

	passwordHash := sql.NullString{}
	if req.Password != "" {
		h, err := hashPassword(req.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		passwordHash = sql.NullString{String: h, Valid: true}
	}

	expiresAt := sql.NullTime{}
	if req.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *req.ExpiresAt, Valid: true}
	}

	share := &NoteShare{
		NoteID:          noteID,
		ShareToken:      generateShareToken(),
		ShareType:       req.ShareType,
		Permission:      req.Permission,
		PasswordHash:    passwordHash,
		ExpiresAt:       expiresAt,
		Status:          "pending",
		ContentSnapshot: sql.NullString{String: noteItem.Content, Valid: true},
		CreatedBy:       userID,
	}

	if share.ShareType == "" {
		share.ShareType = "link"
	}
	if share.Permission == "" {
		share.Permission = "read"
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}

	return share, nil
}

func (s *Service) GetShareStatus(ctx context.Context, userID, noteID uuid.UUID) (*NoteShare, error) {
	noteItem, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if noteItem.OwnerID != userID {
		return nil, note.ErrPermissionDenied
	}

	return s.shareRepo.GetByNoteID(ctx, noteID)
}

func (s *Service) AccessShare(ctx context.Context, token, password string) (*note.Note, string, error) {
	share, err := s.shareRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, "", err
	}

	// Check status
	switch share.Status {
	case "pending":
		return nil, "", ErrSharePending
	case "rejected":
		return nil, "", ErrShareRejected
	}

	// Check expiry
	if share.ExpiresAt.Valid && share.ExpiresAt.Time.Before(time.Now()) {
		return nil, "", ErrShareExpired
	}

	// Check password
	if share.PasswordHash.Valid {
		if password == "" || !verifyPassword(share.PasswordHash.String, password) {
			return nil, "", ErrInvalidPassword
		}
	}

	noteItem, err := s.noteRepo.GetByID(ctx, share.NoteID)
	if err != nil {
		return nil, "", err
	}

	return noteItem, share.Permission, nil
}

func (s *Service) DeleteShare(ctx context.Context, userID, noteID uuid.UUID) error {
	share, err := s.shareRepo.GetByNoteID(ctx, noteID)
	if err != nil {
		return err
	}

	noteItem, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}
	if noteItem.OwnerID != userID {
		return note.ErrPermissionDenied
	}

	return s.shareRepo.Delete(ctx, share.ID)
}

func (s *Service) Approve(ctx context.Context, shareID, reviewerID uuid.UUID) error {
	return s.shareRepo.UpdateStatus(ctx, shareID, "approved", reviewerID, "")
}

func (s *Service) Reject(ctx context.Context, shareID, reviewerID uuid.UUID, reason string) error {
	return s.shareRepo.UpdateStatus(ctx, shareID, "rejected", reviewerID, reason)
}

func (s *Service) ReReview(ctx context.Context, shareID, reviewerID uuid.UUID) error {
	return s.shareRepo.UpdateStatus(ctx, shareID, "pending", reviewerID, "")
}

func (s *Service) Revoke(ctx context.Context, shareID, reviewerID uuid.UUID) error {
	return s.shareRepo.UpdateStatus(ctx, shareID, "revoked", reviewerID, "")
}

// Admin methods

func (s *Service) ListShares(ctx context.Context, status, keyword string, page, size int) ([]AdminShareListItem, int64, error) {
	return s.shareRepo.ListShares(ctx, status, keyword, page, size)
}

func (s *Service) GetShareDetail(ctx context.Context, shareID uuid.UUID) (*AdminShareListItem, error) {
	return s.shareRepo.GetShareDetail(ctx, shareID)
}

func (s *Service) GetShareStats(ctx context.Context) (*ShareStats, error) {
	return s.shareRepo.GetShareStats(ctx)
}

func (s *Service) BatchReview(ctx context.Context, shareIDs []uuid.UUID, reviewerID uuid.UUID, action, reason string) error {
	status := "approved"
	if action == "reject" {
		status = "rejected"
	}
	return s.shareRepo.BatchUpdateStatus(ctx, shareIDs, status, reviewerID, reason)
}
