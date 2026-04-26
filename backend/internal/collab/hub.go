package collab

import (
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Hub manages all collaboration rooms.
type Hub struct {
	rooms   map[string]*Room
	mu      sync.RWMutex
	rdb     *redis.Client
	logger  *zap.Logger
}

func NewHub(rdb *redis.Client, logger *zap.Logger) *Hub {
	return &Hub{
		rooms:  make(map[string]*Room),
		rdb:    rdb,
		logger: logger,
	}
}

// GetOrCreateRoom returns an existing room or creates a new one for the given note.
func (h *Hub) GetOrCreateRoom(noteID string) *Room {
	h.mu.RLock()
	room, ok := h.rooms[noteID]
	h.mu.RUnlock()

	if ok {
		return room
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double check after acquiring write lock
	if room, ok = h.rooms[noteID]; ok {
		return room
	}

	room = NewRoom(noteID, h.rdb, h.logger)
	h.rooms[noteID] = room
	return room
}

// RemoveRoom removes an empty room from the hub.
func (h *Hub) RemoveRoom(noteID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, noteID)
}

// RoomCount returns the number of active rooms (for monitoring).
func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}
