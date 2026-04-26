CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID REFERENCES notes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50),
    mime_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL,
    bucket VARCHAR(100) DEFAULT 'notes',
    size_bytes BIGINT,
    width INT,
    height INT,
    duration_sec FLOAT,
    thumbnail_path VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    status SMALLINT DEFAULT 0,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_files_note ON files(note_id);
CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_type ON files(file_type);
CREATE INDEX idx_files_status ON files(status);
