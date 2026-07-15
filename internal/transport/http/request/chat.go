package request

type ChatAttachment struct {
	FileName string `json:"file_name" binding:"required,max=256"`
	Text     string `json:"text" binding:"omitempty"`
}

type ChatStreamRequest struct {
	ConversationID string `json:"conversation_id" binding:"omitempty"`
	AgentConfigID  string `json:"agent_config_id" binding:"omitempty,uuid"`
	Message        string `json:"message" binding:"required,min=1"`
	// AI 主动开场白（今日回顾「聊聊」带入）：新会话首轮时先作为 assistant 消息落库，
	// 使其进入对话历史，模型能接住这个话题。仅 conversation_id 为空（新会话）时生效
	Greeting string `json:"greeting" binding:"omitempty"`
	// 本轮挂载的技能 id（任务能力包，override 提示词/工具白名单/知识库范围）
	SkillID string `json:"skill_id" binding:"omitempty"`
	// 多模态：图片 file_key 列表
	ImageKeys []string `json:"image_keys" binding:"omitempty"`
	// 对话临时附件（文档文本），仅本次对话上下文使用，不入库
	Attachments     []*ChatAttachment `json:"attachments"`
	EnableKnowledge *bool             `json:"enable_knowledge" binding:"omitempty"`
	EnableMemory    bool              `json:"enable_memory" binding:"omitempty"`
	EnableWebSearch bool              `json:"enable_web_search" binding:"omitempty"`
}
