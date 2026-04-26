-- System configuration
CREATE TABLE system_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

INSERT INTO system_config (config_key, config_value, description) VALUES
('site_name', '{"value": "Krasis"}', 'Site name'),
('allow_signup', '{"value": true}', 'Allow user registration'),
('require_email_verification', '{"value": true}', 'Require email verification'),
('default_role', '{"value": "member"}', 'Default role for new users'),
('max_notes_per_user', '{"value": -1}', 'Max notes per user (-1 = unlimited)'),
('max_storage_per_user_bytes', '{"value": 10737418240}', 'Max storage per user in bytes (10GB)'),
('max_file_size_bytes', '{"value": 104857600}', 'Max file size in bytes (100MB)'),
('session_duration_days', '{"value": 7}', 'Session duration in days'),
('max_devices_per_user', '{"value": 10}', 'Max login devices per user'),
('enable_sharing', '{"value": true}', 'Enable sharing feature'),
('enable_ai', '{"value": true}', 'Enable AI feature'),
('maintenance_mode', '{"value": false}', 'Enable maintenance mode');

-- OAuth configuration
CREATE TABLE oauth_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) UNIQUE NOT NULL,  -- github, google
    enabled BOOLEAN DEFAULT false,
    client_id VARCHAR(255),
    client_secret VARCHAR(255),
    redirect_uri VARCHAR(500),
    config JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- User groups
CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_groups_default ON groups(is_default);

-- Group features (per-group feature toggles and limits)
CREATE TABLE group_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,
    feature_value JSONB NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(group_id, feature_key)
);

CREATE INDEX idx_group_features_group ON group_features(group_id);

-- Insert default groups
INSERT INTO groups (name, description, is_default) VALUES
('free', '免费用户组', true),
('pro', '专业版用户组', false),
('enterprise', '企业版用户组', false);

-- Insert default features for free group
INSERT INTO group_features (group_id, feature_key, feature_value)
SELECT id, 'enable_sharing', '{"value": true}' FROM groups WHERE name = 'free';

INSERT INTO group_features (group_id, feature_key, feature_value)
SELECT id, 'enable_ai', '{"value": true}' FROM groups WHERE name = 'free';

INSERT INTO group_features (group_id, feature_key, feature_value)
SELECT id, 'ai_ask_limit', '{"value": 10, "period": "minute"}' FROM groups WHERE name = 'free';

INSERT INTO group_features (group_id, feature_key, feature_value)
SELECT id, 'version_history_limit', '{"value": 10}' FROM groups WHERE name = 'free';

INSERT INTO group_features (group_id, feature_key, feature_value)
SELECT id, 'storage_limit_bytes', '{"value": 10737418240}' FROM groups WHERE name = 'free';

-- Add group_id to users table
ALTER TABLE users ADD COLUMN group_id UUID REFERENCES groups(id);

-- Set default group for existing users
UPDATE users SET group_id = (SELECT id FROM groups WHERE is_default LIMIT 1) WHERE group_id IS NULL;

-- Audit logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),          -- user, note, share, ai_model, config, etc.
    target_id UUID,
    admin_id UUID NOT NULL REFERENCES users(id),
    admin_username VARCHAR(100),
    changes JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_admin ON audit_logs(admin_id);
CREATE INDEX idx_audit_logs_target ON audit_logs(target_type, target_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
