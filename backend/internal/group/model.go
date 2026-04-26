package group

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Group struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsDefault   bool       `json:"is_default"`
	UserCount   int64      `json:"user_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type GroupFeature struct {
	ID           uuid.UUID       `json:"id"`
	GroupID      uuid.UUID       `json:"group_id"`
	FeatureKey   string          `json:"feature_key"`
	FeatureValue json.RawMessage `json:"feature_value"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context) ([]Group, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.name, g.description, g.is_default, g.created_at, g.updated_at,
			   COUNT(u.id) AS user_count
		FROM groups g
		LEFT JOIN users u ON u.group_id = g.id
		GROUP BY g.id
		ORDER BY g.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		var updatedAt sql.NullTime
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.IsDefault, &g.CreatedAt, &updatedAt, &g.UserCount); err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			g.UpdatedAt = &updatedAt.Time
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []Group{}
	}
	return groups, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Group, error) {
	var g Group
	var updatedAt sql.NullTime
	var userCount int64
	err := r.pool.QueryRow(ctx, `
		SELECT g.id, g.name, g.description, g.is_default, g.created_at, g.updated_at,
			   COUNT(u.id) AS user_count
		FROM groups g
		LEFT JOIN users u ON u.group_id = g.id
		WHERE g.id = $1
		GROUP BY g.id
	`, id).Scan(&g.ID, &g.Name, &g.Description, &g.IsDefault, &g.CreatedAt, &updatedAt, &userCount)
	if err != nil {
		return nil, err
	}
	g.UserCount = userCount
	if updatedAt.Valid {
		g.UpdatedAt = &updatedAt.Time
	}
	return &g, nil
}

func (r *Repository) Create(ctx context.Context, name, description string) (*Group, error) {
	g := &Group{ID: uuid.New(), Name: name, Description: description}
	err := r.pool.QueryRow(ctx,
		"INSERT INTO groups (id, name, description) VALUES ($1, $2, $3) RETURNING created_at",
		g.ID, g.Name, g.Description).Scan(&g.CreatedAt)
	return g, err
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, name, description string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE groups SET name = $1, description = $2, updated_at = NOW() WHERE id = $3",
		name, description, id)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM groups WHERE id = $1 AND is_default = false", id)
	return err
}

func (r *Repository) GetFeatures(ctx context.Context, groupID uuid.UUID) ([]GroupFeature, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, group_id, feature_key, feature_value, updated_at FROM group_features WHERE group_id = $1", groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []GroupFeature
	for rows.Next() {
		var f GroupFeature
		if err := rows.Scan(&f.ID, &f.GroupID, &f.FeatureKey, &f.FeatureValue, &f.UpdatedAt); err != nil {
			return nil, err
		}
		features = append(features, f)
	}
	if features == nil {
		features = []GroupFeature{}
	}
	return features, nil
}

func (r *Repository) UpdateFeatures(ctx context.Context, groupID uuid.UUID, features map[string]json.RawMessage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for key, value := range features {
		_, err := tx.Exec(ctx, `
			INSERT INTO group_features (group_id, feature_key, feature_value)
			VALUES ($1, $2, $3)
			ON CONFLICT (group_id, feature_key) DO UPDATE SET feature_value = $3, updated_at = NOW()
		`, groupID, key, value)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
