package chat

import (
	"context"
	"errors"
	"strings"

	"github.com/boxify/api-go/internal/core/llm"
	flow "github.com/boxify/api-go/internal/domain/flow"
	flowchat "github.com/boxify/api-go/internal/domain/flow/chat"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// TurnInput 是 HTTP SSE 与外部渠道共同使用的单轮聊天输入。
type TurnInput struct {
	UserID                uuid.UUID
	ConversationID        uuid.UUID
	CurrentUserMessageID  uuid.UUID
	Message               string
	Attachments           []*types.MessageAttachment
	AgentConfigID         *uuid.UUID
	EnableKnowledge       *bool
	ToolPolicy            string
	AssistantMessageID    *uuid.UUID
	DiscardPartialOnError bool
}

// TurnResult 返回已持久化的最终 Assistant 消息。
type TurnResult struct {
	AssistantMessage *models.Message
}

// TurnObserver 接收 token、工具和状态事件；不参与消息持久化。
type TurnObserver func(flow.Message)

// RunTurn 执行渠道无关聊天回合，并统一持久化最终或中断的 Assistant 消息。
func RunTurn(ctx context.Context, svcCtx *svc.ServiceContext, input TurnInput, observer TurnObserver) (TurnResult, error) {
	if svcCtx == nil || svcCtx.ConversationRepo == nil || svcCtx.MessageRepo == nil {
		return TurnResult{}, xerr.Internal("聊天回合依赖未初始化", nil)
	}
	if _, err := svcCtx.ConversationRepo.FindByID(ctx, input.UserID, input.ConversationID); err != nil {
		return TurnResult{}, err
	}
	agentConfig, err := turnAgentConfig(ctx, svcCtx, input.UserID, input.AgentConfigID)
	if err != nil {
		return TurnResult{}, err
	}
	persona, err := turnActivePersona(ctx, svcCtx, input.UserID)
	if err != nil {
		return TurnResult{}, err
	}
	runtimeConfig := resolveTurnRuntimeConfig(input.EnableKnowledge, agentConfig, persona)
	messages, err := flowchat.NewOrchestrator(svcCtx).Run(ctx, flowchat.Input{
		UserID: input.UserID, ConversationID: input.ConversationID, CurrentUserMessageID: input.CurrentUserMessageID,
		Message: input.Message, Attachments: input.Attachments, EnableKnowledge: runtimeConfig.EnableKnowledge,
		Temperature: runtimeConfig.Temperature, SystemPrompt: runtimeConfig.SystemPrompt,
		ContextPolicy: runtimeConfig.ContextPolicy, ToolPolicy: input.ToolPolicy,
	})
	if err != nil {
		return TurnResult{}, err
	}
	parts := make([]models.MessagePart, 0, 8)
	result := TurnResult{}
	for message := range messages {
		switch item := message.(type) {
		case *flow.AssistantMessage:
			answer := strings.TrimSpace(item.Answer)
			parts = finalizePartsWithAnswer(parts, answer)
			assistantRow := &models.Message{
				ConversationID: input.ConversationID, Role: string(llm.AssistantRole), Content: answer,
				MetaData: buildAssistantMeta(parts, false),
			}
			if input.AssistantMessageID != nil {
				assistantRow.ID = *input.AssistantMessageID
			}
			assistant, saveErr := svcCtx.MessageRepo.Create(ctx, input.UserID, assistantRow)
			if saveErr != nil {
				return result, saveErr
			}
			result.AssistantMessage = assistant
		case *flow.ErrorMessage:
			partial := strings.TrimSpace(item.Partial)
			if !input.DiscardPartialOnError {
				if partial != "" {
					parts = finalizePartsWithAnswer(parts, partial)
					_, _ = svcCtx.MessageRepo.Create(ctx, input.UserID, &models.Message{
						ConversationID: input.ConversationID, Role: string(llm.AssistantRole), Content: partial,
						MetaData: buildAssistantMeta(parts, true),
					})
				} else if len(parts) > 0 {
					_, _ = svcCtx.MessageRepo.Create(ctx, input.UserID, &models.Message{
						ConversationID: input.ConversationID, Role: string(llm.AssistantRole), Content: "",
						MetaData: buildAssistantMeta(parts, true),
					})
				}
			}
			if observer != nil {
				observer(message)
			}
			if item.Err != nil {
				return result, item.Err
			}
			return result, errors.New(strings.TrimSpace(item.Message))
		case *flow.ToolCallMessage:
			if item != nil {
				parts = appendToolCallPart(parts, item.Tool, item.Input, item.Iteration, item.ToolCallID)
			}
		case *flow.ToolResultMessage:
			if item != nil {
				parts = appendToolResultPart(parts, item.Tool, item.Input, truncateObservation(item.Observation), item.Error, item.Iteration, item.ToolCallID)
			}
		case *flow.PartialMessage:
			if item != nil && item.Text != "" {
				parts = appendTextPart(parts, item.Text)
			}
		}
		if observer != nil {
			observer(message)
		}
	}
	if result.AssistantMessage == nil {
		return result, errors.New("聊天回合未生成最终回复")
	}
	return result, nil
}

func turnAgentConfig(ctx context.Context, svcCtx *svc.ServiceContext, userID uuid.UUID, configID *uuid.UUID) (*models.AgentConfig, error) {
	if svcCtx.AgentConfigRepo == nil {
		return nil, nil
	}
	if configID != nil {
		return svcCtx.AgentConfigRepo.FindByID(ctx, userID, *configID)
	}
	config, err := svcCtx.AgentConfigRepo.FindDefault(ctx, userID)
	if err != nil && xerr.From(err).Kind == xerr.KindNotFound {
		return nil, nil
	}
	return config, err
}

func turnActivePersona(ctx context.Context, svcCtx *svc.ServiceContext, userID uuid.UUID) (*models.AgentPersona, error) {
	if svcCtx.AgentPersonaRepo == nil {
		return nil, nil
	}
	return svcCtx.AgentPersonaRepo.FindActive(ctx, userID)
}

func resolveTurnRuntimeConfig(enableKnowledge *bool, agentConfig *models.AgentConfig, persona *models.AgentPersona) chatRuntimeConfig {
	config := chatRuntimeConfig{
		Temperature:   defaultChatTemperature,
		ContextPolicy: mapper.AgentConfigToContextPolicy(agentConfig),
	}
	if agentConfig != nil {
		if agentConfig.Temperature > 0 {
			config.Temperature = agentConfig.Temperature
		}
		config.EnableKnowledge = agentConfig.EnableKnowledge
	}
	if enableKnowledge != nil {
		config.EnableKnowledge = *enableKnowledge
	}
	agentPrompt, soul, identity := "", "", ""
	if agentConfig != nil {
		agentPrompt = strings.TrimSpace(agentConfig.SystemPrompt)
	}
	if persona != nil {
		soul, identity = persona.Soul, persona.Identity
	}
	config.SystemPrompt = buildChatSystemPrompt(soul, identity, agentPrompt)
	return config
}
