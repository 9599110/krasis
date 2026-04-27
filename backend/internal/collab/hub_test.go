package collab

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestHub_GetOrCreateRoom(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(nil, logger)

	room1 := hub.GetOrCreateRoom("note-1")
	room2 := hub.GetOrCreateRoom("note-1")

	if room1 != room2 {
		t.Fatal("same room should be returned for same note ID")
	}

	room3 := hub.GetOrCreateRoom("note-2")
	if room3 == room1 {
		t.Fatal("different note should get different room")
	}
}

func TestHub_CloseRoom(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(nil, logger)

	room := hub.GetOrCreateRoom("note-1")
	hub.CloseRoom("note-1")

	// Getting a new room should create a fresh room
	room2 := hub.GetOrCreateRoom("note-1")
	if room == room2 {
		t.Fatal("should get new room after close")
	}
}

func TestHub_RoomCount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(nil, logger)

	if hub.RoomCount() != 0 {
		t.Fatal("initial room count should be 0")
	}

	hub.GetOrCreateRoom("note-1")
	hub.GetOrCreateRoom("note-2")

	if hub.RoomCount() != 2 {
		t.Fatalf("room count want 2 got %d", hub.RoomCount())
	}
}

func TestRoom_NewRoom(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)

	if room.noteID != "note-1" {
		t.Fatalf("noteID mismatch")
	}
	if room.GetTextState() != "" {
		t.Fatal("initial state should be empty")
	}
}

func TestRoom_SetTextState(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)

	room.SetTextState("Hello, World!")

	if room.GetTextState() != "Hello, World!" {
		t.Fatalf("state mismatch: %s", room.GetTextState())
	}
}

func TestRoom_Revision(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)

	if room.GetRevision() != 0 {
		t.Fatal("initial revision should be 0")
	}

	room.IncrementRevision()

	if room.GetRevision() != 1 {
		t.Fatalf("revision want 1 got %d", room.GetRevision())
	}
}

func TestRoom_AddRemoveClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)

	client := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "TestUser",
		Sender:   make(chan WSMessage, 10),
	}

	room.register <- client
	time.Sleep(10 * time.Millisecond)

	room.mu.RLock()
	count := len(room.clients)
	room.mu.RUnlock()

	if count != 1 {
		t.Fatalf("client count want 1 got %d", count)
	}

	room.unregister <- client
	time.Sleep(10 * time.Millisecond)

	room.mu.RLock()
	count = len(room.clients)
	room.mu.RUnlock()

	if count != 0 {
		t.Fatalf("client count after remove want 0 got %d", count)
	}
}

func TestRoom_Close(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)

	room.Close()

	select {
	case _, ok := <-room.exit:
		if ok {
			t.Fatal("exit channel should be closed")
		}
	default:
		t.Fatal("exit channel should be closed immediately")
	}
}

func TestClient_SendChannel(t *testing.T) {
	client := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "TestUser",
		Sender:   make(chan WSMessage, 10),
	}

	msg := WSMessage{Type: "test", Payload: map[string]any{"data": "value"}}

	select {
	case client.Sender <- msg:
		// Success
	case <-time.After(time.Second):
		t.Fatal("send channel should not block")
	}

	select {
	case received := <-client.Sender:
		if received.Type != "test" {
			t.Fatalf("message type mismatch")
		}
	case <-time.After(time.Second):
		t.Fatal("receive should not block")
	}
}

func TestWSMessage_JSON(t *testing.T) {
	msg := WSMessage{
		Type:    "sync",
		Payload: map[string]any{"update": "test content"},
	}

	if msg.Type != "sync" {
		t.Fatalf("type mismatch")
	}
	if msg.Payload["update"] != "test content" {
		t.Fatalf("payload mismatch")
	}
}
