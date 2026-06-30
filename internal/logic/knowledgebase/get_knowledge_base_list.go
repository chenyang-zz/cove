package knowledgebase

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

const (
	defaultKnowledgeBaseName        = "默认知识库"
	defaultKnowledgeBaseDescription = "未分类资料默认归入此库"
	defaultKnowledgeBaseIcon        = "📚"
	defaultKnowledgeBaseColor       = "#155EEF"
)

// GetKnowledgeBaseListLogic contains the getKnowledgeBaseList use case.
type GetKnowledgeBaseListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetKnowledgeBaseListLogic creates a GetKnowledgeBaseListLogic.
func NewGetKnowledgeBaseListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKnowledgeBaseListLogic {
	return &GetKnowledgeBaseListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.knowledgebase.getknowledgebaselist"),
	}
}

// GetKnowledgeBaseList 查询知识库列表
func (l *GetKnowledgeBaseListLogic) GetKnowledgeBaseList(userID uuid.UUID) (*response.ListResponse[*response.KnowledgeBaseResponse], error) {
	rows, err := l.svcCtx.KnowledgeBaseRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err = l.ensureDefaultKnowledgeBase(userID, rows)
	if err != nil {
		return nil, err
	}
	out := make([]*response.KnowledgeBaseResponse, 0, len(rows))

	// TODO 获取 doc和img count

	for _, row := range rows {
		out = append(out, mapper.KnowledgeBaseToResponse(row))
	}
	return &response.ListResponse[*response.KnowledgeBaseResponse]{List: out}, nil
}

func (l *GetKnowledgeBaseListLogic) ensureDefaultKnowledgeBase(userID uuid.UUID, rows []*models.KnowledgeBase) ([]*models.KnowledgeBase, error) {
	for _, row := range rows {
		if row != nil && row.IsDefault {
			return rows, nil
		}
	}
	row, err := l.svcCtx.KnowledgeBaseRepo.Create(l.ctx, userID, &models.KnowledgeBase{
		Name:        defaultKnowledgeBaseName,
		Description: defaultKnowledgeBaseDescription,
		Icon:        defaultKnowledgeBaseIcon,
		Color:       defaultKnowledgeBaseColor,
		IsDefault:   true,
		ChatEnabled: true,
	})
	if err != nil {
		return nil, err
	}
	l.log.InfoContext(l.ctx, "创建默认知识库",
		slog.String("user_id", userID.String()),
		slog.String("knowledge_base_id", row.ID.String()),
	)
	return append(rows, row), nil
}
