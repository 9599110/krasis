package collab

import (
	"testing"
)

func TestApplyInsert(t *testing.T) {
	doc := "hello world"
	op := Operation{
		ClientID: "user1",
		Revision: 1,
		Ops: []TextOp{
			{Type: OpInsert, Pos: 5, Text: " beautiful"},
		},
	}
	result, err := Apply(doc, op)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result != "hello beautiful world" {
		t.Fatalf("expected 'hello beautiful world', got '%s'", result)
	}
}

func TestApplyDelete(t *testing.T) {
	doc := "hello beautiful world"
	op := Operation{
		ClientID: "user1",
		Revision: 1,
		Ops: []TextOp{
			{Type: OpDelete, Pos: 5, Length: 10},
		},
	}
	result, err := Apply(doc, op)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", result)
	}
}

func TestApplyReplace(t *testing.T) {
	doc := "hello world"
	op := Operation{
		ClientID: "user1",
		Revision: 1,
		Ops: []TextOp{
			{Type: OpReplace, Pos: 6, Length: 5, Text: "gophers"},
		},
	}
	result, err := Apply(doc, op)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result != "hello gophers" {
		t.Fatalf("expected 'hello gophers', got '%s'", result)
	}
}

func TestApplyMultipleOps(t *testing.T) {
	doc := "hello world"
	op := Operation{
		ClientID: "user1",
		Revision: 1,
		Ops: []TextOp{
			{Type: OpInsert, Pos: 5, Text: " beautiful"},
			{Type: OpReplace, Pos: 16, Length: 5, Text: "earth"},
		},
	}
	result, err := Apply(doc, op)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result != "hello beautiful earth" {
		t.Fatalf("expected 'hello beautiful earth', got '%s'", result)
	}
}

func TestTransformTwoInserts(t *testing.T) {
	// Two users insert at the same position
	a := Operation{ClientID: "A", Revision: 0, Ops: []TextOp{{Type: OpInsert, Pos: 5, Text: "X"}}}
	b := Operation{ClientID: "B", Revision: 0, Ops: []TextOp{{Type: OpInsert, Pos: 5, Text: "Y"}}}

	aPrime, bPrime := Transform(a, b)

	// Since "A" < "B" (client ID comparison), A's insert goes first when positions are equal
	// So a' is NOT shifted (stays at 5), and b' IS shifted to 6
	if aPrime.Ops[0].Pos != 5 {
		t.Fatalf("expected a' pos=5, got %d", aPrime.Ops[0].Pos)
	}
	if bPrime.Ops[0].Pos != 6 {
		t.Fatalf("expected b' pos=6, got %d", bPrime.Ops[0].Pos)
	}
}

func TestTransformDeleteAfterInsert(t *testing.T) {
	// A deletes at pos 10, B inserts at pos 0
	a := Operation{ClientID: "A", Revision: 0, Ops: []TextOp{{Type: OpDelete, Pos: 10, Length: 5}}}
	b := Operation{ClientID: "B", Revision: 0, Ops: []TextOp{{Type: OpInsert, Pos: 0, Text: "hello"}}}

	aPrime, _ := Transform(a, b)

	// A's delete should be shifted by the length of B's insert
	expectedPos := 10 + len([]rune("hello"))
	if aPrime.Ops[0].Pos != expectedPos {
		t.Fatalf("expected a' pos=%d, got %d", expectedPos, aPrime.Ops[0].Pos)
	}
}

func TestTransformTwoDeletes(t *testing.T) {
	// A deletes [5,10), B deletes [7,12) — overlapping deletes
	a := Operation{ClientID: "A", Revision: 0, Ops: []TextOp{{Type: OpDelete, Pos: 5, Length: 5}}}
	b := Operation{ClientID: "B", Revision: 0, Ops: []TextOp{{Type: OpDelete, Pos: 7, Length: 5}}}

	aPrime, _ := Transform(a, b)

	// After B deletes [7,12), A's delete [5,10) should become [5,7) since [7,10) is already deleted by B
	if aPrime.Ops[0].Pos != 5 {
		t.Fatalf("expected a' pos=5, got %d", aPrime.Ops[0].Pos)
	}
	if aPrime.Ops[0].Length != 2 {
		t.Fatalf("expected a' length=2, got %d", aPrime.Ops[0].Length)
	}
}

func TestTransformInsertInsideDelete(t *testing.T) {
	// A inserts at pos 7, B deletes [5,10)
	a := Operation{ClientID: "A", Revision: 0, Ops: []TextOp{{Type: OpInsert, Pos: 7, Text: "X"}}}
	b := Operation{ClientID: "B", Revision: 0, Ops: []TextOp{{Type: OpDelete, Pos: 5, Length: 5}}}

	aPrime, _ := Transform(a, b)

	// A's insert was inside B's delete range, should be moved to the start of the delete
	if aPrime.Ops[0].Pos != 5 {
		t.Fatalf("expected a' pos=5, got %d", aPrime.Ops[0].Pos)
	}
}

func TestParseOperation(t *testing.T) {
	data := []byte(`{"client_id":"user1","revision":1,"note_id":"n1","ops":[{"type":"insert","pos":0,"text":"hello"}]}`)
	op, err := ParseOperation(data)
	if err != nil {
		t.Fatalf("ParseOperation failed: %v", err)
	}
	if op.ClientID != "user1" {
		t.Fatalf("expected client_id=user1, got %s", op.ClientID)
	}
	if len(op.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(op.Ops))
	}
	if op.Ops[0].Type != OpInsert {
		t.Fatalf("expected type=insert, got %s", op.Ops[0].Type)
	}
}

func TestApplyChineseText(t *testing.T) {
	doc := "你好世界"
	op := Operation{
		ClientID: "user1",
		Revision: 1,
		Ops: []TextOp{
			{Type: OpInsert, Pos: 2, Text: "的"},
		},
	}
	result, err := Apply(doc, op)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result != "你好的世界" {
		t.Fatalf("expected '你好的世界', got '%s'", result)
	}
}
