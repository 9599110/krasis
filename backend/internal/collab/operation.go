package collab

import (
	"encoding/json"
	"fmt"
)

// OpType represents a text operation type.
type OpType string

const (
	OpInsert  OpType = "insert"
	OpDelete  OpType = "delete"
	OpRetain  OpType = "retain"
	OpReplace OpType = "replace"
)

// TextOp represents a single atomic text operation at a position.
type TextOp struct {
	Type   OpType `json:"type"`
	Pos    int    `json:"pos"`
	Length int    `json:"length,omitempty"`  // for delete/retain
	Text   string `json:"text,omitempty"`    // for insert/replace
}

// Operation represents a composite operation containing multiple TextOps.
type Operation struct {
	ClientID string    `json:"client_id"`
	Revision int       `json:"revision"`
	NoteID   string    `json:"note_id"`
	Ops      []TextOp  `json:"ops"`
}

// Apply applies an operation to a document string, returning the new document.
// Each TextOp's position is relative to the document state after the previous TextOp.
func Apply(doc string, op Operation) (string, error) {
	result := []rune(doc)

	for _, textOp := range op.Ops {
		switch textOp.Type {
		case OpInsert:
			pos := textOp.Pos
			if pos < 0 || pos > len(result) {
				return doc, fmt.Errorf("insert position %d out of bounds (doc len %d)", pos, len(result))
			}
			insert := []rune(textOp.Text)
			result = append(result[:pos], append(insert, result[pos:]...)...)

		case OpDelete:
			pos := textOp.Pos
			end := pos + textOp.Length
			if pos < 0 || end > len(result) {
				return doc, fmt.Errorf("delete range [%d, %d) out of bounds (doc len %d)", pos, end, len(result))
			}
			result = append(result[:pos], result[end:]...)

		case OpReplace:
			pos := textOp.Pos
			end := pos + textOp.Length
			if pos < 0 || end > len(result) {
				return doc, fmt.Errorf("replace range [%d, %d) out of bounds (doc len %d)", pos, end, len(result))
			}
			replacement := []rune(textOp.Text)
			result = append(result[:pos], append(replacement, result[end:]...)...)
		}
	}

	return string(result), nil
}

// Transform transforms two operations against each other.
// Given operation A and operation B that were concurrent,
// Transform(A, B) returns A' such that applying B then A' yields
// the same result as applying A then B'.
// Returns (A', B').
func Transform(a, b Operation) (Operation, Operation) {
	aPrime := Operation{
		ClientID: a.ClientID,
		Revision: b.Revision + 1,
		NoteID:   a.NoteID,
	}
	bPrime := Operation{
		ClientID: b.ClientID,
		Revision: b.Revision + 1,
		NoteID:   b.NoteID,
	}

	aPrime.Ops = transformOps(a.Ops, b.Ops, a.ClientID, b.ClientID)
	bPrime.Ops = transformOps(b.Ops, a.Ops, b.ClientID, a.ClientID)

	return aPrime, bPrime
}

// transformOps transforms opA against opB, returning the transformed ops.
func transformOps(aOps, bOps []TextOp, aClientID, bClientID string) []TextOp {
	result := make([]TextOp, 0, len(aOps))
	for _, a := range aOps {
		transformed := a
		for _, b := range bOps {
			transformed = transformSingle(transformed, b, aClientID, bClientID)
		}
		result = append(result, transformed)
	}
	return result
}

// transformSingle transforms a single TextOp against another concurrent TextOp.
func transformSingle(a, b TextOp, aClientID, bClientID string) TextOp {
	switch a.Type {
	case OpInsert:
		switch b.Type {
		case OpInsert:
			// Two inserts at the same position: break tie by client ID
			// Lower client ID goes first. If b goes first, shift a.
			if b.Pos < a.Pos || (b.Pos == a.Pos && bClientID < aClientID) {
				a.Pos += len([]rune(b.Text))
			}
		case OpDelete:
			// If a's insert position falls within b's delete range,
			// move the insert to the start of the delete range
			deleteEnd := b.Pos + b.Length
			if a.Pos >= b.Pos && a.Pos < deleteEnd {
				a.Pos = b.Pos
			} else if a.Pos >= deleteEnd {
				a.Pos -= b.Length
			}
		case OpReplace:
			replaceEnd := b.Pos + b.Length
			if a.Pos >= b.Pos && a.Pos < replaceEnd {
				a.Pos = b.Pos
			} else if a.Pos >= replaceEnd {
				a.Pos -= b.Length
			}
		}

	case OpDelete:
		switch b.Type {
		case OpInsert:
			// b inserted before or at a's delete position: expand a's range
			if b.Pos <= a.Pos {
				insertLen := len([]rune(b.Text))
				a.Pos += insertLen
			} else if b.Pos >= a.Pos+a.Length {
				// b inserted after a's delete range: no change
			} else {
				// b inserted inside a's delete range: expand a's length
				a.Length += len([]rune(b.Text))
			}
		case OpDelete:
			// Two deletes: adjust a's range based on b's delete
			aEnd := a.Pos + a.Length
			bEnd := b.Pos + b.Length
			if b.Pos <= a.Pos && bEnd >= aEnd {
				// b completely covers a: a becomes no-op
				a.Length = 0
			} else if b.Pos <= a.Pos && bEnd < aEnd {
				// b deletes before part of a
				a.Pos = b.Pos
				a.Length = aEnd - b.Pos
			} else if b.Pos > a.Pos && b.Pos < aEnd && bEnd >= aEnd {
				// b deletes end of a
				a.Length = b.Pos - a.Pos
			} else if b.Pos > a.Pos && bEnd < aEnd {
				// b deletes middle of a
				a.Length -= b.Length
			} else if b.Pos >= aEnd {
				// b after a: shift a's position
				a.Pos -= b.Length
			}
		case OpReplace:
			replaceEnd := b.Pos + b.Length
			aEnd := a.Pos + a.Length
			if b.Pos <= a.Pos && replaceEnd >= aEnd {
				// b completely covers a: a becomes no-op
				a.Length = 0
			} else if b.Pos <= a.Pos && replaceEnd < aEnd {
				a.Pos = b.Pos
				a.Length = aEnd - b.Pos
			} else if b.Pos > a.Pos && b.Pos < aEnd && replaceEnd >= aEnd {
				a.Length = b.Pos - a.Pos
			} else if b.Pos > a.Pos && replaceEnd < aEnd {
				a.Length -= b.Length
			} else if b.Pos >= aEnd {
				a.Pos -= b.Length
			}
		}

	case OpReplace:
		switch b.Type {
		case OpInsert:
			if b.Pos <= a.Pos {
				insertLen := len([]rune(b.Text))
				a.Pos += insertLen
			} else if b.Pos < a.Pos+a.Length {
				a.Length += len([]rune(b.Text))
			}
		case OpDelete:
			deleteEnd := b.Pos + b.Length
			aEnd := a.Pos + a.Length
			if b.Pos <= a.Pos && deleteEnd >= aEnd {
				a.Length = 0
			} else if b.Pos <= a.Pos && deleteEnd < aEnd {
				a.Pos = b.Pos
				a.Length = aEnd - b.Pos
			} else if b.Pos >= aEnd {
				a.Pos -= b.Length
			} else if b.Pos > a.Pos && deleteEnd < aEnd {
				a.Length -= b.Length
			}
		case OpReplace:
			replaceEnd := b.Pos + b.Length
			aEnd := a.Pos + a.Length
			if b.Pos <= a.Pos && replaceEnd >= aEnd {
				a.Length = 0
			} else if b.Pos <= a.Pos && replaceEnd < aEnd {
				a.Pos = b.Pos
				a.Length = aEnd - b.Pos
			} else if b.Pos >= aEnd {
				a.Pos -= b.Length
			} else if b.Pos > a.Pos && replaceEnd < aEnd {
				a.Length -= b.Length
			}
		}
	}

	return a
}

// ParseOperation unmarshals a JSON message into an Operation.
func ParseOperation(data []byte) (Operation, error) {
	var op Operation
	if err := json.Unmarshal(data, &op); err != nil {
		return Operation{}, fmt.Errorf("parse operation: %w", err)
	}
	if op.ClientID == "" {
		return Operation{}, fmt.Errorf("operation missing client_id")
	}
	if len(op.Ops) == 0 {
		return Operation{}, fmt.Errorf("operation has no ops")
	}
	return op, nil
}

// MarshalJSON returns the JSON encoding of the operation.
func (o Operation) MarshalJSON() ([]byte, error) {
	type Alias Operation
	return json.Marshal((*Alias)(&o))
}
