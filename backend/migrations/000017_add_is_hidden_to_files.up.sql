-- Add is_hidden column to files for hidden-until-unhide encrypted file support
ALTER TABLE files ADD COLUMN is_hidden BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_files_hidden ON files(is_hidden);
