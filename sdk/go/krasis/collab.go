package krasis

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// CollabModule manages WebSocket collaboration sessions.
type CollabModule struct {
	wsBaseURL            string
	token                func() string
	conn                 *websocket.Conn
	noteID               string
	handlers             []CollabHandler
	mu                   sync.Mutex
	reconnectAttempts    int
	maxReconnectAttempts int
	reconnectTimer       *time.Timer
}

// CollabHandler is a callback for collaboration events.
type CollabHandler func(event CollabEvent)

// CollabEvent represents a collaboration event.
type CollabEvent struct {
	Type    string
	Payload map[string]any
	UserID  string
}

// NewCollabModule creates a new collaboration module.
func NewCollabModule(wsBaseURL string, token func() string) *CollabModule {
	return &CollabModule{
		wsBaseURL:            wsBaseURL,
		token:                token,
		maxReconnectAttempts: 5,
	}
}

// Connect establishes a WebSocket connection to a note.
func (c *CollabModule) Connect(noteID string) {
	c.mu.Lock()
	c.noteID = noteID
	c.reconnectAttempts = 0
	c.mu.Unlock()
	c.doConnect()
}

func (c *CollabModule) doConnect() {
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}

	url := fmt.Sprintf("%s/ws/collab?note_id=%s&token=%s", c.wsBaseURL, c.noteID, c.token())
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		c.mu.Unlock()
		c.emit(CollabEvent{Type: "error", Payload: map[string]any{"error": err.Error()}})
		c.scheduleReconnect()
		return
	}
	c.conn = conn
	c.mu.Unlock()

	c.emit(CollabEvent{Type: "open"})

	go c.readLoop()
}

func (c *CollabModule) readLoop() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.emit(CollabEvent{Type: "close"})
			c.scheduleReconnect()
			return
		}

		var msg struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
			UserID  string         `json:"user_id"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.emit(CollabEvent{Type: msg.Type, Payload: msg.Payload, UserID: msg.UserID})
	}
}

// On registers a handler for collaboration events.
func (c *CollabModule) On(handler CollabHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

// SendSync broadcasts a document update.
func (c *CollabModule) SendSync(update string, version int) {
	c.send(map[string]any{
		"type": "sync",
		"payload": map[string]any{
			"update":  update,
			"version": version,
		},
	})
}

// SendAwareness broadcasts awareness state.
func (c *CollabModule) SendAwareness(payload map[string]any) {
	c.send(map[string]any{
		"type":    "awareness",
		"payload": payload,
	})
}

// SendPresenceQuery requests the list of online users.
func (c *CollabModule) SendPresenceQuery() {
	c.send(map[string]any{
		"type":    "awareness_query",
		"payload": map[string]any{},
	})
}

func (c *CollabModule) send(msg map[string]any) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return
	}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

func (c *CollabModule) emit(event CollabEvent) {
	c.mu.Lock()
	handlers := make([]CollabHandler, len(c.handlers))
	copy(handlers, c.handlers)
	c.mu.Unlock()
	for _, h := range handlers {
		h(event)
	}
}

func (c *CollabModule) scheduleReconnect() {
	c.mu.Lock()
	if c.reconnectAttempts >= c.maxReconnectAttempts {
		c.mu.Unlock()
		return
	}
	c.reconnectAttempts++
	delay := time.Duration(math.Min(float64(1000)*math.Pow(2, float64(c.reconnectAttempts)), 30000)) * time.Millisecond
	c.mu.Unlock()

	c.reconnectTimer = time.AfterFunc(delay, c.doConnect)
}

// Disconnect closes the connection and stops reconnecting.
func (c *CollabModule) Disconnect() {
	if c.reconnectTimer != nil {
		c.reconnectTimer.Stop()
	}
	c.mu.Lock()
	c.reconnectAttempts = c.maxReconnectAttempts
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}
