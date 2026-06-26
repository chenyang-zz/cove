package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Username     string
	Nickname     *string
	Email        *string
	Avatar       *string
	PasswordHash string
}

type UserRepository interface {
	Create(ctx context.Context, user User) (User, error)
	FindByLogin(ctx context.Context, login string) (User, error)
	FindByID(ctx context.Context, id uuid.UUID) (User, error)
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token RefreshToken) (RefreshToken, error)
	FindByHash(ctx context.Context, hash string) (RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error
}
