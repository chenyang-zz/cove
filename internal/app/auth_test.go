package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/app"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

type fakeUserRepository struct {
	byID    map[uuid.UUID]repository.User
	byLogin map[string]repository.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		byID:    map[uuid.UUID]repository.User{},
		byLogin: map[string]repository.User{},
	}
}

func (r *fakeUserRepository) Create(ctx context.Context, user repository.User) (repository.User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if _, ok := r.byLogin[user.Username]; ok {
		return repository.User{}, xerr.UserExists()
	}
	if user.Email != nil {
		if _, ok := r.byLogin[*user.Email]; ok {
			return repository.User{}, xerr.UserExists()
		}
	}
	r.byID[user.ID] = user
	r.byLogin[user.Username] = user
	if user.Email != nil {
		r.byLogin[*user.Email] = user
	}
	return user, nil
}

func (r *fakeUserRepository) FindByLogin(ctx context.Context, login string) (repository.User, error) {
	user, ok := r.byLogin[login]
	if !ok {
		return repository.User{}, xerr.NotFound("用户不存在")
	}
	return user, nil
}

func (r *fakeUserRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return repository.User{}, xerr.NotFound("用户不存在")
	}
	return user, nil
}

type fakeRefreshTokenRepository struct {
	byHash map[string]repository.RefreshToken
}

func newFakeRefreshTokenRepository() *fakeRefreshTokenRepository {
	return &fakeRefreshTokenRepository{byHash: map[string]repository.RefreshToken{}}
}

func (r *fakeRefreshTokenRepository) Create(ctx context.Context, token repository.RefreshToken) (repository.RefreshToken, error) {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	r.byHash[token.TokenHash] = token
	return token, nil
}

func (r *fakeRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (repository.RefreshToken, error) {
	token, ok := r.byHash[hash]
	if !ok {
		return repository.RefreshToken{}, xerr.InvalidToken()
	}
	return token, nil
}

func (r *fakeRefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	for hash, token := range r.byHash {
		if token.ID == id {
			token.RevokedAt = &revokedAt
			r.byHash[hash] = token
			return nil
		}
	}
	return xerr.InvalidToken()
}

func TestAuthServiceRegistersUserAndReturnsRefreshToken(t *testing.T) {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	service := app.NewAuthService(users, refreshTokens, "test-secret")

	out, err := service.Register(context.Background(), app.RegisterInput{
		Username: "  Alice  ",
		Email:    ptr("  ALICE@example.COM  "),
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Register error = %v", err)
	}
	if out.UserID == uuid.Nil {
		t.Fatal("user id is nil")
	}
	if out.Username != "alice" {
		t.Fatalf("username = %q, want alice", out.Username)
	}
	if out.Email == nil || *out.Email != "alice@example.com" {
		t.Fatalf("email = %v, want alice@example.com", out.Email)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatalf("tokens must be returned: access=%q refresh=%q", out.AccessToken, out.RefreshToken)
	}
	created := users.byID[out.UserID]
	if created.PasswordHash == "secret123" || !app.CheckPassword(created.PasswordHash, "secret123") {
		t.Fatal("password was not hashed correctly")
	}
}

func TestAuthServiceLoginSupportsUsernameAndEmail(t *testing.T) {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	service := app.NewAuthService(users, refreshTokens, "test-secret")
	registered, err := service.Register(context.Background(), app.RegisterInput{
		Username: "alice",
		Email:    ptr("alice@example.com"),
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Register error = %v", err)
	}

	for _, login := range []string{"alice", "alice@example.com"} {
		out, err := service.Login(context.Background(), app.LoginInput{
			Login:    login,
			Password: "secret123",
		})
		if err != nil {
			t.Fatalf("Login(%q) error = %v", login, err)
		}
		if out.UserID != registered.UserID || out.AccessToken == "" || out.RefreshToken == "" {
			t.Fatalf("Login(%q) output = %+v, want registered user with tokens", login, out)
		}
	}
}

func TestAuthServiceRejectsInvalidCredentials(t *testing.T) {
	service := app.NewAuthService(newFakeUserRepository(), newFakeRefreshTokenRepository(), "test-secret")

	_, err := service.Login(context.Background(), app.LoginInput{Login: "missing", Password: "secret123"})
	if xerr.From(err).Kind != xerr.KindUnauthorized {
		t.Fatalf("Login missing error = %v, want unauthorized", err)
	}
}

func TestAuthServiceRefreshRotatesTokenAndRejectsReuse(t *testing.T) {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	service := app.NewAuthService(users, refreshTokens, "test-secret")
	registered, err := service.Register(context.Background(), app.RegisterInput{
		Username: "alice",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Register error = %v", err)
	}

	refreshed, err := service.Refresh(context.Background(), app.RefreshInput{RefreshToken: registered.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh error = %v", err)
	}
	if refreshed.UserID != registered.UserID || refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatalf("Refresh output = %+v, want same user with new tokens", refreshed)
	}
	if refreshed.RefreshToken == registered.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	_, err = service.Refresh(context.Background(), app.RefreshInput{RefreshToken: registered.RefreshToken})
	if xerr.From(err).Kind != xerr.KindUnauthorized {
		t.Fatalf("Refresh reused token error = %v, want unauthorized", err)
	}
}

func TestAuthServiceRefreshRejectsExpiredToken(t *testing.T) {
	users := newFakeUserRepository()
	refreshTokens := newFakeRefreshTokenRepository()
	service := app.NewAuthService(users, refreshTokens, "test-secret")
	user := repository.User{ID: uuid.New(), Username: "alice", PasswordHash: "hash"}
	users.byID[user.ID] = user
	users.byLogin[user.Username] = user
	rawToken := "expired-token"
	refreshTokens.byHash[app.HashRefreshToken(rawToken)] = repository.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: app.HashRefreshToken(rawToken),
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	_, err := service.Refresh(context.Background(), app.RefreshInput{RefreshToken: rawToken})
	if xerr.From(err).Kind != xerr.KindUnauthorized {
		t.Fatalf("Refresh expired token error = %v, want unauthorized", err)
	}
}

func TestAuthServiceRequiresRepositories(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewAuthService did not panic")
		}
	}()
	_ = app.NewAuthService(nil, newFakeRefreshTokenRepository(), "test-secret")
}

func ptr(value string) *string {
	return &value
}

var _ repository.UserRepository = (*fakeUserRepository)(nil)
var _ repository.RefreshTokenRepository = (*fakeRefreshTokenRepository)(nil)
