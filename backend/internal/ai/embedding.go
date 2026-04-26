package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Embedder interface for generating text embeddings
type Embedder interface {
	Encode(ctx context.Context, text string) ([]float32, error)
	EncodeBatch(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
	Name() string
}

// EmbeddingFactory creates embedder instances based on model configuration
type EmbeddingFactory struct {
	httpClient *http.Client
}

func NewEmbeddingFactory() *EmbeddingFactory {
	return &EmbeddingFactory{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (f *EmbeddingFactory) GetEmbedder(model *AIModel) Embedder {
	switch model.Provider {
	case "openai":
		return NewOpenAIEmbedder(model, f.httpClient)
	case "ollama":
		return NewOllamaEmbedder(model, f.httpClient)
	default:
		return NewOpenAIEmbedder(model, f.httpClient)
	}
}

// OpenAIEmbedder calls OpenAI-compatible embedding API
type OpenAIEmbedder struct {
	modelName string
	apiKey    string
	endpoint  string
	client    *http.Client
	dim       int
}

func NewOpenAIEmbedder(model *AIModel, client *http.Client) *OpenAIEmbedder {
	endpoint := model.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/embeddings"
	}
	dim := 1536
	if model.Dimensions > 0 {
		dim = model.Dimensions
	}
	return &OpenAIEmbedder{
		modelName: model.ModelName,
		apiKey:    model.APIKey,
		endpoint:  endpoint,
		client:    client,
		dim:       dim,
	}
}

func (e *OpenAIEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	results, err := e.EncodeBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (e *OpenAIEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"input": texts,
		"model": e.modelName,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", e.endpoint, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embedding API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		emb := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			emb[j] = float32(v)
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

func (e *OpenAIEmbedder) Dimensions() int { return e.dim }
func (e *OpenAIEmbedder) Name() string    { return e.modelName }

// OllamaEmbedder calls local Ollama embedding API
type OllamaEmbedder struct {
	endpoint  string
	modelName string
	client    *http.Client
}

func NewOllamaEmbedder(model *AIModel, client *http.Client) *OllamaEmbedder {
	endpoint := model.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	return &OllamaEmbedder{
		endpoint:  endpoint,
		modelName: model.ModelName,
		client:    client,
	}
}

func (e *OllamaEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":  e.modelName,
		"prompt": text,
	})

	resp, err := e.client.Post(e.endpoint+"/api/embeddings", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	emb := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		emb[i] = float32(v)
	}
	return emb, nil
}

func (e *OllamaEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Encode(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

func (e *OllamaEmbedder) Dimensions() int { return 0 }
func (e *OllamaEmbedder) Name() string    { return e.modelName }
