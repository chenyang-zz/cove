package http_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/app"
	"github.com/boxify/api-go/internal/repository"
	httptransport "github.com/boxify/api-go/internal/transport/http"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func newTestRouter(t *testing.T, enableDebugPanicRoute ...bool) http.Handler {
	t.Helper()
	cipher, err := app.NewSecretCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	deps := httptransport.Dependencies{
		AuthService:        app.NewAuthService(newTestUserRepository(), newTestRefreshTokenRepository(), "test-secret"),
		ChatService:        app.NewChatService(),
		ModelConfigService: app.NewModelConfigService(cipher),
	}
	if len(enableDebugPanicRoute) > 0 {
		deps.EnableDebugPanicRoute = enableDebugPanicRoute[0]
	}
	return httptransport.NewRouter(deps)
}

type testUserRepository struct {
	byID    map[uuid.UUID]repository.User
	byLogin map[string]repository.User
}

func newTestUserRepository() *testUserRepository {
	return &testUserRepository{
		byID:    map[uuid.UUID]repository.User{},
		byLogin: map[string]repository.User{},
	}
}

func (r *testUserRepository) Create(ctx context.Context, user repository.User) (repository.User, error) {
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

func (r *testUserRepository) FindByLogin(ctx context.Context, login string) (repository.User, error) {
	user, ok := r.byLogin[login]
	if !ok {
		return repository.User{}, xerr.NotFound("用户不存在")
	}
	return user, nil
}

func (r *testUserRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return repository.User{}, xerr.NotFound("用户不存在")
	}
	return user, nil
}

type testRefreshTokenRepository struct {
	byHash map[string]repository.RefreshToken
}

func newTestRefreshTokenRepository() *testRefreshTokenRepository {
	return &testRefreshTokenRepository{byHash: map[string]repository.RefreshToken{}}
}

func (r *testRefreshTokenRepository) Create(ctx context.Context, token repository.RefreshToken) (repository.RefreshToken, error) {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	r.byHash[token.TokenHash] = token
	return token, nil
}

func (r *testRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (repository.RefreshToken, error) {
	token, ok := r.byHash[hash]
	if !ok {
		return repository.RefreshToken{}, xerr.InvalidToken()
	}
	return token, nil
}

func (r *testRefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	for hash, token := range r.byHash {
		if token.ID == id {
			token.RevokedAt = &revokedAt
			r.byHash[hash] = token
			return nil
		}
	}
	return xerr.InvalidToken()
}

func TestRouterHealthUsesUnifiedResponse(t *testing.T) {
	router := newTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"code":0,"message":"ok","data":{"status":"ok"}}` {
		t.Fatalf("body = %s", got)
	}
}

func TestRouterRequiresExplicitDependencies(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewRouter did not panic for missing dependencies")
		}
	}()
	_ = httptransport.NewRouter(httptransport.Dependencies{})
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	router := newTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(w.Body.String(), `"code":40100`) {
		t.Fatalf("body = %s, want auth error code", w.Body.String())
	}
}

func TestChatStreamSetsSSEHeadersAndEvents(t *testing.T) {
	router := newTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", got)
	}

	scanner := bufio.NewScanner(strings.NewReader(w.Body.String()))
	events := map[string]bool{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events[strings.TrimPrefix(line, "event: ")] = true
		}
	}
	for _, name := range []string{"meta", "token", "done"} {
		if !events[name] {
			t.Fatalf("missing SSE event %q in body:\n%s", name, w.Body.String())
		}
	}
}
