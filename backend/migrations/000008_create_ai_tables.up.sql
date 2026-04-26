-- AI conversations
CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255),
    model VARCHAR(50) DEFAULT 'gpt-4',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_conv_user ON ai_conversations(user_id);

-- AI messages
CREATE TABLE ai_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,           -- user, assistant, system
    content TEXT NOT NULL,
    refs JSONB DEFAULT '[]',             -- referenced note snippets
    token_count INT,
    model VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_msg_conv ON ai_messages(conversation_id);

-- Note embeddings tracking (actual vectors stored in Qdrant)
CREATE TABLE note_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    chunk_text TEXT NOT NULL,
    vector_id VARCHAR(255),              -- Qdrant point ID
    token_count INT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    UNIQUE(note_id, chunk_index)
);

CREATE INDEX idx_note_emb_note ON note_embeddings(note_id);

-- AI models configuration
CREATE TABLE ai_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    provider VARCHAR(50) NOT NULL,           -- openai, azure, anthropic, ollama, local
    model_type VARCHAR(20) NOT NULL,         -- llm, embedding
    endpoint VARCHAR(500),
    api_key VARCHAR(500),
    model_name VARCHAR(100) NOT NULL,
    api_version VARCHAR(50),
    max_tokens INT DEFAULT 4096,
    temperature FLOAT DEFAULT 0.7,
    top_p FLOAT,
    dimensions INT,
    is_enabled BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    priority INT DEFAULT 100,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_ai_models_type ON ai_models(model_type);
CREATE INDEX idx_ai_models_enabled ON ai_models(is_enabled);

-- AI system config
CREATE TABLE ai_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

INSERT INTO ai_config (config_key, config_value, description) VALUES
('chunk_size', '{"value": 500}', 'Text chunk size in tokens'),
('chunk_overlap', '{"value": 50}', 'Chunk overlap size'),
('top_k', '{"value": 5}', 'RAG retrieval top K'),
('score_threshold', '{"value": 0.7}', 'Similarity threshold'),
('enable_rag', '{"value": true}', 'Enable RAG'),
('max_context_tokens', '{"value": 8000}', 'Max context tokens'),
('system_prompt', '{"value": "你是一个智能笔记助手。请根据以下上下文回答用户的问题。"}', 'System prompt'),
('enable_streaming', '{"value": true}', 'Enable streaming responses');
