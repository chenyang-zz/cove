package modelconfig

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

func TestCreateModelEncryptsAndPersistsAPIKey(t *testing.T) {
	ctx := context.Background()
	cipher := newTestCipher(t)
	repo := &fakeModelConfigRepository{}
	userID := uuid.New()

	out, err := NewCreateModelLogic(ctx, &svc.ServiceContext{
		ModelConfigRepo: repo,
		SecretCipher:    cipher,
	}).CreateModel(userID, &request.CreateModelRequest{
		Type:       "chat",
		Provider:   "deepseek",
		Name:       "DeepSeek Chat",
		ModelName:  "deepseek-chat",
		ApiKey:     "sk-secret",
		BaseUrl:    "https://api.deepseek.com",
		Capability: []string{"function_call"},
		IsDefault:  true,
	})
	if err != nil {
		t.Fatalf("CreateModel error = %v", err)
	}
	if repo.created.APIKeyEncrypted == "" || repo.created.APIKeyEncrypted == "sk-secret" {
		t.Fatalf("APIKeyEncrypted = %q, want encrypted value", repo.created.APIKeyEncrypted)
	}
	plain, err := cipher.Decrypt(repo.created.APIKeyEncrypted)
	if err != nil {
		t.Fatalf("Decrypt APIKeyEncrypted error = %v", err)
	}
	if plain != "sk-secret" {
		t.Fatalf("decrypted API key = %q, want sk-secret", plain)
	}
	if out.APIKeyMasked != "*****cret" {
		t.Fatalf("APIKeyMasked = %q, want *****cret", out.APIKeyMasked)
	}
	if out.Name != "DeepSeek Chat" || out.ModelName != "deepseek-chat" {
		t.Fatalf("response = %+v", out)
	}
}

func TestListModelsReturnsMaskedKeysAndSkipsDecryptFailures(t *testing.T) {
	ctx := context.Background()
	cipher := newTestCipher(t)
	encrypted, err := cipher.Encrypt("sk-secret")
	if err != nil {
		t.Fatalf("Encrypt error = %v", err)
	}
	userID := uuid.New()
	repo := &fakeModelConfigRepository{
		rows: []*models.ModelConfig{
			{
				ID:              uuid.New(),
				UserID:          userID,
				Type:            "chat",
				Provider:        "deepseek",
				Name:            "DeepSeek Chat",
				ModelName:       "deepseek-chat",
				APIKeyEncrypted: encrypted,
			},
			{
				ID:              uuid.New(),
				UserID:          userID,
				Type:            "embedding",
				Name:            "Bad Key",
				APIKeyEncrypted: "not-encrypted",
			},
		},
	}

	out, err := NewListModelsLogic(ctx, &svc.ServiceContext{
		ModelConfigRepo: repo,
		SecretCipher:    cipher,
	}).ListModels(userID, &request.ListModelsRequest{Type: "chat"})
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}
	if repo.listType == nil || *repo.listType != domain.ModelType("chat") {
		t.Fatalf("listType = %v, want chat", repo.listType)
	}
	if len(out.List) != 1 {
		t.Fatalf("List len = %d, want 1", len(out.List))
	}
	if out.List[0].APIKeyMasked != "*****cret" {
		t.Fatalf("APIKeyMasked = %q, want *****cret", out.List[0].APIKeyMasked)
	}
}

func newTestCipher(t *testing.T) *security.SecretCipher {
	t.Helper()
	cipher, err := security.NewSecretCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewSecretCipher error = %v", err)
	}
	return cipher
}

type fakeModelConfigRepository struct {
	created  *models.ModelConfig
	updated  *models.ModelConfig
	rows     []*models.ModelConfig
	listType *domain.ModelType
}

func (r *fakeModelConfigRepository) Create(ctx context.Context, row *models.ModelConfig) (*models.ModelConfig, error) {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	r.created = row
	return row, nil
}

func (r *fakeModelConfigRepository) Update(ctx context.Context, row *models.ModelConfig) (*models.ModelConfig, error) {
	r.updated = row
	return row, nil
}

func (r *fakeModelConfigRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	return nil
}

func (r *fakeModelConfigRepository) List(ctx context.Context, userID uuid.UUID, modelType *domain.ModelType) ([]*models.ModelConfig, error) {
	r.listType = modelType
	return r.rows, nil
}

func (r *fakeModelConfigRepository) FindByID(ctx context.Context, userID uuid.UUID, configID uuid.UUID) (*models.ModelConfig, error) {
	for _, row := range r.rows {
		if row.ID == configID && row.UserID == userID {
			return row, nil
		}
	}
	return nil, nil
}
