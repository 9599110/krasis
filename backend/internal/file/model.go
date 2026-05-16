package file

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
)

var ErrFileNotFound = errors.New("file not found")

type File struct {
	ID            uuid.UUID              `json:"id"`
	NoteID        types.NullUUID          `json:"note_id"`
	FolderID      types.NullUUID          `json:"folder_id"`
	UserID        uuid.UUID              `json:"user_id"`
	FileName      string                 `json:"file_name"`
	FileType      types.NullString        `json:"file_type"`
	MimeType      types.NullString        `json:"mime_type"`
	StoragePath   string                 `json:"storage_path"`
	Bucket        string                 `json:"bucket"`
	SizeBytes     types.NullInt64         `json:"size_bytes"`
	Width         types.NullInt32         `json:"width"`
	Height        types.NullInt32         `json:"height"`
	DurationSec   types.NullFloat64       `json:"duration_sec"`
	ThumbnailPath types.NullString        `json:"thumbnail_url"`
	Metadata      map[string]interface{}  `json:"metadata"`
	Status        int16                   `json:"status"`
	IsHidden      bool                    `json:"is_hidden"`
	CreatedAt     time.Time               `json:"created_at"`
}

type PresignResult struct {
	FileID    string `json:"file_id"`
	UploadURL string `json:"upload_url"`
	ExpiresIn int    `json:"expires_in"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, file *File) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO files (id, note_id, folder_id, user_id, file_name, file_type, mime_type,
					       storage_path, bucket, size_bytes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0)
		RETURNING created_at
	`, file.ID, file.NoteID, file.FolderID, file.UserID, file.FileName, file.FileType,
		file.MimeType, file.StoragePath, file.Bucket, file.SizeBytes,
	).Scan(&file.CreatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*File, error) {
	var f File
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, folder_id, user_id, file_name, file_type, mime_type,
		       storage_path, bucket, size_bytes, width, height, duration_sec,
		       thumbnail_path, metadata, status, is_hidden, created_at
		FROM files WHERE id = $1
	`, id).Scan(
		&f.ID, &f.NoteID, &f.FolderID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
		&f.StoragePath, &f.Bucket, &f.SizeBytes, &f.Width, &f.Height,
		&f.DurationSec, &f.ThumbnailPath, &f.Metadata, &f.Status, &f.IsHidden, &f.CreatedAt,
	)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return &f, nil
}

func (r *Repository) ListByNote(ctx context.Context, noteID uuid.UUID) ([]*File, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, folder_id, user_id, file_name, file_type, mime_type,
		       storage_path, bucket, size_bytes, thumbnail_path, status, created_at
		FROM files WHERE note_id = $1 AND status = 1 AND is_hidden = false
		ORDER BY created_at
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(
			&f.ID, &f.NoteID, &f.FolderID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
			&f.StoragePath, &f.Bucket, &f.SizeBytes, &f.ThumbnailPath, &f.Status, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, nil
}

func (r *Repository) ListByFolder(ctx context.Context, folderID uuid.UUID) ([]*File, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, folder_id, user_id, file_name, file_type, mime_type,
		       storage_path, bucket, size_bytes, thumbnail_path, status, created_at
		FROM files WHERE folder_id = $1 AND status = 1 AND is_hidden = false
		ORDER BY created_at DESC
	`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(
			&f.ID, &f.NoteID, &f.FolderID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
			&f.StoragePath, &f.Bucket, &f.SizeBytes, &f.ThumbnailPath, &f.Status, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE files SET status = $1, processed_at = NOW() WHERE id = $2",
		status, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

func (r *Repository) ListHiddenByFolder(ctx context.Context, folderID uuid.UUID) ([]*File, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, folder_id, user_id, file_name, file_type, mime_type,
		       storage_path, bucket, size_bytes, thumbnail_path, status, created_at
		FROM files WHERE folder_id = $1 AND status = 1 AND is_hidden = true
		ORDER BY created_at DESC
	`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(
			&f.ID, &f.NoteID, &f.FolderID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
			&f.StoragePath, &f.Bucket, &f.SizeBytes, &f.ThumbnailPath, &f.Status, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		f.IsHidden = true
		files = append(files, &f)
	}
	return files, nil
}

// UnhideFile sets a file's is_hidden flag from TRUE back to FALSE, making it visible in normal listings.
func (r *Repository) UnhideFile(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE files SET is_hidden = false WHERE id = $1 AND is_hidden = true",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

// MarkHidden sets a file as hidden (is_hidden = true).
func (r *Repository) MarkHidden(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE files SET is_hidden = true WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

// ListImagesByUser returns paginated image files for a user across all folders.
func (r *Repository) ListImagesByUser(ctx context.Context, userID uuid.UUID, page, size int) ([]*File, int64, error) {
	offset := (page - 1) * size

	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM files
		 WHERE user_id = $1 AND status = 1 AND is_hidden = false
		 AND (mime_type LIKE 'image/%' OR file_name ILIKE '%.png' OR file_name ILIKE '%.jpg'
		      OR file_name ILIKE '%.jpeg' OR file_name ILIKE '%.gif' OR file_name ILIKE '%.webp'
		      OR file_name ILIKE '%.bmp' OR file_name ILIKE '%.svg' OR file_name ILIKE '%.avif'
		      OR file_name ILIKE '%.heic' OR file_name ILIKE '%.heif')`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, folder_id, user_id, file_name, file_type, mime_type,
		       storage_path, bucket, size_bytes, width, height, thumbnail_path,
		       status, created_at
		FROM files
		WHERE user_id = $1 AND status = 1 AND is_hidden = false
		  AND (mime_type LIKE 'image/%' OR file_name ILIKE '%.png' OR file_name ILIKE '%.jpg'
		       OR file_name ILIKE '%.jpeg' OR file_name ILIKE '%.gif' OR file_name ILIKE '%.webp'
		       OR file_name ILIKE '%.bmp' OR file_name ILIKE '%.svg' OR file_name ILIKE '%.avif'
		       OR file_name ILIKE '%.heic' OR file_name ILIKE '%.heif')
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(
			&f.ID, &f.NoteID, &f.FolderID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
			&f.StoragePath, &f.Bucket, &f.SizeBytes, &f.Width, &f.Height,
			&f.ThumbnailPath, &f.Status, &f.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		files = append(files, &f)
	}
	return files, total, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM files WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}
