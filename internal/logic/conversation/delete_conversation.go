package conversation

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

// DeleteConversationLogic contains the deleteConversation use case.
type DeleteConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteConversationLogic creates a DeleteConversationLogic.
func NewDeleteConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConversationLogic {
	return &DeleteConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.conversation.deleteconversation"),
	}
}

// DeleteConversation 删除会话
func (l *DeleteConversationLogic) DeleteConversation(userID uuid.UUID, input *request.UriConversationIDRequest) error {
	conversationID, err := conversationIDFromInput(input)
	if err != nil {
		return err
	}
	if err := l.svcCtx.ConversationRepo.Delete(l.ctx, userID, conversationID); err != nil {
		return err
	}

	return nil
}
