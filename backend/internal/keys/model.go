package keys

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserKey represents a user's encrypted ECC key pair stored on the server.
type UserKey struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	EncryptedPrivateKey string    `json:"encrypted_private_key"` // base64
	Salt                string    `json:"salt"`                  // base64
	IV                  string    `json:"iv"`                    // base64
	PublicKeyRaw        string    `json:"public_key_raw"`        // base64
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Upsert(ctx context.Context, key *UserKey) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_keys (user_id, encrypted_private_key, salt, iv, public_key_raw)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id)
		DO UPDATE SET encrypted_private_key = $2, salt = $3, iv = $4, public_key_raw = $5, updated_at = NOW()
	`, key.UserID, key.EncryptedPrivateKey, key.Salt, key.IV, key.PublicKeyRaw)
	return err
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserKey, error) {
	var k UserKey
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, encrypted_private_key, salt, iv, public_key_raw, created_at, updated_at
		FROM user_keys WHERE user_id = $1
	`, userID).Scan(&k.ID, &k.UserID, &k.EncryptedPrivateKey, &k.Salt, &k.IV, &k.PublicKeyRaw, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *Repository) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM user_keys WHERE user_id = $1", userID)
	return err
}

func (r *Repository) GetPublicKey(ctx context.Context, userID uuid.UUID) (string, error) {
	var pub string
	err := r.pool.QueryRow(ctx, "SELECT public_key_raw FROM user_keys WHERE user_id = $1", userID).Scan(&pub)
	if err != nil {
		return "", err
	}
	return pub, nil
}
