package types

import (
	"fmt"

	"github.com/google/uuid"
)

type TaskName string

type QueueName string

const (
	TaskParseDocument     TaskName = "parse:document"
	TaskParseImage        TaskName = "parse:image"
	TaskMemoryExtract     TaskName = "memory:extract"
	TaskMemoryConsolidate TaskName = "memory:consolidate"
	TaskResearchRun       TaskName = "research:run"
	TaskGatewayTurn       TaskName = "gateway:turn"
	TaskGatewayDeliver    TaskName = "gateway:deliver"
)

const (
	QueueDefault  QueueName = "default"
	QueueParse    QueueName = "parse"
	QueueMemory   QueueName = "memory"
	QueueResearch QueueName = "research"
	QueueBeat     QueueName = "beat"
	QueueGateway  QueueName = "gateway"
)

type Task struct {
	Name    TaskName
	Queue   QueueName
	Payload any
}

type ParseDocumentPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	DocumentID uuid.UUID `json:"document_id"`
}

type ParseImagePayload struct {
	UserID  uuid.UUID `json:"user_id"`
	ImageID uuid.UUID `json:"image_id"`
}

// GatewayTurnPayload 标识一个已通过策略门控的入站事件。
type GatewayTurnPayload struct {
	InboxEventID uuid.UUID `json:"inbox_event_id"`
}

// GatewayDeliverPayload 标识一个待可靠投递的发件箱消息。
type GatewayDeliverPayload struct {
	OutboxMessageID uuid.UUID `json:"outbox_message_id"`
}

func TaskNames() []TaskName {
	return []TaskName{
		TaskParseDocument,
		TaskParseImage,
		TaskMemoryExtract,
		TaskMemoryConsolidate,
		TaskResearchRun,
		TaskGatewayTurn,
		TaskGatewayDeliver,
	}
}

// NewGatewayTurnTask 创建网关回合任务。
func NewGatewayTurnTask(inboxEventID uuid.UUID) (*Task, error) {
	if inboxEventID == uuid.Nil {
		return nil, fmt.Errorf("inbox_event_id is required")
	}
	return &Task{Name: TaskGatewayTurn, Queue: QueueGateway, Payload: &GatewayTurnPayload{InboxEventID: inboxEventID}}, nil
}

// NewGatewayDeliverTask 创建网关发件箱投递任务。
func NewGatewayDeliverTask(outboxMessageID uuid.UUID) (*Task, error) {
	if outboxMessageID == uuid.Nil {
		return nil, fmt.Errorf("outbox_message_id is required")
	}
	return &Task{Name: TaskGatewayDeliver, Queue: QueueGateway, Payload: &GatewayDeliverPayload{OutboxMessageID: outboxMessageID}}, nil
}

func NewParseDocumentTask(userID uuid.UUID, documentID uuid.UUID) (*Task, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if documentID == uuid.Nil {
		return nil, fmt.Errorf("document_id is required")
	}
	return &Task{
		Name:  TaskParseDocument,
		Queue: QueueParse,
		Payload: &ParseDocumentPayload{
			UserID:     userID,
			DocumentID: documentID,
		},
	}, nil
}

func NewParseImageTask(userID uuid.UUID, imageID uuid.UUID) (*Task, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if imageID == uuid.Nil {
		return nil, fmt.Errorf("image_id is required")
	}
	return &Task{
		Name:  TaskParseImage,
		Queue: QueueParse,
		Payload: &ParseImagePayload{
			UserID:  userID,
			ImageID: imageID,
		},
	}, nil
}
