package collab

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/krasis/krasis/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket
	},
}

// Handler handles WebSocket upgrade and message routing.
type Handler struct {
	hub        *Hub
	jwtManager *auth.JWTManager
	logger     *zap.Logger
}

func NewHandler(hub *Hub, jwtManager *auth.JWTManager, logger *zap.Logger) *Handler {
	return &Handler{
		hub:        hub,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// HandleCollab upgrades HTTP to WebSocket and manages the connection lifecycle.
func (h *Handler) HandleCollab(c *gin.Context) {
	noteID := c.Query("note_id")
	if noteID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "note_id is required"})
		return
	}

	// Authenticate via token query param
	tokenStr := c.Query("token")
	if tokenStr == "" {
		// Try Authorization header
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	claims, err := h.jwtManager.Validate(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	clientID := uuid.New().String()
	room := h.hub.GetOrCreateRoom(noteID)

	client := &Client{
		ID:       clientID,
		UserID:   claims.UserID,
		Username: claims.UserID, // Will be overridden by awareness message
		Sender:   make(chan WSMessage, 64),
	}

	room.register <- client

	var wg sync.WaitGroup
	wg.Add(2)

	// Reader goroutine
	go func() {
		defer wg.Done()
		h.readLoop(conn, client, room)
	}()

	// Writer goroutine
	go func() {
		defer wg.Done()
		h.writeLoop(conn, client, room)
	}()

	// Wait for one of the goroutines to finish, then clean up
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-c.Request.Context().Done():
	}

	room.unregister <- client
	conn.Close()
}

func (h *Handler) readLoop(conn *websocket.Conn, client *Client, room *Room) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Info("read loop recovered", zap.Any("recover", r))
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				h.logger.Warn("unexpected close", zap.Error(err))
			}
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Warn("invalid message", zap.Error(err))
			continue
		}

		h.handleMessage(msg, client, room)
	}
}

func (h *Handler) handleMessage(msg WSMessage, client *Client, room *Room) {
	switch msg.Type {
	case "sync":
		if payload, ok := msg.Payload["update"]; ok {
			// Check if this is an OT operation (byte array with ops) or raw state update
			switch p := payload.(type) {
			case []byte:
				room.UpdateState(p)
				msg.UserID = client.UserID
				room.Broadcast(msg, client.ID)
			case map[string]any:
				// OT operation-based sync
				opsJSON, _ := json.Marshal(p)
				op, err := ParseOperation(opsJSON)
				if err != nil {
					h.logger.Warn("invalid operation", zap.Error(err))
					client.Sender <- WSMessage{
						Type: "error",
						Payload: map[string]any{
							"message": "invalid operation: " + err.Error(),
						},
					}
					return
				}

				transformedOp, ackOp := room.ApplyOperation(op)
				msg.UserID = client.UserID
				msg.Payload["update"] = ackOp
				room.Broadcast(msg, client.ID)

				// Send ack to the author with the transformed version
				client.Sender <- WSMessage{
					Type: "ack",
					Payload: map[string]any{
						"revision": transformedOp.Revision,
					},
				}
			}
		}

	case "operation":
		// Dedicated operation message type for OT
		opsJSON, err := json.Marshal(msg.Payload)
		if err != nil {
			h.logger.Warn("failed to marshal operation", zap.Error(err))
			return
		}
		op, err := ParseOperation(opsJSON)
		if err != nil {
			h.logger.Warn("invalid operation", zap.Error(err))
			client.Sender <- WSMessage{
				Type: "error",
				Payload: map[string]any{
					"message": "invalid operation: " + err.Error(),
				},
			}
			return
		}

		transformedOp, ackOp := room.ApplyOperation(op)
		msg.UserID = client.UserID
		msg.Payload = ackOp
		room.Broadcast(msg, client.ID)

		client.Sender <- WSMessage{
			Type: "ack",
			Payload: map[string]any{
				"revision": transformedOp.Revision,
			},
		}

	case "awareness":
		// Broadcast awareness to all other clients
		msg.Payload["user_id"] = client.UserID
		msg.Payload["username"] = client.Username
		room.Broadcast(msg, client.ID)

	case "awareness_query":
		// Send current awareness of all clients back to the requester
		room.mu.RLock()
		users := make([]map[string]any, 0, len(room.clients))
		for _, c := range room.clients {
			users = append(users, map[string]any{
				"user_id":  c.UserID,
				"username": c.Username,
			})
		}
		room.mu.RUnlock()

		client.Sender <- WSMessage{
			Type: "presence",
			Payload: map[string]any{
				"users": users,
			},
		}

	default:
		h.logger.Warn("unknown message type", zap.String("type", msg.Type))
	}
}

func (h *Handler) writeLoop(conn *websocket.Conn, client *Client, room *Room) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.Sender:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(msg); err != nil {
				h.logger.Warn("write message failed", zap.Error(err))
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
