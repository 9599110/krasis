package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// VectorStore interface for vector database operations
type VectorStore interface {
	Search(ctx context.Context, vector []float32, userID string, topK int, threshold float64) ([]*Chunk, error)
	Upsert(ctx context.Context, noteID string, chunks []*Chunk, vectors [][]float32) error
	DeleteByNote(ctx context.Context, noteID string) error
}

// FTSResult holds a keyword search result for RAG fallback
type FTSResult struct {
	ID         uuid.UUID
	Title      string
	Highlights []string
	Score      float64
}

// KeywordSearcher interface for full-text search fallback
type KeywordSearcher interface {
	SearchByKeyword(ctx context.Context, query string, userID string, topK int) ([]*FTSResult, error)
}

type AIService struct {
	repo           *Repository
	modelManager   *ModelConfigManager
	vectorStore    VectorStore
	keywordSearcher KeywordSearcher
	llmFactory     *LLMFactory
	embedFactory   *EmbeddingFactory
}

func NewAIService(repo *Repository, modelManager *ModelConfigManager, vectorStore VectorStore, keywordSearcher KeywordSearcher) *AIService {
	return &AIService{
		repo:           repo,
		modelManager:   modelManager,
		vectorStore:    vectorStore,
		keywordSearcher: keywordSearcher,
		llmFactory:     NewLLMFactory(),
		embedFactory:   NewEmbeddingFactory(),
	}
}

type AskRequest struct {
	Question       string `json:"question"`
	ConversationID string `json:"conversation_id,omitempty"`
	ModelID        string `json:"model_id,omitempty"`
	TopK           int    `json:"top_k,omitempty"`
	Stream         bool   `json:"stream,omitempty"`
}

type AskResponse struct {
	Answer         string      `json:"answer"`
	References     []*Chunk    `json:"references,omitempty"`
	ConversationID string      `json:"conversation_id"`
	MessageID      string      `json:"message_id,omitempty"`
}

func (s *AIService) Ask(ctx context.Context, userID string, req *AskRequest) (*AskResponse, error) {
	// Get AI config
	aiCfg, err := s.repo.GetAIConfig(ctx)
	if err != nil {
		aiCfg = &AIConfig{
			TopK: 5, EnableRAG: true, EnableStreaming: true,
			SystemPrompt: "你是一个智能笔记助手。请根据以下上下文回答用户的问题。",
		}
	}

	// 1. Hybrid retrieval: vector search first, FTS fallback
	chunks, err := s.hybridRetrieve(ctx, userID, req.Question, aiCfg, req.TopK)
	if err != nil {
		return nil, fmt.Errorf("retrieval error: %w", err)
	}

	// 2. Get LLM model
	var llmModel *AIModel
	if req.ModelID != "" {
		llmModel = s.modelManager.GetLLM(req.ModelID)
	}
	if llmModel == nil {
		llmModel = s.modelManager.GetDefaultLLM()
	}
	if llmModel == nil {
		return nil, ErrNoLLMModelConfigured
	}

	// 3. Build prompt
	var messages []MessageParam
	if len(chunks) == 0 {
		// Fall back to pure chat when no retrieval context is found.
		messages = []MessageParam{
			{Role: "system", Content: aiCfg.SystemPrompt},
			{Role: "user", Content: req.Question},
		}
	} else {
		context := s.buildContext(chunks)
		messages = s.buildMessages(aiCfg.SystemPrompt, req.Question, context)
	}

	// 4. Generate response
	llm := s.llmFactory.GetLLM(llmModel)
	answer, err := llm.Generate(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM generate error: %w", err)
	}

	return &AskResponse{
		Answer:     answer,
		References: chunks,
	}, nil
}

func (s *AIService) AskStream(ctx context.Context, userID string, req *AskRequest) (<-chan string, error) {
	aiCfg, err := s.repo.GetAIConfig(ctx)
	if err != nil {
		aiCfg = &AIConfig{
			TopK: 5, EnableRAG: true, EnableStreaming: true,
			SystemPrompt: "你是一个智能笔记助手。请根据以下上下文回答用户的问题。",
		}
	}

	// 1. Hybrid retrieval
	chunks, err := s.hybridRetrieve(ctx, userID, req.Question, aiCfg, req.TopK)
	if err != nil {
		return nil, fmt.Errorf("retrieval error: %w", err)
	}

	var llmModel *AIModel
	if req.ModelID != "" {
		llmModel = s.modelManager.GetLLM(req.ModelID)
	}
	if llmModel == nil {
		llmModel = s.modelManager.GetDefaultLLM()
	}
	if llmModel == nil {
		return nil, ErrNoLLMModelConfigured
	}

	var messages []MessageParam
	if len(chunks) == 0 {
		messages = []MessageParam{
			{Role: "system", Content: aiCfg.SystemPrompt},
			{Role: "user", Content: req.Question},
		}
	} else {
		context := s.buildContext(chunks)
		messages = s.buildMessages(aiCfg.SystemPrompt, req.Question, context)
	}

	llm := s.llmFactory.GetLLM(llmModel)
	return llm.GenerateStream(ctx, messages, nil)
}

func (s *AIService) buildContext(chunks []*Chunk) string {
	var sb strings.Builder
	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("\n[%d] 来源：%s\n内容：%s\n", i+1, chunk.NoteTitle, chunk.Text))
	}
	return sb.String()
}

func (s *AIService) buildMessages(systemPrompt, question, context string) []MessageParam {
	messages := []MessageParam{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("上下文：\n%s\n\n问题：%s\n\n请基于上下文回答，如果上下文中没有相关信息，请说明无法回答。", context, question)},
	}
	return messages
}

// hybridRetrieve performs hybrid retrieval: vector search first, then FTS fallback.
func (s *AIService) hybridRetrieve(ctx context.Context, userID, query string, aiCfg *AIConfig, reqTopK int) ([]*Chunk, error) {
	topK := reqTopK
	if topK <= 0 {
		topK = aiCfg.TopK
	}

	// 1. Vector search
	var chunks []*Chunk
	if aiCfg.EnableRAG && s.vectorStore != nil {
		embedModel := s.modelManager.GetDefaultEmbedding()
		if embedModel != nil {
			embedder := s.embedFactory.GetEmbedder(embedModel)
			queryVector, err := embedder.Encode(ctx, query)
			if err == nil {
				chunks, err = s.vectorStore.Search(ctx, queryVector, userID, topK, aiCfg.ScoreThreshold)
				if err != nil {
					chunks = nil // Reset on error, try FTS
				}
			}
		}
	}

	// 2. FTS fallback if vector results are insufficient
	needMore := topK - len(chunks)
	if needMore > 0 && s.keywordSearcher != nil {
		ftsResults, err := s.keywordSearcher.SearchByKeyword(ctx, query, userID, topK)
		if err == nil && len(ftsResults) > 0 {
			// Convert FTS results to chunks, deduplicating by note ID
			vectorNoteIDs := make(map[string]bool)
			for _, c := range chunks {
				vectorNoteIDs[c.NoteID] = true
			}

			for _, r := range ftsResults {
				if len(chunks) >= topK {
					break
				}
				if !vectorNoteIDs[r.ID.String()] {
					// Join highlights into text
					text := strings.Join(r.Highlights, "\n")
					chunks = append(chunks, &Chunk{
						NoteID:    r.ID.String(),
						NoteTitle: r.Title,
						Text:      text,
						ChunkIndex: 0,
						TokenCount: len([]rune(text)),
					})
				}
			}
		}
	}

	return chunks, nil
}

func (s *AIService) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListConversations(ctx, uid)
}

func (s *AIService) ListMessages(ctx context.Context, conversationID, userID string) ([]*Message, error) {
	cid, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMessages(ctx, cid, uid)
}

func (s *AIService) CreateConversation(ctx context.Context, userID, title, model string) (*Conversation, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateConversation(ctx, uid, title, model)
}

func (s *AIService) SaveMessage(ctx context.Context, msg *Message) error {
	return s.repo.CreateMessage(ctx, msg)
}

// GetModelManager returns the model configuration manager
func (s *AIService) GetModelManager() *ModelConfigManager {
	return s.modelManager
}

// DeleteNoteIndex removes a note's embeddings from the vector store
func (s *AIService) DeleteNoteIndex(ctx context.Context, noteID string) error {
	if s.vectorStore != nil {
		s.vectorStore.DeleteByNote(ctx, noteID)
	}
	s.repo.DeleteEmbeddingsByNote(ctx, uuid.MustParse(noteID))
	return nil
}

// IndexNote chunks note content into embeddings and stores in vector DB
func (s *AIService) IndexNote(ctx context.Context, noteID, title, content, userID string) error {
	embedModel := s.modelManager.GetDefaultEmbedding()
	if embedModel == nil {
		return nil // Skip if no embedding model
	}

	aiCfg, err := s.repo.GetAIConfig(ctx)
	if err != nil {
		aiCfg = &AIConfig{ChunkSize: 500, ChunkOverlap: 50}
	}

	// Delete old embeddings for this note
	if s.vectorStore != nil {
		s.vectorStore.DeleteByNote(ctx, noteID)
	}

	// Chunk and embed
	chunks := SplitNoteIntoChunks(noteID, title, content, aiCfg.ChunkSize, aiCfg.ChunkOverlap)
	if len(chunks) == 0 {
		return nil
	}

	embedder := s.embedFactory.GetEmbedder(embedModel)
	vectors := make([][]float32, len(chunks))
	for i, chunk := range chunks {
		vec, err := embedder.Encode(ctx, chunk.Text)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", i, err)
		}
		vectors[i] = vec
	}

	// Store in vector DB
	if s.vectorStore != nil {
		if err := s.vectorStore.Upsert(ctx, noteID, chunks, vectors); err != nil {
			return err
		}
	}

	// Save embedding references
	for i, chunk := range chunks {
		vectorID := fmt.Sprintf("%s_%d", noteID, i)
		s.repo.SaveEmbedding(ctx, uuid.MustParse(noteID), i, chunk.Text, vectorID, chunk.TokenCount)
	}

	return nil
}
