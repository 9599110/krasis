package search

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/krasis/krasis/pkg/types"
)

func TestService_Search_EmptyQuery(t *testing.T) {
	// Empty query returns ErrEmptyQuery without touching the repo
	svc := &Service{}
	_, _, err := svc.Search(nil, "", "all", 1, 20)
	if err != ErrEmptyQuery {
		t.Fatalf("want ErrEmptyQuery got %v", err)
	}
}

func TestSearchResult_Struct(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	r := SearchResult{
		Type:       "note",
		ID:         id,
		Title:      "My Note",
		Highlights: []string{"Some <em>content</em>"},
		Score:      0.95,
		UpdatedAt:  types.NullTime{Time: now, Valid: true},
	}
	if r.Type != "note" {
		t.Fatalf("type want 'note' got %s", r.Type)
	}
	if len(r.Highlights) != 1 {
		t.Fatalf("highlights count want 1 got %d", len(r.Highlights))
	}
	if r.Score != 0.95 {
		t.Fatalf("score want 0.95 got %f", r.Score)
	}
}
