-- Composite index for note list queries: filter by owner_id + is_deleted, sort by updated_at
CREATE INDEX idx_notes_owner_deleted ON notes(owner_id, is_deleted);
CREATE INDEX idx_notes_updated_at ON notes(updated_at DESC) WHERE is_deleted = false;
