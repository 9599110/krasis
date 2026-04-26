package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GenerateOptions for LLM generation
type GenerateOptions struct {
	Temperature float64
	MaxTokens   int
	Stream      bool
}

// LLM interface for text generation
type LLM interface {
	Generate(ctx context.Context, messages []MessageParam, options *GenerateOptions) (string, error)
	GenerateStream(ctx context.Context, messages []MessageParam, options *GenerateOptions) (<-chan string, error)
}

// MessageParam represents a single message in a conversation
type MessageParam struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMFactory creates LLM instances based on model configuration
type LLMFactory struct {
	httpClient *http.Client
}

func NewLLMFactory() *LLMFactory {
	return &LLMFactory{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (f *LLMFactory) GetLLM(model *AIModel) LLM {
	switch model.Provider {
	case "openai":
		return NewOpenAILLM(model, f.httpClient)
	case "azure":
		return NewAzureLLM(model, f.httpClient)
	case "anthropic":
		return NewAnthropicLLM(model, f.httpClient)
	case "ollama":
		return NewOllamaLLM(model, f.httpClient)
	default:
		return NewOpenAILLM(model, f.httpClient)
	}
}

// OpenAILLM calls OpenAI-compatible chat completions API
type OpenAILLM struct {
	apiKey      string
	modelName   string
	endpoint    string
	maxTokens   int
	temperature float64
	client      *http.Client
}

func NewOpenAILLM(model *AIModel, client *http.Client) *OpenAILLM {
	endpoint := model.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	maxTokens := model.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	return &OpenAILLM{
		apiKey:      model.APIKey,
		modelName:   model.ModelName,
		endpoint:    endpoint,
		maxTokens:   maxTokens,
		temperature: model.Temperature,
		client:      client,
	}
}

func (l *OpenAILLM) Generate(ctx context.Context, messages []MessageParam, options *GenerateOptions) (string, error) {
	temperature := l.temperature
	maxTokens := l.maxTokens
	if options != nil {
		if options.Temperature > 0 {
			temperature = options.Temperature
		}
		if options.MaxTokens > 0 {
			maxTokens = options.MaxTokens
		}
	}

	reqBody := map[string]interface{}{
		"model":       l.modelName,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM API returned status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}
	return result.Choices[0].Message.Content, nil
}

func (l *OpenAILLM) GenerateStream(ctx context.Context, messages []MessageParam, options *GenerateOptions) (<-chan string, error) {
	temperature := l.temperature
	maxTokens := l.maxTokens
	if options != nil {
		if options.Temperature > 0 {
			temperature = options.Temperature
		}
		if options.MaxTokens > 0 {
			maxTokens = options.MaxTokens
		}
	}

	reqBody := map[string]interface{}{
		"model":       l.modelName,
		"messages":    messages,
		"stream":      true,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if data == "[DONE]" {
				break
			}

			var delta struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if json.Unmarshal([]byte(data), &delta) == nil {
				if len(delta.Choices) > 0 && delta.Choices[0].Delta.Content != "" {
					ch <- delta.Choices[0].Delta.Content
				}
			}
		}
	}()

	return ch, nil
}

// OllamaLLM calls local Ollama API
type OllamaLLM struct {
	endpoint    string
	modelName   string
	temperature float64
	maxTokens   int
	client      *http.Client
}

func NewOllamaLLM(model *AIModel, client *http.Client) *OllamaLLM {
	endpoint := model.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	return &OllamaLLM{
		endpoint:    endpoint,
		modelName:   model.ModelName,
		temperature: model.Temperature,
		maxTokens:   model.MaxTokens,
		client:      client,
	}
}

func (l *OllamaLLM) Generate(ctx context.Context, messages []MessageParam, options *GenerateOptions) (string, error) {
	// Ollama uses prompt format, combine messages
	var prompt string
	for _, m := range messages {
		prompt += m.Content + "\n"
	}

	reqBody := map[string]interface{}{
		"model":  l.modelName,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": l.temperature,
			"num_predict": l.maxTokens,
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint+"/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Response, nil
}

func (l *OllamaLLM) GenerateStream(ctx context.Context, messages []MessageParam, options *GenerateOptions) (<-chan string, error) {
	var prompt string
	for _, m := range messages {
		prompt += m.Content + "\n"
	}

	reqBody := map[string]interface{}{
		"model":  l.modelName,
		"prompt": prompt,
		"stream": true,
		"options": map[string]interface{}{
			"temperature": l.temperature,
			"num_predict": l.maxTokens,
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint+"/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for decoder.More() {
			var response struct {
				Response string `json:"response"`
				Done     bool   `json:"done"`
			}
			if err := decoder.Decode(&response); err != nil {
				break
			}
			ch <- response.Response
			if response.Done {
				break
			}
		}
	}()

	return ch, nil
}

// AzureLLM calls Azure OpenAI Service
type AzureLLM struct {
	apiKey      string
	endpoint    string
	modelName   string
	apiVersion  string
	maxTokens   int
	temperature float64
	client      *http.Client
}

func NewAzureLLM(model *AIModel, client *http.Client) *AzureLLM {
	apiVersion := model.APIVersion
	if apiVersion == "" {
		apiVersion = "2024-02-01"
	}
	return &AzureLLM{
		apiKey:      model.APIKey,
		endpoint:    model.Endpoint,
		modelName:   model.ModelName,
		apiVersion:  apiVersion,
		maxTokens:   model.MaxTokens,
		temperature: model.Temperature,
		client:      client,
	}
}

func (l *AzureLLM) Generate(ctx context.Context, messages []MessageParam, options *GenerateOptions) (string, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		l.endpoint, l.modelName, l.apiVersion)

	reqBody := map[string]interface{}{
		"messages":    messages,
		"max_tokens":  l.maxTokens,
		"temperature": l.temperature,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from Azure LLM")
	}
	return result.Choices[0].Message.Content, nil
}

func (l *AzureLLM) GenerateStream(ctx context.Context, messages []MessageParam, options *GenerateOptions) (<-chan string, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		l.endpoint, l.modelName, l.apiVersion)

	reqBody := map[string]interface{}{
		"messages":    messages,
		"stream":      true,
		"max_tokens":  l.maxTokens,
		"temperature": l.temperature,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if data == "[DONE]" {
				break
			}

			var delta struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if json.Unmarshal([]byte(data), &delta) == nil {
				if len(delta.Choices) > 0 && delta.Choices[0].Delta.Content != "" {
					ch <- delta.Choices[0].Delta.Content
				}
			}
		}
	}()

	return ch, nil
}

// AnthropicLLM calls Claude API
type AnthropicLLM struct {
	apiKey      string
	modelName   string
	endpoint    string
	maxTokens   int
	temperature float64
	client      *http.Client
}

func NewAnthropicLLM(model *AIModel, client *http.Client) *AnthropicLLM {
	endpoint := model.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1/messages"
	}
	return &AnthropicLLM{
		apiKey:      model.APIKey,
		modelName:   model.ModelName,
		endpoint:    endpoint,
		maxTokens:   model.MaxTokens,
		temperature: model.Temperature,
		client:      client,
	}
}

func (l *AnthropicLLM) Generate(ctx context.Context, messages []MessageParam, options *GenerateOptions) (string, error) {
	maxTokens := l.maxTokens
	if options != nil && options.MaxTokens > 0 {
		maxTokens = options.MaxTokens
	}

	// Convert messages to Anthropic format (system + messages)
	var systemMsg string
	var msgs []map[string]interface{}
	for _, m := range messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": m.Content})
		}
	}

	reqBody := map[string]interface{}{
		"model":         l.modelName,
		"messages":      msgs,
		"max_tokens":    maxTokens,
		"temperature":   l.temperature,
	}
	if systemMsg != "" {
		reqBody["system"] = systemMsg
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", l.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}
	return result.Content[0].Text, nil
}

func (l *AnthropicLLM) GenerateStream(ctx context.Context, messages []MessageParam, options *GenerateOptions) (<-chan string, error) {
	maxTokens := l.maxTokens
	if options != nil && options.MaxTokens > 0 {
		maxTokens = options.MaxTokens
	}

	var systemMsg string
	var msgs []map[string]interface{}
	for _, m := range messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			msgs = append(msgs, map[string]interface{}{"role": m.Role, "content": m.Content})
		}
	}

	reqBody := map[string]interface{}{
		"model":         l.modelName,
		"messages":      msgs,
		"max_tokens":    maxTokens,
		"temperature":   l.temperature,
		"stream":        true,
	}
	if systemMsg != "" {
		reqBody["system"] = systemMsg
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", l.endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", l.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if data == "" {
				continue
			}

			var event struct {
				Type string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if json.Unmarshal([]byte(data), &event) == nil {
				if event.Type == "content_block_delta" && event.Delta.Text != "" {
					ch <- event.Delta.Text
				}
			}
		}
	}()

	return ch, nil
}
