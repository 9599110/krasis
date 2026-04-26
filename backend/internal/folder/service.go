package folder

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface abstracts the database layer for testing.
type RepositoryInterface interface {
	Create(ctx context.Context, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string) (*Folder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Folder, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*Folder, error)
	Update(ctx context.Context, id, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string, sortOrder int) error
	Delete(ctx context.Context, id, ownerID uuid.UUID) error
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string) (*Folder, error) {
	return s.repo.Create(ctx, ownerID, name, parentID, color)
}

func (s *Service) List(ctx context.Context, ownerID uuid.UUID) ([]*Folder, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}

func (s *Service) Update(ctx context.Context, ownerID uuid.UUID, id uuid.UUID, name string, parentID *uuid.UUID, color string, sortOrder int) error {
	return s.repo.Update(ctx, id, ownerID, name, parentID, color, sortOrder)
}

func (s *Service) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	return s.repo.Delete(ctx, id, ownerID)
}
