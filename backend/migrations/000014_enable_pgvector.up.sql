-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Add embedding vector column to note_embeddings (replaces Qdrant vector_id)
ALTER TABLE note_embeddings ADD COLUMN IF NOT EXISTS embedding vector(1536);

-- Drop the Qdrant-specific vector_id column
ALTER TABLE note_embeddings DROP COLUMN IF EXISTS vector_id;

-- Create HNSW index for fast approximate cosine similarity search
-- HNSW is available in pgvector 0.5.0+ (this system runs 0.8.2)
CREATE INDEX IF NOT EXISTS idx_note_embeddings_vector
    ON note_embeddings
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
