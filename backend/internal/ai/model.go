package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/types"
	"go.uber.org/zap"
)

var (
	ErrNoLLMModelConfigured   = errors.New("no LLM model configured")
	ErrNoEmbeddingModelConfig = errors.New("no embedding model configured")
)

type AIModel struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Provider    string          `json:"provider"`
	ModelType   string          `json:"type"`
	Endpoint    string          `json:"endpoint"`
	APIKey      string          `json:"-"`
	ModelName   string          `json:"model_name"`
	APIVersion  string          `json:"api_version,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	TopP        float64         `json:"top_p,omitempty"`
	Dimensions  int             `json:"dimensions,omitempty"`
	IsEnabled   bool            `json:"is_enabled"`
	IsDefault   bool            `json:"is_default"`
	Priority    int             `json:"priority"`
	Config      json.RawMessage `json:"config,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   sql.NullTime    `json:"updated_at,omitempty"`
}

type AIConfig struct {
	ChunkSize        int    `json:"chunk_size"`
	ChunkOverlap     int    `json:"chunk_overlap"`
	TopK             int    `json:"top_k"`
	ScoreThreshold   float64 `json:"score_threshold"`
	EnableRAG        bool   `json:"enable_rag"`
	MaxContextTokens int    `json:"max_context_tokens"`
	SystemPrompt     string `json:"system_prompt"`
	EnableStreaming  bool   `json:"enable_streaming"`
}

type Conversation struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Title     string     `json:"title"`
	Model     string     `json:"model"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt types.NullTime `json:"updated_at"`
}

type Message struct {
	ID             uuid.UUID      `json:"id"`
	ConversationID uuid.UUID      `json:"conversation_id"`
	Role           string         `json:"role"`
	Content        string         `json:"content"`
	References     json.RawMessage `json:"references,omitempty"`
	TokenCount     int            `json:"token_count,omitempty"`
	Model          string         `json:"model,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetAIConfig(ctx context.Context) (*AIConfig, error) {
	rows, err := r.pool.Query(ctx, "SELECT config_key, config_value FROM ai_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cfg := &AIConfig{
		ChunkSize: 500, ChunkOverlap: 50, TopK: 5,
		ScoreThreshold: 0.7, EnableRAG: true,
		MaxContextTokens: 8000, EnableStreaming: true,
		SystemPrompt: "你是一个智能笔记助手。请根据以下上下文回答用户的问题。",
	}

	for rows.Next() {
		var key string
		var val []byte
		rows.Scan(&key, &val)
		var v struct {
			Value interface{} `json:"value"`
		}
		json.Unmarshal(val, &v)

		switch key {
		case "chunk_size":
			if f, ok := v.Value.(float64); ok {
				cfg.ChunkSize = int(f)
			}
		case "chunk_overlap":
			if f, ok := v.Value.(float64); ok {
				cfg.ChunkOverlap = int(f)
			}
		case "top_k":
			if f, ok := v.Value.(float64); ok {
				cfg.TopK = int(f)
			}
		case "score_threshold":
			if f, ok := v.Value.(float64); ok {
				cfg.ScoreThreshold = f
			}
		case "enable_rag":
			if b, ok := v.Value.(bool); ok {
				cfg.EnableRAG = b
			}
		case "max_context_tokens":
			if f, ok := v.Value.(float64); ok {
				cfg.MaxContextTokens = int(f)
			}
		case "system_prompt":
			if s, ok := v.Value.(string); ok {
				cfg.SystemPrompt = s
			}
		case "enable_streaming":
			if b, ok := v.Value.(bool); ok {
				cfg.EnableStreaming = b
			}
		}
	}
	return cfg, nil
}

func (r *Repository) ListModels(ctx context.Context, modelType string) ([]*AIModel, error) {
	query := "SELECT id, name, provider, model_type, endpoint, api_key, model_name, api_version, max_tokens, temperature, top_p, dimensions, is_enabled, is_default, priority, config, created_at, updated_at FROM ai_models ORDER BY model_type, priority"
	args := []interface{}{}

	if modelType != "" {
		query = "SELECT id, name, provider, model_type, endpoint, api_key, model_name, api_version, max_tokens, temperature, top_p, dimensions, is_enabled, is_default, priority, config, created_at, updated_at FROM ai_models WHERE model_type = $1 ORDER BY priority"
		args = append(args, modelType)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*AIModel
	for rows.Next() {
		var m AIModel
		err := rows.Scan(
			&m.ID, &m.Name, &m.Provider, &m.ModelType, &m.Endpoint,
			&m.APIKey, &m.ModelName, &m.APIVersion, &m.MaxTokens,
			&m.Temperature, &m.TopP, &m.Dimensions, &m.IsEnabled,
			&m.IsDefault, &m.Priority, &m.Config, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		models = append(models, &m)
	}
	return models, nil
}

func (r *Repository) GetModel(ctx context.Context, id uuid.UUID) (*AIModel, error) {
	var m AIModel
	err := r.pool.QueryRow(ctx,
		"SELECT id, name, provider, model_type, endpoint, api_key, model_name, api_version, max_tokens, temperature, top_p, dimensions, is_enabled, is_default, priority, config, created_at, updated_at FROM ai_models WHERE id = $1",
		id,
	).Scan(
		&m.ID, &m.Name, &m.Provider, &m.ModelType, &m.Endpoint,
		&m.APIKey, &m.ModelName, &m.APIVersion, &m.MaxTokens,
		&m.Temperature, &m.TopP, &m.Dimensions, &m.IsEnabled,
		&m.IsDefault, &m.Priority, &m.Config, &m.CreatedAt, &m.UpdatedAt,
	)
	return &m, err
}

func (r *Repository) GetModelByName(ctx context.Context, name string) (*AIModel, error) {
	var m AIModel
	err := r.pool.QueryRow(ctx,
		"SELECT id, name, provider, model_type, endpoint, api_key, model_name, api_version, max_tokens, temperature, top_p, dimensions, is_enabled, is_default, priority, config, created_at, updated_at FROM ai_models WHERE name = $1 AND is_enabled = true",
		name,
	).Scan(
		&m.ID, &m.Name, &m.Provider, &m.ModelType, &m.Endpoint,
		&m.APIKey, &m.ModelName, &m.APIVersion, &m.MaxTokens,
		&m.Temperature, &m.TopP, &m.Dimensions, &m.IsEnabled,
		&m.IsDefault, &m.Priority, &m.Config, &m.CreatedAt, &m.UpdatedAt,
	)
	return &m, err
}

func (r *Repository) CreateModel(ctx context.Context, m *AIModel) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO ai_models (name, provider, model_type, endpoint, api_key, model_name,
			api_version, max_tokens, temperature, top_p, dimensions, is_enabled, is_default,
			priority, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, m.Name, m.Provider, m.ModelType, m.Endpoint, m.APIKey, m.ModelName,
		m.APIVersion, m.MaxTokens, m.Temperature, m.TopP, m.Dimensions, m.IsEnabled,
		m.IsDefault, m.Priority, m.Config,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *Repository) UpdateModel(ctx context.Context, id uuid.UUID, m *AIModel) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE ai_models SET name = $1, provider = $2, model_type = $3, endpoint = $4,
			api_key = $5, model_name = $6, api_version = $7, max_tokens = $8,
			temperature = $9, top_p = $10, dimensions = $11, is_enabled = $12,
			is_default = $13, priority = $14, config = $15, updated_at = NOW()
		WHERE id = $16
	`, m.Name, m.Provider, m.ModelType, m.Endpoint, m.APIKey, m.ModelName,
		m.APIVersion, m.MaxTokens, m.Temperature, m.TopP, m.Dimensions, m.IsEnabled,
		m.IsDefault, m.Priority, m.Config, id,
	)
	return err
}

func (r *Repository) DeleteModel(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM ai_models WHERE id = $1", id)
	return err
}

func (r *Repository) CreateConversation(ctx context.Context, userID uuid.UUID, title, model string) (*Conversation, error) {
	var c Conversation
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ai_conversations (user_id, title, model) VALUES ($1, $2, $3)
		RETURNING id, user_id, title, model, created_at, updated_at
	`, userID, title, model).Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func (r *Repository) ListConversations(ctx context.Context, userID uuid.UUID) ([]*Conversation, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, user_id, title, model, created_at, updated_at FROM ai_conversations WHERE user_id = $1 ORDER BY updated_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, &c)
	}
	return convs, nil
}

func (r *Repository) GetConversation(ctx context.Context, id, userID uuid.UUID) (*Conversation, error) {
	var c Conversation
	err := r.pool.QueryRow(ctx,
		"SELECT id, user_id, title, model, created_at, updated_at FROM ai_conversations WHERE id = $1 AND user_id = $2",
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func (r *Repository) CreateMessage(ctx context.Context, msg *Message) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO ai_messages (conversation_id, role, content, references, token_count, model)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, msg.ConversationID, msg.Role, msg.Content, msg.References, msg.TokenCount, msg.Model,
	).Scan(&msg.ID, &msg.CreatedAt)
}

func (r *Repository) ListMessages(ctx context.Context, conversationID, userID uuid.UUID) ([]*Message, error) {
	// Verify conversation ownership
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT true FROM ai_conversations WHERE id = $1 AND user_id = $2",
		conversationID, userID,
	).Scan(&exists)
	if err != nil || !exists {
		return nil, errors.New("conversation not found")
	}

	rows, err := r.pool.Query(ctx,
		"SELECT id, conversation_id, role, content, references, token_count, model, created_at FROM ai_messages WHERE conversation_id = $1 ORDER BY created_at",
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.References, &m.TokenCount, &m.Model, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	return messages, nil
}

func (r *Repository) SaveEmbedding(ctx context.Context, noteID uuid.UUID, chunkIndex int, chunkText, vectorID string, tokenCount int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_embeddings (note_id, chunk_index, chunk_text, vector_id, token_count)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (note_id, chunk_index) DO UPDATE SET chunk_text = $3, vector_id = $4, token_count = $5, updated_at = NOW()
	`, noteID, chunkIndex, chunkText, vectorID, tokenCount)
	return err
}

func (r *Repository) DeleteEmbeddingsByNote(ctx context.Context, noteID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM note_embeddings WHERE note_id = $1", noteID)
	return err
}

func (r *Repository) GetEmbeddingIDsByNote(ctx context.Context, noteID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, "SELECT vector_id FROM note_embeddings WHERE note_id = $1", noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repository) UpdateConfigValue(ctx context.Context, key string, value []byte) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_config (config_key, config_value) VALUES ($1, $2)
		ON CONFLICT (config_key) DO UPDATE SET config_value = $2, updated_at = NOW()
	`, key, value)
	return err
}

// ModelConfigManager manages AI model configurations with in-memory cache
type ModelConfigManager struct {
	repo            *Repository
	mu              sync.RWMutex
	models          []*AIModel
	embeddingModels []*AIModel
	llmModels       []*AIModel
}

func NewModelConfigManager(repo *Repository) *ModelConfigManager {
	return &ModelConfigManager{repo: repo}
}

func (m *ModelConfigManager) LoadModels(ctx context.Context) error {
	models, err := m.repo.ListModels(ctx, "")
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.models = models
	m.embeddingModels = nil
	m.llmModels = nil

	for _, model := range models {
		switch model.ModelType {
		case "embedding":
			m.embeddingModels = append(m.embeddingModels, model)
		case "llm":
			m.llmModels = append(m.llmModels, model)
		}
	}
	return nil
}

// StartReloader periodically reloads model config from the database.
func (m *ModelConfigManager) StartReloader(ctx context.Context, interval time.Duration, logger *zap.Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := m.LoadModels(ctx); err != nil {
					logger.Warn("failed to reload AI models", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *ModelConfigManager) GetDefaultLLM() *AIModel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, model := range m.llmModels {
		if model.IsEnabled && model.IsDefault {
			return model
		}
	}
	for _, model := range m.llmModels {
		if model.IsEnabled {
			return model
		}
	}
	return nil
}

func (m *ModelConfigManager) GetDefaultEmbedding() *AIModel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, model := range m.embeddingModels {
		if model.IsEnabled && model.IsDefault {
			return model
		}
	}
	for _, model := range m.embeddingModels {
		if model.IsEnabled {
			return model
		}
	}
	return nil
}

func (m *ModelConfigManager) GetLLM(id string) *AIModel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		for _, model := range m.llmModels {
			if model.IsEnabled && model.Name == id {
				return model
			}
		}
		return nil
	}

	for _, model := range m.llmModels {
		if model.IsEnabled && model.ID == parsedID {
			return model
		}
	}
	return nil
}
