CREATE TABLE user_oauth (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_access_token TEXT,
    provider_refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_user ON user_oauth(user_id);
CREATE INDEX idx_oauth_provider ON user_oauth(provider, provider_user_id);
