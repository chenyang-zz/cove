package mapper_test

import (
	"testing"
	"time"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

func TestNewModelConfigFromCreate(t *testing.T) {
	userID := uuid.New()
	input := &request.CreateModelRequest{
		Type:       "chat",
		Provider:   "deepseek",
		Name:       "DeepSeek Chat",
		ModelName:  "deepseek-chat",
		ApiKey:     "sk-secret",
		BaseUrl:    "https://api.deepseek.com",
		Capability: []string{"function_call", "vision"},
		IsDefault:  true,
	}

	got := mapper.NewModelConfigFromCreate(userID, input, "encrypted-key")

	if got.UserID != userID || got.Type != input.Type || got.Provider != input.Provider {
		t.Fatalf("model identity fields = %+v", got)
	}
	if got.Name != input.Name || got.ModelName != input.ModelName || got.BaseURL != input.BaseUrl {
		t.Fatalf("model text fields = %+v", got)
	}
	if got.APIKeyEncrypted != "encrypted-key" {
		t.Fatalf("APIKeyEncrypted = %q, want encrypted-key", got.APIKeyEncrypted)
	}
	if len(got.Capability) != 2 || got.Capability[0] != "function_call" || got.Capability[1] != "vision" {
		t.Fatalf("Capability = %+v", got.Capability)
	}
	if !got.IsDefault {
		t.Fatal("IsDefault = false, want true")
	}
}

func TestApplyUpdateModelConfigOnlyAssignsPresentNonEmptyFields(t *testing.T) {
	row := &models.ModelConfig{
		Name:            "Old Name",
		ModelName:       "old-model",
		BaseURL:         "https://old.example.com",
		APIKeyEncrypted: "old-encrypted",
		Capability:      models.StringList{"old"},
	}
	empty := ""
	name := "New Name"
	baseURL := "https://new.example.com"
	apiKey := "sk-plain"

	mapper.ApplyUpdateModelConfig(row, &request.UpdateModelRequest{
		Name:       &name,
		ModelName:  &empty,
		BaseUrl:    &baseURL,
		ApiKey:     &apiKey,
		Capability: []string{"vision"},
	})

	if row.Name != "New Name" {
		t.Fatalf("Name = %q, want New Name", row.Name)
	}
	if row.ModelName != "old-model" {
		t.Fatalf("ModelName = %q, want old-model", row.ModelName)
	}
	if row.BaseURL != "https://new.example.com" {
		t.Fatalf("BaseURL = %q, want new URL", row.BaseURL)
	}
	if row.APIKeyEncrypted != "old-encrypted" {
		t.Fatalf("APIKeyEncrypted = %q, want old-encrypted", row.APIKeyEncrypted)
	}
	if len(row.Capability) != 1 || row.Capability[0] != "vision" {
		t.Fatalf("Capability = %+v, want [vision]", row.Capability)
	}
}

func TestModelConfigToResponseUsesMaskedKeyOnly(t *testing.T) {
	id := uuid.New()
	createdAt := time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	row := &models.ModelConfig{
		ID:              id,
		Type:            "chat",
		Provider:        "deepseek",
		Name:            "DeepSeek Chat",
		ModelName:       "deepseek-chat",
		APIKeyEncrypted: "encrypted-key",
		BaseURL:         "https://api.deepseek.com",
		Capability:      models.StringList{"function_call"},
		IsDefault:       true,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	got := mapper.ModelConfigToResponse(row, "*****cret")

	if got.ID != id || got.Type != row.Type || got.Provider != row.Provider {
		t.Fatalf("response identity fields = %+v", got)
	}
	if got.APIKeyMasked != "*****cret" {
		t.Fatalf("APIKeyMasked = %q, want masked key", got.APIKeyMasked)
	}
	if got.APIKeyMasked == row.APIKeyEncrypted {
		t.Fatal("response leaked APIKeyEncrypted")
	}
	if len(got.Capability) != 1 || got.Capability[0] != "function_call" {
		t.Fatalf("Capability = %+v", got.Capability)
	}
	if !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("timestamps = %s/%s", got.CreatedAt, got.UpdatedAt)
	}
}

func TestModelConfigsToListResponseKeepsOrderAndSkipsUnmaskedRows(t *testing.T) {
	first := &models.ModelConfig{ID: uuid.New(), Name: "first"}
	skipped := &models.ModelConfig{ID: uuid.New(), Name: "skipped"}
	third := &models.ModelConfig{ID: uuid.New(), Name: "third"}

	got := mapper.ModelConfigsToListResponse([]*models.ModelConfig{first, skipped, third}, func(row *models.ModelConfig) (string, bool) {
		if row == skipped {
			return "", false
		}
		return "masked-" + row.Name, true
	})

	if len(got.List) != 2 {
		t.Fatalf("List len = %d, want 2", len(got.List))
	}
	if got.List[0].ID != first.ID || got.List[0].APIKeyMasked != "masked-first" {
		t.Fatalf("first response = %+v", got.List[0])
	}
	if got.List[1].ID != third.ID || got.List[1].APIKeyMasked != "masked-third" {
		t.Fatalf("third response = %+v", got.List[1])
	}
}
