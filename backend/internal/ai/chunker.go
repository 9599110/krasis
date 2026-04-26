package ai

import (
	"unicode/utf8"
)

// Chunk represents a text chunk from a note
type Chunk struct {
	NoteID      string  `json:"note_id"`
	NoteTitle   string  `json:"note_title"`
	Text        string  `json:"text"`
	ChunkIndex  int     `json:"chunk_index"`
	TokenCount  int     `json:"token_count"`
}

// SplitNoteIntoChunks divides note content into overlapping chunks
func SplitNoteIntoChunks(noteID, title, content string, chunkSize, overlap int) []*Chunk {
	if content == "" {
		return nil
	}

	runes := []rune(content)
	if len(runes) <= chunkSize {
		return []*Chunk{{
			NoteID:     noteID,
			NoteTitle:  title,
			Text:       content,
			ChunkIndex: 0,
			TokenCount: len(runes),
		}}
	}

	var chunks []*Chunk
	start := 0
	idx := 0

	for start < len(runes) {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		// Try to split at a natural boundary (newline)
		if end < len(runes) {
			for i := end; i > start+chunkSize/2; i-- {
				if runes[i] == '\n' {
					end = i
					break
				}
			}
		}

		chunks = append(chunks, &Chunk{
			NoteID:     noteID,
			NoteTitle:  title,
			Text:       string(runes[start:end]),
			ChunkIndex: idx,
			TokenCount: end - start,
		})

		start = end - overlap
		idx++

		if start >= len(runes) || end >= len(runes) {
			break
		}
	}

	return chunks
}

// CountTokens returns approximate token count (characters / 4 for rough estimate)
func CountTokens(text string) int {
	return utf8.RuneCountInString(text) / 4
}
