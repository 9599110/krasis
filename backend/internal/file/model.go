package file

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
)

var ErrFileNotFound = errors.New("file not found")

type File struct {
	ID            uuid.UUID      `json:"id"`
	NoteID        types.NullUUID  `json:"note_id"`
	UserID        uuid.UUID      `json:"user_id"`
	FileName      string         `json:"file_name"`
	FileType      sql.NullString `json:"file_type"`
	MimeType      sql.NullString `json:"mime_type"`
	StoragePath   string         `json:"storage_path"`
	Bucket        string         `json:"bucket"`
	SizeBytes     sql.NullInt64  `json:"size_bytes"`
	Width         sql.NullInt32  `json:"width"`
	Height        sql.NullInt32  `json:"height"`
	DurationSec   sql.NullFloat64 `json:"duration_sec"`
	ThumbnailPath sql.NullString `json:"thumbnail_url"`
	Metadata      map[string]interface{} `json:"metadata"`
	Status        int16          `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
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
		INSERT INTO files (id, note_id, user_id, file_name, file_type, mime_type,
						   storage_path, bucket, size_bytes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0)
		RETURNING created_at
	`, file.ID, file.NoteID, file.UserID, file.FileName, file.FileType,
		file.MimeType, file.StoragePath, file.Bucket, file.SizeBytes,
	).Scan(&file.CreatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*File, error) {
	var f File
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, user_id, file_name, file_type, mime_type,
			   storage_path, bucket, size_bytes, width, height, duration_sec,
			   thumbnail_path, metadata, status, created_at
		FROM files WHERE id = $1
	`, id).Scan(
		&f.ID, &f.NoteID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
		&f.StoragePath, &f.Bucket, &f.SizeBytes, &f.Width, &f.Height,
		&f.DurationSec, &f.ThumbnailPath, &f.Metadata, &f.Status, &f.CreatedAt,
	)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return &f, nil
}

func (r *Repository) ListByNote(ctx context.Context, noteID uuid.UUID) ([]*File, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, user_id, file_name, file_type, mime_type,
			   storage_path, bucket, size_bytes, thumbnail_path, status, created_at
		FROM files WHERE note_id = $1 AND status = 1
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
			&f.ID, &f.NoteID, &f.UserID, &f.FileName, &f.FileType, &f.MimeType,
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
