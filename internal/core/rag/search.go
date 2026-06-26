package rag

import (
	"context"

	"github.com/boxify/api-go/internal/domain"
	"github.com/google/uuid"
)

type HybridSearcher interface {
	Search(ctx context.Context, userID uuid.UUID, query string, kbIDs []string, topK int) ([]domain.Citation, error)
}

type NoopSearcher struct{}

func (NoopSearcher) Search(ctx context.Context, userID uuid.UUID, query string, kbIDs []string, topK int) ([]domain.Citation, error) {
	return nil, nil
}
