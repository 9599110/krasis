package note

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
	"go.uber.org/zap"
)

var (
	ErrNoteNotFound     = errors.New("note not found")
	ErrNoteDeleted      = errors.New("note has been deleted")
	ErrVersionConflict  = errors.New("version conflict")
	ErrPermissionDenied = errors.New("permission denied")
)

type Note struct {
	ID          uuid.UUID     `json:"id"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	ContentHTML sql.NullString `json:"content_html,omitempty"`
	OwnerID     uuid.UUID     `json:"owner_id"`
	FolderID    types.NullUUID `json:"folder_id"`
	Version     int           `json:"version"`
	IsPublic    bool          `json:"is_public"`
	IsDeleted   bool          `json:"-"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   types.NullTime `json:"updated_at"`
}

type NoteVersion struct {
	ID            uuid.UUID     `json:"id"`
	NoteID        uuid.UUID     `json:"note_id"`
	Title         sql.NullString `json:"title"`
	Content       sql.NullString `json:"content,omitempty"`
	ContentHTML   sql.NullString `json:"content_html,omitempty"`
	Version       int           `json:"version"`
	ChangedBy     types.NullUUID `json:"changed_by,omitempty"`
	ChangeSummary sql.NullString `json:"change_summary,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

type NoteRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewNoteRepository(pool *pgxpool.Pool, logger *zap.Logger) *NoteRepository {
	return &NoteRepository{pool: pool, logger: logger}
}

func (r *NoteRepository) Create(ctx context.Context, ownerID uuid.UUID, title, content string, folderID *uuid.UUID) (*Note, error) {
	var note Note
	query := `
		INSERT INTO notes (owner_id, title, content, folder_id, version)
		VALUES ($1, $2, $3, $4, 1)
		RETURNING id, title, content, owner_id, folder_id, version, created_at, updated_at
	`

	var fid sql.NullString
	if folderID != nil {
		fid = sql.NullString{String: folderID.String(), Valid: true}
	}

	err := r.pool.QueryRow(ctx, query, ownerID, title, content, fid).Scan(
		&note.ID, &note.Title, &note.Content, &note.OwnerID, &note.FolderID,
		&note.Version, &note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}

	return &note, nil
}

func (r *NoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*Note, error) {
	var note Note
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, content, content_html, owner_id, folder_id, version,
				is_public, is_deleted, created_at, updated_at
		 FROM notes WHERE id = $1`,
		id,
	).Scan(
		&note.ID, &note.Title, &note.Content, &note.ContentHTML, &note.OwnerID,
		&note.FolderID, &note.Version, &note.IsPublic, &note.IsDeleted,
		&note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, page, size int, sort, order string) ([]*Note, int64, error) {
	where := "WHERE owner_id = $1 AND is_deleted = false"
	args := []interface{}{ownerID}
	argIdx := 2

	if folderID != nil {
		where += fmt.Sprintf(" AND folder_id = $%d", argIdx)
		args = append(args, folderID)
		argIdx++
	}

	// Count
	var total int64
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM notes "+where, args...,
	).Scan(&total)
	if err != nil {
		r.logger.Error("ListByOwner count failed",
			zap.Error(err),
			zap.String("owner_id", ownerID.String()),
		)
		return nil, 0, err
	}

	// Sort validation — whitelist to prevent SQL injection
	allowedSort := map[string]bool{"title": true, "created_at": true, "updated_at": true}
	if !allowedSort[sort] {
		sort = "updated_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	offset := (page - 1) * size
	args = append(args, size, offset)
	query := fmt.Sprintf(`
		SELECT id, title, content, owner_id, folder_id, version,
			   is_public, created_at, updated_at
		FROM notes %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sort, order, argIdx, argIdx+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("ListByOwner query failed",
			zap.String("query", query),
			zap.Error(err),
			zap.String("owner_id", ownerID.String()),
			zap.Int("page", page),
			zap.Int("size", size),
		)
		return nil, 0, err
	}
	defer rows.Close()

	notes := make([]*Note, 0, size)
	for rows.Next() {
		var n Note
		var content sql.NullString
		if err := rows.Scan(
			&n.ID, &n.Title, &content, &n.OwnerID, &n.FolderID,
			&n.Version, &n.IsPublic, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if content.Valid {
			n.Content = content.String
		}
		notes = append(notes, &n)
	}

	return notes, total, nil
}

func (r *NoteRepository) UpdateWithOptimisticLock(ctx context.Context, id uuid.UUID, title, content string, currentVersion int) (*Note, error) {
	var note Note
	err := r.pool.QueryRow(ctx, `
		UPDATE notes
		SET title = $1, content = $2, version = version + 1, updated_at = NOW()
		WHERE id = $3 AND version = $4 AND is_deleted = false
		RETURNING id, title, content, owner_id, folder_id, version, created_at, updated_at
	`, title, content, id, currentVersion).Scan(
		&note.ID, &note.Title, &note.Content, &note.OwnerID, &note.FolderID,
		&note.Version, &note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			// Check if note exists but version mismatch
			existing, _ := r.GetByID(ctx, id)
			if existing != nil {
				return nil, ErrVersionConflict
			}
			return nil, ErrNoteNotFound
		}
		return nil, err
	}
	return &note, nil
}

func (r *NoteRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE notes SET is_deleted = true, deleted_at = NOW() WHERE id = $1 AND is_deleted = false",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func (r *NoteRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func (r *NoteRepository) SaveVersion(ctx context.Context, noteID, changedBy uuid.UUID, title, content string, version int, summary string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_versions (note_id, title, content, version, changed_by, change_summary)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, noteID, title, content, version, changedBy, summary)
	return err
}

func (r *NoteRepository) GetVersions(ctx context.Context, noteID uuid.UUID) ([]*NoteVersion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, title, content, content_html, version, changed_by,
			   change_summary, created_at
		FROM note_versions WHERE note_id = $1
		ORDER BY version DESC
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*NoteVersion
	for rows.Next() {
		var v NoteVersion
		if err := rows.Scan(
			&v.ID, &v.NoteID, &v.Title, &v.Content, &v.ContentHTML,
			&v.Version, &v.ChangedBy, &v.ChangeSummary, &v.CreatedAt,
		); err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}
	return versions, nil
}

func (r *NoteRepository) GetVersion(ctx context.Context, noteID uuid.UUID, version int) (*NoteVersion, error) {
	var v NoteVersion
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, title, content, content_html, version, changed_by,
			   change_summary, created_at
		FROM note_versions WHERE note_id = $1 AND version = $2
	`, noteID, version).Scan(
		&v.ID, &v.NoteID, &v.Title, &v.Content, &v.ContentHTML,
		&v.Version, &v.ChangedBy, &v.ChangeSummary, &v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *NoteRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notes WHERE is_deleted = false").Scan(&count)
	return count, err
}

func (r *NoteRepository) CountToday(ctx context.Context, action string) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM notes WHERE DATE(created_at) = CURRENT_DATE"
	if action == "updated" {
		query = "SELECT COUNT(*) FROM notes WHERE DATE(updated_at) = CURRENT_DATE"
	}
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *NoteRepository) TotalStorageUsed(ctx context.Context) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COALESCE(SUM(file_size), 0) FROM files").Scan(&total)
	return total, err
}

func (r *NoteRepository) RestoreVersion(ctx context.Context, noteID uuid.UUID, version int, title, content string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var newVersion int
	err = tx.QueryRow(ctx, `
		UPDATE notes SET title = $1, content = $2, version = version + 1, updated_at = NOW()
		WHERE id = $3 AND is_deleted = false
		RETURNING version
	`, title, content, noteID).Scan(&newVersion)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO note_versions (note_id, title, content, version, change_summary)
		VALUES ($1, $2, $3, $4, '版本回滚')
	`, noteID, title, content, newVersion)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
