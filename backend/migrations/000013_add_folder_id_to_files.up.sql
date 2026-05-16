ALTER TABLE files
    ADD COLUMN folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    ALTER COLUMN note_id DROP NOT NULL;

CREATE INDEX idx_files_folder ON files(folder_id);
