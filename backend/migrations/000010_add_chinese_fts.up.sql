-- Enable zhparser extension for Chinese full-text search
CREATE EXTENSION IF NOT EXISTS zhparser;

-- Create Chinese text search configuration
CREATE TEXT SEARCH CONFIGURATION IF NOT EXISTS chinese_zh (PARSER = zhparser);

-- Add Chinese search vector column to notes table
ALTER TABLE notes ADD COLUMN IF NOT EXISTS search_vector_zh tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('chinese_zh', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('chinese_zh', coalesce(content, '')), 'B')
    ) STORED;

-- Create GIN index for Chinese full-text search
CREATE INDEX IF NOT EXISTS idx_notes_search_zh ON notes USING GIN (search_vector_zh);
