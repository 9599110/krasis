package vector

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/internal/ai"
)

// PGVectorStore stores and searches embeddings in PostgreSQL via pgvector extension.
// It implements ai.VectorStore interface.
type PGVectorStore struct {
	pool      *pgxpool.Pool
	dimension int // embedding dimension (e.g. 1536 for text-embedding-ada-002)
}

// NewPGVectorStore creates a new PGVectorStore.
// dimension should match the embedding model's output dimension.
func NewPGVectorStore(pool *pgxpool.Pool, dimension int) *PGVectorStore {
	if dimension <= 0 {
		dimension = 1536 // default for OpenAI text-embedding-ada-002
	}
	return &PGVectorStore{
		pool:      pool,
		dimension: dimension,
	}
}

// formatVector converts a float32 slice to pgvector text format like [1.0,2.0,3.0]
func formatVector(v []float32) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, val := range v {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%f", val))
	}
	sb.WriteString("]")
	return sb.String()
}

// Upsert inserts or updates note chunks with their embeddings in the note_embeddings table.
func (s *PGVectorStore) Upsert(ctx context.Context, noteID string, chunks []*ai.Chunk, vectors [][]float32) error {
	uid, err := uuid.Parse(noteID)
	if err != nil {
		return fmt.Errorf("parse note_id: %w", err)
	}

	for i, chunk := range chunks {
		vec := vectors[i]
		if len(vec) != s.dimension {
			return fmt.Errorf("chunk %d: expected dimension %d, got %d", i, s.dimension, len(vec))
		}

		// Format vector as text for pgvector's ::vector cast
		vecStr := formatVector(vec)

		_, err := s.pool.Exec(ctx, `
			INSERT INTO note_embeddings (note_id, chunk_index, chunk_text, embedding, token_count)
			VALUES ($1, $2, $3, $4::vector, $5)
			ON CONFLICT (note_id, chunk_index)
			DO UPDATE SET chunk_text = $3, embedding = $4::vector, token_count = $5, updated_at = NOW()
		`, uid, chunk.ChunkIndex, chunk.Text, vecStr, chunk.TokenCount)
		if err != nil {
			return fmt.Errorf("upsert chunk %d: %w", i, err)
		}
	}
	return nil
}

// Search performs cosine similarity search via pgvector's <=> operator.
// Results are filtered by user (owner_id in notes table).
// threshold is a similarity threshold (0.0 - 1.0), results below it are excluded.
func (s *PGVectorStore) Search(ctx context.Context, queryVector []float32, userID string, topK int, threshold float64) ([]*ai.Chunk, error) {
	if len(queryVector) != s.dimension {
		return nil, fmt.Errorf("query vector dimension %d does not match store dimension %d", len(queryVector), s.dimension)
	}

	// Format query vector as text for ::vector cast
	queryVecStr := formatVector(queryVector)

	// Convert similarity threshold to distance threshold
	// similarity = 1 - cosine_distance, so distance = 1 - similarity
	maxDistance := 1.0 - threshold

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user_id: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT ne.note_id::text, n.title, ne.chunk_text, ne.chunk_index, ne.token_count
		FROM note_embeddings ne
		JOIN notes n ON n.id = ne.note_id
		WHERE n.owner_id = $1
		  AND n.is_deleted = false
		  AND (ne.embedding <=> $2::vector) <= $3
		ORDER BY ne.embedding <=> $2::vector
		LIMIT $4
	`, uid, queryVecStr, maxDistance, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search query: %w", err)
	}
	defer rows.Close()

	var chunks []*ai.Chunk
	for rows.Next() {
		var c ai.Chunk
		if err := rows.Scan(&c.NoteID, &c.NoteTitle, &c.Text, &c.ChunkIndex, &c.TokenCount); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		chunks = append(chunks, &c)
	}

	return chunks, nil
}

// DeleteByNote removes all embeddings for a given note.
func (s *PGVectorStore) DeleteByNote(ctx context.Context, noteID string) error {
	uid, err := uuid.Parse(noteID)
	if err != nil {
		return fmt.Errorf("parse note_id: %w", err)
	}
	_, err = s.pool.Exec(ctx, "DELETE FROM note_embeddings WHERE note_id = $1", uid)
	return err
}
