/**
 * @Time   : 2026/6/30 18:00
 * @Author : chenyangzhao542@gmail.com
 * @File   : knowledgebase.go
 **/

package request

type CreateKnowledgeBaseRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=128"`
	Description string `json:"description" binding:"omitempty,max=512"`
	Icon        string `json:"icon" binding:"omitempty,max=32"`
	Color       string `json:"color" binding:"omitempty,max=16"`
}

type UriKnowledgeBaseIDRequest struct {
	KID string `uri:"k_id" binding:"required,uuid"`
}
type UpdateKnowledgeBaseRequest struct {
	UriKnowledgeBaseIDRequest
	Name        *string `json:"name" binding:"omitempty,min=1,max=128"`
	Description *string `json:"description" binding:"omitempty,max=512"`
	Icon        *string `json:"icon" binding:"omitempty,max=32"`
	Color       *string `json:"color" binding:"omitempty,max=16"`
}

type EnabledChatRequest struct {
	UriKnowledgeBaseIDRequest
	ChatEnabled *bool `json:"chat_enabled" binding:"required"`
}
