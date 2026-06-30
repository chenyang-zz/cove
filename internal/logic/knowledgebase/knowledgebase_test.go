package knowledgebase

import (
	"context"
	"reflect"
	"testing"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

type fakeKnowledgeBaseRepository struct {
	rows      map[uuid.UUID]*models.KnowledgeBase
	created   *models.KnowledgeBase
	deletedID uuid.UUID
	partial   *models.KnowledgeBase
	fields    []string
	updateID  uuid.UUID
}

func newFakeKnowledgeBaseRepository(rows ...*models.KnowledgeBase) *fakeKnowledgeBaseRepository {
	repo := &fakeKnowledgeBaseRepository{rows: map[uuid.UUID]*models.KnowledgeBase{}}
	for _, row := range rows {
		repo.rows[row.ID] = row
	}
	return repo
}

func (r *fakeKnowledgeBaseRepository) Create(ctx context.Context, userID uuid.UUID, row *models.KnowledgeBase) (*models.KnowledgeBase, error) {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	row.UserID = userID
	r.created = row
	r.rows[row.ID] = row
	return row, nil
}

func (r *fakeKnowledgeBaseRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.KnowledgeBase, error) {
	out := make([]*models.KnowledgeBase, 0, len(r.rows))
	for _, row := range r.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *fakeKnowledgeBaseRepository) FindByID(ctx context.Context, userID uuid.UUID, knowledgeBaseID uuid.UUID) (*models.KnowledgeBase, error) {
	row, ok := r.rows[knowledgeBaseID]
	if !ok || row.UserID != userID {
		return nil, xerr.NotFound("知识库不存在")
	}
	return row, nil
}

func (r *fakeKnowledgeBaseRepository) Update(ctx context.Context, userID uuid.UUID, row *models.KnowledgeBase) (*models.KnowledgeBase, error) {
	r.rows[row.ID] = row
	return row, nil
}

func (r *fakeKnowledgeBaseRepository) UpdateFields(ctx context.Context, userID uuid.UUID, knowledgeBaseID uuid.UUID, row *models.KnowledgeBase, fields *repository.KnowledgeBaseUpdateFields) (*models.KnowledgeBase, error) {
	r.updateID = knowledgeBaseID
	r.partial = row
	r.fields = fields.Columns()
	if len(r.fields) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	existing, err := r.FindByID(ctx, userID, knowledgeBaseID)
	if err != nil {
		return nil, err
	}
	for _, column := range r.fields {
		switch column {
		case "name":
			existing.Name = row.Name
		case "description":
			existing.Description = row.Description
		case "icon":
			existing.Icon = row.Icon
		case "color":
			existing.Color = row.Color
		case "chat_enabled":
			existing.ChatEnabled = row.ChatEnabled
		}
	}
	return existing, nil
}

func (r *fakeKnowledgeBaseRepository) Delete(ctx context.Context, userID uuid.UUID, knowledgeBaseID uuid.UUID) error {
	row, ok := r.rows[knowledgeBaseID]
	if !ok || row.UserID != userID {
		return xerr.NotFound("知识库不存在")
	}
	r.deletedID = knowledgeBaseID
	delete(r.rows, knowledgeBaseID)
	return nil
}

func TestCreateKnowledgeBasePersistsDisplayFields(t *testing.T) {
	// 验证创建知识库会保存展示字段，并且新建知识库默认不参与聊天检索。
	ctx := context.Background()
	userID := uuid.New()
	repo := newFakeKnowledgeBaseRepository()
	logic := NewCreateKnowledgeBaseLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo})

	out, err := logic.CreateKnowledgeBase(userID, &request.CreateKnowledgeBaseRequest{
		Name:        "  产品资料  ",
		Description: "内部材料",
		Icon:        "book",
		Color:       "#22c55e",
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase error = %v", err)
	}
	if repo.created == nil {
		t.Fatal("repository Create was not called")
	}
	if repo.created.UserID != userID {
		t.Fatalf("created user id = %s, want %s", repo.created.UserID, userID)
	}
	if repo.created.Name != "产品资料" || repo.created.Description != "内部材料" || repo.created.Icon != "book" || repo.created.Color != "#22c55e" {
		t.Fatalf("created row = %+v, want display fields persisted", repo.created)
	}
	if repo.created.IsDefault || repo.created.ChatEnabled {
		t.Fatalf("created flags = default:%v chat:%v, want false/false", repo.created.IsDefault, repo.created.ChatEnabled)
	}
	if out.ID != repo.created.ID || out.Color != "#22c55e" || out.DocCount != 0 || out.ImageCount != 0 {
		t.Fatalf("response = %+v, want created row with zero counts", out)
	}
}

func TestGetAndListKnowledgeBaseMapRows(t *testing.T) {
	// 验证详情和列表接口会按用户隔离读取，并把模型转换为响应结构。
	ctx := context.Background()
	userID := uuid.New()
	row := &models.KnowledgeBase{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        "默认库",
		Description: "默认描述",
		Icon:        "home",
		Color:       "#0ea5e9",
		IsDefault:   true,
		ChatEnabled: true,
	}
	repo := newFakeKnowledgeBaseRepository(row, &models.KnowledgeBase{ID: uuid.New(), UserID: uuid.New(), Name: "other"})

	got, err := NewGetKnowledgeBaseLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo}).GetKnowledgeBase(userID, &request.UriKnowledgeBaseIDRequest{KID: row.ID.String()})
	if err != nil {
		t.Fatalf("GetKnowledgeBase error = %v", err)
	}
	if got.ID != row.ID || got.Name != "默认库" || got.Color != "#0ea5e9" || !got.IsDefault || !got.ChatEnabled {
		t.Fatalf("detail response = %+v, want mapped row", got)
	}

	list, err := NewGetKnowledgeBaseListLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo}).GetKnowledgeBaseList(userID)
	if err != nil {
		t.Fatalf("GetKnowledgeBaseList error = %v", err)
	}
	if len(list.List) != 1 || list.List[0].ID != row.ID {
		t.Fatalf("list = %+v, want only current user's row", list.List)
	}
}

func TestGetKnowledgeBaseListCreatesDefaultWhenMissing(t *testing.T) {
	// 验证列表接口在当前用户没有默认知识库时会自动创建默认知识库。
	ctx := context.Background()
	userID := uuid.New()
	existing := &models.KnowledgeBase{ID: uuid.New(), UserID: userID, Name: "项目资料", Color: "#000000"}
	repo := newFakeKnowledgeBaseRepository(existing)

	list, err := NewGetKnowledgeBaseListLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo}).GetKnowledgeBaseList(userID)
	if err != nil {
		t.Fatalf("GetKnowledgeBaseList error = %v", err)
	}
	if repo.created == nil {
		t.Fatal("default knowledge base was not created")
	}
	if repo.created.UserID != userID || repo.created.Name != "默认知识库" || repo.created.Description != "未分类资料默认归入此库" ||
		repo.created.Icon != "📚" || repo.created.Color != "#155EEF" || !repo.created.IsDefault || !repo.created.ChatEnabled {
		t.Fatalf("created default = %+v, want configured default knowledge base", repo.created)
	}
	if len(list.List) != 2 {
		t.Fatalf("list len = %d, want 2", len(list.List))
	}
	foundDefault := false
	for _, item := range list.List {
		if item.ID == repo.created.ID && item.IsDefault && item.ChatEnabled && item.Color == "#155EEF" {
			foundDefault = true
		}
	}
	if !foundDefault {
		t.Fatalf("list = %+v, want created default response", list.List)
	}
}

func TestGetKnowledgeBaseListDoesNotDuplicateDefault(t *testing.T) {
	// 验证已有默认知识库时列表接口不会重复创建默认库。
	ctx := context.Background()
	userID := uuid.New()
	defaultRow := &models.KnowledgeBase{ID: uuid.New(), UserID: userID, Name: "默认知识库", IsDefault: true, ChatEnabled: true}
	repo := newFakeKnowledgeBaseRepository(defaultRow)

	list, err := NewGetKnowledgeBaseListLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo}).GetKnowledgeBaseList(userID)
	if err != nil {
		t.Fatalf("GetKnowledgeBaseList error = %v", err)
	}
	if repo.created != nil {
		t.Fatalf("created = %+v, want no duplicate default", repo.created)
	}
	if len(list.List) != 1 || list.List[0].ID != defaultRow.ID {
		t.Fatalf("list = %+v, want existing default only", list.List)
	}
}

func TestUpdateKnowledgeBaseSendsOnlyChangedFields(t *testing.T) {
	// 验证更新知识库只提交请求中出现的字段，包含 color 展示字段。
	ctx := context.Background()
	userID := uuid.New()
	row := &models.KnowledgeBase{ID: uuid.New(), UserID: userID, Name: "旧名称", Color: "#000000"}
	repo := newFakeKnowledgeBaseRepository(row)
	name := "新名称"
	color := "#f97316"
	logic := NewUpdateKnowledgeBaseLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo})

	out, err := logic.UpdateKnowledgeBase(userID, &request.UpdateKnowledgeBaseRequest{
		UriKnowledgeBaseIDRequest: request.UriKnowledgeBaseIDRequest{KID: row.ID.String()},
		Name:                      &name,
		Color:                     &color,
	})
	if err != nil {
		t.Fatalf("UpdateKnowledgeBase error = %v", err)
	}
	if repo.updateID != row.ID {
		t.Fatalf("update id = %s, want %s", repo.updateID, row.ID)
	}
	if !reflect.DeepEqual(repo.fields, []string{"name", "color"}) {
		t.Fatalf("updated fields = %#v, want name/color", repo.fields)
	}
	if repo.partial.Name != "新名称" || repo.partial.Color != "#f97316" {
		t.Fatalf("partial row = %+v, want patched fields", repo.partial)
	}
	if out.Name != "新名称" || out.Color != "#f97316" {
		t.Fatalf("response = %+v, want updated fields", out)
	}
}

func TestDeleteKnowledgeBaseRejectsDefaultAndDeletesNormal(t *testing.T) {
	// 验证默认知识库不可删除，普通知识库会调用仓储删除。
	ctx := context.Background()
	userID := uuid.New()
	defaultID := uuid.New()
	normalID := uuid.New()
	repo := newFakeKnowledgeBaseRepository(
		&models.KnowledgeBase{ID: defaultID, UserID: userID, IsDefault: true},
		&models.KnowledgeBase{ID: normalID, UserID: userID},
	)
	logic := NewDeleteKnowledgeBaseLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo})

	err := logic.DeleteKnowledgeBase(userID, &request.UriKnowledgeBaseIDRequest{KID: defaultID.String()})
	if err == nil || xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("delete default error = %v, want bad request", err)
	}
	if err := logic.DeleteKnowledgeBase(userID, &request.UriKnowledgeBaseIDRequest{KID: normalID.String()}); err != nil {
		t.Fatalf("DeleteKnowledgeBase normal error = %v", err)
	}
	if repo.deletedID != normalID {
		t.Fatalf("deleted id = %s, want %s", repo.deletedID, normalID)
	}
}

func TestEnabledChatAcceptsFalse(t *testing.T) {
	// 验证聊天开关允许显式关闭，false 不会被当作缺失字段，并返回更新后的响应。
	ctx := context.Background()
	userID := uuid.New()
	row := &models.KnowledgeBase{ID: uuid.New(), UserID: userID, Name: "资料库", ChatEnabled: true}
	repo := newFakeKnowledgeBaseRepository(row)
	disabled := false
	logic := NewEnabledChatLogic(ctx, &svc.ServiceContext{KnowledgeBaseRepo: repo})

	out, err := logic.EnabledChat(userID, &request.EnabledChatRequest{
		UriKnowledgeBaseIDRequest: request.UriKnowledgeBaseIDRequest{KID: row.ID.String()},
		ChatEnabled:               &disabled,
	})
	if err != nil {
		t.Fatalf("EnabledChat false error = %v", err)
	}
	if !reflect.DeepEqual(repo.fields, []string{"chat_enabled"}) {
		t.Fatalf("updated fields = %#v, want chat_enabled", repo.fields)
	}
	if row.ChatEnabled {
		t.Fatal("ChatEnabled remained true, want false")
	}
	if out == nil || out.ID != row.ID || out.Name != "资料库" || out.ChatEnabled {
		t.Fatalf("response = %+v, want updated knowledge base with chat disabled", out)
	}
}
