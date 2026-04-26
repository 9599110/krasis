package ai

import (
	"testing"

	"github.com/google/uuid"
)

func TestModelConfigManager_NoModels(t *testing.T) {
	m := NewModelConfigManager(nil)
	if m.GetDefaultLLM() != nil {
		t.Fatal("want nil LLM when no models configured")
	}
	if m.GetDefaultEmbedding() != nil {
		t.Fatal("want nil embedding when no models configured")
	}
}

func TestModelConfigManager_GetLLM_ByUUID(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	m := NewModelConfigManager(nil)
	m.models = []*AIModel{
		{ID: id, ModelType: "llm", ModelName: "gpt-4", IsDefault: true, IsEnabled: true},
	}
	m.llmModels = m.models

	got := m.GetLLM("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if got == nil {
		t.Fatal("want model by ID got nil")
	}
	if got.ModelName != "gpt-4" {
		t.Fatalf("want gpt-4 got %s", got.ModelName)
	}
}

func TestModelConfigManager_GetLLM_ByName(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	m := NewModelConfigManager(nil)
	m.llmModels = []*AIModel{
		{ID: id, ModelType: "llm", Name: "gpt-4", IsEnabled: true},
	}

	got := m.GetLLM("gpt-4")
	if got == nil {
		t.Fatal("want model by name got nil")
	}
	if got.Name != "gpt-4" {
		t.Fatalf("want gpt-4 got %s", got.Name)
	}
}

func TestModelConfigManager_GetLLM_InvalidUUID(t *testing.T) {
	m := NewModelConfigManager(nil)
	m.llmModels = []*AIModel{
		{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), ModelType: "llm"},
	}

	got := m.GetLLM("invalid-uuid")
	if got != nil {
		t.Fatal("want nil for invalid UUID that doesn't match any name")
	}
}

func TestModelConfigManager_GetDefaultLLM_FirstAvailable(t *testing.T) {
	m := NewModelConfigManager(nil)
	m.llmModels = []*AIModel{
		{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), ModelType: "llm", ModelName: "first", IsEnabled: true},
	}
	got := m.GetDefaultLLM()
	if got == nil || got.ModelName != "first" {
		t.Fatalf("want first model got %+v", got)
	}
}

func TestModelConfigManager_GetDefaultEmbedding_FirstAvailable(t *testing.T) {
	m := NewModelConfigManager(nil)
	m.embeddingModels = []*AIModel{
		{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), ModelType: "embedding", ModelName: "text-embedding", IsEnabled: true},
	}
	got := m.GetDefaultEmbedding()
	if got == nil || got.ModelName != "text-embedding" {
		t.Fatalf("want embedding model got %+v", got)
	}
}
