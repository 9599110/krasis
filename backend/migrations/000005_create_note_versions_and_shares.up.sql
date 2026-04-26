CREATE TABLE note_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    title VARCHAR(500),
    content TEXT,
    content_html TEXT,
    version INT NOT NULL,
    changed_by UUID REFERENCES users(id),
    change_summary TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_note_versions_note ON note_versions(note_id);
CREATE INDEX idx_note_versions_version ON note_versions(note_id, version DESC);

CREATE TABLE note_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    share_token VARCHAR(64) UNIQUE NOT NULL,
    share_type VARCHAR(20) DEFAULT 'link',
    share_with_user_id UUID REFERENCES users(id),
    share_with_email VARCHAR(255),
    permission VARCHAR(20) DEFAULT 'read',
    password_hash VARCHAR(255),
    expires_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'pending',
    content_snapshot TEXT,
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id),
    rejection_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_note_shares_token ON note_shares(share_token);
CREATE INDEX idx_note_shares_note ON note_shares(note_id);
CREATE INDEX idx_note_shares_status ON note_shares(status);
