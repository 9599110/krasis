package collab

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// StateKey returns the Redis key for a note's collaboration state.
func StateKey(noteID string) string {
	return "collab:state:" + noteID
}

// PersistState saves the document state to Redis with a TTL.
func PersistState(ctx context.Context, rdb *redis.Client, noteID string, state []byte) error {
	return rdb.Set(ctx, StateKey(noteID), state, 24*time.Hour).Err()
}

// LoadState retrieves the document state from Redis.
func LoadState(ctx context.Context, rdb *redis.Client, noteID string) ([]byte, error) {
	return rdb.Get(ctx, StateKey(noteID)).Bytes()
}
