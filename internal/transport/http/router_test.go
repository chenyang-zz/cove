package http_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	httptransport "github.com/boxify/api-go/internal/transport/http"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func newTestRouter(t *testing.T, enableDebugPanicRoute ...bool) http.Handler {
	t.Helper()
	cipher, err := security.NewSecretCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		UserRepo:         newTestUserRepository(),
		RefreshTokenRepo: newTestRefreshTokenRepository(),
		ModelConfigRepo:  &testModelConfigRepository{},
		ConversationRepo: newTestConversationRepository(),
		MessageRepo:      newTestMessageRepository(),
		Realtime:         testRealtimeBroker{},
		SecretCipher:     cipher,
		TokenIssuer:      security.NewTokenIssuer("test-secret", time.Hour),
	}
	deps := httptransport.Dependencies{
		Svc: svcCtx,
	}
	if len(enableDebugPanicRoute) > 0 {
		deps.EnableDebugPanicRoute = enableDebugPanicRoute[0]
	}
	return httptransport.NewRouter(deps)
}

type testRealtimeBroker struct{}

func (testRealtimeBroker) Publish(ctx context.Context, topic string, event domain.Event) error {
	return nil
}

func (testRealtimeBroker) Subscribe(ctx context.Context, topic string) (realtime.Subscription, error) {
	events := make(chan domain.Event, 2)
	events <- domain.NewTokenEvent("345")
	events <- domain.NewDoneEvent("ok")
	close(events)
	return testRealtimeSubscription{events: events}, nil
}

type testRealtimeSubscription struct {
	events <-chan domain.Event
}

func (s testRealtimeSubscription) Events() <-chan domain.Event {
	return s.events
}

func (s testRealtimeSubscription) Close(ctx context.Context) error {
	return nil
}

type testModelConfigRepository struct {
	rows []*models.ModelConfig
}

func (r *testModelConfigRepository) Create(ctx context.Context, row *models.ModelConfig) (*models.ModelConfig, error) {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	r.rows = append(r.rows, row)
	return row, nil
}

func (r *testModelConfigRepository) Update(ctx context.Context, row *models.ModelConfig) (*models.ModelConfig, error) {
	for i, existing := range r.rows {
		if existing.ID == row.ID {
			r.rows[i] = row
			return row, nil
		}
	}
	r.rows = append(r.rows, row)
	return row, nil
}

func (r *testModelConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	for i, row := range r.rows {
		if row.ID == id {
			r.rows = append(r.rows[:i], r.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *testModelConfigRepository) List(ctx context.Context, userID uuid.UUID, modelType *domain.ModelType) ([]*models.ModelConfig, error) {
	out := make([]*models.ModelConfig, 0, len(r.rows))
	for _, row := range r.rows {
		if row.UserID == userID && (modelType == nil || row.Type == string(*modelType)) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *testModelConfigRepository) FindByID(ctx context.Context, userID uuid.UUID, configID uuid.UUID) (*models.ModelConfig, error) {
	for _, row := range r.rows {
		if row.ID == configID && row.UserID == userID {
			return row, nil
		}
	}
	return nil, xerr.NotFound("模型配置不存在")
}

type testConversationRepository struct {
	rows []*models.Conversation
}

func newTestConversationRepository() *testConversationRepository {
	return &testConversationRepository{}
}

func (r *testConversationRepository) Create(ctx context.Context, userID uuid.UUID, row *models.Conversation) (*models.Conversation, error) {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	row.UserID = userID
	r.rows = append(r.rows, row)
	return row, nil
}

func (r *testConversationRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.Conversation, error) {
	out := make([]*models.Conversation, 0, len(r.rows))
	for _, row := range r.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *testConversationRepository) FindByID(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*models.Conversation, error) {
	for _, row := range r.rows {
		if row.ID == conversationID && row.UserID == userID {
			return row, nil
		}
	}
	return nil, xerr.NotFound("会话不存在")
}

func (r *testConversationRepository) Update(ctx context.Context, userID uuid.UUID, row *models.Conversation) (*models.Conversation, error) {
	for i, existing := range r.rows {
		if existing.ID == row.ID && existing.UserID == userID {
			row.UserID = userID
			r.rows[i] = row
			return row, nil
		}
	}
	return nil, xerr.NotFound("会话不存在")
}

func (r *testConversationRepository) UpdateFields(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, row *models.Conversation, fields *repository.ConversationUpdateFields) (*models.Conversation, error) {
	existing, err := r.FindByID(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	for _, column := range fields.Columns() {
		if column == "title" {
			existing.Title = row.Title
		}
	}
	return existing, nil
}

func (r *testConversationRepository) Delete(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	for i, row := range r.rows {
		if row.ID == conversationID && row.UserID == userID {
			r.rows = append(r.rows[:i], r.rows[i+1:]...)
			return nil
		}
	}
	return xerr.NotFound("会话不存在")
}

type testMessageRepository struct {
	rows []*models.Message
}

func newTestMessageRepository() *testMessageRepository {
	return &testMessageRepository{}
}

func (r *testMessageRepository) Create(ctx context.Context, userID uuid.UUID, row *models.Message) (*models.Message, error) {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	r.rows = append(r.rows, row)
	return row, nil
}

func (r *testMessageRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.Message, error) {
	return append([]*models.Message(nil), r.rows...), nil
}

func (r *testMessageRepository) FindByID(ctx context.Context, userID uuid.UUID, messageID uuid.UUID) (*models.Message, error) {
	for _, row := range r.rows {
		if row.ID == messageID {
			return row, nil
		}
	}
	return nil, xerr.NotFound("消息不存在")
}

func (r *testMessageRepository) Update(ctx context.Context, userID uuid.UUID, row *models.Message) (*models.Message, error) {
	for i, existing := range r.rows {
		if existing.ID == row.ID {
			r.rows[i] = row
			return row, nil
		}
	}
	return nil, xerr.NotFound("消息不存在")
}

func (r *testMessageRepository) UpdateFields(ctx context.Context, userID uuid.UUID, messageID uuid.UUID, row *models.Message, fields *repository.MessageUpdateFields) (*models.Message, error) {
	existing, err := r.FindByID(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}
	for _, column := range fields.Columns() {
		switch column {
		case "conversation_id":
			existing.ConversationID = row.ConversationID
		case "role":
			existing.Role = row.Role
		case "sender_persona_id":
			existing.SenderPersonaID = row.SenderPersonaID
		case "sender_user_id":
			existing.SenderUserID = row.SenderUserID
		case "meta_data":
			existing.MetaData = row.MetaData
		}
	}
	return existing, nil
}

func (r *testMessageRepository) Delete(ctx context.Context, userID uuid.UUID, messageID uuid.UUID) error {
	for i, row := range r.rows {
		if row.ID == messageID {
			r.rows = append(r.rows[:i], r.rows[i+1:]...)
			return nil
		}
	}
	return xerr.NotFound("消息不存在")
}

func (r *testMessageRepository) Count(ctx context.Context, conversationID uuid.UUID) (int64, error) {
	var count int64
	for _, row := range r.rows {
		if row.ConversationID == conversationID {
			count++
		}
	}
	return count, nil
}

type testUserRepository struct {
	byID    map[uuid.UUID]*models.User
	byLogin map[string]*models.User
}

func newTestUserRepository() *testUserRepository {
	return &testUserRepository{
		byID:    map[uuid.UUID]*models.User{},
		byLogin: map[string]*models.User{},
	}
}

func (r *testUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if _, ok := r.byLogin[user.Username]; ok {
		return nil, xerr.UserExists()
	}
	if user.Email != nil {
		if _, ok := r.byLogin[*user.Email]; ok {
			return nil, xerr.UserExists()
		}
	}
	r.byID[user.ID] = user
	r.byLogin[user.Username] = user
	if user.Email != nil {
		r.byLogin[*user.Email] = user
	}
	return user, nil
}

func (r *testUserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	if _, ok := r.byID[user.ID]; !ok {
		return nil, xerr.NotFound("用户不存在")
	}
	r.byID[user.ID] = user
	r.byLogin[user.Username] = user
	if user.Email != nil {
		r.byLogin[*user.Email] = user
	}
	return user, nil
}

func (r *testUserRepository) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	user, ok := r.byLogin[login]
	if !ok {
		return nil, xerr.NotFound("用户不存在")
	}
	return user, nil
}

func (r *testUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, xerr.NotFound("用户不存在")
	}
	return user, nil
}

type testRefreshTokenRepository struct {
	byHash map[string]*models.RefreshToken
}

func newTestRefreshTokenRepository() *testRefreshTokenRepository {
	return &testRefreshTokenRepository{byHash: map[string]*models.RefreshToken{}}
}

func (r *testRefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) (*models.RefreshToken, error) {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	r.byHash[token.TokenHash] = token
	return token, nil
}

func (r *testRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	token, ok := r.byHash[hash]
	if !ok {
		return nil, xerr.InvalidToken()
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
	var got struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List []struct {
				ServerName string `json:"server_name"`
				IsHealthy  bool   `json:"is_healthy"`
				Error      string `json:"error"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if got.Code != 0 || got.Message != "ok" {
		t.Fatalf("body = %+v, want success envelope", got)
	}
	if len(got.Data.List) != 5 {
		t.Fatalf("health list len = %d, want 5", len(got.Data.List))
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
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
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
