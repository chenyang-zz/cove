package knowledgebase

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// DeleteKnowledgeBaseLogic contains the deleteKnowledgeBase use case.
type DeleteKnowledgeBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteKnowledgeBaseLogic creates a DeleteKnowledgeBaseLogic.
func NewDeleteKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKnowledgeBaseLogic {
	return &DeleteKnowledgeBaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.knowledgebase.deleteknowledgebase"),
	}
}

// DeleteKnowledgeBase 删除知识库
func (l *DeleteKnowledgeBaseLogic) DeleteKnowledgeBase(userID uuid.UUID, input *request.UriKnowledgeBaseIDRequest) error {
	knowledgeBaseID, err := knowledgebaseIDFromInput(input)
	if err != nil {
		return err
	}
	row, err := l.svcCtx.KnowledgeBaseRepo.FindByID(l.ctx, userID, knowledgeBaseID)
	if err != nil {
		return err
	}
	if row.IsDefault {
		return xerr.BadRequest("默认知识库不可删除")
	}

	// TODO 删除 配套ES Chunk 以及OSS文件

	if err := l.svcCtx.KnowledgeBaseRepo.Delete(l.ctx, userID, knowledgeBaseID); err != nil {
		return err
	}
	l.log.InfoContext(l.ctx, "删除知识库",
		slog.String("user_id", userID.String()),
		slog.String("knowledge_base_id", knowledgeBaseID.String()),
	)
	return nil
}
