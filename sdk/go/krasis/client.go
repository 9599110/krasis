package krasis

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

// Config holds the SDK configuration.
type Config struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// Client is the base HTTP client for all API requests.
type Client struct {
	baseURL string
	token   string
	mu      sync.RWMutex
	http    *http.Client
}

// NewClient creates a new API client.
func NewClient(cfg Config) *Client {
	c := &Client{
		baseURL: cfg.BaseURL,
		token:   cfg.Token,
	}
	if cfg.HTTP != nil {
		c.http = cfg.HTTP
	} else {
		c.http = &http.Client{Timeout: 30 * time.Second}
	}
	return c
}

// SetToken updates the authentication token.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
}

// ClearToken removes the authentication token.
func (c *Client) ClearToken() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

// Token returns the current authentication token.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// IsAuthenticated reports whether a token is set.
func (c *Client) IsAuthenticated() bool {
	return c.Token() != ""
}

// Get performs an HTTP GET request.
func (c *Client) Get(path string, out any) error {
	return c.do(http.MethodGet, path, nil, out)
}

// Post performs an HTTP POST request.
func (c *Client) Post(path string, body any, out any) error {
	return c.do(http.MethodPost, path, body, out)
}

// Put performs an HTTP PUT request.
func (c *Client) Put(path string, body any, out any, headers map[string]string) error {
	return c.doWithHeaders(http.MethodPut, path, body, out, headers)
}

// Delete performs an HTTP DELETE request.
func (c *Client) Delete(path string, out any) error {
	return c.do(http.MethodDelete, path, nil, out)
}

func (c *Client) do(method, path string, body any, out any) error {
	return c.doWithHeaders(method, path, body, out, nil)
}

func (c *Client) doWithHeaders(method, path string, body any, out any, headers map[string]string) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := c.Token(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil {
			return nil
		}
		var wrapper struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
			return err
		}
		return json.Unmarshal(wrapper.Data, out)
	}

	// Error handling
	var errResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	json.NewDecoder(resp.Body).Decode(&errResp)

	switch resp.StatusCode {
	case 401:
		return ErrAuthentication
	case 404:
		return ErrNotFound
	case 409:
		return &VersionConflictError{ServerVersion: 0}
	case 429:
		return ErrRateLimit
	default:
		return &APIError{
			Code:       errResp.Code,
			Message:    errResp.Message,
			StatusCode: resp.StatusCode,
		}
	}
}

// Paginated is a generic response wrapper for paginated results.
type Paginated[T any] struct {
	Items []T   `json:"items"`
	Total int   `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}
