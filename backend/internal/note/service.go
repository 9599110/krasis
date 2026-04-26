package note

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// NoteIndexer is the interface for RAG note indexing (defined here to avoid circular dep)
type NoteIndexer interface {
	IndexNote(ctx context.Context, noteID, title, content, userID string) error
	DeleteNoteIndex(ctx context.Context, noteID string) error
}

// NoteRepositoryInterface abstracts the database layer for testing.
type NoteRepositoryInterface interface {
	Create(ctx context.Context, ownerID uuid.UUID, title, content string, folderID *uuid.UUID) (*Note, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Note, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, page, size int, sort, order string) ([]*Note, int64, error)
	UpdateWithOptimisticLock(ctx context.Context, id uuid.UUID, title, content string, version int) (*Note, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	PermanentDelete(ctx context.Context, id uuid.UUID) error
	SaveVersion(ctx context.Context, noteID, changedBy uuid.UUID, title, content string, version int, summary string) error
	GetVersions(ctx context.Context, noteID uuid.UUID) ([]*NoteVersion, error)
	GetVersion(ctx context.Context, noteID uuid.UUID, version int) (*NoteVersion, error)
	RestoreVersion(ctx context.Context, noteID uuid.UUID, version int, title, content string) error
	Count(ctx context.Context) (int64, error)
	CountToday(ctx context.Context, action string) (int64, error)
	TotalStorageUsed(ctx context.Context) (int64, error)
}

type NoteService struct {
	repo       NoteRepositoryInterface
	indexer    NoteIndexer
}

func NewNoteService(repo NoteRepositoryInterface, indexer NoteIndexer) *NoteService {
	return &NoteService{repo: repo, indexer: indexer}
}

func (s *NoteService) Create(ctx context.Context, ownerID uuid.UUID, title, content string, folderID *uuid.UUID) (*Note, error) {
	if strings.TrimSpace(title) == "" {
		title = "Untitled"
	}

	note, err := s.repo.Create(ctx, ownerID, title, content, folderID)
	if err != nil {
		return nil, err
	}

	// Save initial version
	if err := s.repo.SaveVersion(ctx, note.ID, ownerID, note.Title, note.Content, 1, "创建笔记"); err != nil {
		return note, fmt.Errorf("save version: %w", err)
	}

	// Index for RAG search (async)
	if s.indexer != nil {
		go s.indexer.IndexNote(context.Background(), note.ID.String(), note.Title, note.Content, ownerID.String())
	}

	return note, nil
}

func (s *NoteService) GetByID(ctx context.Context, userID, noteID uuid.UUID) (*Note, error) {
	note, err := s.repo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note.OwnerID != userID {
		return nil, ErrPermissionDenied
	}
	if note.IsDeleted {
		return nil, ErrNoteDeleted
	}
	return note, nil
}

func (s *NoteService) List(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID, page, size int, sort, order string) ([]*Note, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListByOwner(ctx, userID, folderID, page, size, sort, order)
}

func (s *NoteService) Update(ctx context.Context, userID, noteID uuid.UUID, title, content string, version int) (*Note, error) {
	note, err := s.repo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note.OwnerID != userID {
		return nil, ErrPermissionDenied
	}
	if note.IsDeleted {
		return nil, ErrNoteDeleted
	}

	updatedNote, err := s.repo.UpdateWithOptimisticLock(ctx, noteID, title, content, version)
	if err != nil {
		if err == ErrVersionConflict {
			latest, _ := s.repo.GetByID(ctx, noteID)
			return nil, &ConflictError{
				CurrentVersion: latest.Version,
				Note:           latest,
			}
		}
		return nil, err
	}

	// Save version history asynchronously
	go s.repo.SaveVersion(context.Background(), noteID, userID, updatedNote.Title, updatedNote.Content, updatedNote.Version, "内容更新")

	// Re-index for RAG search (async)
	if s.indexer != nil {
		go s.indexer.IndexNote(context.Background(), noteID.String(), updatedNote.Title, updatedNote.Content, userID.String())
	}

	return updatedNote, nil
}

func (s *NoteService) Delete(ctx context.Context, userID, noteID uuid.UUID, permanent bool) error {
	note, err := s.repo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}
	if note.OwnerID != userID {
		return ErrPermissionDenied
	}

	if permanent {
		if s.indexer != nil {
			go s.indexer.DeleteNoteIndex(context.Background(), noteID.String())
		}
		return s.repo.PermanentDelete(ctx, noteID)
	}
	if s.indexer != nil {
		go s.indexer.DeleteNoteIndex(context.Background(), noteID.String())
	}
	return s.repo.SoftDelete(ctx, noteID)
}

func (s *NoteService) GetVersions(ctx context.Context, userID, noteID uuid.UUID) ([]*NoteVersion, error) {
	note, err := s.repo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note.OwnerID != userID {
		return nil, ErrPermissionDenied
	}
	return s.repo.GetVersions(ctx, noteID)
}

func (s *NoteService) RestoreVersion(ctx context.Context, userID, noteID uuid.UUID, version int) error {
	note, err := s.repo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}
	if note.OwnerID != userID {
		return ErrPermissionDenied
	}

	v, err := s.repo.GetVersion(ctx, noteID, version)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	title := ""
	if v.Title.Valid {
		title = v.Title.String
	}
	content := ""
	if v.Content.Valid {
		content = v.Content.String
	}

	if err := s.repo.RestoreVersion(ctx, noteID, version, title, content); err != nil {
		return err
	}

	// Re-index for RAG search (async)
	if s.indexer != nil {
		go s.indexer.IndexNote(context.Background(), noteID.String(), title, content, userID.String())
	}
	return nil
}

type ConflictError struct {
	CurrentVersion int
	Note           *Note
}

func (e *ConflictError) Error() string {
	return "version conflict"
}
