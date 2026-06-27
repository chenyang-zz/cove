package conversation

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// ListMessagesLogic contains the listMessages use case.
type ListMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListMessagesLogic creates a ListMessagesLogic.
func NewListMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMessagesLogic {
	return &ListMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.conversation.listmessages"),
	}
}

// ListMessages 获取消息列表
func (l *ListMessagesLogic) ListMessages(userID uuid.UUID, input *request.UriConversationIDRequest) (*response.ListResponse[*response.MessageResponse], error) {
	conversationID, err := conversationIDFromInput(input)
	if err != nil {
		return nil, err
	}

	messages, err := l.svcCtx.MessageRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}

	// 带上当前用户对各消息的反馈（赞/踩），供前端高亮
	feedbacks, err := l.svcCtx.MessageFeedbackRepo.ListByConversationID(l.ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	rateByMsgID := make(map[uuid.UUID]string, len(feedbacks))
	for _, feedback := range feedbacks {
		rateByMsgID[feedback.MessageID] = feedback.Rating
	}

	// user 消息里存的图片 key 转成可访问 url（历史还原图片显示）
	imagesMap := make(map[uuid.UUID][]string, len(messages))
	needSignerUrl := l.svcCtx.URLSigner != nil
	for _, message := range messages {
		if message.MetaData.ImageKeys == nil || len(message.MetaData.ImageKeys) == 0 {
			continue
		}
		images := make([]string, 0, len(message.MetaData.ImageKeys))
		if needSignerUrl {
			for _, imageKey := range message.MetaData.ImageKeys {
				images = append(images, l.svcCtx.URLSigner.URL(imageKey))
			}
		} else {
			images = append(images, message.MetaData.ImageKeys...)
		}

		imagesMap[message.ID] = images
	}

	return mapper.MessagesToListResponse(messages, imagesMap, rateByMsgID), nil
}
