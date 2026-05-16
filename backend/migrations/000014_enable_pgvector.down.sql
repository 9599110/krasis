DROP INDEX IF EXISTS idx_note_embeddings_vector;

ALTER TABLE note_embeddings ADD COLUMN IF NOT EXISTS vector_id VARCHAR(255);

ALTER TABLE note_embeddings DROP COLUMN IF EXISTS embedding;
