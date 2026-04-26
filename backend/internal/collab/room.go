package collab

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client represents a connected WebSocket user in a room.
type Client struct {
	ID       string
	UserID   string
	Username string
	Sender   chan WSMessage
}

// Room manages all clients connected to a single note.
type Room struct {
	noteID        string
	clients       map[string]*Client
	state         []byte // current document state (Yjs update bytes)
	textState     string // current document text for OT
	revision      int    // operation revision counter
	operationLog  []Operation
	mu            sync.RWMutex
	rdb           *redis.Client
	logger        *zap.Logger
	broadcast     chan WSMessage
	register      chan *Client
	unregister    chan *Client
	done          chan struct{}
	once          sync.Once
}

func NewRoom(noteID string, rdb *redis.Client, logger *zap.Logger) *Room {
	r := &Room{
		noteID:     noteID,
		clients:    make(map[string]*Client),
		rdb:        rdb,
		logger:     logger,
		broadcast:  make(chan WSMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
	}

	// Load persisted state from Redis
	if r.rdb != nil {
		go r.loadState()
	}
	// Start the run loop
	go r.run()
	return r
}

func (r *Room) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case client := <-r.register:
			r.mu.Lock()
			r.clients[client.ID] = client
			count := len(r.clients)
			r.mu.Unlock()

			// Send current state to the new client
			if len(r.state) > 0 {
				client.Sender <- WSMessage{
					Type: "sync",
					Payload: map[string]any{
						"update":  r.state,
						"version": 1,
					},
				}
			}

			// Broadcast presence update
			r.broadcastPresence()

			r.logger.Info("client joined room",
				zap.String("note_id", r.noteID),
				zap.String("user_id", client.UserID),
				zap.Int("clients", count),
			)

		case client := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.clients[client.ID]; ok {
				delete(r.clients, client.ID)
				close(client.Sender)
			}
			count := len(r.clients)
			r.mu.Unlock()

			r.broadcastPresence()

			r.logger.Info("client left room",
				zap.String("note_id", r.noteID),
				zap.String("user_id", client.UserID),
				zap.Int("clients", count),
			)

			// Remove empty room after a delay
			if count == 0 {
				go func() {
					time.Sleep(5 * time.Minute)
					r.mu.RLock()
					empty := len(r.clients) == 0
					r.mu.RUnlock()
					if empty {
						close(r.done)
					}
				}()
			}

		case msg := <-r.broadcast:
			r.mu.RLock()
			for id, client := range r.clients {
				if msg.UserID != "" && msg.UserID == id {
					continue // skip sender
				}
				select {
				case client.Sender <- msg:
				default:
					// buffer full, skip
				}
			}
			r.mu.RUnlock()

		case <-ticker.C:
			if r.rdb != nil {
				r.persistState()
			}
		}
	}
}

func (r *Room) Broadcast(msg WSMessage, excludeClientID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, client := range r.clients {
		if id == excludeClientID {
			continue
		}
		select {
		case client.Sender <- msg:
		default:
		}
	}
}

func (r *Room) BroadcastAll(msg WSMessage) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		select {
		case client.Sender <- msg:
		default:
		}
	}
}

func (r *Room) broadcastPresence() {
	r.mu.RLock()
	users := make([]map[string]string, 0, len(r.clients))
	for _, c := range r.clients {
		users = append(users, map[string]string{
			"user_id":  c.UserID,
			"username": c.Username,
		})
	}
	r.mu.RUnlock()

	r.BroadcastAll(WSMessage{
		Type: "presence",
		Payload: map[string]any{
			"users": users,
		},
	})
}

func (r *Room) loadState() {
	ctx := context.Background()
	data, err := r.rdb.Get(ctx, "collab:state:"+r.noteID).Bytes()
	if err == nil && len(data) > 0 {
		r.mu.Lock()
		r.state = data
		r.mu.Unlock()
	}
}

func (r *Room) persistState() {
	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()

	if len(state) == 0 {
		return
	}

	ctx := context.Background()
	r.rdb.Set(ctx, "collab:state:"+r.noteID, state, 24*time.Hour)
}

func (r *Room) UpdateState(update []byte) {
	r.mu.Lock()
	r.state = update
	r.mu.Unlock()
}

// ApplyOperation applies an OT operation to the room's text state.
// It transforms the incoming operation against any concurrent operations
// since the client's base revision, then applies it atomically.
// Returns (transformedOp, broadcastPayload).
func (r *Room) ApplyOperation(op Operation) (Operation, map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Transform against any operations that happened since the client's revision
	clientRev := op.Revision
	serverRev := r.revision

	if clientRev < serverRev {
		// Client is behind: transform against intervening operations
		for i := clientRev; i < serverRev; i++ {
			if i >= len(r.operationLog) {
				break
			}
			concurrent := r.operationLog[i]
			var transformedOp, _ Operation
			op, transformedOp = Transform(op, concurrent)
			_ = transformedOp
		}
	}

	// Apply the (potentially transformed) operation to text state
	newText, err := Apply(r.textState, op)
	if err != nil {
		r.logger.Error("failed to apply operation", zap.Error(err))
		op = Operation{
			ClientID: op.ClientID,
			Revision: r.revision + 1,
			NoteID:   r.noteID,
			Ops:      nil, // no-op
		}
	} else {
		r.textState = newText
	}

	op.Revision = r.revision + 1
	r.operationLog = append(r.operationLog, op)

	// Cap the log to prevent memory growth (keep last 1000 ops)
	if len(r.operationLog) > 1000 {
		r.operationLog = r.operationLog[len(r.operationLog)-1000:]
	}

	payload := map[string]any{
		"ops":      op.Ops,
		"revision": op.Revision,
		"client_id": op.ClientID,
	}

	return op, payload
}

// SetTextState sets the current text state (e.g., loaded from database).
func (r *Room) SetTextState(text string) {
	r.mu.Lock()
	r.textState = text
	r.mu.Unlock()
}

// GetTextState returns the current text state.
func (r *Room) GetTextState() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.textState
}

// GetRevision returns the current operation revision.
func (r *Room) GetRevision() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.revision
}

func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

type WSMessage struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
	UserID  string         `json:"user_id,omitempty"`
}

func (m WSMessage) MarshalJSON() ([]byte, error) {
	type Alias WSMessage
	return json.Marshal((*Alias)(&m))
}
