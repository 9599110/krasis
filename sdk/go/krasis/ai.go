package krasis

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AIModule handles AI question operations.
type AIModule struct {
	client *Client
}

// NewAIModule creates a new AI module.
func NewAIModule(c *Client) *AIModule {
	return &AIModule{client: c}
}

// Ask sends a question and returns the answer.
func (a *AIModule) Ask(req AskRequest) (*AskResponse, error) {
	var resp AskResponse
	err := a.client.Post("/ai/ask", req, &resp)
	return &resp, err
}

// AskStream sends a question and returns a stream of tokens.
func (a *AIModule) AskStream(ctx context.Context, req AskRequest) (<-chan string, <-chan error, error) {
	req.Stream = true

	data, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.client.baseURL+"/ai/ask/stream", bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token := a.client.Token(); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := a.client.http.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, nil, &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	tokenCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer resp.Body.Close()
		defer close(tokenCh)
		defer close(errCh)

		scanner := bufio.NewScanner(resp.Body)
		var lastEvent string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				lastEvent = strings.TrimPrefix(line, "event: ")
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				payload := strings.TrimPrefix(line, "data: ")
				if lastEvent == "token" {
					var d struct {
						Token string `json:"token"`
					}
					if err := json.Unmarshal([]byte(payload), &d); err == nil && d.Token != "" {
						select {
						case tokenCh <- d.Token:
						case <-ctx.Done():
							return
						}
					}
				} else if lastEvent == "done" {
					return
				}
			}
		}
	}()

	return tokenCh, errCh, nil
}

// ListConversations returns all conversations.
func (a *AIModule) ListConversations() ([]Conversation, error) {
	var convs []Conversation
	err := a.client.Get("/ai/conversations", &convs)
	return convs, err
}

// GetMessages returns all messages in a conversation.
func (a *AIModule) GetMessages(conversationID string) ([]Message, error) {
	var msgs []Message
	err := a.client.Get("/ai/conversations/"+conversationID+"/messages", &msgs)
	return msgs, err
}

// CreateConversation creates a new conversation.
func (a *AIModule) CreateConversation(title, model string) (*Conversation, error) {
	body := map[string]string{}
	if title != "" {
		body["title"] = title
	}
	if model != "" {
		body["model"] = model
	}
	var conv Conversation
	err := a.client.Post("/ai/conversations", body, &conv)
	return &conv, err
}
