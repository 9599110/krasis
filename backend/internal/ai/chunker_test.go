package ai

import (
	"encoding/json"
	"testing"
)

func TestSplitNoteIntoChunks_Empty(t *testing.T) {
	chunks := SplitNoteIntoChunks("n1", "title", "", 500, 50)
	if len(chunks) != 0 {
		t.Fatalf("want 0 chunks got %d", len(chunks))
	}
}

func TestSplitNoteIntoChunks_SingleChunk(t *testing.T) {
	chunks := SplitNoteIntoChunks("n1", "title", "hello world", 500, 50)
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk got %d", len(chunks))
	}
	if chunks[0].Text != "hello world" {
		t.Fatalf("text want 'hello world' got %q", chunks[0].Text)
	}
	if chunks[0].NoteID != "n1" {
		t.Fatalf("noteID want n1 got %s", chunks[0].NoteID)
	}
	if chunks[0].NoteTitle != "title" {
		t.Fatalf("title want 'title' got %s", chunks[0].NoteTitle)
	}
	if chunks[0].TokenCount != 11 {
		t.Fatalf("tokenCount want 11 got %d", chunks[0].TokenCount)
	}
}

func TestSplitNoteIntoChunks_MultipleChunks(t *testing.T) {
	content := ""
	for i := 0; i < 200; i++ {
		content += "a"
	}
	chunks := SplitNoteIntoChunks("n1", "t", content, 50, 10)
	if len(chunks) == 0 {
		t.Fatal("expected multiple chunks")
	}
}

func TestChunk_JSONMarshal(t *testing.T) {
	c := Chunk{
		NoteID:     "note-1",
		NoteTitle:  "Test",
		Text:       "content",
		ChunkIndex: 0,
		TokenCount: 7,
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty JSON")
	}
	var decoded Chunk
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.NoteID != "note-1" {
		t.Fatalf("NoteID want note-1 got %s", decoded.NoteID)
	}
}

func TestCountTokens(t *testing.T) {
	// CountTokens uses utf8.RuneCountInString / 4
	tokens := CountTokens("hello")
	// "hello" = 5 runes / 4 = 1
	if tokens != 1 {
		t.Fatalf("want 1 token got %d", tokens)
	}
	if CountTokens("") != 0 {
		t.Fatal("empty string should return 0")
	}
}
