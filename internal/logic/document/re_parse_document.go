package document

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// ReParseDocumentLogic contains the reParseDocument use case.
type ReParseDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewReParseDocumentLogic creates a ReParseDocumentLogic.
func NewReParseDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReParseDocumentLogic {
	return &ReParseDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.document.reparsedocument"),
	}
}

// ReParseDocument 重新提交文档解析
func (l *ReParseDocumentLogic) ReParseDocument(userID uuid.UUID, input *request.UriDocumentIDRequest) (*response.DocumentResponse, error) {
	documentID, err := parseDocumentID(input.DocumentID)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.DocumentRepo.UpdateFields(l.ctx, userID, documentID, &models.Document{
		Status:   domain.DocumentStatusPending,
		Progress: 0,
		ErrorMsg: nil,
	}, repository.NewDocumentUpdateFields().Status().Progress().ErrorMsg())
	if err != nil {
		return nil, err
	}
	l.log.InfoContext(l.ctx, "重新提交文档解析",
		slog.String("user_id", userID.String()),
		slog.String("document_id", documentID.String()),
	)
	if err := enqueueParseDocumentTask(l.ctx, l.svcCtx.TaskProducer, userID, documentID); err != nil {
		markDocumentParseDispatchFailed(l.ctx, l.svcCtx.DocumentRepo, userID, documentID, err)
		return nil, err
	}
	l.log.InfoContext(l.ctx, "文档解析任务已入队",
		slog.String("document_id", documentID.String()),
	)
	return mapper.DocumentToResponse(row, nil), nil
}
