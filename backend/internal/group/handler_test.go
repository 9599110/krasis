package group

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGroupModel_Fields(t *testing.T) {
	now := time.Now()
	g := Group{
		ID:          uuid.New(),
		Name:        "Admins",
		Description: "Admin group",
		IsDefault:   true,
		UserCount:   5,
		CreatedAt:   now,
		UpdatedAt:   &now,
	}
	if g.Name != "Admins" {
		t.Fatalf("name want Admins got %s", g.Name)
	}
	if g.UserCount != 5 {
		t.Fatalf("userCount want 5 got %d", g.UserCount)
	}
}

func TestGroupFeature_Fields(t *testing.T) {
	f := GroupFeature{
		ID:           uuid.New(),
		GroupID:      uuid.New(),
		FeatureKey:   "max_notes",
		FeatureValue: json.RawMessage(`{"value": 100}`),
	}
	if f.FeatureKey != "max_notes" {
		t.Fatalf("key want max_notes got %s", f.FeatureKey)
	}
}

func TestGroupJSONSerialization(t *testing.T) {
	now := time.Now()
	g := Group{ID: uuid.New(), Name: "Test", IsDefault: true, UserCount: 3, CreatedAt: now}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Group
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "Test" {
		t.Fatalf("name want Test got %s", decoded.Name)
	}
}
