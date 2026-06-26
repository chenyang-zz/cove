package app

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type ModelConfigService struct {
	mu     sync.Mutex
	cipher *SecretCipher
	rows   map[uuid.UUID][]ModelConfig
}

type ModelConfig struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"-"`
	Type            string    `json:"type"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	BaseURL         string    `json:"base_url"`
	APIKeyEncrypted string    `json:"-"`
	APIKeyMasked    string    `json:"api_key_masked"`
	IsDefault       bool      `json:"is_default"`
}

type CreateModelConfigInput struct {
	UserID    uuid.UUID
	Type      string
	Provider  string
	Model     string
	BaseURL   string
	APIKey    string
	IsDefault bool
}

func NewModelConfigService(cipher *SecretCipher) *ModelConfigService {
	if cipher == nil {
		panic("secret cipher is required")
	}
	return &ModelConfigService{cipher: cipher, rows: map[uuid.UUID][]ModelConfig{}}
}

func (s *ModelConfigService) Create(ctx context.Context, input CreateModelConfigInput) (ModelConfig, error) {
	encrypted, err := s.cipher.Encrypt(input.APIKey)
	if err != nil {
		return ModelConfig{}, err
	}
	row := ModelConfig{
		ID:              uuid.New(),
		UserID:          input.UserID,
		Type:            input.Type,
		Provider:        input.Provider,
		Model:           input.Model,
		BaseURL:         input.BaseURL,
		APIKeyEncrypted: encrypted,
		APIKeyMasked:    MaskSecret(input.APIKey),
		IsDefault:       input.IsDefault,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if row.IsDefault {
		existing := s.rows[input.UserID]
		for i := range existing {
			if existing[i].Type == row.Type {
				existing[i].IsDefault = false
			}
		}
		s.rows[input.UserID] = existing
	}
	s.rows[input.UserID] = append(s.rows[input.UserID], row)
	return row, nil
}

func (s *ModelConfigService) List(ctx context.Context, userID uuid.UUID) ([]ModelConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.rows[userID]
	out := make([]ModelConfig, len(rows))
	copy(out, rows)
	return out, nil
}
