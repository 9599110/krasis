package collab

import (
	"go.uber.org/zap"
)

func TestRoom_ApplyOperation_Insert(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)
	room.SetTextState("Hello")

	op := Operation{
		ClientID: "user-1",
		Revision: 0,
		NoteID:   "note-1",
		Ops: []TextOp{
			{Type: OpInsert, Pos: 5, Text: ", World"},
		},
	}

	transformed, payload := room.ApplyOperation(op)

	expected := "Hello, World"
	if room.GetTextState() != expected {
		t.Fatalf("expected %q, got %q", expected, room.GetTextState())
	}

	if transformed.Revision != 1 {
		t.Fatalf("revision want 1 got %d", transformed.Revision)
	}

	if _, ok := payload["revision"].(int); !ok {
		t.Fatal("payload should contain revision")
	}
}

func TestRoom_ApplyOperation_Delete(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)
	room.SetTextState("Hello, World")

	op := Operation{
		ClientID: "user-1",
		Revision: 0,
		NoteID:   "note-1",
		Ops: []TextOp{
			{Type: OpDelete, Pos: 5, Length: 2},
		},
	}

	room.ApplyOperation(op)

	expected := "HelloWorld"
	if room.GetTextState() != expected {
		t.Fatalf("expected %q, got %q", expected, room.GetTextState())
	}
}

func TestRoom_ApplyOperation_Retain(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	room := NewRoom("note-1", nil, logger)
	room.SetTextState("Hello, World")

	op := Operation{
		ClientID: "user-1",
		Revision: 0,
		NoteID:   "note-1",
		Ops: []TextOp{
			{Type: OpRetain, Length: 5},
		},
	}

	room.ApplyOperation(op)

	// Retain should not change the text
	expected := "Hello, World"
	if room.GetTextState() != expected {
		t.Fatalf("expected %q, got %q", expected, room.GetTextState())
	}
}

func TestTransform_InsertInsert(t *testing.T) {
	opA := Operation{
		ClientID: "A",
		Revision: 0,
		Ops:      []TextOp{{Type: OpInsert, Pos: 0, Text: "A"}},
	}
	opB := Operation{
		ClientID: "B",
		Revision: 0,
		Ops:      []TextOp{{Type: OpInsert, Pos: 0, Text: "B"}},
	}

	aPrime, bPrime := Transform(opA, opB)

	// Both should have their inserts at position 0
	if len(aPrime.Ops) != 1 || aPrime.Ops[0].Text != "A" {
		t.Fatalf("aPrime mismatch")
	}
	if len(bPrime.Ops) != 1 || bPrime.Ops[0].Text != "B" || bPrime.Ops[0].Pos != 1 {
		t.Fatalf("bPrime mismatch: %+v", bPrime.Ops[0])
	}
}

func TestTransform_InsertDelete(t *testing.T) {
	opInsert := Operation{
		ClientID: "A",
		Revision: 0,
		Ops:      []TextOp{{Type: OpInsert, Pos: 0, Text: "X"}},
	}
	opDelete := Operation{
		ClientID: "B",
		Revision: 0,
		Ops:      []TextOp{{Type: OpDelete, Pos: 0, Length: 1}},
	}

	insertPrime, deletePrime := Transform(opInsert, opDelete)

	// Insert should shift position
	if insertPrime.Ops[0].Pos != 1 {
		t.Fatalf("insertPrime pos want 1 got %d", insertPrime.Ops[0].Pos)
	}

	// Delete should stay at position 0 (already deleted before insert was applied)
	if deletePrime.Ops[0].Pos != 0 {
		t.Fatalf("deletePrime pos want 0 got %d", deletePrime.Ops[0].Pos)
	}
}

func TestApply_Insert(t *testing.T) {
	doc := "Hello"

	op := Operation{
		Ops: []TextOp{{Type: OpInsert, Pos: 5, Text: ", World"}},
	}

	result, err := Apply(doc, op)
	if err != nil {
		t.Fatal(err)
	}

	if result != "Hello, World" {
		t.Fatalf("expected %q, got %q", "Hello, World", result)
	}
}

func TestApply_Delete(t *testing.T) {
	doc := "Hello, World"

	op := Operation{
		Ops: []TextOp{{Type: OpDelete, Pos: 5, Length: 2}},
	}

	result, err := Apply(doc, op)
	if err != nil {
		t.Fatal(err)
	}

	if result != "HelloWorld" {
		t.Fatalf("expected %q, got %q", "HelloWorld", result)
	}
}

func TestApply_Retain(t *testing.T) {
	doc := "Hello"

	op := Operation{
		Ops: []TextOp{{Type: OpRetain, Length: 5}},
	}

	result, err := Apply(doc, op)
	if err != nil {
		t.Fatal(err)
	}

	if result != doc {
		t.Fatalf("expected %q, got %q", doc, result)
	}
}

func TestApply_InvalidPos(t *testing.T) {
	doc := "Hello"

	op := Operation{
		Ops: []TextOp{{Type: OpInsert, Pos: 100, Text: "X"}},
	}

	_, err := Apply(doc, op)
	if err == nil {
		t.Fatal("expected error for invalid position")
	}
}

func TestTextOp_String(t *testing.T) {
	tests := []struct {
		op      TextOp
		prefix  string
	}{
		{TextOp{Type: OpInsert, Pos: 5, Text: "test"}, "insert"},
		{TextOp{Type: OpDelete, Pos: 5, Length: 3}, "delete"},
		{TextOp{Type: OpRetain, Length: 5}, "retain"},
	}

	for _, tt := range tests {
		s := tt.op.String()
		if s == "" {
			t.Fatalf("String() should not be empty for %s", tt.prefix)
		}
	}
}
