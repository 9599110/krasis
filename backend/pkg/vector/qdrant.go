package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/krasis/krasis/internal/ai"
)

// QdrantClient is a minimal Qdrant HTTP client for vector operations
type QdrantClient struct {
	endpoint   string
	apiKey     string
	collection string
	httpClient *http.Client
}

func NewQdrantClient(endpoint, apiKey, collection string) *QdrantClient {
	return &QdrantClient{
		endpoint:   endpoint,
		apiKey:     apiKey,
		collection: collection,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

type point struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

type searchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	Filter      any       `json:"filter,omitempty"`
}

type searchResult struct {
	Result []struct {
		ID      string                 `json:"id"`
		Score   float64                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

func (c *QdrantClient) Upsert(ctx context.Context, noteID string, chunks []*ai.Chunk, vectors [][]float32) error {
	points := make([]point, len(chunks))
	for i, chunk := range chunks {
		points[i] = point{
			ID:     fmt.Sprintf("%s_%d", noteID, chunk.ChunkIndex),
			Vector: vectors[i],
			Payload: map[string]interface{}{
				"note_id":     noteID,
				"user_id":     chunk.NoteID,
				"note_title":  chunk.NoteTitle,
				"chunk_text":  chunk.Text,
				"chunk_index": chunk.ChunkIndex,
			},
		}
	}

	body := map[string]interface{}{"points": points}
	return c.do(ctx, "PUT", "/points", body, nil)
}

func (c *QdrantClient) Search(ctx context.Context, vector []float32, userID string, topK int, threshold float64) ([]*ai.Chunk, error) {
	req := searchRequest{
		Vector:      vector,
		Limit:       topK,
		WithPayload: true,
	}
	if userID != "" {
		req.Filter = map[string]any{
			"must": []map[string]any{
				{"key": "user_id", "match": map[string]any{"value": userID}},
			},
		}
	}

	var result searchResult
	if err := c.do(ctx, "POST", "/points/search", req, &result); err != nil {
		return nil, err
	}

	var chunks []*ai.Chunk
	for _, r := range result.Result {
		if r.Score < threshold {
			continue
		}
		chunk := &ai.Chunk{
			NoteID:    strField(r.Payload, "note_id"),
			NoteTitle: strField(r.Payload, "note_title"),
			Text:      strField(r.Payload, "chunk_text"),
		}
		if v, ok := r.Payload["chunk_index"].(float64); ok {
			chunk.ChunkIndex = int(v)
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (c *QdrantClient) DeleteByNote(ctx context.Context, noteID string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{"key": "note_id", "match": map[string]any{"value": noteID}},
			},
		},
	}
	return c.do(ctx, "POST", "/points/delete", body, nil)
}

func (c *QdrantClient) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	data, _ := json.Marshal(body)
	url := c.endpoint + "/collections/" + c.collection + path

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Qdrant returned status %d", resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func strField(payload map[string]interface{}, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}
