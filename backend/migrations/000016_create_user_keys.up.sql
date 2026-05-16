CREATE TABLE IF NOT EXISTS user_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    encrypted_private_key TEXT NOT NULL,  -- base64 encoded
    salt                TEXT NOT NULL,    -- base64 encoded
    iv                  TEXT NOT NULL,    -- base64 encoded
    public_key_raw      TEXT NOT NULL,    -- base64 encoded
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ,
    UNIQUE(user_id)
);

CREATE INDEX idx_user_keys_user_id ON user_keys(user_id);
