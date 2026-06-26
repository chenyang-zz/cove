package repository

import "context"

type MemoryExample struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Text   string `json:"text"`
}

type MemoryExampleRepository interface {
	Upsert(ctx context.Context, item MemoryExample) (MemoryExample, error)
	FindByID(ctx context.Context, userID string, id string) (MemoryExample, error)
	Delete(ctx context.Context, userID string, id string) error
}
