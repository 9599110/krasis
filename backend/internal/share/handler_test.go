package share

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateShareToken(t *testing.T) {
	t1 := generateShareToken()
	t2 := generateShareToken()
	if t1 == t2 {
		t.Fatal("tokens should be unique")
	}
	if len(t1) != 32 {
		t.Fatalf("token length want 32 got %d", len(t1))
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "secret123" {
		t.Fatal("password should be hashed, not plaintext")
	}
	if !verifyPassword(hash, "secret123") {
		t.Fatal("should verify correct password")
	}
	if verifyPassword(hash, "wrong") {
		t.Fatal("should reject wrong password")
	}
}

func TestVerifyPassword_EmptyHash(t *testing.T) {
	if verifyPassword("", "test") {
		t.Fatal("empty hash should never verify")
	}
}

func TestNoteShare_StatusValues(t *testing.T) {
	now := time.Now()
	s := &NoteShare{
		ID:         uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		NoteID:     uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		ShareToken: "abc123",
		Status:     "pending",
		CreatedAt:  now,
	}
	if s.Status != "pending" {
		t.Fatalf("status want pending got %s", s.Status)
	}
}
