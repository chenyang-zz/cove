/**
 * @Time   : 2026/6/27 15:56
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

func ConversationToResponse(row *models.Conversation) *response.ConversationResponse {
	if row == nil {
		return nil
	}
	res := &response.ConversationResponse{
		ID:              row.ID,
		Title:           row.Title,
		IsGroup:         row.IsGroup,
		MemberPersonIDs: row.MemberPersonIDs,
		EnableTools:     row.EnableTools,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	if res.MemberPersonIDs == nil {
		res.MemberPersonIDs = []string{}
	}

	return res
}

func ConversationsToListResponse(rows []*models.Conversation) *response.ListResponse[*response.ConversationResponse] {
	out := make([]*response.ConversationResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ConversationToResponse(row))
	}
	return &response.ListResponse[*response.ConversationResponse]{List: out}
}

func MessageToResponse(row *models.Message, imagesMap map[uuid.UUID][]string, ratingMap map[uuid.UUID]string) *response.MessageResponse {

	images := make([]string, 0)
	metadata := &response.MessageMetaData{
		ImageKeys:  row.MetaData.ImageKeys,
		SenderName: row.MetaData.SenderName,
	}
	if imgs, exist := imagesMap[row.ID]; exist {
		images = imgs
	}

	res := &response.MessageResponse{
		ID:        row.ID,
		Role:      row.Role,
		Content:   row.Content,
		MetaData:  metadata,
		Images:    images,
		CreatedAt: row.CreatedAt,
	}

	if row.SenderPersonID != uuid.Nil {
		res.SenderPersonID = &row.SenderPersonID
	}
	if metadata.SenderName != "" {
		res.SenderName = &row.MetaData.SenderName
	}
	if rating, exist := ratingMap[row.ID]; exist {
		res.Feedback = &rating
	}

	return res
}

func MessagesToListResponse(rows []*models.Message, imagesMap map[uuid.UUID][]string, ratingMap map[uuid.UUID]string) *response.ListResponse[*response.
	MessageResponse] {
	out := make([]*response.MessageResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, MessageToResponse(row, imagesMap, ratingMap))
	}
	return &response.ListResponse[*response.MessageResponse]{List: out}
}
