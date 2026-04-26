package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	PasswordHash *string    `json:"-"`
	AvatarURL    string     `json:"avatar_url"`
	Status       int16      `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    types.NullTime `json:"updated_at,omitempty"`
}

type UserOAuth struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               uuid.UUID  `json:"user_id"`
	Provider             string     `json:"provider"`
	ProviderUserID       string     `json:"provider_user_id"`
	ProviderAccessToken  string     `json:"-"`
	ProviderRefreshToken string     `json:"-"`
	TokenExpiresAt       sql.NullTime   `json:"-"`
	CreatedAt            time.Time      `json:"-"`
	UpdatedAt            sql.NullTime   `json:"-"`
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		"SELECT id, email, username, password_hash, avatar_url, status, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		"SELECT id, email, username, password_hash, avatar_url, status, created_at, updated_at FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, email, username, avatarURL string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		"INSERT INTO users (email, username, avatar_url) VALUES ($1, $2, $3) RETURNING id, email, username, password_hash, avatar_url, status, created_at, updated_at",
		email, username, avatarURL,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) CreateWithPasswordHash(ctx context.Context, email, username, passwordHash, avatarURL string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		"INSERT INTO users (email, username, password_hash, avatar_url) VALUES ($1, $2, $3, $4) RETURNING id, email, username, password_hash, avatar_url, status, created_at, updated_at",
		email, username, passwordHash, avatarURL,
	).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, username, avatarURL string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET username = $1, avatar_url = $2, updated_at = NOW() WHERE id = $3",
		username, avatarURL, id,
	)
	return err
}

func (r *UserRepository) GetRole(ctx context.Context, userID uuid.UUID) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx,
		`SELECT r.name FROM roles r
		 JOIN user_roles ur ON r.id = ur.role_id
		 WHERE ur.user_id = $1
		 ORDER BY r.name ASC
		 LIMIT 1`,
		userID,
	).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "member", nil
		}
		return "", err
	}
	return role, nil
}

func (r *UserRepository) AssignDefaultRole(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1, id FROM roles WHERE name = 'member'
		 ON CONFLICT DO NOTHING`,
		userID,
	)
	return err
}

type UserWithRole struct {
	User
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"last_login_at,omitempty"`
}

func (r *UserRepository) ListUsers(ctx context.Context, keyword, role string, page, size int) ([]*UserWithRole, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if keyword != "" {
		where += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	if role != "" {
		where += fmt.Sprintf(" AND r.name = $%d", argIdx)
		args = append(args, role)
		argIdx++
	}

	var total int64
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		%s
	`, where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	args = append(args, size, offset)

	query := fmt.Sprintf(`
		SELECT u.id, u.email, u.username, u.avatar_url, u.status, u.created_at, u.updated_at,
			   COALESCE(r.name, 'member') AS role
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		%s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*UserWithRole
	for rows.Next() {
		var u UserWithRole
		if err := rows.Scan(
			&u.ID, &u.Email, &u.Username, &u.AvatarURL, &u.Status,
			&u.CreatedAt, &u.UpdatedAt, &u.Role,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}

	return users, total, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_roles SET role_id = (SELECT id FROM roles WHERE name = $1)
		WHERE user_id = $2
	`, roleName, userID)
	return err
}

func (r *UserRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, status int16) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2",
		status, userID,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	return err
}

func (r *UserRepository) BatchUpdateStatus(ctx context.Context, userIDs []uuid.UUID, status int16) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET status = $1, updated_at = NOW() WHERE id = ANY($2)",
		status, userIDs,
	)
	return err
}

func (r *UserRepository) GetGroupID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var groupID uuid.UUID
	err := r.pool.QueryRow(ctx,
		"SELECT group_id FROM users WHERE id = $1", userID,
	).Scan(&groupID)
	if err != nil {
		return uuid.Nil, err
	}
	return groupID, nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
