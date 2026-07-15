package agentconfig

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// TestCreateAgentConfigUsesExplicitDefaults 验证空对象创建会写入完整业务默认值而不依赖数据库默认行为。
func TestCreateAgentConfigUsesExplicitDefaults(t *testing.T) {
	repo := newAgentConfigTestRepository()
	userID := uuid.New()
	logic := NewCreateAgentConfigLogic(context.Background(), &svc.ServiceContext{AgentConfigRepo: repo})

	result, err := logic.CreateAgentConfig(userID, &request.CreateAgentConfigRequest{})
	if err != nil {
		t.Fatalf("CreateAgentConfig(defaults) error = %v, want nil", err)
	}
	if result.ID == uuid.Nil || result.Name != "默认配置" || !result.IsDefault || result.Temperature != 0.7 || !result.EnableKnowledge || !result.EnableMemory || !result.EnableActiveRecall {
		t.Fatalf("CreateAgentConfig(defaults) result = %#v, want complete defaults", result)
	}
	if !result.ContextEnabled || result.ContextWindowTokens != 32768 || result.ContextSummaryMaxTokens != 1024 {
		t.Fatalf("CreateAgentConfig(defaults) context = %#v, want default context policy", result)
	}
}

// TestCreateAgentConfigAppliesOverridesAndValidatesPolicy 验证创建时可覆盖默认值且会拒绝无效上下文比例。
func TestCreateAgentConfigAppliesOverridesAndValidatesPolicy(t *testing.T) {
	repo := newAgentConfigTestRepository()
	logic := NewCreateAgentConfigLogic(context.Background(), &svc.ServiceContext{AgentConfigRepo: repo})
	name, temperature, knowledge := "  日常助手  ", 1.2, false
	window, recent := 65536, 12000

	result, err := logic.CreateAgentConfig(uuid.New(), &request.CreateAgentConfigRequest{AgentConfigFieldsRequest: request.AgentConfigFieldsRequest{
		Name: &name, Temperature: &temperature, EnableKnowledge: &knowledge,
		ContextWindowTokens: &window, ContextKeepRecentTokens: &recent,
	}})
	if err != nil {
		t.Fatalf("CreateAgentConfig(overrides) error = %v, want nil", err)
	}
	if result.Name != "日常助手" || result.Temperature != temperature || result.EnableKnowledge || result.ContextWindowTokens != window || result.ContextKeepRecentTokens != recent {
		t.Fatalf("CreateAgentConfig(overrides) result = %#v, want submitted overrides", result)
	}

	target := 0.9
	if _, err := logic.CreateAgentConfig(uuid.New(), &request.CreateAgentConfigRequest{AgentConfigFieldsRequest: request.AgentConfigFieldsRequest{ContextTargetRatio: &target}}); err == nil {
		t.Fatal("CreateAgentConfig(invalid ratios) error = nil, want validation error")
	}
}

// TestListAndGetAgentConfigsUseUserAndIDScope 验证列表和详情只返回当前用户拥有的配置且列表为空时不会隐式创建。
func TestListAndGetAgentConfigsUseUserAndIDScope(t *testing.T) {
	repo := newAgentConfigTestRepository()
	userA, userB := uuid.New(), uuid.New()
	rowA, _ := repo.Create(context.Background(), userA, defaultAgentConfig())
	_, _ = repo.Create(context.Background(), userB, defaultAgentConfig())
	svcCtx := &svc.ServiceContext{AgentConfigRepo: repo}

	list, err := NewListAgentConfigsLogic(context.Background(), svcCtx).ListAgentConfigs(userA)
	if err != nil || len(list.List) != 1 || list.List[0].ID != rowA.ID {
		t.Fatalf("ListAgentConfigs(userA) = %#v, error=%v, want one owned config", list, err)
	}
	detail, err := NewGetAgentConfigLogic(context.Background(), svcCtx).GetAgentConfig(userA, &request.UriAgentConfigIDRequest{AgentConfigID: rowA.ID.String()})
	if err != nil || detail.ID != rowA.ID {
		t.Fatalf("GetAgentConfig(owner) = %#v, error=%v, want owned config", detail, err)
	}
	if _, err := NewGetAgentConfigLogic(context.Background(), svcCtx).GetAgentConfig(userB, &request.UriAgentConfigIDRequest{AgentConfigID: rowA.ID.String()}); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("GetAgentConfig(other user) error = %v, want not found", err)
	}

	empty, err := NewListAgentConfigsLogic(context.Background(), svcCtx).ListAgentConfigs(uuid.New())
	if err != nil || len(empty.List) != 0 || len(repo.rows) != 2 {
		t.Fatalf("ListAgentConfigs(empty) = %#v, error=%v rows=%d, want empty without create", empty, err, len(repo.rows))
	}
}

// TestUpdateAgentConfigUsesIDAndRejectsInvalidOrEmptyPatch 验证更新按 ID 定位、持久化选定字段并拒绝无效或空 patch。
func TestUpdateAgentConfigUsesIDAndRejectsInvalidOrEmptyPatch(t *testing.T) {
	repo := newAgentConfigTestRepository()
	userID := uuid.New()
	row, _ := repo.Create(context.Background(), userID, defaultAgentConfig())
	logic := NewUpdateAgentConfigLogic(context.Background(), &svc.ServiceContext{AgentConfigRepo: repo})
	name, window, recent := "技术助手", 65536, 12000

	result, err := logic.UpdateAgentConfig(userID, &request.UpdateAgentConfigRequest{
		UriAgentConfigIDRequest:  request.UriAgentConfigIDRequest{AgentConfigID: row.ID.String()},
		AgentConfigFieldsRequest: request.AgentConfigFieldsRequest{Name: &name, ContextWindowTokens: &window, ContextKeepRecentTokens: &recent},
	})
	if err != nil || repo.updates != 1 || result.Name != name || result.ContextWindowTokens != window || result.ContextKeepRecentTokens != recent {
		t.Fatalf("UpdateAgentConfig(valid) result=%#v error=%v updates=%d, want persisted fields", result, err, repo.updates)
	}

	target := 0.9
	if _, err := logic.UpdateAgentConfig(userID, &request.UpdateAgentConfigRequest{
		UriAgentConfigIDRequest:  request.UriAgentConfigIDRequest{AgentConfigID: row.ID.String()},
		AgentConfigFieldsRequest: request.AgentConfigFieldsRequest{ContextTargetRatio: &target},
	}); err == nil {
		t.Fatal("UpdateAgentConfig(invalid ratios) error = nil, want validation error")
	}
	if _, err := logic.UpdateAgentConfig(userID, &request.UpdateAgentConfigRequest{UriAgentConfigIDRequest: request.UriAgentConfigIDRequest{AgentConfigID: row.ID.String()}}); xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("UpdateAgentConfig(empty patch) error = %v, want bad request", err)
	}
	blankName := "   "
	if _, err := logic.UpdateAgentConfig(userID, &request.UpdateAgentConfigRequest{
		UriAgentConfigIDRequest:  request.UriAgentConfigIDRequest{AgentConfigID: row.ID.String()},
		AgentConfigFieldsRequest: request.AgentConfigFieldsRequest{Name: &blankName},
	}); xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("UpdateAgentConfig(blank name) error = %v, want bad request", err)
	}
}

// TestDeleteAgentConfigUsesUserAndIDScope 验证删除只能移除当前用户拥有的指定配置。
func TestDeleteAgentConfigUsesUserAndIDScope(t *testing.T) {
	repo := newAgentConfigTestRepository()
	owner := uuid.New()
	row, _ := repo.Create(context.Background(), owner, defaultAgentConfig())
	logic := NewDeleteAgentConfigLogic(context.Background(), &svc.ServiceContext{AgentConfigRepo: repo})
	input := &request.UriAgentConfigIDRequest{AgentConfigID: row.ID.String()}

	if err := logic.DeleteAgentConfig(uuid.New(), input); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("DeleteAgentConfig(other user) error = %v, want not found", err)
	}
	if err := logic.DeleteAgentConfig(owner, input); err != nil {
		t.Fatalf("DeleteAgentConfig(owner) error = %v, want nil", err)
	}
	if len(repo.rows) != 0 {
		t.Fatalf("DeleteAgentConfig(owner) rows = %d, want 0", len(repo.rows))
	}
}

// TestSetDefaultAgentConfigSwitchesUniqueDefault 验证设置默认配置会清除旧默认项，并且响应返回新的默认配置。
func TestSetDefaultAgentConfigSwitchesUniqueDefault(t *testing.T) {
	repo := newAgentConfigTestRepository()
	userID := uuid.New()
	first, _ := repo.Create(context.Background(), userID, defaultAgentConfig())
	second, _ := repo.Create(context.Background(), userID, defaultAgentConfig())
	logic := NewSetDefaultAgentConfigLogic(context.Background(), &svc.ServiceContext{AgentConfigRepo: repo})

	result, err := logic.SetDefaultAgentConfig(userID, &request.UriAgentConfigIDRequest{AgentConfigID: second.ID.String()})
	if err != nil {
		t.Fatalf("SetDefaultAgentConfig() error = %v, want nil", err)
	}
	if !result.IsDefault || repo.rows[first.ID].IsDefault || !repo.rows[second.ID].IsDefault {
		t.Fatalf("SetDefaultAgentConfig() result=%#v first=%v second=%v, want only second default", result, repo.rows[first.ID].IsDefault, repo.rows[second.ID].IsDefault)
	}
}

type agentConfigTestRepository struct {
	rows    map[uuid.UUID]*models.AgentConfig
	updates int
}

func newAgentConfigTestRepository() *agentConfigTestRepository {
	return &agentConfigTestRepository{rows: make(map[uuid.UUID]*models.AgentConfig)}
}

func (r *agentConfigTestRepository) Create(_ context.Context, userID uuid.UUID, row *models.AgentConfig) (*models.AgentConfig, error) {
	cloned := *row
	if cloned.ID == uuid.Nil {
		cloned.ID = uuid.New()
	}
	cloned.UserID = userID
	cloned.Name = strings.TrimSpace(cloned.Name)
	if cloned.Name == "" {
		count := r.countForUser(userID)
		if count == 0 {
			cloned.Name = "默认配置"
		} else {
			cloned.Name = fmt.Sprintf("智能体配置 %d", count+1)
		}
	}
	if r.nameExists(userID, cloned.Name, uuid.Nil) {
		return nil, xerr.Conflict("智能体配置名称已存在")
	}
	cloned.IsDefault = r.defaultForUser(userID) == nil
	r.rows[cloned.ID] = &cloned
	return &cloned, nil
}

func (r *agentConfigTestRepository) List(_ context.Context, userID uuid.UUID) ([]*models.AgentConfig, error) {
	out := make([]*models.AgentConfig, 0)
	for _, row := range r.rows {
		if row.UserID == userID {
			cloned := *row
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *agentConfigTestRepository) FindByID(_ context.Context, userID uuid.UUID, id uuid.UUID) (*models.AgentConfig, error) {
	row := r.rows[id]
	if row == nil || row.UserID != userID {
		return nil, xerr.NotFound("智能体配置不存在")
	}
	cloned := *row
	return &cloned, nil
}

func (r *agentConfigTestRepository) FindDefault(_ context.Context, userID uuid.UUID) (*models.AgentConfig, error) {
	row := r.defaultForUser(userID)
	if row == nil {
		return nil, xerr.NotFound("默认智能体配置不存在")
	}
	cloned := *row
	return &cloned, nil
}

func (r *agentConfigTestRepository) SetDefault(_ context.Context, userID uuid.UUID, id uuid.UUID) (*models.AgentConfig, error) {
	row := r.rows[id]
	if row == nil || row.UserID != userID {
		return nil, xerr.NotFound("智能体配置不存在")
	}
	for _, candidate := range r.rows {
		if candidate.UserID == userID {
			candidate.IsDefault = candidate.ID == id
		}
	}
	cloned := *row
	return &cloned, nil
}

func (r *agentConfigTestRepository) Update(_ context.Context, userID uuid.UUID, row *models.AgentConfig) (*models.AgentConfig, error) {
	if _, err := r.FindByID(context.Background(), userID, row.ID); err != nil {
		return nil, err
	}
	cloned := *row
	r.rows[row.ID] = &cloned
	return &cloned, nil
}

func (r *agentConfigTestRepository) UpdateFields(_ context.Context, userID uuid.UUID, id uuid.UUID, row *models.AgentConfig, fields *repository.AgentConfigUpdateFields) (*models.AgentConfig, error) {
	if len(fields.Columns()) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	if _, err := r.FindByID(context.Background(), userID, id); err != nil {
		return nil, err
	}
	if r.nameExists(userID, row.Name, id) {
		return nil, xerr.Conflict("智能体配置名称已存在")
	}
	cloned := *row
	cloned.ID, cloned.UserID = id, userID
	r.rows[id] = &cloned
	r.updates++
	return &cloned, nil
}

func (r *agentConfigTestRepository) Delete(_ context.Context, userID uuid.UUID, id uuid.UUID) error {
	row, err := r.FindByID(context.Background(), userID, id)
	if err != nil {
		return err
	}
	delete(r.rows, id)
	if row.IsDefault {
		for _, candidate := range r.rows {
			if candidate.UserID == userID {
				candidate.IsDefault = true
				break
			}
		}
	}
	return nil
}

func (r *agentConfigTestRepository) defaultForUser(userID uuid.UUID) *models.AgentConfig {
	for _, row := range r.rows {
		if row.UserID == userID && row.IsDefault {
			return row
		}
	}
	return nil
}

func (r *agentConfigTestRepository) countForUser(userID uuid.UUID) int {
	count := 0
	for _, row := range r.rows {
		if row.UserID == userID {
			count++
		}
	}
	return count
}

func (r *agentConfigTestRepository) nameExists(userID uuid.UUID, name string, exceptID uuid.UUID) bool {
	for _, row := range r.rows {
		if row.UserID == userID && row.ID != exceptID && row.Name == name {
			return true
		}
	}
	return false
}
