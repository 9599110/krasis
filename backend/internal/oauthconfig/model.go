package oauthconfig

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthProviderConfig struct {
	ID           uuid.UUID       `json:"-"`
	Provider     string          `json:"provider"`
	Enabled      bool            `json:"enabled"`
	ClientID     string          `json:"client_id"`
	ClientSecret string          `json:"client_secret"`
	RedirectURI  string          `json:"redirect_uri"`
	Config       json.RawMessage `json:"config"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetAll(ctx context.Context) (map[string]OAuthProviderConfig, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, provider, enabled, client_id, client_secret, redirect_uri, config, updated_at FROM oauth_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]OAuthProviderConfig)
	for rows.Next() {
		var c OAuthProviderConfig
		if err := rows.Scan(&c.ID, &c.Provider, &c.Enabled, &c.ClientID, &c.ClientSecret, &c.RedirectURI, &c.Config, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result[c.Provider] = c
	}
	return result, nil
}

func (r *Repository) GetByProvider(ctx context.Context, provider string) (*OAuthProviderConfig, error) {
	var c OAuthProviderConfig
	err := r.pool.QueryRow(ctx,
		"SELECT id, provider, enabled, client_id, client_secret, redirect_uri, config, updated_at FROM oauth_config WHERE provider = $1",
		provider).Scan(&c.ID, &c.Provider, &c.Enabled, &c.ClientID, &c.ClientSecret, &c.RedirectURI, &c.Config, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) Upsert(ctx context.Context, provider string, enabled bool, clientID, clientSecret, redirectURI string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_config (provider, enabled, client_id, client_secret, redirect_uri, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (provider) DO UPDATE SET enabled = $2, client_id = $3, client_secret = $4, redirect_uri = $5, updated_at = NOW()
	`, provider, enabled, clientID, clientSecret, redirectURI)
	return err
}
