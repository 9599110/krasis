-- Enable full-text search extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Add search_vector column to notes table
ALTER TABLE notes ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content, '')), 'B')
    ) STORED;

-- Create GIN index for full-text search
CREATE INDEX idx_notes_search ON notes USING GIN (search_vector);

-- Create trigram index for fuzzy search on title
CREATE INDEX idx_notes_title_trgm ON notes USING GIN (title gin_trgm_ops);
