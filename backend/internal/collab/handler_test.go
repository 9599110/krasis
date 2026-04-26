package collab

import (
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

func TestHandleMessageSync(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(nil, logger)
	handler := NewHandler(hub, nil, logger)

	room := hub.GetOrCreateRoom("test-note-1")

	client := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "TestUser",
		Sender:   make(chan WSMessage, 10),
	}
	room.register <- client

	room.mu.RLock()
	count := len(room.clients)
	room.mu.RUnlock()
	if count != 1 {
		t.Fatalf("expected 1 client, got %d", count)
	}

	handler.handleMessage(WSMessage{
		Type: "sync",
		Payload: map[string]any{
			"update": []byte("initial state"),
		},
	}, client, room)

	room.mu.RLock()
	state := string(room.state)
	room.mu.RUnlock()

	if state != "initial state" {
		t.Fatalf("expected state 'initial state', got '%s'", state)
	}
}

func TestRoomBroadcast(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("test-broadcast", nil, logger)

	client1 := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "Alice",
		Sender:   make(chan WSMessage, 10),
	}
	client2 := &Client{
		ID:       "client-2",
		UserID:   "user-2",
		Username: "Bob",
		Sender:   make(chan WSMessage, 10),
	}

	// Directly add clients to the room map (bypass async registration)
	room.mu.Lock()
	room.clients[client1.ID] = client1
	room.clients[client2.ID] = client2
	room.mu.Unlock()

	room.Broadcast(WSMessage{
		Type: "awareness",
		Payload: map[string]any{
			"user_id": "user-1",
		},
	}, client1.ID)

	// Client2 should receive it
	select {
	case msg := <-client2.Sender:
		if msg.Type != "awareness" {
			t.Fatalf("expected awareness, got %s", msg.Type)
		}
	default:
		t.Fatal("client2 did not receive broadcast")
	}

	// Client1 should NOT receive it (excluded)
	select {
	case msg := <-client1.Sender:
		t.Fatalf("client1 should not receive excluded message, got %v", msg)
	default:
		// expected
	}
}

func TestRoomPresence(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("test-presence", nil, logger)

	client1 := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "Alice",
		Sender:   make(chan WSMessage, 10),
	}
	client2 := &Client{
		ID:       "client-2",
		UserID:   "user-2",
		Username: "Bob",
		Sender:   make(chan WSMessage, 10),
	}

	room.mu.Lock()
	room.clients[client1.ID] = client1
	room.clients[client2.ID] = client2
	room.mu.Unlock()

	room.broadcastPresence()

	// Both clients should receive presence
	for _, ch := range []chan WSMessage{client1.Sender, client2.Sender} {
		select {
		case msg := <-ch:
			if msg.Type != "presence" {
				t.Fatalf("expected presence, got %s", msg.Type)
			}
		default:
			t.Fatal("client did not receive presence")
		}
	}
}

func TestHandleMessageOperation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("test-op", nil, logger)

	room.SetTextState("hello world")

	client1 := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "Alice",
		Sender:   make(chan WSMessage, 10),
	}
	room.register <- client1

	op := Operation{
		ClientID: "user-1",
		Revision: 0,
		NoteID:   "test-op",
		Ops: []TextOp{
			{Type: OpInsert, Pos: 5, Text: " beautiful"},
		},
	}

	transformedOp, payload := room.ApplyOperation(op)

	// Check state was updated
	if room.GetTextState() != "hello beautiful world" {
		t.Fatalf("expected 'hello beautiful world', got '%s'", room.GetTextState())
	}

	// Check revision incremented
	if transformedOp.Revision != 1 {
		t.Fatalf("expected revision=1, got %d", transformedOp.Revision)
	}

	// Check payload has expected fields
	if rev, ok := payload["revision"].(int); !ok || rev != 1 {
		t.Fatalf("expected payload revision=1, got %v", payload["revision"])
	}
}

func TestHandleMessageUnknownType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(nil, logger)
	handler := NewHandler(hub, nil, logger)

	room := hub.GetOrCreateRoom("test-note-5")

	client := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		Username: "TestUser",
		Sender:   make(chan WSMessage, 10),
	}
	room.register <- client

	// Should not panic or error on unknown type
	handler.handleMessage(WSMessage{
		Type:    "unknown_type",
		Payload: map[string]any{},
	}, client, room)
}

func TestOperationConvergence(t *testing.T) {
	// Simulate concurrent edits: both users type at the same position
	doc := "hello"

	opA := Operation{
		ClientID: "A",
		Revision: 0,
		NoteID:   "note-1",
		Ops:      []TextOp{{Type: OpInsert, Pos: 5, Text: " world"}},
	}
	opB := Operation{
		ClientID: "B",
		Revision: 0,
		NoteID:   "note-1",
		Ops:      []TextOp{{Type: OpInsert, Pos: 5, Text: "!"}},
	}

	// Transform and apply
	aPrime, bPrime := Transform(opA, opB)

	resultA, err := Apply(doc, opA)
	if err != nil {
		t.Fatalf("Apply A failed: %v", err)
	}
	resultA, err = Apply(resultA, bPrime)
	if err != nil {
		t.Fatalf("Apply B' failed: %v", err)
	}

	resultB, err := Apply(doc, opB)
	if err != nil {
		t.Fatalf("Apply B failed: %v", err)
	}
	resultB, err = Apply(resultB, aPrime)
	if err != nil {
		t.Fatalf("Apply A' failed: %v", err)
	}

	if resultA != resultB {
		t.Fatalf("convergence failed: A->B'=%q, B->A'=%q", resultA, resultB)
	}
}

func TestOTJSONRoundTrip(t *testing.T) {
	op := Operation{
		ClientID: "user-1",
		Revision: 42,
		NoteID:   "note-123",
		Ops: []TextOp{
			{Type: OpInsert, Pos: 0, Text: "hello"},
			{Type: OpDelete, Pos: 5, Length: 3},
		},
	}

	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	parsed, err := ParseOperation(data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if parsed.ClientID != op.ClientID {
		t.Fatalf("client_id mismatch")
	}
	if parsed.Revision != op.Revision {
		t.Fatalf("revision mismatch")
	}
	if len(parsed.Ops) != len(op.Ops) {
		t.Fatalf("ops count mismatch")
	}
	if parsed.Ops[0].Type != OpInsert || parsed.Ops[1].Type != OpDelete {
		t.Fatalf("op types mismatch")
	}
}
