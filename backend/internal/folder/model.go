package folder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
)

var (
	ErrFolderNotFound     = errors.New("folder not found")
	ErrPermissionDenied   = errors.New("permission denied")
)

type Folder struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	ParentID  types.NullUUID `json:"parent_id"`
	OwnerID   uuid.UUID     `json:"owner_id"`
	Color     sql.NullString `json:"color"`
	SortOrder int           `json:"sort_order"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt types.NullTime `json:"updated_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string) (*Folder, error) {
	var f Folder
	var parentIDVal interface{}
	if parentID != nil {
		parentIDVal = parentID
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO folders (owner_id, name, parent_id, color)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, parent_id, owner_id, color, sort_order, created_at, updated_at
	`, ownerID, name, parentIDVal, color).Scan(
		&f.ID, &f.Name, &f.ParentID, &f.OwnerID, &f.Color,
		&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	return &f, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Folder, error) {
	var f Folder
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, parent_id, owner_id, color, sort_order, created_at, updated_at
		FROM folders WHERE id = $1
	`, id).Scan(
		&f.ID, &f.Name, &f.ParentID, &f.OwnerID, &f.Color,
		&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, ErrFolderNotFound
	}
	return &f, nil
}

func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*Folder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, parent_id, owner_id, color, sort_order, created_at, updated_at
		FROM folders WHERE owner_id = $1
		ORDER BY sort_order, created_at
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(
			&f.ID, &f.Name, &f.ParentID, &f.OwnerID, &f.Color,
			&f.SortOrder, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, nil
}

func (r *Repository) Update(ctx context.Context, id, ownerID uuid.UUID, name string, parentID *uuid.UUID, color string, sortOrder int) error {
	var parentIDVal interface{}
	if parentID != nil {
		parentIDVal = parentID
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE folders SET name = $1, parent_id = $2, color = $3, sort_order = $4, updated_at = NOW()
		WHERE id = $5 AND owner_id = $6
	`, name, parentIDVal, color, sortOrder, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFolderNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM folders WHERE id = $1 AND owner_id = $2",
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFolderNotFound
	}
	return nil
}
